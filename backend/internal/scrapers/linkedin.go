package scrapers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/mrcodeeu/homepage/internal/models"
	"github.com/mrcodeeu/homepage/internal/storage"
	"github.com/pquerna/otp/totp"
)

const (
	cacheKeyLinkedIn        = "linkedin_data"
	cacheKeyLinkedInCookies = "linkedin_cookies"
	linkedInLoginURL        = "https://www.linkedin.com/login"
	linkedInTimeoutSec      = 120
)

// LinkedInScraper implements the Scraper interface for LinkedIn profiles using chromedp
type LinkedInScraper struct {
	email      string
	password   string
	totpSecret string
	profileURL string
	cache      storage.Cache
	cacheTTL   time.Duration
	headless   bool
}

// NewLinkedInScraper creates a new LinkedIn scraper with chromedp
func NewLinkedInScraper(email, password, totpSecret, profileURL string, cache storage.Cache) *LinkedInScraper {
	return &LinkedInScraper{
		email:      email,
		password:   password,
		totpSecret: totpSecret,
		profileURL: profileURL,
		cache:      cache,
		cacheTTL:   24 * time.Hour,
		headless:   true, // Always headless for CI/CD compatibility
	}
}

// Name returns the scraper name
func (l *LinkedInScraper) Name() string {
	return "linkedin"
}

// GetCached returns cached data or scrapes if needed
func (l *LinkedInScraper) GetCached() (any, error) {
	cached, err := l.cache.Get(cacheKeyLinkedIn)
	if err != nil {
		return nil, fmt.Errorf("cache error: %w", err)
	}

	if cached != nil {
		var data models.LinkedInData
		if err := json.Unmarshal(cached, &data); err != nil {
			return l.Refresh()
		}
		return data, nil
	}

	return l.Refresh()
}

// Scrape fetches fresh data from LinkedIn using chromedp
func (l *LinkedInScraper) Scrape() (any, error) {
	if l.email == "" || l.password == "" {
		return nil, fmt.Errorf("LinkedIn credentials not set (need LINKEDIN_EMAIL and LINKEDIN_PASSWORD)")
	}

	log.Println("Starting LinkedIn scraper with chromedp...")

	// Create chromedp context with options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", l.headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),            // Required for GitHub Actions
		chromedp.Flag("disable-dev-shm-usage", true), // Required for Docker/CI
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	// Set timeout
	ctx, cancel = context.WithTimeout(ctx, linkedInTimeoutSec*time.Second)
	defer cancel()

	// Try to restore cookies and check if already logged in
	cookiesRestored, err := l.restoreCookies(ctx)
	if err != nil {
		log.Printf("Warning: Failed to restore cookies: %v", err)
	}

	// Check if we're already logged in
	loggedIn := false
	if cookiesRestored {
		log.Println("Checking if already logged in with cached cookies...")
		loggedIn = l.isLoggedIn(ctx)
		if loggedIn {
			log.Println("Already logged in with cached cookies!")
		} else {
			log.Println("Cached cookies expired or invalid, will perform fresh login")
		}
	}

	// Login if not already logged in
	if !loggedIn {
		if err := l.login(ctx); err != nil {
			return nil, fmt.Errorf("login failed: %w", err)
		}
		log.Println("Login successful")

		// Save cookies for future use
		if err := l.saveCookies(ctx); err != nil {
			log.Printf("Warning: Failed to save cookies: %v", err)
		}
	}

	log.Println("Navigating to profile...")

	// Navigate to profile and extract data
	data, err := l.extractProfileData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to extract profile data: %w", err)
	}

	log.Println("Profile data extracted successfully")
	return data, nil
}

// Refresh forces a fresh scrape and updates cache
func (l *LinkedInScraper) Refresh() (any, error) {
	data, err := l.Scrape()
	if err != nil {
		return nil, err
	}

	// Update cache
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := l.cache.Set(cacheKeyLinkedIn, jsonData, l.cacheTTL); err != nil {
		log.Printf("Warning: failed to update cache: %v", err)
	}

	return data, nil
}

