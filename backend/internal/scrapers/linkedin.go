package scrapers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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
	linkedInTimeoutSec      = 180 // 3 minutes
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

// LinkedInCookie represents a browser cookie for persistence
type LinkedInCookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"`
	HTTPOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	SameSite string  `json:"sameSite"`
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

// downloadImageAsBase64 downloads an image and converts it to a base64 data URI
func downloadImageAsBase64(imageURL string) string {
	if imageURL == "" {
		return ""
	}

	if strings.HasPrefix(imageURL, "data:") {
		return imageURL
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(imageURL)
	if err != nil {
		log.Printf("Failed to download image %s: %v", imageURL, err)
		return ""
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("Warning: failed to close response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to download image %s: status %d", imageURL, resp.StatusCode)
		return ""
	}

	const maxImageSize = 10 * 1024 * 1024 // 10MB
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxImageSize))
	if err != nil {
		log.Printf("Failed to read image data: %v", err)
		return ""
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	base64Data := base64.StdEncoding.EncodeToString(data)
	dataURI := fmt.Sprintf("data:%s;base64,%s", contentType, base64Data)
	log.Printf("Converted image to base64 data URI (%d bytes)", len(data))
	return dataURI
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
			log.Printf("Warning: failed to unmarshal cached LinkedIn data, performing fresh scrape: %v", err)
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

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", l.headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, linkedInTimeoutSec*time.Second)
	defer cancel()

	log.Println("Navigating to LinkedIn...")
	if err := chromedp.Run(ctx, chromedp.Navigate("https://www.linkedin.com")); err != nil {
		return nil, fmt.Errorf("failed to navigate to LinkedIn (check network and Chrome installation): %w", err)
	}

	// Try to restore cookies
	cookiesRestored := l.restoreCookies(ctx)
	if cookiesRestored {
		log.Println("Restored cookies from cache, checking if session is valid...")
		if err := chromedp.Run(ctx, chromedp.Navigate("https://www.linkedin.com/feed/")); err == nil {
			var currentURL string
			if err := chromedp.Run(ctx, chromedp.Location(&currentURL)); err != nil {
				log.Printf("Failed to get current URL after cookie restore: %v, proceeding to fresh login", err)
			} else if !strings.Contains(currentURL, "login") && !strings.Contains(currentURL, "checkpoint") {
				log.Println("Cookie session is valid, skipping login...")
				data, err := l.extractProfileData(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to extract profile data: %w", err)
				}
				log.Println("Profile data extracted successfully")
				return data, nil
			}
		}
		log.Println("Cookie session expired or invalid, performing fresh login...")
	}

	// Perform login
	if err := l.login(ctx); err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}
	log.Println("Login successful")

	l.saveCookies(ctx)
	log.Println("Navigating to profile...")

	data, err := l.extractProfileData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to extract profile data: %w", err)
	}

	log.Println("Profile data extracted successfully")
	return data, nil
}

// saveCookies saves LinkedIn cookies to cache
func (l *LinkedInScraper) saveCookies(ctx context.Context) {
	var allCookies []LinkedInCookie

	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			cookies, err := network.GetCookies().Do(ctx)
			if err != nil {
				return err
			}

			for _, c := range cookies {
				if strings.Contains(c.Domain, "linkedin.com") {
					allCookies = append(allCookies, LinkedInCookie{
						Name:     c.Name,
						Value:    c.Value,
						Domain:   c.Domain,
						Path:     c.Path,
						Expires:  c.Expires,
						HTTPOnly: c.HTTPOnly,
						Secure:   c.Secure,
						SameSite: string(c.SameSite),
					})
				}
			}
			return nil
		}),
	)

	if err != nil {
		log.Printf("Warning: failed to extract cookies: %v", err)
		return
	}

	if len(allCookies) > 0 {
		cookieData, err := json.Marshal(allCookies)
		if err != nil {
			log.Printf("Warning: failed to marshal cookies: %v", err)
			return
		}
		if err := l.cache.Set(cacheKeyLinkedInCookies, cookieData, 7*24*time.Hour); err != nil {
			log.Printf("Warning: failed to save cookies to cache: %v", err)
		} else {
			log.Printf("Saved %d LinkedIn cookies to cache", len(allCookies))
		}
	}
}

// restoreCookies restores LinkedIn cookies from cache
func (l *LinkedInScraper) restoreCookies(ctx context.Context) bool {
	cached, err := l.cache.Get(cacheKeyLinkedInCookies)
	if err != nil || cached == nil {
		return false
	}

	var cookies []LinkedInCookie
	if err := json.Unmarshal(cached, &cookies); err != nil {
		log.Printf("Warning: failed to unmarshal cached cookies: %v", err)
		return false
	}

	if len(cookies) == 0 {
		return false
	}

	err = chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			for _, c := range cookies {
				var sameSite network.CookieSameSite
				switch c.SameSite {
				case "Strict":
					sameSite = network.CookieSameSiteStrict
				case "Lax":
					sameSite = network.CookieSameSiteLax
				case "None":
					sameSite = network.CookieSameSiteNone
				default:
					sameSite = network.CookieSameSiteLax
				}

				err := network.SetCookie(c.Name, c.Value).
					WithDomain(c.Domain).
					WithPath(c.Path).
					WithHTTPOnly(c.HTTPOnly).
					WithSecure(c.Secure).
					WithSameSite(sameSite).
					Do(ctx)
				if err != nil {
					log.Printf("Warning: failed to set cookie %s: %v", c.Name, err)
				}
			}
			return nil
		}),
	)

	if err != nil {
		log.Printf("Warning: failed to restore cookies: %v", err)
		return false
	}

	log.Printf("Restored %d cookies from cache", len(cookies))
	return true
}

