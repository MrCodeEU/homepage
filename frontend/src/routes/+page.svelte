<script lang="ts">
	import { onMount } from 'svelte';
	import { getLinkedInData, getProjects, getStravaData, type LinkedInData, type Project, type StravaData } from '$lib/api';
	import { LogoAnimation } from '$lib/components';
	import logoSvg from '$lib/assets/logo.svg?raw';

	let linkedIn: LinkedInData | null = null;
	let projects: Project[] = [];
	let strava: StravaData | null = null;
	let loading = true;
	let error: string | null = null;

	// Format date from YYYY-MM to readable format
	function formatDate(dateStr: string): string {
		if (!dateStr || dateStr === 'Present') return 'Present';
		const [year, month] = dateStr.split('-');
		if (!month) return year;
		const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
		return `${monthNames[parseInt(month) - 1]} ${year}`;
	}

	// Format distance in km
	function formatDistance(meters: number): string {
		return (meters / 1000).toFixed(1);
	}

	// Format time in minutes or hours
	function formatTime(seconds: number): string {
		const hours = Math.floor(seconds / 3600);
		const minutes = Math.floor((seconds % 3600) / 60);
		if (hours > 0) return `${hours}h ${minutes}m`;
		return `${minutes}m`;
	}

	// Format pace (min/km)
	function formatPace(pace: number): string {
		if (!pace || pace <= 0) return '-';
		const mins = Math.floor(pace);
		const secs = Math.round((pace - mins) * 60);
		return `${mins}:${secs.toString().padStart(2, '0')}/km`;
	}

	// Get display name (first and last name only)
	function getDisplayName(fullName: string | undefined): string {
		if (!fullName) return 'Michael Reinegger';
		const parts = fullName.split(' ');
		if (parts.length <= 2) return fullName;
		// Return first and last name only
		return `${parts[0]} ${parts[1]}`;
	}

	// Language color mapping for GitHub
	function getLanguageColor(language: string): string {
		const colors: Record<string, string> = {
			'TypeScript': '#3178c6',
			'JavaScript': '#f1e05a',
			'Python': '#3572A5',
			'Go': '#00ADD8',
			'Rust': '#dea584',
			'Java': '#b07219',
			'C++': '#f34b7d',
			'C': '#555555',
			'C#': '#178600',
			'Ruby': '#701516',
			'PHP': '#4F5D95',
			'Swift': '#ffac45',
			'Kotlin': '#A97BFF',
			'Svelte': '#ff3e00',
			'Vue': '#41b883',
			'HTML': '#e34c26',
			'CSS': '#563d7c',
			'Shell': '#89e051',
			'Dockerfile': '#384d54'
		};
		return colors[language] || '#8b8b8b';
	}

	onMount(async () => {
		try {
			[linkedIn, projects, strava] = await Promise.all([
				getLinkedInData(),
				getProjects(),
				getStravaData()
			]);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load data';
			console.error('Error loading data:', err);
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>{linkedIn?.profile.name || 'Michael Reinegger'} - Portfolio</title>
	<meta name="description" content={linkedIn?.profile.headline || 'Software Engineer Portfolio'} />
</svelte:head>

<!-- Background Logo Animation -->
<LogoAnimation
	svgContent={logoSvg}
	animationDuration={0.8}
	reverseDuration={0.3}
	perspective={1200}
	zIndex={-1}
	logoSize={400}
	scrollThreshold={15}
	windIntensity={1}
	pathCurviness={1.2}
/>

<main class="page">
	{#if loading}
		<section class="hero">
			<div class="loading">
				<div class="spinner"></div>
				<p>Loading...</p>
			</div>
		</section>
	{:else if error}
		<section class="hero">
			<div class="error-container">
				<h2>Error</h2>
				<p>{error}</p>
			</div>
		</section>
	{:else}
		<!-- Hero Section with Logo in center -->
		<section class="hero">
			<div class="hero-content">
				<h1>{getDisplayName(linkedIn?.profile.name)}</h1>
				<h2 class="headline">{linkedIn?.profile.headline || 'Software Engineer'}</h2>
				<p class="location">📍 {linkedIn?.profile.location || ''}</p>
				{#if linkedIn?.profile.summary}
					<p class="summary">{linkedIn.profile.summary}</p>
				{/if}
				<div class="scroll-indicator">
					<span>Scroll to explore</span>
					<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M12 5v14M5 12l7 7 7-7"/>
					</svg>
				</div>
			</div>
		</section>

		<!-- Skills Section -->
		{#if linkedIn?.skills && linkedIn.skills.length > 0}
			<section class="section skills-section">
				<h2 class="section-title">Skills</h2>
				<div class="skills-grid">
					{#each linkedIn.skills as skill}
						<span class="skill-tag">{skill}</span>
					{/each}
				</div>
			</section>
		{/if}

		<!-- Experience Section -->
		{#if linkedIn?.experience && linkedIn.experience.length > 0}
			<section class="section experience-section">
				<h2 class="section-title">Experience</h2>
				<div class="timeline">
					{#each linkedIn.experience as exp}
						<article class="timeline-item">
							<div class="timeline-marker"></div>
							<div class="timeline-content">
								<div class="timeline-header">
									{#if exp.company_logo}
										<img src={exp.company_logo} alt={exp.company} class="company-logo" />
									{:else}
										<div class="company-logo-placeholder">
											{exp.company ? exp.company.charAt(0) : '?'}
										</div>
									{/if}
									<div class="timeline-info">
										<h3>{exp.title}</h3>
										<p class="company">{exp.company}</p>
										<p class="meta">
											<span class="date">{formatDate(exp.start_date)} - {formatDate(exp.end_date)}</span>
											{#if exp.duration}
												<span class="duration">· {exp.duration}</span>
											{/if}
										</p>
										{#if exp.location}
											<p class="location-small">📍 {exp.location}</p>
										{/if}
									</div>
								</div>
								{#if exp.description}
									<p class="description">{exp.description}</p>
								{/if}
							</div>
						</article>
					{/each}
				</div>
			</section>
		{/if}

		<!-- Education Section -->
		{#if linkedIn?.education && linkedIn.education.length > 0}
			<section class="section education-section">
				<h2 class="section-title">Education</h2>
				<div class="education-grid">
					{#each linkedIn.education as edu}
						<article class="education-card">
							<div class="education-header">
								{#if edu.school_logo}
									<img src={edu.school_logo} alt={edu.school} class="school-logo" />
								{:else}
									<div class="school-logo-placeholder">
										{edu.school ? edu.school.charAt(0) : '?'}
									</div>
								{/if}
								<div class="education-info">
									<h3>{edu.school}</h3>
									<p class="degree">{edu.degree}{edu.field ? `, ${edu.field}` : ''}</p>
									<p class="date">{formatDate(edu.start_date)} - {formatDate(edu.end_date)}</p>
								</div>
							</div>
							{#if edu.description}
								<p class="description">{edu.description}</p>
							{/if}
						</article>
					{/each}
				</div>
			</section>
		{/if}

		<!-- Projects Section -->
		{#if projects && projects.length > 0}
			<section class="section projects-section">
				<h2 class="section-title">Projects</h2>
				<div class="projects-grid">
					{#each projects as project, idx}
						<article class="project-card">
							{#if project.images && project.images.length > 0}
								<div class="project-image-container">
									<img
										src={project.images[0]}
										alt={project.name}
										class="project-image"
										on:error={(e) => {
											const target = e.currentTarget;
											if (target instanceof HTMLImageElement) {
												target.style.display = 'none';
											}
										}}
									/>
									{#if project.featured}
										<span class="featured-badge">Featured</span>
									{/if}
								</div>
							{:else}
								<div class="project-image-placeholder" style="background: linear-gradient(135deg, hsl({idx * 45}, 70%, 60%), hsl({idx * 45 + 40}, 70%, 50%));">
									{#if project.featured}
										<span class="featured-badge">Featured</span>
									{/if}
								</div>
							{/if}
							<div class="project-content">
								<h3>
									<a href={project.url} target="_blank" rel="noopener noreferrer">
										{project.name}
									</a>
								</h3>
								<p class="description">{project.description || 'No description available'}</p>
								<div class="project-meta">
									{#if project.language}
										<span class="language">
											<span class="language-dot" style="background: {getLanguageColor(project.language)}"></span>
											{project.language}
										</span>
									{/if}
									<span class="stars">⭐ {project.stars}</span>
								</div>
								{#if project.topics && project.topics.length > 0}
									<div class="topics">
										{#each project.topics.slice(0, 4) as topic}
											<span class="topic-tag">{topic}</span>
										{/each}
									</div>
								{/if}
							</div>
						</article>
					{/each}
				</div>
			</section>
		{/if}

		<!-- Strava Section -->
		{#if strava}
			<section class="section strava-section">
				<h2 class="section-title">🏃 Running Stats</h2>
				
				<!-- Stats Overview -->
				<div class="stats-grid">
					<div class="stat-card">
						<div class="stat-icon">🏃</div>
						<p class="stat-value">{strava.total_stats.count}</p>
						<p class="stat-label">Total Runs</p>
					</div>
					<div class="stat-card">
						<div class="stat-icon">📏</div>
						<p class="stat-value">{formatDistance(strava.total_stats.distance)} km</p>
						<p class="stat-label">Total Distance</p>
					</div>
					<div class="stat-card">
						<div class="stat-icon">⏱️</div>
						<p class="stat-value">{formatTime(strava.total_stats.moving_time)}</p>
						<p class="stat-label">Total Time</p>
					</div>
					<div class="stat-card">
						<div class="stat-icon">⛰️</div>
						<p class="stat-value">{Math.round(strava.total_stats.elevation_gain)} m</p>
						<p class="stat-label">Elevation Gain</p>
					</div>
				</div>

				<!-- Year to Date Stats -->
				<div class="ytd-section">
					<h3>Year to Date</h3>
					<div class="ytd-stats">
						<span>{strava.year_to_date_stats.count} runs</span>
						<span>•</span>
						<span>{formatDistance(strava.year_to_date_stats.distance)} km</span>
						<span>•</span>
						<span>{formatTime(strava.year_to_date_stats.moving_time)}</span>
					</div>
				</div>

				<!-- Personal Records -->
				{#if strava.personal_records && strava.personal_records.length > 0}
					<div class="records-section">
						<h3>Personal Records</h3>
						<div class="records-grid">
							{#each strava.personal_records as record}
								<div class="record-card">
									<span class="record-type">{record.type}</span>
									<span class="record-time">{formatTime(record.time)}</span>
								</div>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Recent Activities -->
				{#if strava.recent_activities && strava.recent_activities.length > 0}
					<div class="recent-section">
						<h3>Recent Runs</h3>
						<div class="activities-list">
							{#each strava.recent_activities.slice(0, 5) as activity}
								<div class="activity-item">
									<div class="activity-info">
										<h4>{activity.name}</h4>
										<p class="activity-date">{new Date(activity.start_date).toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' })}</p>
									</div>
									<div class="activity-stats">
										<span class="activity-distance">{formatDistance(activity.distance)} km</span>
										<span class="activity-time">{formatTime(activity.moving_time)}</span>
										<span class="activity-pace">{formatPace(activity.average_pace)}</span>
									</div>
								</div>
							{/each}
						</div>
					</div>
				{/if}
			</section>
		{/if}

		<!-- Footer -->
		<footer class="footer">
			<p>Built with ❤️ using SvelteKit & Go</p>
			<div class="social-links">
				<a href="https://github.com/MrCodeEU" target="_blank" rel="noopener noreferrer">GitHub</a>
				<a href="https://linkedin.com/in/mrcodeeu" target="_blank" rel="noopener noreferrer">LinkedIn</a>
			</div>
		</footer>
	{/if}
</main>

<style>
	:global(html, body) {
		margin: 0;
		padding: 0;
		min-height: 100%;
	}

	:global(body) {
		font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
		background: linear-gradient(180deg, #0f0f1a 0%, #1a1a2e 50%, #16213e 100%);
		background-attachment: fixed;
		color: #e0e0e0;
	}

	.page {
		position: relative;
		z-index: 1;
	}

	/* Loading & Error States */
	.loading {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 1rem;
	}

	.spinner {
		width: 50px;
		height: 50px;
		border: 4px solid rgba(102, 126, 234, 0.2);
		border-top-color: #667eea;
		border-radius: 50%;
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.error-container {
		background: rgba(229, 62, 62, 0.1);
		border: 1px solid rgba(229, 62, 62, 0.3);
		padding: 2rem;
		border-radius: 12px;
		text-align: center;
	}

	.error-container h2 {
		color: #e53e3e;
		margin: 0 0 1rem;
	}

	/* Hero Section */
	.hero {
		min-height: 100vh;
		display: flex;
		align-items: center;
		justify-content: center;
		text-align: center;
		padding: 2rem;
		padding-top: 40vh;
	}

	.hero-content {
		max-width: 700px;
	}

	.hero h1 {
		font-size: clamp(2rem, 8vw, 3.5rem);
		margin: 0 0 0.5rem;
		background: linear-gradient(135deg, #667eea 0%, #764ba2 50%, #f093fb 100%);
		-webkit-background-clip: text;
		-webkit-text-fill-color: transparent;
		background-clip: text;
		line-height: 1.2;
	}

	.hero .headline {
		font-size: clamp(1.1rem, 4vw, 1.5rem);
		color: #a0a0a0;
		font-weight: 400;
		margin: 0 0 1rem;
	}

	.hero .location {
		color: #808080;
		font-size: 1rem;
		margin: 0 0 1.5rem;
	}

	.hero .summary {
		font-size: 1.1rem;
		color: #b0b0b0;
		line-height: 1.6;
		margin: 0 0 3rem;
	}

	.scroll-indicator {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.5rem;
		color: #667eea;
		animation: bounce 2s infinite;
	}

	.scroll-indicator span {
		font-size: 0.9rem;
	}

	@keyframes bounce {
		0%, 100% { transform: translateY(0); }
		50% { transform: translateY(10px); }
	}

	/* Section Styles */
	.section {
		padding: 4rem 2rem;
		max-width: 1200px;
		margin: 0 auto;
	}

	.section-title {
		text-align: center;
		font-size: clamp(1.5rem, 5vw, 2rem);
		margin: 0 0 3rem;
		color: #fff;
		background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
		-webkit-background-clip: text;
		-webkit-text-fill-color: transparent;
		background-clip: text;
	}

	/* Skills Section */
	.skills-grid {
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: 0.75rem;
	}

	.skill-tag {
		background: rgba(102, 126, 234, 0.15);
		color: #667eea;
		padding: 0.5rem 1.25rem;
		border-radius: 25px;
		font-size: 0.9rem;
		border: 1px solid rgba(102, 126, 234, 0.3);
		transition: all 0.3s ease;
	}

	.skill-tag:hover {
		background: rgba(102, 126, 234, 0.25);
		transform: translateY(-2px);
	}

	/* Experience Timeline */
	.timeline {
		position: relative;
		padding-left: 2rem;
	}

	.timeline::before {
		content: '';
		position: absolute;
		left: 0;
		top: 0;
		bottom: 0;
		width: 2px;
		background: linear-gradient(180deg, #667eea 0%, #764ba2 100%);
	}

	.timeline-item {
		position: relative;
		margin-bottom: 2rem;
	}

	.timeline-marker {
		position: absolute;
		left: -2rem;
		top: 0.5rem;
		width: 12px;
		height: 12px;
		border-radius: 50%;
		background: #667eea;
		transform: translateX(-50%);
		box-shadow: 0 0 0 4px rgba(102, 126, 234, 0.2);
	}

	.timeline-content {
		background: rgba(255, 255, 255, 0.03);
		border-radius: 16px;
		padding: 1.5rem;
		border: 1px solid rgba(255, 255, 255, 0.08);
		margin-left: 1rem;
	}

	.timeline-header {
		display: flex;
		gap: 1rem;
		align-items: flex-start;
	}

	.company-logo {
		width: 50px;
		height: 50px;
		border-radius: 8px;
		object-fit: cover;
		flex-shrink: 0;
	}

	.company-logo-placeholder {
		width: 50px;
		height: 50px;
		border-radius: 8px;
		background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
		display: flex;
		align-items: center;
		justify-content: center;
		font-weight: bold;
		font-size: 1.25rem;
		color: white;
		flex-shrink: 0;
	}

	.timeline-info h3 {
		margin: 0 0 0.25rem;
		color: #fff;
		font-size: 1.1rem;
	}

	.timeline-info .company {
		color: #667eea;
		margin: 0 0 0.25rem;
		font-weight: 500;
	}

	.timeline-info .meta {
		color: #808080;
		font-size: 0.85rem;
		margin: 0;
	}

	.timeline-info .location-small {
		color: #707070;
		font-size: 0.8rem;
		margin: 0.25rem 0 0;
	}

	.timeline-content .description {
		color: #a0a0a0;
		margin: 1rem 0 0;
		line-height: 1.6;
	}

	/* Education Section */
	.education-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
		gap: 1.5rem;
	}

	.education-card {
		background: rgba(255, 255, 255, 0.03);
		border-radius: 16px;
		padding: 1.5rem;
		border: 1px solid rgba(255, 255, 255, 0.08);
		transition: transform 0.3s ease;
	}

	.education-card:hover {
		transform: translateY(-4px);
	}

	.education-header {
		display: flex;
		gap: 1rem;
		align-items: flex-start;
	}

	.school-logo {
		width: 50px;
		height: 50px;
		border-radius: 8px;
		object-fit: cover;
		flex-shrink: 0;
	}

	.school-logo-placeholder {
		width: 50px;
		height: 50px;
		border-radius: 8px;
		background: linear-gradient(135deg, #764ba2 0%, #f093fb 100%);
		display: flex;
		align-items: center;
		justify-content: center;
		font-weight: bold;
		font-size: 1.25rem;
		color: white;
		flex-shrink: 0;
	}

	.education-info h3 {
		margin: 0 0 0.25rem;
		color: #fff;
		font-size: 1.1rem;
	}

	.education-info .degree {
		color: #667eea;
		margin: 0 0 0.25rem;
		font-weight: 500;
	}

	.education-info .date {
		color: #808080;
		font-size: 0.85rem;
		margin: 0;
	}

	.education-card .description {
		color: #a0a0a0;
		margin: 1rem 0 0;
		line-height: 1.6;
	}

	/* Projects Section */
	.projects-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
		gap: 1.5rem;
	}

	.project-card {
		background: rgba(255, 255, 255, 0.05);
		border-radius: 16px;
		overflow: hidden;
		backdrop-filter: blur(10px);
		border: 1px solid rgba(255, 255, 255, 0.1);
		transition: transform 0.3s ease, box-shadow 0.3s ease;
		display: flex;
		flex-direction: column;
	}

	.project-card:hover {
		transform: translateY(-8px);
		box-shadow: 0 20px 40px rgba(0, 0, 0, 0.3);
	}

	.project-image-container {
		position: relative;
		height: 180px;
		overflow: hidden;
	}

	.project-image {
		width: 100%;
		height: 100%;
		object-fit: cover;
		transition: transform 0.3s ease;
	}

	.project-card:hover .project-image {
		transform: scale(1.05);
	}

	.project-image-placeholder {
		height: 180px;
		position: relative;
	}

	.featured-badge {
		position: absolute;
		top: 12px;
		right: 12px;
		background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
		color: white;
		padding: 0.4rem 0.8rem;
		border-radius: 20px;
		font-size: 0.75rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.5px;
	}

	.project-content {
		padding: 1.5rem;
		flex: 1;
		display: flex;
		flex-direction: column;
	}

	.project-content h3 {
		margin: 0 0 0.75rem;
	}

	.project-content h3 a {
		color: #fff;
		text-decoration: none;
		transition: color 0.3s ease;
	}

	.project-content h3 a:hover {
		color: #667eea;
	}

	.project-content .description {
		color: #a0a0a0;
		font-size: 0.9rem;
		line-height: 1.6;
		flex: 1;
		margin: 0 0 1rem;
	}

	.project-meta {
		display: flex;
		justify-content: space-between;
		align-items: center;
		font-size: 0.85rem;
		color: #808080;
		margin-bottom: 1rem;
	}

	.language {
		display: flex;
		align-items: center;
		gap: 0.4rem;
	}

	.language-dot {
		width: 10px;
		height: 10px;
		border-radius: 50%;
	}

	.topics {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}

	.topic-tag {
		background: rgba(102, 126, 234, 0.15);
		color: #667eea;
		padding: 0.25rem 0.75rem;
		border-radius: 20px;
		font-size: 0.75rem;
	}

	/* Strava Section */
	.strava-section {
		background: rgba(252, 76, 2, 0.03);
		border-radius: 24px;
		margin: 2rem auto;
		border: 1px solid rgba(252, 76, 2, 0.1);
	}

	.stats-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
		gap: 1rem;
		margin-bottom: 2rem;
	}

	.stat-card {
		text-align: center;
		padding: 1.5rem 1rem;
		background: rgba(252, 76, 2, 0.1);
		border-radius: 16px;
		border: 1px solid rgba(252, 76, 2, 0.2);
	}

	.stat-icon {
		font-size: 1.5rem;
		margin-bottom: 0.5rem;
	}

	.stat-value {
		font-size: 1.75rem;
		font-weight: bold;
		color: #fc4c02;
		margin: 0;
	}

	.stat-label {
		margin: 0.25rem 0 0;
		color: #a0a0a0;
		font-size: 0.85rem;
	}

	.ytd-section {
		text-align: center;
		margin-bottom: 2rem;
	}

	.ytd-section h3 {
		color: #fff;
		margin: 0 0 0.5rem;
		font-size: 1.1rem;
	}

	.ytd-stats {
		display: flex;
		justify-content: center;
		gap: 0.75rem;
		color: #a0a0a0;
		font-size: 0.9rem;
	}

	.records-section {
		margin-bottom: 2rem;
	}

	.records-section h3 {
		color: #fff;
		margin: 0 0 1rem;
		font-size: 1.1rem;
	}

	.records-grid {
		display: flex;
		flex-wrap: wrap;
		gap: 0.75rem;
	}

	.record-card {
		background: rgba(255, 255, 255, 0.05);
		padding: 0.75rem 1.25rem;
		border-radius: 12px;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.25rem;
	}

	.record-type {
		color: #808080;
		font-size: 0.8rem;
		text-transform: uppercase;
	}

	.record-time {
		color: #fc4c02;
		font-weight: bold;
	}

	.recent-section h3 {
		color: #fff;
		margin: 0 0 1rem;
		font-size: 1.1rem;
	}

	.activities-list {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.activity-item {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 1rem;
		background: rgba(255, 255, 255, 0.03);
		border-radius: 12px;
		border-left: 3px solid #fc4c02;
		flex-wrap: wrap;
		gap: 0.5rem;
	}

	.activity-info h4 {
		margin: 0 0 0.25rem;
		color: #fff;
		font-size: 1rem;
	}

	.activity-date {
		color: #808080;
		font-size: 0.8rem;
		margin: 0;
	}

	.activity-stats {
		display: flex;
		gap: 1rem;
		font-size: 0.9rem;
	}

	.activity-distance {
		color: #fc4c02;
		font-weight: 500;
	}

	.activity-time {
		color: #a0a0a0;
	}

	.activity-pace {
		color: #808080;
	}

	/* Footer */
	.footer {
		padding: 3rem 2rem;
		text-align: center;
		color: #606060;
		border-top: 1px solid rgba(255, 255, 255, 0.05);
		margin-top: 2rem;
	}

	.footer p {
		margin: 0 0 1rem;
	}

	.social-links {
		display: flex;
		justify-content: center;
		gap: 2rem;
	}

	.social-links a {
		color: #667eea;
		text-decoration: none;
		transition: color 0.3s ease;
	}

	.social-links a:hover {
		color: #764ba2;
	}

	/* Mobile Responsiveness */
	@media (max-width: 768px) {
		.hero {
			min-height: 90vh;
			padding: 1.5rem;
		}

		.section {
			padding: 3rem 1.5rem;
		}

		.timeline {
			padding-left: 1.5rem;
		}

		.timeline-content {
			margin-left: 0.5rem;
			padding: 1rem;
		}

		.timeline-header {
			flex-direction: column;
			gap: 0.75rem;
		}

		.company-logo,
		.company-logo-placeholder,
		.school-logo,
		.school-logo-placeholder {
			width: 40px;
			height: 40px;
		}

		.education-grid {
			grid-template-columns: 1fr;
		}

		.projects-grid {
			grid-template-columns: 1fr;
		}

		.stats-grid {
			grid-template-columns: repeat(2, 1fr);
		}

		.activity-item {
			flex-direction: column;
			align-items: flex-start;
		}

		.activity-stats {
			width: 100%;
			justify-content: space-between;
		}
	}

	@media (max-width: 480px) {
		.hero h1 {
			font-size: 1.75rem;
		}

		.hero .headline {
			font-size: 1rem;
		}

		.stats-grid {
			grid-template-columns: 1fr 1fr;
		}

		.stat-card {
			padding: 1rem;
		}

		.stat-value {
			font-size: 1.4rem;
		}

		.ytd-stats {
			flex-wrap: wrap;
		}
	}
</style>
