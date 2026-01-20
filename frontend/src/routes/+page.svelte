<script lang="ts">
	import { onMount } from 'svelte';
	import { getLinkedInData, getProjects, getStravaData, type LinkedInData, type Project, type StravaData, type ProjectLink } from '$lib/api';
	import { LogoAnimation, ImageCarousel } from '$lib/components';
	import logoSvg from '$lib/assets/logo.svg?raw';

	// Icon mapping for link types - supports custom icon or auto-detection from name
	// Custom icons use format: "mdi:icon-name" for Material Design Icons
	function getLinkIcon(name: string, customIcon?: string): string {
		// Use custom icon if provided (strip mdi: prefix for internal use)
		if (customIcon) {
			return customIcon.replace(/^mdi:/, '');
		}
		// Auto-detect from name as fallback
		const nameLower = name.toLowerCase();
		if (nameLower.includes('github')) return 'github';
		if (nameLower.includes('live') || nameLower.includes('prod')) return 'globe';
		if (nameLower.includes('staging') || nameLower.includes('dev') || nameLower.includes('test')) return 'flask';
		if (nameLower.includes('docs') || nameLower.includes('documentation')) return 'book';
		if (nameLower.includes('demo')) return 'play';
		if (nameLower.includes('api')) return 'api';
		if (nameLower.includes('download')) return 'download';
		return 'link';
	}

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
	windIntensity={2.5}
	pathCurviness={3}
/>