// Refresh forces a fresh scrape and updates cache
func (l *LinkedInScraper) Refresh() (any, error) {
	data, err := l.Scrape()
	if err != nil {
		return nil, err
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := l.cache.Set(cacheKeyLinkedIn, jsonData, l.cacheTTL); err != nil {
		log.Printf("Warning: failed to update cache: %v", err)
	}

	return data, nil
}

// login performs LinkedIn login
func (l *LinkedInScraper) login(ctx context.Context) error {
	log.Println("Navigating to LinkedIn login page...")

	if err := chromedp.Run(ctx,
		chromedp.Navigate(linkedInLoginURL),
		chromedp.WaitVisible(`#username`, chromedp.ByID),
	); err != nil {
		return fmt.Errorf("failed to load login page: %w", err)
	}

	log.Println("Entering credentials...")

	if err := chromedp.Run(ctx,
		chromedp.SendKeys(`#username`, l.email, chromedp.ByID),
		chromedp.SendKeys(`#password`, l.password, chromedp.ByID),
		chromedp.Click(`button[type="submit"]`, chromedp.ByQuery),
	); err != nil {
		return fmt.Errorf("failed to submit login form: %w", err)
	}

	log.Println("Waiting for login to complete...")

	if err := chromedp.Run(ctx,
		chromedp.WaitNotPresent(`#username`, chromedp.ByID),
	); err != nil {
		var currentURL string
		_ = chromedp.Run(ctx, chromedp.Location(&currentURL))
		if strings.Contains(currentURL, "challenge") || strings.Contains(currentURL, "checkpoint") {
			return fmt.Errorf("LinkedIn security challenge detected - manual verification may be required")
		}
		return fmt.Errorf("login may have failed: %w", err)
	}

	time.Sleep(2 * time.Second)

	if err := l.handle2FA(ctx); err != nil {
		return fmt.Errorf("2FA handling failed: %w", err)
	}

	time.Sleep(2 * time.Second)
	return nil
}

// handle2FA checks for and handles TOTP-based two-factor authentication
func (l *LinkedInScraper) handle2FA(ctx context.Context) error {
	var currentURL string
	if err := chromedp.Run(ctx, chromedp.Location(&currentURL)); err != nil {
		return fmt.Errorf("failed to get current URL: %w", err)
	}

	log.Printf("Current URL after login: %s", currentURL)

	is2FAPage := strings.Contains(currentURL, "checkpoint") ||
		strings.Contains(currentURL, "challenge") ||
		strings.Contains(currentURL, "two-step-verification")

	if !is2FAPage {
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

	if l.totpSecret == "" {
		return fmt.Errorf("2FA required but TOTP secret not configured (set LINKEDIN_TOTP_SECRET)")
	}

	otpCode, err := totp.GenerateCode(l.totpSecret, time.Now())
	if err != nil {
		return fmt.Errorf("failed to generate TOTP code: %w", err)
	}

	log.Println("Generated TOTP code successfully")

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

	log.Println("Entering TOTP code...")
	if err := chromedp.Run(ctx,
		chromedp.WaitVisible(foundSelector, chromedp.ByQuery),
		chromedp.SendKeys(foundSelector, otpCode, chromedp.ByQuery),
	); err != nil {
		return fmt.Errorf("failed to enter OTP code: %w", err)
	}

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
		log.Println("Could not find submit button, trying Enter key...")
		if err := chromedp.Run(ctx,
			chromedp.SendKeys(foundSelector, "\n", chromedp.ByQuery),
		); err != nil {
			return fmt.Errorf("failed to submit 2FA form: %w", err)
		}
	}

	log.Println("Waiting for 2FA verification to complete...")
	time.Sleep(3 * time.Second)

	if err := chromedp.Run(ctx, chromedp.Location(&currentURL)); err != nil {
		return fmt.Errorf("failed to get URL after 2FA: %w", err)
	}

	if strings.Contains(currentURL, "checkpoint") || strings.Contains(currentURL, "challenge") {
		return fmt.Errorf("2FA verification may have failed - still on verification page")
	}

	log.Println("2FA verification completed successfully")
	return nil
}

// extractProfileData navigates to profile and detail pages to extract all data using stable selectors
func (l *LinkedInScraper) extractProfileData(ctx context.Context) (*models.LinkedInData, error) {
	data := &models.LinkedInData{
		Profile:    models.LinkedInProfile{},
		Experience: []models.LinkedInExperience{},
		Education:  []models.LinkedInEducation{},
		Skills:     []string{},
	}

	baseURL := cleanProfileURL(l.profileURL)

	// Extract profile information from main profile page
	log.Printf("Extracting profile information from: %s", l.profileURL)

	// Check if we're already on the profile page
	var currentURL string
	if err := chromedp.Run(ctx, chromedp.Location(&currentURL)); err != nil {
		log.Printf("Failed to get current URL: %v, navigating to profile", err)
	}

	if !strings.Contains(currentURL, "/in/") {
		log.Println("Navigating to profile page...")
		if err := chromedp.Run(ctx,
			chromedp.Navigate(l.profileURL),
		); err != nil {
			return nil, fmt.Errorf("failed to navigate to profile: %w", err)
		}
		time.Sleep(3 * time.Second)
	} else {
		log.Println("Already on profile page, skipping navigation")
	}

	// Wait for page to load with timeout
	log.Println("Waiting for main content to load...")
	waitCtx, waitCancel := context.WithTimeout(ctx, 10*time.Second)
	defer waitCancel()

	if err := chromedp.Run(waitCtx,
		chromedp.WaitVisible(`main`, chromedp.ByQuery),
	); err != nil {
		log.Printf("Warning: Failed to wait for main element: %v", err)
		// Continue anyway, maybe the page loaded differently
	}

	time.Sleep(2 * time.Second)

	// Extract profile basics — fail hard if this doesn't work since it indicates the page didn't load
	log.Println("Extracting profile data...")

	profile, err := l.extractProfileBasics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to extract profile basics (page may not have loaded): %w", err)
	}
	data.Profile = profile

	log.Printf("Extracted profile: name='%s', headline='%s', location='%s'",
		data.Profile.Name, data.Profile.Headline, data.Profile.Location)

	// Extract experience from details page
	experience, err := l.extractExperienceData(ctx, baseURL)
	if err != nil {
		log.Printf("Warning: failed to extract experience: %v", err)
	} else {
		data.Experience = experience
		log.Printf("Extracted %d experience entries", len(data.Experience))
	}

	// Extract education from details page
	education, err := l.extractEducationData(ctx, baseURL)
	if err != nil {
		log.Printf("Warning: failed to extract education: %v", err)
	} else {
		data.Education = education
		log.Printf("Extracted %d education entries", len(data.Education))
	}

	// Extract skills from details page
	skills, err := l.extractSkillsData(ctx, baseURL)
	if err != nil {
		log.Printf("Warning: failed to extract skills: %v", err)
	} else {
		data.Skills = skills
		log.Printf("Extracted %d skills", len(data.Skills))
	}

	return data, nil
}