// login performs LinkedIn login with email and password
func (l *LinkedInScraper) login(ctx context.Context) error {
	log.Println("Navigating to LinkedIn login page...")

	// Navigate to login page and wait for it to load
	if err := chromedp.Run(ctx,
		chromedp.Navigate(linkedInLoginURL),
		chromedp.WaitVisible(`#username`, chromedp.ByID),
	); err != nil {
		return fmt.Errorf("failed to load login page: %w", err)
	}

	log.Println("Entering credentials...")

	// Enter credentials and submit
	if err := chromedp.Run(ctx,
		chromedp.SendKeys(`#username`, l.email, chromedp.ByID),
		chromedp.SendKeys(`#password`, l.password, chromedp.ByID),
		chromedp.Click(`button[type="submit"]`, chromedp.ByQuery),
	); err != nil {
		return fmt.Errorf("failed to submit login form: %w", err)
	}

	// Wait for login to complete - check for feed or profile elements
	log.Println("Waiting for login to complete...")

	// Wait for navigation away from login page
	if err := chromedp.Run(ctx,
		chromedp.WaitNotPresent(`#username`, chromedp.ByID),
	); err != nil {
		// Check if we're on a challenge page
		var currentURL string
		_ = chromedp.Run(ctx, chromedp.Location(&currentURL))
		if strings.Contains(currentURL, "challenge") || strings.Contains(currentURL, "checkpoint") {
			return fmt.Errorf("LinkedIn security challenge detected - manual verification may be required")
		}
		return fmt.Errorf("login may have failed: %w", err)
	}

	// Wait a moment for potential 2FA page to load
	time.Sleep(2 * time.Second)

	// Check if 2FA is required and handle it
	if err := l.handle2FA(ctx); err != nil {
		return fmt.Errorf("2FA handling failed: %w", err)
	}

	// Short pause to ensure login is fully processed
	time.Sleep(2 * time.Second)

	return nil
}

// handle2FA checks for and handles TOTP-based two-factor authentication
func (l *LinkedInScraper) handle2FA(ctx context.Context) error {
	// Check current URL to see if we're on a 2FA/verification page
	var currentURL string
	if err := chromedp.Run(ctx, chromedp.Location(&currentURL)); err != nil {
		return fmt.Errorf("failed to get current URL: %w", err)
	}

	log.Printf("Current URL after login: %s", currentURL)

	// Check if we're on a 2FA challenge page
	is2FAPage := strings.Contains(currentURL, "checkpoint") ||
		strings.Contains(currentURL, "challenge") ||
		strings.Contains(currentURL, "two-step-verification")

	if !is2FAPage {
		// Also check if there's a 2FA input field on the page
		var otpInputExists bool
		_ = chromedp.Run(ctx,
			chromedp.Evaluate(`document.querySelector('input[name="pin"]') !== null || 
				document.querySelector('input#input__phone_verification_pin') !== null ||
				document.querySelector('input[aria-label*="verification"]') !== null ||
				document.querySelector('input[aria-label*="code"]') !== null ||
				document.querySelector('input[type="tel"]') !== null`, &otpInputExists),
		)
		if !otpInputExists {
			log.Println("No 2FA required, proceeding...")
			return nil
		}
	}

	log.Println("2FA verification page detected, generating TOTP code...")

	// Check if we have TOTP secret configured
	if l.totpSecret == "" {
		return fmt.Errorf("2FA required but TOTP secret not configured (set LINKEDIN_TOTP_SECRET)")
	}

	// Generate TOTP code
	otpCode, err := totp.GenerateCode(l.totpSecret, time.Now())
	if err != nil {
		return fmt.Errorf("failed to generate TOTP code: %w", err)
	}

	log.Printf("Generated TOTP code: %s", otpCode)

	// Wait for the OTP input field to be visible
	// LinkedIn uses various selectors for the OTP input
	otpSelectors := []string{
		`input[name="pin"]`,
		`input#input__phone_verification_pin`,
		`input[aria-label*="verification"]`,
		`input[aria-label*="code"]`,
		`input[type="tel"]`,
		`input.verification-code-input`,
		`input[data-test="verification-code-input"]`,
	}

	var foundSelector string
	for _, selector := range otpSelectors {
		var exists bool
		if err := chromedp.Run(ctx,
			chromedp.Evaluate(fmt.Sprintf(`document.querySelector('%s') !== null`, selector), &exists),
		); err == nil && exists {
			foundSelector = selector
			log.Printf("Found OTP input with selector: %s", selector)
			break
		}
	}

	if foundSelector == "" {
		// Try waiting a bit more for the input to appear
		time.Sleep(2 * time.Second)
		for _, selector := range otpSelectors {
			var exists bool
			if err := chromedp.Run(ctx,
				chromedp.Evaluate(fmt.Sprintf(`document.querySelector('%s') !== null`, selector), &exists),
			); err == nil && exists {
				foundSelector = selector
				log.Printf("Found OTP input with selector (after wait): %s", selector)
				break
			}
		}
	}

	if foundSelector == "" {
		return fmt.Errorf("could not find OTP input field on 2FA page")
	}

	// Enter the OTP code
	log.Println("Entering TOTP code...")
	if err := chromedp.Run(ctx,
		chromedp.WaitVisible(foundSelector, chromedp.ByQuery),
		chromedp.SendKeys(foundSelector, otpCode, chromedp.ByQuery),
	); err != nil {
		return fmt.Errorf("failed to enter OTP code: %w", err)
	}

	// Submit the 2FA form - try different submit methods
	submitSelectors := []string{
		`button[type="submit"]`,
		`button[data-test="submit-button"]`,
		`button.btn-primary`,
		`button[aria-label*="Submit"]`,
		`button[aria-label*="Verify"]`,
	}

	var submitErr error
	for _, selector := range submitSelectors {
		submitErr = chromedp.Run(ctx,
			chromedp.Click(selector, chromedp.ByQuery),
		)
		if submitErr == nil {
			log.Printf("Clicked submit button with selector: %s", selector)
			break
		}
	}

	if submitErr != nil {
		// Try pressing Enter as fallback
		log.Println("Could not find submit button, trying Enter key...")
		if err := chromedp.Run(ctx,
			chromedp.SendKeys(foundSelector, "\n", chromedp.ByQuery),
		); err != nil {
			return fmt.Errorf("failed to submit 2FA form: %w", err)
		}
	}

	// Wait for 2FA verification to complete
	log.Println("Waiting for 2FA verification to complete...")
	time.Sleep(3 * time.Second)

	// Verify we're no longer on the 2FA page
	if err := chromedp.Run(ctx, chromedp.Location(&currentURL)); err != nil {
		return fmt.Errorf("failed to get URL after 2FA: %w", err)
	}

	if strings.Contains(currentURL, "checkpoint") || strings.Contains(currentURL, "challenge") {
		return fmt.Errorf("2FA verification may have failed - still on verification page")
	}

	log.Println("2FA verification completed successfully")
	return nil
}