<main class="relative z-10">
	{#if loading}
		<section class="hero min-h-screen flex items-center justify-center text-center p-8 pt-[30vh]">
			<div class="flex flex-col items-center justify-center gap-4">
				<div class="spinner"></div>
				<p class="text-detroit-text-muted">Loading...</p>
			</div>
		</section>
	{:else if error}
		<section class="hero min-h-screen flex items-center justify-center text-center p-8 pt-[30vh]">
			<div class="error-container detroit-clip-md">
				<h2 class="text-detroit-accent uppercase tracking-[0.2em] font-light mb-4">Error</h2>
				<p class="text-detroit-text-muted">{error}</p>
			</div>
		</section>
	{:else}
		<!-- Hero Section with Logo in center -->
		<section class="hero min-h-screen flex items-center justify-center text-center p-8 pt-[30vh] relative overflow-hidden">
			<!-- Decorative triangles -->
			<div class="detroit-triangle-up absolute top-[10%] right-[5%] w-[200px] h-[200px] detroit-animate-float"></div>
			<div class="detroit-triangle-down absolute bottom-[15%] left-[5%] w-[150px] h-[150px] detroit-animate-float" style="animation-direction: reverse; animation-duration: 8s;"></div>

			<div class="hero-content max-w-[800px] relative z-10 p-12 bg-detroit-bg-card backdrop-blur-[10px] detroit-clip-lg border border-detroit-border-primary">
				<h1 class="detroit-text-gradient-primary text-[clamp(2.5rem,8vw,4.5rem)] mb-3 leading-[1.1] font-extralight tracking-[0.1em] uppercase">
					{getDisplayName(linkedIn?.profile.name)}
				</h1>
				<h2 class="text-[clamp(0.9rem,2.5vw,1.2rem)] text-white/55 font-light mb-6 tracking-[0.15em] uppercase">
					{linkedIn?.profile.headline || 'Software Engineer'}
				</h2>
				<p class="text-detroit-primary/70 text-sm mb-6 tracking-[0.08em]">
					{linkedIn?.profile.location || ''}
				</p>
				{#if linkedIn?.profile.summary}
					<p class="text-detroit-text-muted text-[0.95rem] leading-[1.9] mb-10 font-light">
						{linkedIn.profile.summary}
					</p>
				{/if}
				<div class="scroll-indicator flex flex-col items-center gap-3 text-detroit-primary/60">
					<span class="text-[0.7rem] uppercase tracking-[0.25em]">Scroll to explore</span>
					<svg class="opacity-60" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M12 5v14M5 12l7 7 7-7"/>
					</svg>
				</div>
			</div>
		</section>

		<!-- Skills Section -->
		{#if linkedIn?.skills && linkedIn.skills.length > 0}
			<section class="section py-24 px-8 max-w-[1200px] mx-auto relative detroit-divider">
				<h2 class="detroit-title text-[clamp(2rem,5vw,3rem)] mb-14 text-white/95 font-extralight tracking-[0.2em] uppercase">Skills</h2>
				<div class="flex flex-wrap gap-4">
					{#each linkedIn.skills as skill}
						<span class="detroit-tag py-[0.7rem] px-7 text-xs uppercase tracking-[0.1em] font-normal">{skill}</span>
					{/each}
				</div>
			</section>
		{/if}

		<!-- Experience Section -->
		{#if linkedIn?.experience && linkedIn.experience.length > 0}
			<section class="section py-24 px-8 max-w-[1200px] mx-auto relative detroit-divider">
				<h2 class="detroit-title text-[clamp(2rem,5vw,3rem)] mb-14 text-white/95 font-extralight tracking-[0.2em] uppercase">Experience</h2>
				<div class="detroit-timeline">
					{#each linkedIn.experience as exp}
						<article class="timeline-item relative mb-12">
							<div class="detroit-timeline-marker top-6"></div>
							<div class="timeline-content detroit-card detroit-clip-md detroit-corners-md p-8">
								<!-- Corner accent borders -->
								<span class="corner-tr"></span>
								<span class="corner-bl"></span>
								<div class="flex gap-6 items-start max-md:flex-col max-md:gap-4">
									{#if exp.company_logo}
										<img src={exp.company_logo} alt={exp.company} class="company-logo w-[55px] h-[55px] detroit-clip-xs object-cover shrink-0" />
									{:else}
										<div class="company-logo-placeholder w-[55px] h-[55px] detroit-clip-xs flex items-center justify-center font-normal text-xl text-detroit-primary shrink-0 border border-detroit-primary/30">
											{exp.company ? exp.company.charAt(0) : '?'}
										</div>
									{/if}
									<div>
										<h3 class="mb-1 text-white/95 text-lg font-normal tracking-[0.04em]">{exp.title}</h3>
										<p class="text-detroit-primary mb-1 font-normal text-[0.95rem]">{exp.company}</p>
										<p class="text-white/40 text-sm tracking-[0.06em]">
											<span>{formatDate(exp.start_date)} - {formatDate(exp.end_date)}</span>
											{#if exp.duration}
												<span class="ml-1">· {exp.duration}</span>
											{/if}
										</p>
										{#if exp.location}
											<p class="text-detroit-primary/50 text-xs mt-1">{exp.location}</p>
										{/if}
									</div>
								</div>
								{#if exp.description}
									<p class="text-white/50 mt-6 leading-[1.8] text-sm">{exp.description}</p>
								{/if}
							</div>
						</article>
					{/each}
				</div>
			</section>
		{/if}

		<!-- Education Section -->
		{#if linkedIn?.education && linkedIn.education.length > 0}
			<section class="section py-24 px-8 max-w-[1200px] mx-auto relative detroit-divider">
				<h2 class="detroit-title text-[clamp(2rem,5vw,3rem)] mb-14 text-white/95 font-extralight tracking-[0.2em] uppercase">Education</h2>
				<div class="grid grid-cols-[repeat(auto-fit,minmax(340px,1fr))] gap-8 max-md:grid-cols-1">
					{#each linkedIn.education as edu}
						<article class="detroit-card-secondary detroit-clip-md detroit-corners-md detroit-corners-md-secondary p-8 education-card">
							<!-- Corner accent borders -->
							<span class="corner-tr"></span>
							<span class="corner-bl"></span>
							<div class="flex gap-6 items-start">
								{#if edu.school_logo}
									<img src={edu.school_logo} alt={edu.school} class="w-[55px] h-[55px] detroit-clip-xs object-cover shrink-0" />
								{:else}
									<div class="w-[55px] h-[55px] detroit-clip-xs flex items-center justify-center font-normal text-xl text-detroit-secondary shrink-0 border border-detroit-secondary/30 bg-gradient-to-br from-detroit-secondary/20 to-detroit-accent/20">
										{edu.school ? edu.school.charAt(0) : '?'}
									</div>
								{/if}
								<div>
									<h3 class="mb-1 text-white/95 text-lg font-normal">{edu.school}</h3>
									<p class="text-detroit-secondary mb-1 font-normal text-[0.95rem]">{edu.degree}{edu.field ? `, ${edu.field}` : ''}</p>
									<p class="text-white/40 text-sm">{formatDate(edu.start_date)} - {formatDate(edu.end_date)}</p>
								</div>
							</div>
							{#if edu.description}
								<p class="text-white/50 mt-6 leading-[1.8] text-sm">{edu.description}</p>
							{/if}
						</article>
					{/each}
				</div>
			</section>
		{/if}

		<!-- Projects Section -->
		{#if projects && projects.length > 0}
			<section class="section py-24 px-8 max-w-[1400px] mx-auto relative detroit-divider">
				<h2 class="detroit-title text-[clamp(2rem,5vw,3rem)] mb-14 text-white/95 font-extralight tracking-[0.2em] uppercase">Projects</h2>
				<div class="grid grid-cols-[repeat(auto-fill,minmax(420px,1fr))] gap-10 max-md:grid-cols-1">
					{#each projects as project, idx}
						<article class="detroit-card detroit-clip-lg detroit-border-frame detroit-corners-lg flex flex-col">
							<!-- Corner accent borders -->
							<span class="corner-tr"></span>
							<span class="corner-bl"></span>
							{#if project.images && project.images.length > 0}
								<div class="relative">
									<ImageCarousel
										images={project.images}
										alt={project.name}
										interval={3000}
										height="280px"
									/>
									{#if project.featured}
										<span class="detroit-badge absolute top-5 left-0 py-2 px-7 pl-5 text-[0.6rem] font-medium uppercase tracking-[0.2em] z-30 bg-black/60 backdrop-blur-sm">Featured</span>
									{/if}
								</div>
							{:else}
								<div class="h-[280px] relative detroit-clip-lg project-image-container" style="background: linear-gradient(135deg, hsl({idx * 45}, 70%, 60%), hsl({idx * 45 + 40}, 70%, 50%));">
									{#if project.featured}
										<span class="detroit-badge absolute top-5 left-0 py-2 px-7 pl-5 text-[0.6rem] font-medium uppercase tracking-[0.2em] z-10 bg-black/60 backdrop-blur-sm">Featured</span>
									{/if}
								</div>
							{/if}
							<div class="p-8 flex-1 flex flex-col">
								<h3 class="mb-4">
									<span class="text-white/95 font-normal tracking-[0.04em] text-xl">
										{project.name}
									</span>
								</h3>
								<p class="text-detroit-text-muted text-sm leading-[1.8] flex-1 mb-4">{project.description || 'No description available'}</p>

								<!-- Badges Section -->
								{#if project.badges && project.badges.length > 0}
									<div class="flex flex-wrap gap-2 mb-4">
										{#each project.badges as badge}
											<img
												src={badge}
												alt="Badge"
												class="h-5 object-contain"
												on:error={(e) => {
													const target = e.currentTarget;
													if (target instanceof HTMLImageElement) {
														target.style.display = 'none';
													}
												}}
											/>
										{/each}
									</div>
								{/if}

								<div class="flex justify-between items-center text-sm text-white/40 mb-4 pt-4 border-t border-detroit-border-primary">
									{#if project.language}
										<span class="flex items-center gap-2">
											<span class="w-2.5 h-2.5 detroit-clip-diamond" style="background: {getLanguageColor(project.language)}"></span>
											{project.language}
										</span>
									{/if}
									<span class="flex items-center gap-1">
										<svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
											<path d="M12 .587l3.668 7.568 8.332 1.151-6.064 5.828 1.48 8.279-7.416-3.967-7.417 3.967 1.481-8.279-6.064-5.828 8.332-1.151z"/>
										</svg>
										{project.stars}
									</span>
								</div>
								{#if project.topics && project.topics.length > 0}
									<div class="flex flex-wrap gap-2 mb-6">
										{#each project.topics.slice(0, 4) as topic}
											<span class="detroit-tag-secondary py-1 px-4 text-[0.6rem] uppercase tracking-[0.08em]">{topic}</span>
										{/each}
									</div>
								{/if}

								<!-- Links Section -->
								<div class="flex flex-wrap gap-3 pt-4 border-t border-detroit-border-primary">
									<!-- GitHub link (always shown) -->
									<a
										href={project.url}
										target="_blank"
										rel="noopener noreferrer"
										class="project-link-btn flex items-center gap-2 py-2 px-4 bg-white/[0.03] border border-detroit-primary/30 detroit-clip-xs text-xs uppercase tracking-[0.08em] text-detroit-primary/70 hover:text-detroit-primary hover:bg-detroit-primary/10 hover:border-detroit-primary/50 transition-all duration-300"
									>
										<svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
											<path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
										</svg>
										GitHub
									</a>

									<!-- Custom links from .portfolio -->
									{#if project.links && project.links.length > 0}
										{#each project.links as link}
											{@const iconType = getLinkIcon(link.name, link.icon)}
											<a
												href={link.url}
												target="_blank"
												rel="noopener noreferrer"
												class="project-link-btn flex items-center gap-2 py-2 px-4 bg-white/[0.03] border border-detroit-accent/30 detroit-clip-xs text-xs uppercase tracking-[0.08em] text-detroit-accent/70 hover:text-detroit-accent hover:bg-detroit-accent/10 hover:border-detroit-accent/50 transition-all duration-300"
											>
												{#if iconType === 'globe'}
													<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
														<circle cx="12" cy="12" r="10"/>
														<path d="M2 12h20M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
													</svg>
												{:else if iconType === 'flask'}
													<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
														<path d="M9 3h6M10 3v6l-5 8.5a2 2 0 0 0 1.7 3h10.6a2 2 0 0 0 1.7-3L14 9V3"/>
														<path d="M8.5 14h7"/>
													</svg>
												{:else if iconType === 'book'}
													<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
														<path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/>
														<path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>
													</svg>
												{:else if iconType === 'play'}
													<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
														<polygon points="5 3 19 12 5 21 5 3"/>
													</svg>
												{:else if iconType === 'rocket-launch' || iconType === 'rocket'}
													<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
														<path d="M4.5 16.5c-1.5 1.26-2 5-2 5s3.74-.5 5-2c.71-.84.7-2.13-.09-2.91a2.18 2.18 0 0 0-2.91-.09z"/>
														<path d="m12 15-3-3a22 22 0 0 1 2-3.95A12.88 12.88 0 0 1 22 2c0 2.72-.78 7.5-6 11a22.35 22.35 0 0 1-4 2z"/>
														<path d="M9 12H4s.55-3.03 2-4c1.62-1.08 5 0 5 0"/>
														<path d="M12 15v5s3.03-.55 4-2c1.08-1.62 0-5 0-5"/>
													</svg>
												{:else if iconType === 'api'}
													<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
														<path d="M4 6h16M4 12h16M4 18h16"/>
													</svg>
												{:else if iconType === 'download'}
													<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
														<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
														<polyline points="7 10 12 15 17 10"/>
														<line x1="12" y1="15" x2="12" y2="3"/>
													</svg>
												{:else}
													<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
														<path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
														<path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
													</svg>
												{/if}
												{link.name}
											</a>
										{/each}
									{/if}
								</div>
							</div>
						</article>
					{/each}
				</div>
			</section>
		{/if}

		<!-- Strava Section -->
		{#if strava}
			<section class="strava-section detroit-clip-oct py-24 px-8 max-w-[1200px] mx-auto my-12 bg-detroit-bg-card-light backdrop-blur-[20px] border border-detroit-primary/15 relative">
				<h2 class="detroit-title text-[clamp(2rem,5vw,3rem)] mb-14 text-white/95 font-extralight tracking-[0.2em] uppercase">Running Stats</h2>

				<!-- Stats Overview -->
				<div class="grid grid-cols-[repeat(auto-fit,minmax(150px,1fr))] gap-5 mb-10">
					<div class="stat-card text-center py-7 px-4 bg-detroit-primary/5 border border-detroit-primary/15 detroit-clip-oct transition-all duration-300 hover:bg-detroit-primary/10 hover:-translate-y-1 hover:shadow-[0_10px_30px_rgba(0,212,255,0.1)]">
						<div class="text-2xl mb-2 opacity-80"></div>
						<p class="text-3xl font-light text-detroit-primary m-0 tracking-tight">{strava.total_stats.count}</p>
						<p class="mt-1 text-white/40 text-[0.7rem] uppercase tracking-[0.12em]">Total Runs</p>
					</div>
					<div class="stat-card text-center py-7 px-4 bg-detroit-primary/5 border border-detroit-primary/15 detroit-clip-oct transition-all duration-300 hover:bg-detroit-primary/10 hover:-translate-y-1 hover:shadow-[0_10px_30px_rgba(0,212,255,0.1)]">
						<div class="text-2xl mb-2 opacity-80"></div>
						<p class="text-3xl font-light text-detroit-primary m-0 tracking-tight">{formatDistance(strava.total_stats.distance)} km</p>
						<p class="mt-1 text-white/40 text-[0.7rem] uppercase tracking-[0.12em]">Total Distance</p>
					</div>
					<div class="stat-card text-center py-7 px-4 bg-detroit-primary/5 border border-detroit-primary/15 detroit-clip-oct transition-all duration-300 hover:bg-detroit-primary/10 hover:-translate-y-1 hover:shadow-[0_10px_30px_rgba(0,212,255,0.1)]">
						<div class="text-2xl mb-2 opacity-80"></div>
						<p class="text-3xl font-light text-detroit-primary m-0 tracking-tight">{formatTime(strava.total_stats.moving_time)}</p>
						<p class="mt-1 text-white/40 text-[0.7rem] uppercase tracking-[0.12em]">Total Time</p>
					</div>
					<div class="stat-card text-center py-7 px-4 bg-detroit-primary/5 border border-detroit-primary/15 detroit-clip-oct transition-all duration-300 hover:bg-detroit-primary/10 hover:-translate-y-1 hover:shadow-[0_10px_30px_rgba(0,212,255,0.1)]">
						<div class="text-2xl mb-2 opacity-80"></div>
						<p class="text-3xl font-light text-detroit-primary m-0 tracking-tight">{Math.round(strava.total_stats.elevation_gain)} m</p>
						<p class="mt-1 text-white/40 text-[0.7rem] uppercase tracking-[0.12em]">Elevation Gain</p>
					</div>
				</div>

				<!-- Year to Date Stats -->
				<div class="text-center mb-10 py-7 bg-detroit-primary/5 detroit-clip-hex-square border border-detroit-primary/15">
					<h3 class="text-white/80 mb-3 text-sm font-normal uppercase tracking-[0.2em]">Year to Date</h3>
					<div class="flex justify-center gap-8 text-white/50 text-sm flex-wrap">
						<span>{strava.year_to_date_stats.count} runs</span>
						<span class="w-1.5 h-1.5 bg-detroit-primary/60 detroit-clip-diamond self-center"></span>
						<span>{formatDistance(strava.year_to_date_stats.distance)} km</span>
						<span class="w-1.5 h-1.5 bg-detroit-primary/60 detroit-clip-diamond self-center"></span>
						<span>{formatTime(strava.year_to_date_stats.moving_time)}</span>
					</div>
				</div>

				<!-- Personal Records -->
				{#if strava.personal_records && strava.personal_records.length > 0}
					<div class="mb-10">
						<h3 class="text-white/80 mb-5 text-sm font-normal uppercase tracking-[0.12em]">Personal Records</h3>
						<div class="flex flex-wrap gap-4">
							{#each strava.personal_records as record}
								<div class="bg-white/[0.02] py-4 px-8 border border-detroit-primary/15 detroit-clip-skew flex flex-col items-center gap-1 transition-all duration-300 hover:bg-detroit-primary/5 hover:translate-x-1">
									<span class="text-white/40 text-[0.65rem] uppercase tracking-[0.12em]">{record.type}</span>
									<span class="text-detroit-primary font-normal text-lg">{formatTime(record.time)}</span>
								</div>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Recent Activities -->
				{#if strava.recent_activities && strava.recent_activities.length > 0}
					<div>
						<h3 class="text-white/80 mb-5 text-sm font-normal uppercase tracking-[0.12em]">Recent Runs</h3>
						<div class="flex flex-col gap-4">
							{#each strava.recent_activities.slice(0, 5) as activity}
								<div class="activity-item flex justify-between items-center py-5 px-6 bg-white/[0.02] border border-detroit-primary/15 detroit-clip-sm flex-wrap gap-4 transition-all duration-300 hover:bg-detroit-primary/[0.04] hover:border-detroit-primary/25 hover:translate-x-2 max-md:flex-col max-md:items-start">
									<div>
										<h4 class="mb-1 text-white/90 text-[0.95rem] font-normal">{activity.name}</h4>
										<p class="text-white/35 text-xs tracking-[0.04em]">{new Date(activity.start_date).toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' })}</p>
									</div>
									<div class="flex gap-8 text-sm max-md:w-full max-md:justify-between">
										<span class="text-detroit-primary font-normal">{formatDistance(activity.distance)} km</span>
										<span class="text-white/50">{formatTime(activity.moving_time)}</span>
										<span class="text-white/35">{formatPace(activity.average_pace)}</span>
									</div>
								</div>
							{/each}
						</div>
					</div>
				{/if}
			</section>
		{/if}

		<!-- Footer -->
		<footer class="py-20 px-8 text-center text-white/30 relative mt-16 detroit-divider">
			<p class="mb-8 text-sm tracking-[0.06em]">Built with SvelteKit & Go</p>
			<div class="flex justify-center gap-16 max-sm:gap-8">
				<a href="https://github.com/MrCodeEU" target="_blank" rel="noopener noreferrer" class="text-detroit-primary/70 no-underline transition-all duration-300 text-sm uppercase tracking-[0.15em] relative py-2 px-4 detroit-clip-skew hover:text-detroit-primary hover:bg-detroit-primary/5">GitHub</a>
				<a href="https://linkedin.com/in/mrcodeeu" target="_blank" rel="noopener noreferrer" class="text-detroit-primary/70 no-underline transition-all duration-300 text-sm uppercase tracking-[0.15em] relative py-2 px-4 detroit-clip-skew hover:text-detroit-primary hover:bg-detroit-primary/5">LinkedIn</a>
			</div>
		</footer>
	{/if}
</main>

<style>
	/* Minimal custom styles for pseudo-elements and complex animations */

	/* Spinner animation */
	.spinner {
		width: 60px;
		height: 60px;
		border: 2px solid rgba(0, 212, 255, 0.2);
		border-top-color: #00d4ff;
		clip-path: polygon(50% 0%, 100% 25%, 100% 75%, 50% 100%, 0% 75%, 0% 25%);
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	/* Error container gradient border */
	.error-container {
		background: rgba(164, 0, 84, 0.1);
		border: 1px solid rgba(164, 0, 84, 0.4);
		padding: 2.5rem;
		text-align: center;
		position: relative;
	}

	.error-container::before {
		content: '';
		position: absolute;
		top: 0;
		left: 0;
		right: 40px;
		height: 3px;
		background: linear-gradient(90deg, #a40054, #f28d1d, transparent);
	}

	/* Hero content gradient borders - inverted so colors meet at corners */
	.hero-content::before {
		content: '';
		position: absolute;
		top: 0;
		left: 0;
		right: 50px;
		height: 4px;
		background: linear-gradient(90deg, #00d4ff 0%, #00d4ff 50%, #f28d1d 80%, #f28d1d 100%);
	}

	.hero-content::after {
		content: '';
		position: absolute;
		bottom: 0;
		right: 0;
		left: 50px;
		height: 4px;
		background: linear-gradient(90deg, #a40054 0%, #a40054 20%, #f28d1d 50%, #f28d1d 100%);
	}

	/* Scroll indicator bounce */
	.scroll-indicator {
		animation: bounce 2s infinite;
	}

	@keyframes bounce {
		0%, 100% { transform: translateY(0); }
		50% { transform: translateY(12px); }
	}

	/* Timeline content gradient borders - colors meet at corner */
	.timeline-content::before {
		content: '';
		position: absolute;
		top: 0;
		left: 0;
		width: 4px;
		height: calc(100% - 40px);
		background: linear-gradient(180deg, #00d4ff 0%, #00d4ff 50%, #f28d1d 80%, #f28d1d 100%);
	}

	.timeline-content::after {
		content: '';
		position: absolute;
		top: 0;
		left: 0;
		right: 40px;
		height: 3px;
		background: linear-gradient(90deg, #00d4ff 0%, #00d4ff 50%, #f28d1d 80%, #f28d1d 100%);
	}

	/* Education card gradient borders - colors meet at corner */
	.education-card::before {
		content: '';
		position: absolute;
		top: 0;
		left: 0;
		right: 40px;
		height: 3px;
		background: linear-gradient(90deg, #f28d1d 0%, #f28d1d 50%, #a40054 80%, #a40054 100%);
	}

	.education-card::after {
		content: '';
		position: absolute;
		top: 40px;
		right: 0;
		width: 3px;
		height: calc(100% - 80px);
		background: linear-gradient(180deg, #a40054 0%, #a40054 30%, #f28d1d 70%, transparent 100%);
	}

	/* Project image container clip */
	.project-image-container {
		clip-path: polygon(0 0, calc(100% - 50px) 0, 100% 50px, 100% 100%, 0 100%);
	}

	/* Strava section gradient borders - using primary cyan color */
	.strava-section::before {
		content: '';
		position: absolute;
		top: 0;
		left: 35px;
		right: 35px;
		height: 3px;
		background: linear-gradient(90deg, transparent 0%, #00d4ff 20%, #00d4ff 60%, #00a8cc 80%, transparent 100%);
	}

	.strava-section::after {
		content: '';
		position: absolute;
		bottom: 0;
		left: 35px;
		right: 35px;
		height: 3px;
		background: linear-gradient(90deg, transparent 0%, #00a8cc 20%, #00d4ff 40%, #00d4ff 80%, transparent 100%);
	}

	/* Activity item left border - using primary cyan color */
	.activity-item::before {
		content: '';
		position: absolute;
		left: 0;
		top: 0;
		bottom: 25px;
		width: 4px;
		background: linear-gradient(180deg, #00d4ff 0%, #00d4ff 50%, #00a8cc 80%, transparent 100%);
	}

	/* Company logo placeholder gradient */
	.company-logo-placeholder {
		background: linear-gradient(135deg, rgba(0, 212, 255, 0.2) 0%, rgba(0, 136, 170, 0.2) 100%);
	}

	/* Stat card top border - using primary cyan color */
	.stat-card::before {
		content: '';
		position: absolute;
		top: 0;
		left: 15px;
		right: 15px;
		height: 2px;
		background: linear-gradient(90deg, transparent, rgba(0, 212, 255, 0.5), #00d4ff, rgba(0, 212, 255, 0.5), transparent);
	}

	/* Mobile adjustments */
	@media (max-width: 768px) {
		.hero-content {
			padding: 2rem;
			clip-path: polygon(0 0, calc(100% - 30px) 0, 100% 30px, 100% 100%, 30px 100%, 0 calc(100% - 30px));
		}

		.strava-section {
			clip-path: polygon(0 20px, 20px 0, calc(100% - 20px) 0, 100% 20px, 100% calc(100% - 20px), calc(100% - 20px) 100%, 20px 100%, 0 calc(100% - 20px));
		}

		.strava-section::before,
		.strava-section::after {
			left: 20px;
			right: 20px;
		}
	}

	@media (max-width: 480px) {
		.hero-content {
			clip-path: polygon(0 0, calc(100% - 20px) 0, 100% 20px, 100% 100%, 20px 100%, 0 calc(100% - 20px));
		}
	}
</style>