// extractProfileBasics extracts basic profile info using semantic selectors
func (l *LinkedInScraper) extractProfileBasics(ctx context.Context) (models.LinkedInProfile, error) {
	var profile models.LinkedInProfile

	// DEBUG: Log the page structure to understand what selectors to use
	var debugHTMLProfile string
	debugScriptProfile := `(function() {
		const main = document.querySelector('main');
		if (!main) return 'No main element found';
		
		const viewNames = [];
		document.querySelectorAll('[data-view-name]').forEach(function(el) {
			viewNames.push(el.getAttribute('data-view-name'));
		});
		
		const testIds = [];
		document.querySelectorAll('[data-testid]').forEach(function(el) {
			testIds.push(el.getAttribute('data-testid'));
		});
		
		const h1Text = document.querySelector('h1') ? document.querySelector('h1').textContent.trim() : 'No h1 found';
		
		const h2Texts = [];
		document.querySelectorAll('h2').forEach(function(h2, i) {
			if (i < 3) h2Texts.push(h2.textContent.trim().substring(0, 50));
		});
		
		const pTags = [];
		document.querySelectorAll('main p').forEach(function(p, i) {
			if (i < 5) pTags.push(p.textContent.trim().substring(0, 50));
		});
		
		return JSON.stringify({
			viewNames: viewNames,
			testIds: testIds,
			h1Text: h1Text,
			h2Texts: h2Texts,
			allPTags: pTags
		}, null, 2);
	})()`
	_ = chromedp.Run(ctx, chromedp.Evaluate(debugScriptProfile, &debugHTMLProfile))
	log.Printf("DEBUG: Profile page structure: %s", debugHTMLProfile)

	// Use JavaScript to extract profile data based on semantic structure
	// This is more resilient than CSS class selectors
	var result map[string]interface{}

	profileScript := `(function() {
		const data = {};
		
		// Try to find name - LinkedIn now uses h2 for the name in profile top card
		// Look for h2 that contains the name (not notification count)
		const h2Elements = document.querySelectorAll('h2');
		for (let i = 0; i < h2Elements.length; i++) {
			const text = h2Elements[i].textContent.trim();
			// Name is usually longer than 3 chars and doesn't contain notification text
			if (text && text.length > 3 && !text.includes('Benachrichtigungen') && 
			    !text.includes('Notifications') && !text.match(/^\d+/)) {
				data.name = text;
				break;
			}
		}
		
		// If no name found in h2, try h1
		if (!data.name) {
			const nameEl = document.querySelector('h1');
			if (nameEl) data.name = nameEl.textContent.trim();
		}
		
		// Try to find headline - look for text that looks like a job title/position
		// Headline is usually in a p element after the name
		const mainSection = document.querySelector('main');
		if (mainSection) {
			const paragraphs = mainSection.querySelectorAll('p');
			for (let i = 0; i < paragraphs.length; i++) {
				const text = paragraphs[i].textContent.trim();
				// Headlines typically contain job-related keywords or are structured like titles
				// Skip pronouns (er/ihm, she/her, etc.) and very short text
				if (text && text.length > 10 && text.length < 150 && 
				    !text.includes('@') && !text.includes('Kontakt') && 
				    !text.includes('Follower') && !text.includes('follower') &&
				    !text.match(/^(er\/ihm|she\/her|he\/him|they\/them)$/i)) {
					data.headline = text;
					break;
				}
			}
		}
		
		// Try to find location - look for text with location patterns
		if (mainSection) {
			const allText = mainSection.querySelectorAll('p, span');
			for (let i = 0; i < allText.length; i++) {
				const text = allText[i].textContent.trim();
				// Location patterns: contains comma and location keywords
				if (text && text.length < 100 && 
				    (text.includes('Österreich') || text.includes('Austria') || 
				     text.includes('Germany') || text.includes('Deutschland') ||
				     text.match(/^[A-Z][a-z]+,?\s+[A-Z]/))) {
					data.location = text;
					break;
				}
			}
		}
		
		// Try to find profile photo - look for images near the profile section
		const imgSelectors = [
			'img[alt*="profile"]',
			'img[alt*="Profil"]',
			'img[alt*="photo"]',
			'img[alt*="Photo"]',
			'[data-view-name="profile-top-card-member-photo"] img',
			'button img[class*="profile"]',
			'figure img'
		];
		for (let i = 0; i < imgSelectors.length; i++) {
			const img = document.querySelector(imgSelectors[i]);
			if (img && img.src && !img.src.includes('data:')) {
				data.photoURL = img.src;
				break;
			}
		}
		
		return data;
	})()`
	err := chromedp.Run(ctx, chromedp.Evaluate(profileScript, &result))

	if err != nil {
		return profile, fmt.Errorf("failed to evaluate profile script: %w", err)
	}

	log.Printf("DEBUG: Profile extraction result: %+v", result)

	if name, ok := result["name"].(string); ok && name != "" {
		profile.Name = name
	}
	if headline, ok := result["headline"].(string); ok && headline != "" {
		profile.Headline = headline
	}
	if location, ok := result["location"].(string); ok && location != "" {
		profile.Location = location
	}
	if photoURL, ok := result["photoURL"].(string); ok && photoURL != "" {
		profile.PhotoURL = downloadImageAsBase64(photoURL)
	}

	return profile, nil
}