// extractProfileData navigates to the profile and extracts all data
func (l *LinkedInScraper) extractProfileData(ctx context.Context) (*models.LinkedInData, error) {
	log.Printf("Navigating to profile: %s", l.profileURL)

	// Navigate to profile page
	if err := chromedp.Run(ctx,
		chromedp.Navigate(l.profileURL),
		chromedp.WaitVisible(`main`, chromedp.ByQuery),
	); err != nil {
		return nil, fmt.Errorf("failed to load profile page: %w", err)
	}

	// Wait for profile content to load
	time.Sleep(3 * time.Second)

	// Scroll down to load lazy-loaded sections
	for i := 0; i < 5; i++ {
		_ = chromedp.Run(ctx,
			chromedp.Evaluate(`window.scrollBy(0, 800)`, nil),
		)
		time.Sleep(500 * time.Millisecond)
	}

	// Scroll back to top
	_ = chromedp.Run(ctx,
		chromedp.Evaluate(`window.scrollTo(0, 0)`, nil),
	)
	time.Sleep(1 * time.Second)

	data := &models.LinkedInData{
		Profile:    models.LinkedInProfile{},
		Experience: []models.LinkedInExperience{},
		Education:  []models.LinkedInEducation{},
		Skills:     []string{},
	}

	// Extract profile information
	if err := l.extractProfile(ctx, data); err != nil {
		log.Printf("Warning: failed to extract profile info: %v", err)
	}

	// Extract experience
	if err := l.extractExperience(ctx, data); err != nil {
		log.Printf("Warning: failed to extract experience: %v", err)
	}

	// Extract education
	if err := l.extractEducation(ctx, data); err != nil {
		log.Printf("Warning: failed to extract education: %v", err)
	}

	// Extract skills
	if err := l.extractSkills(ctx, data); err != nil {
		log.Printf("Warning: failed to extract skills: %v", err)
	}

	return data, nil
}

// extractProfile extracts basic profile information
func (l *LinkedInScraper) extractProfile(ctx context.Context, data *models.LinkedInData) error {
	// Extract name - usually in h1 tag in the profile header
	var name string
	if err := chromedp.Run(ctx,
		chromedp.Text(`h1`, &name, chromedp.ByQuery, chromedp.AtLeast(0)),
	); err == nil && name != "" {
		data.Profile.Name = strings.TrimSpace(name)
	}

	// Extract headline - usually the text under the name
	var headline string
	if err := chromedp.Run(ctx,
		chromedp.Text(`.text-body-medium`, &headline, chromedp.ByQuery, chromedp.AtLeast(0)),
	); err == nil && headline != "" {
		data.Profile.Headline = strings.TrimSpace(headline)
	}

	// Extract location
	var location string
	if err := chromedp.Run(ctx,
		chromedp.Text(`.text-body-small[class*="inline"]`, &location, chromedp.ByQuery, chromedp.AtLeast(0)),
	); err == nil && location != "" {
		data.Profile.Location = strings.TrimSpace(location)
	}

	// Try to extract summary/about section
	var summary string
	if err := chromedp.Run(ctx,
		chromedp.Text(`#about ~ div .inline-show-more-text`, &summary, chromedp.ByQuery, chromedp.AtLeast(0)),
	); err == nil && summary != "" {
		data.Profile.Summary = strings.TrimSpace(summary)
	}

	// Extract profile photo URL
	var photoURL string
	if err := chromedp.Run(ctx,
		chromedp.AttributeValue(`img.pv-top-card-profile-picture__image`, "src", &photoURL, nil, chromedp.AtLeast(0)),
	); err == nil && photoURL != "" {
		data.Profile.PhotoURL = photoURL
	}

	log.Printf("Extracted profile: %s - %s", data.Profile.Name, data.Profile.Headline)
	return nil
}

