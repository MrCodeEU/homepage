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
			<section class="section py-24 px-8 max-w-[1200px] mx-auto relative detroit-divider">
				<h2 class="detroit-title text-[clamp(2rem,5vw,3rem)] mb-14 text-white/95 font-extralight tracking-[0.2em] uppercase">Projects</h2>
				<div class="grid grid-cols-[repeat(auto-fill,minmax(340px,1fr))] gap-8 max-md:grid-cols-1">
					{#each projects as project, idx}
						<article class="detroit-card detroit-clip-lg detroit-border-frame detroit-corners-lg flex flex-col">
							<!-- Corner accent borders -->
							<span class="corner-tr"></span>
							<span class="corner-bl"></span>
							{#if project.images && project.images.length > 0}
								<div class="project-image-container relative h-[200px] overflow-hidden detroit-clip-lg">
									<img
										src={project.images[0]}
										alt={project.name}
										class="w-full h-full object-cover transition-all duration-600 saturate-[0.7] brightness-[0.9] hover:scale-110 hover:saturate-100 hover:brightness-100"
										on:error={(e) => {
											const target = e.currentTarget;
											if (target instanceof HTMLImageElement) {
												target.style.display = 'none';
											}
										}}
									/>
									{#if project.featured}
										<span class="detroit-badge absolute top-5 left-0 py-2 px-7 pl-5 text-[0.6rem] font-medium uppercase tracking-[0.2em] z-10">Featured</span>
									{/if}
								</div>
							{:else}
								<div class="h-[200px] relative detroit-clip-lg" style="background: linear-gradient(135deg, hsl({idx * 45}, 70%, 60%), hsl({idx * 45 + 40}, 70%, 50%));">
									{#if project.featured}
										<span class="detroit-badge absolute top-5 left-0 py-2 px-7 pl-5 text-[0.6rem] font-medium uppercase tracking-[0.2em] z-10">Featured</span>
									{/if}
								</div>
							{/if}
							<div class="p-8 flex-1 flex flex-col">
								<h3 class="mb-4">
									<a href={project.url} target="_blank" rel="noopener noreferrer" class="text-white/95 no-underline transition-colors duration-300 font-normal tracking-[0.04em] text-lg hover:text-detroit-primary">
										{project.name}
									</a>
								</h3>
								<p class="text-detroit-text-muted text-sm leading-[1.8] flex-1 mb-6">{project.description || 'No description available'}</p>
								<div class="flex justify-between items-center text-sm text-white/40 mb-4 pt-4 border-t border-detroit-border-primary">
									{#if project.language}
										<span class="flex items-center gap-2">
											<span class="w-2.5 h-2.5 detroit-clip-diamond" style="background: {getLanguageColor(project.language)}"></span>
											{project.language}
										</span>
									{/if}
									<span>{project.stars}</span>
								</div>
								{#if project.topics && project.topics.length > 0}
									<div class="flex flex-wrap gap-2">
										{#each project.topics.slice(0, 4) as topic}
											<span class="detroit-tag-secondary py-1 px-4 text-[0.6rem] uppercase tracking-[0.08em]">{topic}</span>
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
			<section class="strava-section detroit-clip-oct py-24 px-8 max-w-[1200px] mx-auto my-12 bg-detroit-bg-card-light backdrop-blur-[20px] border border-strava/10 relative">
				<h2 class="detroit-title text-[clamp(2rem,5vw,3rem)] mb-14 text-white/95 font-extralight tracking-[0.2em] uppercase">Running Stats</h2>

				<!-- Stats Overview -->
				<div class="grid grid-cols-[repeat(auto-fit,minmax(150px,1fr))] gap-5 mb-10">
					<div class="stat-card text-center py-7 px-4 bg-strava/5 border border-strava/10 detroit-clip-oct transition-all duration-300 hover:bg-strava/10 hover:-translate-y-1 hover:shadow-[0_10px_30px_rgba(252,76,2,0.1)]">
						<div class="text-2xl mb-2 opacity-80"></div>
						<p class="text-3xl font-light text-strava m-0 tracking-tight">{strava.total_stats.count}</p>
						<p class="mt-1 text-white/40 text-[0.7rem] uppercase tracking-[0.12em]">Total Runs</p>
					</div>
					<div class="stat-card text-center py-7 px-4 bg-strava/5 border border-strava/10 detroit-clip-oct transition-all duration-300 hover:bg-strava/10 hover:-translate-y-1 hover:shadow-[0_10px_30px_rgba(252,76,2,0.1)]">
						<div class="text-2xl mb-2 opacity-80"></div>
						<p class="text-3xl font-light text-strava m-0 tracking-tight">{formatDistance(strava.total_stats.distance)} km</p>
						<p class="mt-1 text-white/40 text-[0.7rem] uppercase tracking-[0.12em]">Total Distance</p>
					</div>
					<div class="stat-card text-center py-7 px-4 bg-strava/5 border border-strava/10 detroit-clip-oct transition-all duration-300 hover:bg-strava/10 hover:-translate-y-1 hover:shadow-[0_10px_30px_rgba(252,76,2,0.1)]">
						<div class="text-2xl mb-2 opacity-80"></div>
						<p class="text-3xl font-light text-strava m-0 tracking-tight">{formatTime(strava.total_stats.moving_time)}</p>
						<p class="mt-1 text-white/40 text-[0.7rem] uppercase tracking-[0.12em]">Total Time</p>
					</div>
					<div class="stat-card text-center py-7 px-4 bg-strava/5 border border-strava/10 detroit-clip-oct transition-all duration-300 hover:bg-strava/10 hover:-translate-y-1 hover:shadow-[0_10px_30px_rgba(252,76,2,0.1)]">
						<div class="text-2xl mb-2 opacity-80"></div>
						<p class="text-3xl font-light text-strava m-0 tracking-tight">{Math.round(strava.total_stats.elevation_gain)} m</p>
						<p class="mt-1 text-white/40 text-[0.7rem] uppercase tracking-[0.12em]">Elevation Gain</p>
					</div>
				</div>

				<!-- Year to Date Stats -->
				<div class="text-center mb-10 py-7 bg-strava/5 detroit-clip-hex">
					<h3 class="text-white/80 mb-3 text-sm font-normal uppercase tracking-[0.2em]">Year to Date</h3>
					<div class="flex justify-center gap-8 text-white/50 text-sm flex-wrap">
						<span>{strava.year_to_date_stats.count} runs</span>
						<span class="w-1.5 h-1.5 bg-strava/60 detroit-clip-diamond self-center"></span>
						<span>{formatDistance(strava.year_to_date_stats.distance)} km</span>
						<span class="w-1.5 h-1.5 bg-strava/60 detroit-clip-diamond self-center"></span>
						<span>{formatTime(strava.year_to_date_stats.moving_time)}</span>
					</div>
				</div>

				<!-- Personal Records -->
				{#if strava.personal_records && strava.personal_records.length > 0}
					<div class="mb-10">
						<h3 class="text-white/80 mb-5 text-sm font-normal uppercase tracking-[0.12em]">Personal Records</h3>
						<div class="flex flex-wrap gap-4">
							{#each strava.personal_records as record}
								<div class="bg-white/[0.02] py-4 px-8 border border-strava/10 detroit-clip-skew flex flex-col items-center gap-1 transition-all duration-300 hover:bg-strava/5 hover:translate-x-1">
									<span class="text-white/40 text-[0.65rem] uppercase tracking-[0.12em]">{record.type}</span>
									<span class="text-strava font-normal text-lg">{formatTime(record.time)}</span>
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
								<div class="activity-item flex justify-between items-center py-5 px-6 bg-white/[0.02] border border-strava/10 detroit-clip-sm flex-wrap gap-4 transition-all duration-300 hover:bg-strava/[0.04] hover:border-strava/20 hover:translate-x-2 max-md:flex-col max-md:items-start">
									<div>
										<h4 class="mb-1 text-white/90 text-[0.95rem] font-normal">{activity.name}</h4>
										<p class="text-white/35 text-xs tracking-[0.04em]">{new Date(activity.start_date).toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' })}</p>
									</div>
									<div class="flex gap-8 text-sm max-md:w-full max-md:justify-between">
										<span class="text-strava font-normal">{formatDistance(activity.distance)} km</span>
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

	/* Strava section gradient borders */
	.strava-section::before {
		content: '';
		position: absolute;
		top: 0;
		left: 35px;
		right: 35px;
		height: 3px;
		background: linear-gradient(90deg, transparent 0%, #fc4c02 20%, #fc4c02 60%, #f28d1d 80%, transparent 100%);
	}

	.strava-section::after {
		content: '';
		position: absolute;
		bottom: 0;
		left: 35px;
		right: 35px;
		height: 3px;
		background: linear-gradient(90deg, transparent 0%, #f28d1d 20%, #fc4c02 40%, #fc4c02 80%, transparent 100%);
	}

	/* Activity item left border */
	.activity-item::before {
		content: '';
		position: absolute;
		left: 0;
		top: 0;
		bottom: 25px;
		width: 4px;
		background: linear-gradient(180deg, #fc4c02 0%, #fc4c02 50%, #f28d1d 80%, transparent 100%);
	}

	/* Company logo placeholder gradient */
	.company-logo-placeholder {
		background: linear-gradient(135deg, rgba(0, 212, 255, 0.2) 0%, rgba(0, 136, 170, 0.2) 100%);
	}

	/* Stat card top border */
	.stat-card::before {
		content: '';
		position: absolute;
		top: 0;
		left: 15px;
		right: 15px;
		height: 2px;
		background: linear-gradient(90deg, transparent, rgba(252, 76, 2, 0.5), #fc4c02, rgba(252, 76, 2, 0.5), transparent);
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