// extractExperienceData extracts experience from the details page
func (l *LinkedInScraper) extractExperienceData(ctx context.Context, baseURL string) ([]models.LinkedInExperience, error) {
	var experiences []models.LinkedInExperience

	experienceURL := baseURL + "/details/experience/"
	log.Printf("Extracting experience from: %s", experienceURL)

	if err := chromedp.Run(ctx, chromedp.Navigate(experienceURL)); err != nil {
		return nil, fmt.Errorf("failed to navigate to experience page: %w", err)
	}

	// Wait for main element
	waitCtx, waitCancel := context.WithTimeout(ctx, 15*time.Second)
	defer waitCancel()
	if err := chromedp.Run(waitCtx, chromedp.WaitVisible(`main`, chromedp.ByQuery)); err != nil {
		log.Printf("Warning: timeout waiting for experience page: %v", err)
	}

	// Wait additional time for lazy-loaded content
	time.Sleep(5 * time.Second)

	// Scroll aggressively to trigger lazy loading
	for i := 0; i < 10; i++ {
		_ = chromedp.Run(ctx, chromedp.Evaluate(`window.scrollBy(0, 500)`, nil))
		time.Sleep(300 * time.Millisecond)
	}
	time.Sleep(2 * time.Second)

	// Click "Load more" buttons to load all experience entries
	log.Println("Looking for 'Load more' buttons...")
	for i := 0; i < 5; i++ {
		var clicked bool
		_ = chromedp.Run(ctx, chromedp.Evaluate(`
			(function() {
				// Look for "Load more" / "Weitere laden" buttons
				const buttons = document.querySelectorAll('button');
				for (const btn of buttons) {
					const text = btn.textContent.toLowerCase();
					if (text.includes('load more') || text.includes('weitere laden') || 
					    text.includes('show more') || text.includes('mehr anzeigen')) {
						btn.click();
						return true;
					}
				}
				return false;
			})()
		`, &clicked))
		if clicked {
			log.Println("Clicked 'Load more' button, waiting for content...")
			time.Sleep(3 * time.Second)
		} else {
			break
		}
	}

	// Scroll again after loading more
	for i := 0; i < 5; i++ {
		_ = chromedp.Run(ctx, chromedp.Evaluate(`window.scrollBy(0, 500)`, nil))
		time.Sleep(300 * time.Millisecond)
	}
	time.Sleep(2 * time.Second)

	// DEBUG: Log the page structure
	var debugHTML string
	debugScript := `(function() {
		try {
			const testIds = [];
			document.querySelectorAll('[data-testid]').forEach(function(el) {
				testIds.push(el.getAttribute('data-testid'));
			});
			
			const pTags = [];
			document.querySelectorAll('main p').forEach(function(p, i) {
				if (i < 30) pTags.push((p.textContent || '').trim().substring(0, 80));
			});
			
			// Check for componentkey attributes (new LinkedIn structure)
			const componentKeys = [];
			document.querySelectorAll('[componentkey]').forEach(function(el, i) {
				if (i < 10) componentKeys.push(el.getAttribute('componentkey'));
			});
			
			return JSON.stringify({
				testIds: testIds,
				allPTags: pTags,
				componentKeys: componentKeys
			}, null, 2);
		} catch (e) {
			return 'Error: ' + e.message;
		}
	})()`
	debugErr := chromedp.Run(ctx, chromedp.Evaluate(debugScript, &debugHTML))
	if debugErr != nil {
		log.Printf("DEBUG: Error evaluating experience page structure: %v", debugErr)
	} else {
		log.Printf("DEBUG: Experience page structure: %s", debugHTML)
	}

	// Extract experience data using JavaScript
	// LinkedIn's new structure uses componentkey attributes for experience items
	var expData []map[string]string
	expScript := `(function() {
		const experiences = [];
		
		// Look for experience section by data-testid
		const expSection = document.querySelector('[data-testid*="ExperienceDetailsSection"]');
		
		if (!expSection) {
			console.log('No experience section found');
			return experiences;
		}
		
		// Find all experience items by componentkey attribute (new LinkedIn structure)
		let entries = expSection.querySelectorAll('[componentkey*="entity-collection-item"]');
		
		// Fallback: try role="listitem" for older structure
		if (entries.length === 0) {
			entries = expSection.querySelectorAll('[role="listitem"]');
		}
		
		entries.forEach(function(entry) {
			const exp = {};
			
			// Get all p elements and their text content
			const allPs = entry.querySelectorAll('p');
			const textContents = [];
			allPs.forEach(function(p) {
				const text = p.textContent.trim();
				if (text && text.length > 1) {
					textContents.push(text);
				}
			});
			
			// Extract title - first p element with substantial text
			// In new structure: <p class="_1b2d0c42 f3e5fdd5 ...">Title</p>
			for (let i = 0; i < textContents.length; i++) {
				const text = textContents[i];
				// Skip pronouns, dates, and very short text
				if (text.length > 3 && 
				    !text.match(/^(er\/sie|er\/ihm|sie\/ihr)/i) &&
				    !text.match(/^\d{4}$/) && 
				    !text.match(/^[A-Z][a-z]{2}\.? \d{4}/) &&
				    !text.includes('·')) {
					exp.title = text;
					break;
				}
			}
			
			// Extract company - contains · separator (Company · EmploymentType)
			for (let i = 0; i < textContents.length; i++) {
				const text = textContents[i];
				if (text.includes('·') && !text.includes('–') && !text.includes('-')) {
					// Split by · and take the first part (company name)
					const parts = text.split('·');
					exp.company = parts[0].trim();
					// Employment type is the second part
					if (parts.length > 1) {
						exp.employmentType = parts[1].trim();
					}
					break;
				}
			}
			
			// Extract date range - contains year and dash/en-dash
			for (let i = 0; i < textContents.length; i++) {
				const text = textContents[i];
				if (text.match(/\d{4}/) && (text.includes('–') || text.includes('-') || text.includes(' bis '))) {
					exp.dateRange = text.replace(/\s*·\s*\d+\s*(Monate|Monat|Jahre|Jahr)\s*$/, '').trim();
					break;
				}
			}
			
			// Extract location - contains comma and location keywords
			for (let i = 0; i < textContents.length; i++) {
				const text = textContents[i];
				if (text.includes(',') && (text.includes('Österreich') || text.includes('Austria') || 
				    text.includes('Germany') || text.includes('Deutschland') || text.includes('Wien') ||
				    text.includes('Bezirk') || text.includes('Stadt') || text.includes('Upper Austria'))) {
					exp.location = text;
					break;
				}
			}
			
			// Try to find company logo
			const img = entry.querySelector('img[data-loaded="true"]');
			if (img && img.src && !img.src.includes('data:')) {
				exp.logo = img.src;
			}
			
			if (exp.title) {
				experiences.push(exp);
			}
		});
		
		return experiences;
	})()`
	err := chromedp.Run(ctx, chromedp.Evaluate(expScript, &expData))

	if err != nil {
		return nil, fmt.Errorf("failed to extract experience: %w", err)
	}

	log.Printf("DEBUG: Extracted %d experience entries from JavaScript", len(expData))

	for _, exp := range expData {
		start, end := "", ""
		if dateRange, ok := exp["dateRange"]; ok && dateRange != "" {
			start, end = parseDateRange(dateRange)
		}

		experience := models.LinkedInExperience{
			Title:     exp["title"],
			Company:   exp["company"],
			Location:  exp["location"],
			StartDate: start,
			EndDate:   end,
		}

		if logo, ok := exp["logo"]; ok && logo != "" {
			experience.CompanyLogo = downloadImageAsBase64(logo)
		}

		experiences = append(experiences, experience)
	}

	return experiences, nil
}