// extractExperience extracts work experience
func (l *LinkedInScraper) extractExperience(ctx context.Context, data *models.LinkedInData) error {
	// Scroll to experience section to load it
	_ = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('#experience')?.scrollIntoView({behavior: 'instant', block: 'center'})`, nil),
	)
	time.Sleep(2 * time.Second)

	// Use JavaScript to extract experience data - more reliable than CSS selectors
	var experienceJSON string
	err := chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function() {
				const experiences = [];
				
				// Find the experience section
				const expSection = document.querySelector('#experience');
				if (!expSection) return JSON.stringify([]);
				
				// Find section containing experience
				let section = expSection.closest('section');
				if (!section) {
					let parent = expSection.parentElement;
					while (parent && parent.tagName !== 'SECTION') {
						parent = parent.parentElement;
					}
					section = parent;
				}
				
				if (!section) return JSON.stringify([]);
				
				// Get all list items in the experience section
				const items = section.querySelectorAll('ul > li.artdeco-list__item, ul.pvs-list > li');
				
				items.forEach((item) => {
					const exp = {};
					
					// Get logo/company image
					const logoImg = item.querySelector('img.ivm-view-attr__img--centered, img.EntityPhoto-square-3');
					if (logoImg && logoImg.src && !logoImg.src.includes('ghost')) {
						exp.logo = logoImg.src;
					}
					
					// Get all visible text spans
					const spans = item.querySelectorAll('span[aria-hidden="true"]');
					const texts = Array.from(spans).map(s => s.textContent.trim()).filter(t => t && t.length > 0);
					
					// Title is typically the first bold text
					const titleEl = item.querySelector('.t-bold span[aria-hidden="true"]');
					if (titleEl) {
						exp.title = titleEl.textContent.trim();
					}
					
					// Try to get company from company link first (most reliable)
					const companyLink = item.querySelector('a[href*="/company/"]');
					if (companyLink) {
						const companySpan = companyLink.querySelector('span[aria-hidden="true"]');
						if (companySpan) {
							exp.company = companySpan.textContent.trim().split(' · ')[0].trim();
						}
					}
					
					// If no company from link, look in normal text spans
					if (!exp.company) {
						const normalSpans = item.querySelectorAll('.t-14.t-normal:not(.t-black--light) span[aria-hidden="true"]');
						for (const span of normalSpans) {
							const text = span.textContent.trim();
							// Skip if it looks like a date or duration
							if (/\d{4}|heute|present|monat|year|·.*zeit/i.test(text)) continue;
							// Skip if same as title
							if (text === exp.title) continue;
							// This is likely the company
							const parts = text.split(' · ');
							exp.company = parts[0].trim();
							break;
						}
					}
					
					// If still no company, try to find from the item's structure
					// Sometimes company is the second visible text after title
					if (!exp.company && texts.length > 1) {
						for (let i = 1; i < texts.length; i++) {
							const text = texts[i];
							// Skip if it looks like a date, duration, or location
							if (/\d{4}|heute|present|monat|jahr|year/i.test(text)) continue;
							if (text === exp.title) continue;
							// Check if it's not a location (locations often have commas and country names)
							if (!/,.*(?:österreich|austria|germany|deutschland|schweiz|switzerland)/i.test(text)) {
								exp.company = text.split(' · ')[0].trim();
								break;
							}
						}
					}
					
					// Date range - look for text with date patterns in light-colored spans
					const lightSpans = item.querySelectorAll('.t-14.t-normal.t-black--light span[aria-hidden="true"]');
					for (const span of lightSpans) {
						const text = span.textContent.trim();
						// Check for date patterns (German and English)
						if (/\d{4}|heute|present|jan|feb|mär|apr|mai|jun|jul|aug|sep|okt|nov|dez/i.test(text)) {
							// This contains dates - split by · to get date range and duration
							const parts = text.split(' · ');
							exp.dateRange = parts[0].trim();
							if (parts.length > 1) {
								exp.duration = parts[1].trim();
							}
							break;
						}
					}
					
					// Location - second light span that doesn't contain dates
					let foundDate = false;
					for (const span of lightSpans) {
						const text = span.textContent.trim();
						if (/\d{4}|heute|present/i.test(text)) {
							foundDate = true;
							continue;
						}
						if (foundDate && text && !exp.location) {
							exp.location = text;
							break;
						}
					}
					
					// Description - look for longer text or show-more content
					const descEl = item.querySelector('.inline-show-more-text span[aria-hidden="true"]');
					if (descEl) {
						exp.description = descEl.textContent.trim();
					}
					
					// Validate: title and company should be different
					if (exp.title && exp.company && exp.title === exp.company) {
						exp.company = '';  // Clear if same as title
					}
					
					// Only add if we have meaningful data
					if (exp.title && exp.title.length > 1) {
						experiences.push(exp);
					}
				});
				
				return JSON.stringify(experiences);
			})()
		`, &experienceJSON),
	)

	if err != nil {
		log.Printf("Error extracting experience via JS: %v", err)
		return nil
	}

	var rawExperiences []struct {
		Title       string `json:"title"`
		Company     string `json:"company"`
		Logo        string `json:"logo"`
		DateRange   string `json:"dateRange"`
		Duration    string `json:"duration"`
		Location    string `json:"location"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal([]byte(experienceJSON), &rawExperiences); err != nil {
		log.Printf("Error parsing experience JSON: %v", err)
		return nil
	}

	log.Printf("Found %d experience items via JS extraction", len(rawExperiences))

	// Track seen entries to avoid duplicates
	seen := make(map[string]bool)

	for _, raw := range rawExperiences {
		// Skip if title equals company (bad extraction)
		if raw.Title == raw.Company && raw.Company != "" {
			log.Printf("Skipping entry where title equals company: %s", raw.Title)
			continue
		}

		// Create unique key for deduplication
		key := fmt.Sprintf("%s|%s|%s", raw.Title, raw.Company, raw.DateRange)
		if seen[key] {
			continue
		}
		seen[key] = true

		exp := models.LinkedInExperience{
			Title:       raw.Title,
			Company:     raw.Company,
			CompanyLogo: raw.Logo,
			Location:    raw.Location,
			Description: raw.Description,
			Duration:    raw.Duration,
		}
		exp.StartDate, exp.EndDate = parseLinkedInDateRange(raw.DateRange)

		if exp.Title != "" {
			data.Experience = append(data.Experience, exp)
			log.Printf("Extracted experience: %s at %s (%s - %s)", exp.Title, exp.Company, exp.StartDate, exp.EndDate)
		}
	}

	if len(data.Experience) == 0 {
		log.Println("No experience items extracted")
	}

	return nil
}

// extractEducation extracts education history
func (l *LinkedInScraper) extractEducation(ctx context.Context, data *models.LinkedInData) error {
	// Scroll to education section
	_ = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('#education')?.scrollIntoView({behavior: 'instant', block: 'center'})`, nil),
	)
	time.Sleep(2 * time.Second)

	// Use JavaScript to extract education data
	var educationJSON string
	err := chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function() {
				const education = [];
				
				// Find the education section
				const eduSection = document.querySelector('#education');
				if (!eduSection) return JSON.stringify([]);
				
				// Find section containing education
				let section = eduSection.closest('section');
				if (!section) {
					let parent = eduSection.parentElement;
					while (parent && parent.tagName !== 'SECTION') {
						parent = parent.parentElement;
					}
					section = parent;
				}
				
				if (!section) return JSON.stringify([]);
				
				// Get all list items in the education section
				const items = section.querySelectorAll('ul > li.artdeco-list__item, ul.pvs-list > li');
				
				items.forEach((item) => {
					const edu = {};
					
					// Get logo/school image
					const logoImg = item.querySelector('img.ivm-view-attr__img--centered, img.EntityPhoto-square-3');
					if (logoImg && logoImg.src && !logoImg.src.includes('ghost')) {
						edu.logo = logoImg.src;
					}
					
					// Get all spans with aria-hidden for text content
					const spans = item.querySelectorAll('span[aria-hidden="true"]');
					const texts = Array.from(spans).map(s => s.textContent.trim()).filter(t => t && t.length > 0);
					
					// School name - first bold text
					const schoolEl = item.querySelector('.t-bold span[aria-hidden="true"]');
					if (schoolEl) {
						edu.school = schoolEl.textContent.trim();
					}
					
					// Degree and field - usually in normal text, not light
					const normalSpans = item.querySelectorAll('.t-14.t-normal:not(.t-black--light) span[aria-hidden="true"]');
					for (const span of normalSpans) {
						const text = span.textContent.trim();
						// Skip if it looks like a date
						if (/\d{4}|heute|present/i.test(text)) continue;
						// Skip if same as school
						if (text === edu.school) continue;
						// This is likely degree/field
						const parts = text.split(', ');
						edu.degree = parts[0] || '';
						edu.field = parts.slice(1).join(', ') || '';
						break;
					}
					
					// Date range - look in light-colored spans
					const lightSpans = item.querySelectorAll('.t-14.t-normal.t-black--light span[aria-hidden="true"]');
					for (const span of lightSpans) {
						const text = span.textContent.trim();
						// Check for year patterns
						if (/\d{4}/.test(text) && text.length < 40) {
							edu.dateRange = text;
							break;
						}
					}
					
					// Description/activities
					const descEl = item.querySelector('.inline-show-more-text span[aria-hidden="true"]');
					if (descEl) {
						edu.description = descEl.textContent.trim();
					}
					
					// Validate: school name should look like a school
					// Filter out entries that are just grades or notes
					if (edu.school && 
						edu.school.length > 3 &&
						!/^note:/i.test(edu.school) &&
						!/^\d+[\.,]\d+$/.test(edu.school)) {
						education.push(edu);
					}
				});
				
				return JSON.stringify(education);
			})()
		`, &educationJSON),
	)

	if err != nil {
		log.Printf("Error extracting education via JS: %v", err)
		return nil
	}

	var rawEducation []struct {
		School      string `json:"school"`
		Logo        string `json:"logo"`
		Degree      string `json:"degree"`
		Field       string `json:"field"`
		DateRange   string `json:"dateRange"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal([]byte(educationJSON), &rawEducation); err != nil {
		log.Printf("Error parsing education JSON: %v", err)
		return nil
	}

	log.Printf("Found %d education items via JS extraction", len(rawEducation))

	// Track seen entries to avoid duplicates
	seen := make(map[string]bool)

	for _, raw := range rawEducation {
		// Create unique key for deduplication
		key := fmt.Sprintf("%s|%s|%s", raw.School, raw.Degree, raw.DateRange)
		if seen[key] {
			continue
		}
		seen[key] = true

		edu := models.LinkedInEducation{
			School:      raw.School,
			SchoolLogo:  raw.Logo,
			Degree:      raw.Degree,
			Field:       raw.Field,
			Description: raw.Description,
		}
		edu.StartDate, edu.EndDate = parseLinkedInDateRange(raw.DateRange)

		if edu.School != "" {
			data.Education = append(data.Education, edu)
			log.Printf("Extracted education: %s - %s (%s - %s)", edu.School, edu.Degree, edu.StartDate, edu.EndDate)
		}
	}

	if len(data.Education) == 0 {
		log.Println("No education items extracted")
	}

	return nil
}