// extractEducationData extracts education from the details page
func (l *LinkedInScraper) extractEducationData(ctx context.Context, baseURL string) ([]models.LinkedInEducation, error) {
	var education []models.LinkedInEducation

	educationURL := baseURL + "/details/education/"
	log.Printf("Extracting education from: %s", educationURL)

	if err := chromedp.Run(ctx, chromedp.Navigate(educationURL)); err != nil {
		return nil, fmt.Errorf("failed to navigate to education page: %w", err)
	}

	// Wait for main element
	waitCtx, waitCancel := context.WithTimeout(ctx, 15*time.Second)
	defer waitCancel()
	if err := chromedp.Run(waitCtx, chromedp.WaitVisible(`main`, chromedp.ByQuery)); err != nil {
		log.Printf("Warning: timeout waiting for education page: %v", err)
	}

	// Wait additional time for lazy-loaded content
	time.Sleep(5 * time.Second)

	// Scroll aggressively to trigger lazy loading
	for i := 0; i < 10; i++ {
		_ = chromedp.Run(ctx, chromedp.Evaluate(`window.scrollBy(0, 500)`, nil))
		time.Sleep(300 * time.Millisecond)
	}
	time.Sleep(2 * time.Second)

	// Click "Load more" buttons to load all education entries
	log.Println("Looking for 'Load more' buttons...")
	for i := 0; i < 5; i++ {
		var clicked bool
		_ = chromedp.Run(ctx, chromedp.Evaluate(`
			(function() {
				// Look for "Load more" / "Weitere laden" buttons
				const buttons = document.querySelectorAll('button');
				for (const btn of buttons) {
					const text = btn.textContent.toLowerCase();
					if (text.includes('load more') || text.includes('weitere laden') || 
					    text.includes('show more') || text.includes('mehr anzeigen')) {
						btn.click();
						return true;
					}
				}
				return false;
			})()
		`, &clicked))
		if clicked {
			log.Println("Clicked 'Load more' button, waiting for content...")
			time.Sleep(3 * time.Second)
		} else {
			break
		}
	}

	// Scroll again after loading more
	for i := 0; i < 5; i++ {
		_ = chromedp.Run(ctx, chromedp.Evaluate(`window.scrollBy(0, 500)`, nil))
		time.Sleep(300 * time.Millisecond)
	}
	time.Sleep(2 * time.Second)

	// DEBUG: Log the page structure
	var debugHTMLEdu string
	debugScriptEdu := `(function() {
		try {
			const testIds = [];
			document.querySelectorAll('[data-testid]').forEach(function(el) {
				testIds.push(el.getAttribute('data-testid'));
			});
			
			const pTags = [];
			document.querySelectorAll('main p').forEach(function(p, i) {
				if (i < 30) pTags.push((p.textContent || '').trim().substring(0, 80));
			});
			
			// Check for componentkey attributes (new LinkedIn structure)
			const componentKeys = [];
			document.querySelectorAll('[componentkey]').forEach(function(el, i) {
				if (i < 10) componentKeys.push(el.getAttribute('componentkey'));
			});
			
			return JSON.stringify({
				testIds: testIds,
				allPTags: pTags,
				componentKeys: componentKeys
			}, null, 2);
		} catch (e) {
			return 'Error: ' + e.message;
		}
	})()`
	debugErrEdu := chromedp.Run(ctx, chromedp.Evaluate(debugScriptEdu, &debugHTMLEdu))
	if debugErrEdu != nil {
		log.Printf("DEBUG: Error evaluating education page structure: %v", debugErrEdu)
	} else {
		log.Printf("DEBUG: Education page structure: %s", debugHTMLEdu)
	}

	// Extract education data using JavaScript
	// LinkedIn's new structure uses componentkey attributes for education items
	var eduData []map[string]string
	eduScript := `(function() {
		const education = [];
		
		// Look for education section by data-testid
		const eduSection = document.querySelector('[data-testid*="EducationDetailsSection"]');

		if (!eduSection) {
			console.log('No education section found');
			return education;
		}

		// Find all education items by componentkey attribute (new LinkedIn structure)
		let entries = eduSection.querySelectorAll('[componentkey*="entity-collection-item"]');
		
		// Fallback: try role="listitem" for older structure
		if (entries.length === 0) {
			entries = eduSection.querySelectorAll('[role="listitem"]');
		}
		
		entries.forEach(function(entry) {
			const edu = {};
			
			// Get all p elements and their text content
			const allPs = entry.querySelectorAll('p');
			const textContents = [];
			allPs.forEach(function(p) {
				const text = p.textContent.trim();
				if (text && text.length > 1) {
					textContents.push(text);
				}
			});
			
			// Extract school name - first p element with substantial text
			for (let i = 0; i < textContents.length; i++) {
				const text = textContents[i];
				// Skip dates and very short text
				if (text.length > 3 && 
				    !text.match(/^\d{4}$/) && 
				    !text.match(/^[A-Z][a-z]{2}\.? \d{4}/) &&
				    !text.includes('·')) {
					edu.school = text;
					break;
				}
			}
			
			// Extract degree - contains degree keywords or is second substantial text
			for (let i = 0; i < textContents.length; i++) {
				const text = textContents[i];
				if (text.match(/(Bachelor|Master|Diplom|PhD|Dr\.|MBA|Magister|BSc|MSc|B\.Sc|M\.Sc|Computer Science|Informatik)/i)) {
					edu.degree = text;
					break;
				}
			}
			
			// If no degree found, use second substantial text as degree/field
			if (!edu.degree && textContents.length > 1) {
				for (let i = 0; i < textContents.length; i++) {
					const text = textContents[i];
					if (text !== edu.school && text.length > 3 && 
					    !text.match(/^\d{4}$/) && 
					    !text.match(/^[A-Z][a-z]{2}\.? \d{4}/) &&
					    !text.includes('·')) {
						edu.degree = text;
						break;
					}
				}
			}
			
			// Extract dates - contains year and dash/en-dash
			for (let i = 0; i < textContents.length; i++) {
				const text = textContents[i];
				if (text.match(/\d{4}/) && (text.includes('–') || text.includes('-') || text.includes(' bis '))) {
					edu.dates = text.replace(/\s*·\s*\d+\s*(Monate|Monat|Jahre|Jahr)\s*$/, '').trim();
					break;
				}
			}
			
			// Try to find school logo
			const img = entry.querySelector('img[data-loaded="true"]');
			if (img && img.src && !img.src.includes('data:')) {
				edu.logo = img.src;
			}
			
			if (edu.school) {
				education.push(edu);
			}
		});
		
		return education;
	})()`
	err := chromedp.Run(ctx, chromedp.Evaluate(eduScript, &eduData))

	if err != nil {
		return nil, fmt.Errorf("failed to extract education: %w", err)
	}

	log.Printf("DEBUG: Extracted %d education entries from JavaScript", len(eduData))

	for _, edu := range eduData {
		start, end := "", ""
		if dates, ok := edu["dates"]; ok && dates != "" {
			start, end = parseEducationDates(dates)
		}

		eduItem := models.LinkedInEducation{
			School:    edu["school"],
			Degree:    edu["degree"],
			StartDate: start,
			EndDate:   end,
		}

		if logo, ok := edu["logo"]; ok && logo != "" {
			eduItem.SchoolLogo = downloadImageAsBase64(logo)
		}

		education = append(education, eduItem)
	}

	return education, nil
}

// extractSkillsData extracts skills from the details page
func (l *LinkedInScraper) extractSkillsData(ctx context.Context, baseURL string) ([]string, error) {
	skillsURL := baseURL + "/details/skills/"
	log.Printf("Extracting skills from: %s", skillsURL)

	if err := chromedp.Run(ctx, chromedp.Navigate(skillsURL)); err != nil {
		return nil, fmt.Errorf("failed to navigate to skills page: %w", err)
	}

	// Wait for main element
	waitCtx, waitCancel := context.WithTimeout(ctx, 15*time.Second)
	defer waitCancel()
	if err := chromedp.Run(waitCtx, chromedp.WaitVisible(`main`, chromedp.ByQuery)); err != nil {
		log.Printf("Warning: timeout waiting for skills page: %v", err)
	}

	// Wait additional time for lazy-loaded content
	time.Sleep(5 * time.Second)

	// Scroll aggressively to trigger lazy loading
	for i := 0; i < 10; i++ {
		_ = chromedp.Run(ctx, chromedp.Evaluate(`window.scrollBy(0, 500)`, nil))
		time.Sleep(300 * time.Millisecond)
	}
	time.Sleep(3 * time.Second)

	// DEBUG: Log the page structure
	var debugHTMLSkills string
	debugScriptSkills := `(function() {
		const main = document.querySelector('main');
		if (!main) return 'No main element found';
		
		const testIds = [];
		document.querySelectorAll('[data-testid]').forEach(function(el) {
			testIds.push(el.getAttribute('data-testid'));
		});
		
		const pTags = [];
		document.querySelectorAll('main p').forEach(function(p, i) {
			if (i < 30) pTags.push(p.textContent.trim().substring(0, 80));
		});
		
		const listItems = [];
		document.querySelectorAll('[role="listitem"]').forEach(function(li, i) {
			if (i < 10) listItems.push(li.textContent.trim().substring(0, 100));
		});
		
		return JSON.stringify({
			testIds: testIds,
			allPTags: pTags,
			listItems: listItems
		}, null, 2);
	})()`
	_ = chromedp.Run(ctx, chromedp.Evaluate(debugScriptSkills, &debugHTMLSkills))
	log.Printf("DEBUG: Skills page structure: %s", debugHTMLSkills)

	// Extract skills using JavaScript
	var skillData []string
	skillsScript := `(function() {
		const skills = [];
		const seen = new Set();
		
		// Look for skills section - try multiple selectors
		const skillsSection = document.querySelector('[data-testid*="Skills"]') ||
		                      document.querySelector('[data-view-name*="skill"]') ||
		                      document.querySelector('main');
		
		if (!skillsSection) {
			console.log('No skills section found');
			return skills;
		}
		
		// Find all list items (skills are usually in list items)
		const listItems = skillsSection.querySelectorAll('[role="listitem"]');
		
		listItems.forEach(function(item) {
			// Get the first p element which usually contains the skill name
			const pElements = item.querySelectorAll('p');
			if (pElements.length > 0) {
				const skillName = pElements[0].textContent.trim();
				// Filter out non-skill text
				if (skillName && !seen.has(skillName) && skillName.length > 1 && skillName.length < 100 && 
				    !skillName.includes('·') && !skillName.includes('@') && 
				    !skillName.includes('Warum') && !skillName.includes('Anzeige') &&
				    !skillName.includes('Deutsch') && !skillName.match(/^\d/)) {
					skills.push(skillName);
					seen.add(skillName);
				}
			}
		});
		
		// If no skills found via list items, try all p elements in main
		if (skills.length === 0) {
			const allP = document.querySelectorAll('main p');
			allP.forEach(function(p) {
				const text = p.textContent.trim();
				// Skills are usually short, single words or phrases
				if (text && !seen.has(text) && text.length > 1 && text.length < 50 &&
				    !text.includes('·') && !text.includes('@') && !text.includes(' ') &&
				    !text.includes('Warum') && !text.includes('Anzeige') &&
				    !text.includes('Deutsch') && !text.match(/^\d/)) {
					skills.push(text);
					seen.add(text);
				}
			});
		}
		
		return skills;
	})()`
	err := chromedp.Run(ctx, chromedp.Evaluate(skillsScript, &skillData))

	if err != nil {
		return nil, fmt.Errorf("failed to extract skills: %w", err)
	}

	log.Printf("DEBUG: Extracted %d skills from JavaScript", len(skillData))

	return skillData, nil
}