// extractSkills extracts skills list
func (l *LinkedInScraper) extractSkills(ctx context.Context, data *models.LinkedInData) error {
	// Scroll to skills section
	_ = chromedp.Run(ctx,
		chromedp.Evaluate(`document.querySelector('#skills')?.scrollIntoView({behavior: 'instant', block: 'center'})`, nil),
	)
	time.Sleep(2 * time.Second)

	// Use JavaScript to extract skills
	var skillsJSON string
	err := chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function() {
				const skills = [];
				
				// Find the skills section
				const skillsSection = document.querySelector('#skills');
				if (!skillsSection) return JSON.stringify([]);
				
				// Try multiple selectors for skill items
				const selectors = [
					'#skills ~ div ul > li .t-bold span[aria-hidden="true"]',
					'#skills ~ .pvs-list__outer-container .t-bold span[aria-hidden="true"]',
					'section:has(#skills) .t-bold span[aria-hidden="true"]',
					'#skills ~ div li span.mr1.t-bold span[aria-hidden="true"]',
				];
				
				let items = [];
				for (const sel of selectors) {
					try {
						items = document.querySelectorAll(sel);
						if (items.length > 0) break;
					} catch(e) {}
				}
				
				// Fallback: find section containing skills header
				if (items.length === 0) {
					const sections = document.querySelectorAll('section.artdeco-card');
					for (const sec of sections) {
						const header = sec.querySelector('#skills');
						if (header) {
							items = sec.querySelectorAll('.t-bold span[aria-hidden="true"]');
							break;
						}
					}
				}
				
				const seen = new Set();
				items.forEach((item) => {
					const skill = item.textContent.trim();
					// Filter out non-skill items (headers, section titles, etc.)
					if (skill && 
						skill.length > 1 && 
						skill.length < 60 && 
						!skill.includes('Show all') &&
						!skill.includes('endorsement') &&
						!skill.includes('Skills') &&
						!seen.has(skill.toLowerCase())) {
						seen.add(skill.toLowerCase());
						skills.push(skill);
					}
				});
				
				return JSON.stringify(skills);
			})()
		`, &skillsJSON),
	)

	if err != nil {
		log.Printf("Error extracting skills via JS: %v", err)
		return nil
	}

	var rawSkills []string
	if err := json.Unmarshal([]byte(skillsJSON), &rawSkills); err != nil {
		log.Printf("Error parsing skills JSON: %v", err)
		return nil
	}

	log.Printf("Found %d skills via JS extraction", len(rawSkills))

	for _, skill := range rawSkills {
		if skill != "" && !contains(data.Skills, skill) {
			data.Skills = append(data.Skills, skill)
		}
	}

	if len(data.Skills) == 0 {
		log.Println("No skills extracted")
	}

	return nil
}

// parseDateRange parses LinkedIn date ranges like "Jan 2020 - Present · 4 yrs"
func parseDateRange(dateRange string) (startDate, endDate, duration string) {
	dateRange = strings.TrimSpace(dateRange)

	// Split by · to separate dates from duration
	parts := strings.Split(dateRange, " · ")
	if len(parts) >= 2 {
		duration = strings.TrimSpace(parts[1])
	}

	// Parse the date part
	datePart := parts[0]
	if strings.Contains(datePart, " - ") {
		dates := strings.Split(datePart, " - ")
		if len(dates) >= 1 {
			startDate = strings.TrimSpace(dates[0])
		}
		if len(dates) >= 2 {
			endDate = strings.TrimSpace(dates[1])
		}
	} else if strings.Contains(datePart, " – ") { // Different dash character
		dates := strings.Split(datePart, " – ")
		if len(dates) >= 1 {
			startDate = strings.TrimSpace(dates[0])
		}
		if len(dates) >= 2 {
			endDate = strings.TrimSpace(dates[1])
		}
	} else {
		// Single date, might be just years
		startDate = datePart
	}

	// Normalize "Present" variations
	if strings.EqualFold(endDate, "present") || endDate == "" {
		endDate = "Present"
	}

	return
}

// germanMonthMap maps German month abbreviations to month numbers
var germanMonthMap = map[string]string{
	"jan": "01", "jan.": "01",
	"feb": "02", "feb.": "02",
	"mär": "03", "mär.": "03", "mar": "03", "mar.": "03", "märz": "03",
	"apr": "04", "apr.": "04",
	"mai": "05",
	"jun": "06", "jun.": "06", "juni": "06",
	"jul": "07", "jul.": "07", "juli": "07",
	"aug": "08", "aug.": "08",
	"sep": "09", "sep.": "09", "sept": "09", "sept.": "09",
	"okt": "10", "okt.": "10", "oct": "10", "oct.": "10",
	"nov": "11", "nov.": "11",
	"dez": "12", "dez.": "12", "dec": "12", "dec.": "12",
}

// englishMonthMap maps English month names to month numbers
var englishMonthMap = map[string]string{
	"january": "01", "jan": "01",
	"february": "02", "feb": "02",
	"march": "03", "mar": "03",
	"april": "04", "apr": "04",
	"may":  "05",
	"june": "06", "jun": "06",
	"july": "07", "jul": "07",
	"august": "08", "aug": "08",
	"september": "09", "sep": "09", "sept": "09",
	"october": "10", "oct": "10",
	"november": "11", "nov": "11",
	"december": "12", "dec": "12",
}

// parseLinkedInDateRange parses LinkedIn date ranges and returns properly formatted dates
// Input formats: "Nov. 2025–Heute", "Okt. 2021–Juli 2024", "2020 - 2024", "Jan 2020 - Present"
// Output format: "YYYY-MM" or "YYYY" for start/end, "Present" for ongoing
func parseLinkedInDateRange(dateRange string) (startDate, endDate string) {
	dateRange = strings.TrimSpace(dateRange)
	if dateRange == "" {
		return "", ""
	}

	// Remove duration part (after ·)
	if idx := strings.Index(dateRange, " · "); idx != -1 {
		dateRange = dateRange[:idx]
	}

	// Normalize different dash types and split
	dateRange = strings.ReplaceAll(dateRange, "–", "-") // en-dash
	dateRange = strings.ReplaceAll(dateRange, "—", "-") // em-dash

	var startPart, endPart string

	// Split by dash
	if idx := strings.Index(dateRange, "-"); idx != -1 {
		startPart = strings.TrimSpace(dateRange[:idx])
		endPart = strings.TrimSpace(dateRange[idx+1:])
	} else {
		startPart = dateRange
	}

	// Parse start date
	startDate = parseLinkedInDate(startPart)

	// Parse end date
	if endPart != "" {
		endLower := strings.ToLower(endPart)
		if endLower == "heute" || endLower == "present" || endLower == "current" || endLower == "jetzt" {
			endDate = "Present"
		} else {
			endDate = parseLinkedInDate(endPart)
		}
	}

	return startDate, endDate
}

// parseLinkedInDate parses a single date string into YYYY-MM or YYYY format
func parseLinkedInDate(dateStr string) string {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return ""
	}

	// Extract year (4 digits)
	yearRegex := regexp.MustCompile(`\b(19|20)\d{2}\b`)
	yearMatch := yearRegex.FindString(dateStr)
	if yearMatch == "" {
		return ""
	}

	// Try to extract month
	dateLower := strings.ToLower(dateStr)

	// Check German months
	for monthName, monthNum := range germanMonthMap {
		if strings.Contains(dateLower, monthName) {
			return fmt.Sprintf("%s-%s", yearMatch, monthNum)
		}
	}

	// Check English months
	for monthName, monthNum := range englishMonthMap {
		if strings.Contains(dateLower, monthName) {
			return fmt.Sprintf("%s-%s", yearMatch, monthNum)
		}
	}

	// No month found, return just the year
	return yearMatch
}

// contains checks if a string slice contains a value
func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// ManualData allows setting LinkedIn data manually (fallback)
func (l *LinkedInScraper) ManualData(data models.LinkedInData) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := l.cache.Set(cacheKeyLinkedIn, jsonData, l.cacheTTL); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// SetHeadless allows setting headless mode (for debugging)
func (l *LinkedInScraper) SetHeadless(headless bool) {
	l.headless = headless
}

// ExtractProfileURLUsername extracts username from LinkedIn profile URL
func ExtractProfileURLUsername(url string) string {
	re := regexp.MustCompile(`linkedin\.com/in/([^/]+)/?`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// saveCookies saves the current browser cookies to cache
func (l *LinkedInScraper) saveCookies(ctx context.Context) error {
	var cookies []*network.Cookie
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		c, err := network.GetCookies().Do(ctx)
		if err != nil {
			return err
		}
		cookies = c
		return nil
	})); err != nil {
		return fmt.Errorf("failed to get cookies: %w", err)
	}

	data, err := json.Marshal(cookies)
	if err != nil {
		return fmt.Errorf("failed to marshal cookies: %w", err)
	}

	// Store cookies with 7 day TTL (longer than data cache)
	if err := l.cache.Set(cacheKeyLinkedInCookies, data, 7*24*time.Hour); err != nil {
		return fmt.Errorf("failed to save cookies: %w", err)
	}

	log.Printf("Saved %d LinkedIn cookies to cache", len(cookies))
	return nil
}

// restoreCookies restores cookies from cache to the browser
func (l *LinkedInScraper) restoreCookies(ctx context.Context) (bool, error) {
	cached, err := l.cache.Get(cacheKeyLinkedInCookies)
	if err != nil {
		return false, fmt.Errorf("cache error: %w", err)
	}

	if cached == nil {
		log.Println("No cached LinkedIn cookies found")
		return false, nil
	}

	var cookies []*network.Cookie
	if err := json.Unmarshal(cached, &cookies); err != nil {
		log.Printf("Warning: Failed to unmarshal cached cookies: %v", err)
		return false, nil
	}

	// Set cookies
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		for _, cookie := range cookies {
			// Only restore valid cookies (not expired)
			if cookie.Expires > 0 && time.Unix(int64(cookie.Expires), 0).Before(time.Now()) {
				continue
			}

			// Set cookie - note: we don't set Expires as the cookie already has its expiration
			// and the type conversion is complex
			if err := network.SetCookie(cookie.Name, cookie.Value).
				WithDomain(cookie.Domain).
				WithPath(cookie.Path).
				WithHTTPOnly(cookie.HTTPOnly).
				WithSecure(cookie.Secure).
				Do(ctx); err != nil {
				return err
			}
		}
		return nil
	})); err != nil {
		return false, fmt.Errorf("failed to restore cookies: %w", err)
	}

	log.Printf("Restored %d LinkedIn cookies from cache", len(cookies))
	return true, nil
}

// isLoggedIn checks if we're already logged in to LinkedIn
func (l *LinkedInScraper) isLoggedIn(ctx context.Context) bool {
	var currentURL string
	if err := chromedp.Run(ctx, chromedp.Location(&currentURL)); err != nil {
		return false
	}

	// If we're not on login page and cookies are present, we're likely logged in
	if !strings.Contains(currentURL, "login") && !strings.Contains(currentURL, "authwall") {
		// Try to access a page that requires login
		var feedExists bool
		if err := chromedp.Run(ctx,
			chromedp.Navigate("https://www.linkedin.com/feed/"),
			chromedp.WaitVisible("div[role='main']", chromedp.ByQuery),
		); err == nil {
			feedExists = true
		}
		return feedExists
	}

	return false
}