// parseDateRange parses LinkedIn date ranges like "Nov. 2025–Heute · 4 Monate"
func parseDateRange(dateRange string) (string, string) {
	// Split on the middle dot or dash
	parts := strings.Split(dateRange, "–")
	if len(parts) < 2 {
		parts = strings.Split(dateRange, "-")
	}
	if len(parts) < 2 {
		return "", ""
	}

	start := strings.TrimSpace(parts[0])
	end := strings.TrimSpace(strings.Split(parts[1], "·")[0])

	// Convert to YYYY-MM format
	start = convertToYYYYMM(start)
	if strings.Contains(strings.ToLower(end), "heute") || strings.Contains(strings.ToLower(end), "present") {
		end = "Present"
	} else {
		end = convertToYYYYMM(end)
	}

	return start, end
}

// parseEducationDates parses education date ranges (usually just years)
func parseEducationDates(dates string) (string, string) {
	// Education dates are usually like "2020 - 2024" or just "2020"
	parts := strings.Split(dates, "-")
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(dates), strings.TrimSpace(dates)
}

// convertToYYYYMM converts various date formats to YYYY-MM
func convertToYYYYMM(date string) string {
	date = strings.TrimSpace(date)

	germanMonths := map[string]string{
		"jan.": "01", "feb.": "02", "mär.": "03", "apr.": "04",
		"mai": "05", "jun.": "06", "jul.": "07", "aug.": "08",
		"sep.": "09", "sept.": "09", "okt.": "10", "nov.": "11", "dez.": "12",
	}

	englishMonths := map[string]string{
		"jan": "01", "january": "01", "feb": "02", "february": "02",
		"mar": "03", "march": "03", "apr": "04", "april": "04",
		"may": "05", "jun": "06", "june": "06", "jul": "07", "july": "07",
		"aug": "08", "august": "08", "sep": "09", "september": "09",
		"oct": "10", "october": "10", "nov": "11", "november": "11",
		"dec": "12", "december": "12",
	}

	parts := strings.Fields(date)
	if len(parts) >= 2 {
		month := strings.ToLower(parts[0])
		year := parts[1]

		if monthNum, ok := germanMonths[month]; ok {
			return year + "-" + monthNum
		}
		if monthNum, ok := englishMonths[month]; ok {
			return year + "-" + monthNum
		}
	}

	// If just a year
	if len(date) == 4 {
		return date
	}

	return date
}

// cleanProfileURL removes query parameters and trailing slashes from the profile URL
func cleanProfileURL(url string) string {
	if idx := strings.Index(url, "?"); idx != -1 {
		url = url[:idx]
	}
	url = strings.TrimSuffix(url, "/")
	return url
}
