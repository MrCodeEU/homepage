<script lang="ts">
	import { onMount } from 'svelte';
	import type { LinkedInData, Project, StravaData, StravaDiscipline } from '$lib/api';
	import type { PageData } from './$types';
	// LogoAnimation lazy-loaded after initial paint to avoid pulling GSAP (~132KB) into the critical path
	import type LogoAnimationType from '$lib/components/LogoAnimation.svelte';
	let LogoAnimation = $state<typeof LogoAnimationType | null>(null);
	import {
		Background,
		Card,
		Badge,
		Icon,
		Chip,
		Footer,
		Carousel,
		Avatar,
		Navbar,
		ThemeToggle,
		Pagination,
		Tabs
	} from 'mljr-svelte';
	import logoSvg from '$lib/assets/logo.svg?raw';

	let { data }: { data: PageData } = $props();
	const linkedIn = $derived(data.linkedIn as LinkedInData | null);
	const projects = $derived(data.projects as Project[]);
	const strava = $derived(data.strava as StravaData | null);

	// Owner name shown in the hero — first + last only, bypasses LinkedIn scraping quirks
	const HERO_NAME = 'Michael Reinegger';

	// Navbar items
	const navItems = [
		{ label: 'Skills', href: '#skills', icon: 'mdi:code-tags' },
		{ label: 'Experience', href: '#experience', icon: 'mdi:briefcase' },
		{ label: 'Education', href: '#education', icon: 'mdi:school' },
		{ label: 'Projects', href: '#projects', icon: 'mdi:folder-multiple' },
		{ label: 'Running', href: '#running', icon: 'mdi:run' }
	];

	// School logo overrides (LinkedIn can't scrape these)
	const schoolLogos: Record<string, string> = {
		'Johannes Kepler Universität Linz': '',
		'HTL Steyr': ''
	};
	function getSchoolLogo(edu: { school: string; school_logo?: string }): string {
		return edu.school_logo || schoolLogos[edu.school] || '';
	}

	// Skill chip styling: icon + color variant
	type ChipVariant = 'default' | 'primary' | 'secondary' | 'accent' | 'success' | 'warning' | 'error' | 'outline';
	type SkillInfo = { icon?: string; variant: ChipVariant };

	const SKILL_MAP: Record<string, SkillInfo> = {
		// Languages
		TypeScript: { icon: 'mdi:language-typescript', variant: 'primary' },
		JavaScript: { icon: 'mdi:language-javascript', variant: 'warning' },
		Python: { icon: 'mdi:language-python', variant: 'secondary' },
		Go: { icon: 'mdi:language-go', variant: 'accent' },
		'Rust (Programmiersprache)': { icon: 'mdi:language-rust', variant: 'warning' },
		Rust: { icon: 'mdi:language-rust', variant: 'warning' },
		Java: { icon: 'mdi:language-java', variant: 'error' },
		'C#': { icon: 'mdi:language-csharp', variant: 'secondary' },
		'C++': { icon: 'mdi:language-cpp', variant: 'secondary' },
		Kotlin: { icon: 'mdi:language-kotlin', variant: 'secondary' },
		// Web frontend
		HTML: { icon: 'mdi:language-html5', variant: 'error' },
		'Cascading Style Sheets (CSS)': { icon: 'mdi:language-css3', variant: 'primary' },
		CSS: { icon: 'mdi:language-css3', variant: 'primary' },
		Angular: { icon: 'mdi:angular', variant: 'error' },
		AngularJS: { icon: 'mdi:angular', variant: 'error' },
		Svelte: { icon: 'mdi:code-tags', variant: 'error' },
		'.NET-Framework': { icon: 'mdi:dot-net', variant: 'secondary' },
		'ASP.NET': { icon: 'mdi:dot-net', variant: 'secondary' },
		// DevOps / tools
		Docker: { icon: 'mdi:docker', variant: 'accent' },
		Git: { icon: 'mdi:git', variant: 'warning' },
		'Kontinuierliche Integration': { icon: 'mdi:refresh', variant: 'accent' },
		'IT-Infrastruktur': { icon: 'mdi:server', variant: 'accent' },
		'IT-Management': { icon: 'mdi:monitor-dashboard', variant: 'default' },
		Softwareentwicklung: { icon: 'mdi:code-braces', variant: 'primary' },
		// Security / networking
		Cybersecurity: { icon: 'mdi:shield-lock', variant: 'error' },
		Netzwerksicherheit: { icon: 'mdi:shield-network', variant: 'error' },
		Netzwerkadministration: { icon: 'mdi:lan', variant: 'accent' }
	};

	// Cycle of fallback variants for unknown skills
	const FALLBACK_VARIANTS: ChipVariant[] = ['primary', 'secondary', 'accent', 'success', 'warning', 'outline'];
	function getSkillInfo(skill: string, index: number): SkillInfo {
		if (SKILL_MAP[skill]) return SKILL_MAP[skill];
		return { variant: FALLBACK_VARIANTS[index % FALLBACK_VARIANTS.length] };
	}

	// Icon mapping for project link types
	// Downscale picsum.photos URLs to match display size (cards are ~331×186)
	function optimizeImageUrl(url: string): string {
		return url.replace(
			/^(https:\/\/picsum\.photos\/seed\/[^/]+)\/\d+\/\d+$/,
			'$1/400/225'
		);
	}

	function getLinkIcon(name: string, customIcon?: string): string {
		if (customIcon) return customIcon;
		const nameLower = name.toLowerCase();
		if (nameLower.includes('github')) return 'mdi:github';
		if (nameLower.includes('live') || nameLower.includes('prod')) return 'mdi:web';
		if (nameLower.includes('staging') || nameLower.includes('dev') || nameLower.includes('test'))
			return 'mdi:flask';
		if (nameLower.includes('docs') || nameLower.includes('documentation')) return 'mdi:book-open';
		if (nameLower.includes('demo')) return 'mdi:play';
		if (nameLower.includes('api')) return 'mdi:api';
		if (nameLower.includes('download')) return 'mdi:download';
		return 'mdi:link';
	}

	// Activity tabs — computed in the template inside {#if strava} where it's narrowed
	const DISCIPLINE_ICONS: Record<string, string> = {
		cycling: '🚴',
		training: '🏋️'
	};

	// Pagination — responsive: 4 on mobile, 6 on tablet, 9 on desktop
	let projectsPerPage = $state(9);
	let currentPage = $state(1);
	const pagedProjects = $derived(
		projects.slice((currentPage - 1) * projectsPerPage, currentPage * projectsPerPage)
	);
	const totalPages = $derived(Math.ceil(projects.length / projectsPerPage));

	function formatDate(dateStr: string): string {
		if (!dateStr || dateStr === 'Present') return 'Present';
		const [year, month] = dateStr.split('-');
		if (!month) return year;
		const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
		return `${monthNames[parseInt(month) - 1]} ${year}`;
	}

	function formatDistance(meters: number): string {
		return (meters / 1000).toFixed(1);
	}

	function formatTime(seconds: number): string {
		const hours = Math.floor(seconds / 3600);
		const minutes = Math.floor((seconds % 3600) / 60);
		if (hours > 0) return `${hours}h ${minutes}m`;
		return `${minutes}m`;
	}

	function formatPace(pace: number): string {
		if (!pace || pace <= 0) return '-';
		const mins = Math.floor(pace);
		const secs = Math.round((pace - mins) * 60);
		return `${mins}:${secs.toString().padStart(2, '0')}/km`;
	}

	function getLanguageColor(language: string): string {
		const colors: Record<string, string> = {
			TypeScript: '#3178c6',
			JavaScript: '#f1e05a',
			Python: '#3572A5',
			Go: '#00ADD8',
			Rust: '#dea584',
			Java: '#b07219',
			'C++': '#f34b7d',
			C: '#555555',
			'C#': '#178600',
			Ruby: '#701516',
			PHP: '#4F5D95',
			Swift: '#ffac45',
			Kotlin: '#A97BFF',
			Svelte: '#ff3e00',
			Vue: '#41b883',
			HTML: '#e34c26',
			CSS: '#563d7c',
			Shell: '#89e051',
			Dockerfile: '#384d54'
		};
		return colors[language] || '#8b8b8b';
	}

	// Update projects-per-page based on viewport width + lazy-load LogoAnimation/GSAP
	onMount(() => {
		const update = () => {
			const w = window.innerWidth;
			const ppp = w < 640 ? 4 : w < 1024 ? 6 : 9;
			if (ppp !== projectsPerPage) {
				projectsPerPage = ppp;
				currentPage = 1;
			}
		};
		update();
		window.addEventListener('resize', update);

		// Defer GSAP + LogoAnimation until well after LCP to avoid blocking metrics.
		// requestIdleCallback waits for an idle period; the 3s timeout is a fallback
		// so the animation still appears even on busy pages.
		const loadLogo = () => {
			import('$lib/components/LogoAnimation.svelte').then((mod) => {
				LogoAnimation = mod.default;
			});
		};
		if ('requestIdleCallback' in window) {
			requestIdleCallback(loadLogo, { timeout: 3000 });
		} else {
			setTimeout(loadLogo, 2000);
		}

		return () => window.removeEventListener('resize', update);
	});
</script>

<svelte:head>
	<title>{linkedIn?.profile.name || 'Michael Reinegger'} - Portfolio</title>
	<meta name="description" content={linkedIn?.profile.headline || 'Software Engineer Portfolio'} />
	{#if linkedIn?.profile.photo_url}
		<link rel="preload" href={linkedIn.profile.photo_url} as="image" fetchpriority="high" />
	{/if}
</svelte:head>

<!-- Sticky Navbar -->
<Navbar variant="sticky" items={navItems}>
	{#snippet logo()}
		<div class="flex items-center gap-2" style="color: var(--mljr-primary-500)">
			<Icon icon="mdi:code-braces" size={22} />
			<span class="font-bold text-base">MR</span>
		</div>
	{/snippet}
	{#snippet actions()}
		<ThemeToggle />
	{/snippet}
</Navbar>

<!-- Background with pattern (z-index -1) -->
<Background pattern="dots" opacity={0.06} size={18} />

<!-- Background Logo Animation (z-index -1) — lazy-loaded after first paint -->
{#if LogoAnimation}
	{@const LazyLogo = LogoAnimation}
	<LazyLogo
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
{/if}

<main class="page-container">
	<!-- ═══ HERO ═══ -->
		<section id="hero" style="min-height: calc(100vh - 5rem); display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 2rem 0;">
			<div style="display: flex; flex-direction: column; align-items: center; width: 100%;">
				<div class="transparent-card" style="max-width: 56rem; width: 100%;">
					<!-- flex-wrap: wrap = side-by-side on wide, stacked on narrow -->
					<div class="hero-card-inner" style="display: flex; flex-wrap: wrap; align-items: center; justify-content: center;">

						<!-- Avatar – fixed width, never shrinks -->
						{#if linkedIn?.profile.photo_url}
							<div style="flex-shrink: 0;">
								<Avatar
									src={linkedIn.profile.photo_url}
									alt={linkedIn.profile.name || 'Profile'}
									size="2xl"
									ring="primary"
									shape="circle"
								/>
							</div>
						{/if}

						<!-- Text – grows to fill, wraps below avatar on narrow screens -->
						<div style="flex: 1; min-width: min(100%, 280px);">
							<h1
								style="font-size: clamp(2rem, 5vw, 3.75rem); font-weight: 800; line-height: 1.1; padding-bottom: 0.15em; margin-bottom: 0.6rem; background: linear-gradient(to right, #f97316, #a855f7); -webkit-background-clip: text; background-clip: text; -webkit-text-fill-color: transparent; color: transparent;"
							>
								{HERO_NAME}
							</h1>

							<p style="font-size: 1.15rem; font-weight: 500; margin-bottom: 0.625rem; color: var(--mljr-text-secondary);">
								{linkedIn?.profile.headline || 'Software Engineer'}
							</p>

							{#if linkedIn?.profile.location}
								<p style="display: flex; align-items: center; gap: 0.25rem; margin-bottom: 1.25rem; font-size: 0.875rem; font-weight: 600; color: var(--mljr-primary-500);">
									<Icon icon="mdi:map-marker" size={15} />
									{linkedIn.profile.location}
								</p>
							{/if}

							{#if linkedIn?.profile.summary}
								<p style="line-height: 1.7; color: var(--mljr-text-secondary); font-size: 0.9rem; margin-bottom: 1.75rem;">
									{linkedIn.profile.summary}
								</p>
							{/if}

							<div class="scroll-indicator" style="display: flex; align-items: center; gap: 0.25rem; opacity: 0.55; color: var(--mljr-primary-500);">
								<span style="font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.12em;">Scroll to explore</span>
								<Icon icon="mdi:chevron-down" size={18} />
							</div>
						</div>
					</div>
				</div>
			</div>
		</section>

		<!-- ═══ MAIN CONTENT: Experience (2/3) + Skills/Education sidebar (1/3) ═══ -->
		<div class="main-content-grid pb-16">

			<!-- LEFT COLUMN: Experience -->
			<div>
				{#if linkedIn?.experience && linkedIn.experience.length > 0}
					<section id="experience">
						<h2 class="section-heading">
							<span style="color: var(--mljr-primary-500)"><Icon icon="mdi:briefcase" size={28} /></span>
							Experience
						</h2>
						<div class="experience-grid">
							{#each linkedIn.experience as exp}
								<div class="exp-card transparent-card">
									<!-- Company logo + header -->
									<div class="exp-card-header">
										{#if exp.company_logo}
											<img src={exp.company_logo} alt={exp.company} class="company-logo" loading="lazy" />
										{:else}
											<div class="company-logo logo-placeholder">
												<Icon icon="mdi:office-building" size={24} />
											</div>
										{/if}
										<div class="flex-1 min-w-0">
											<p class="font-semibold text-sm leading-tight truncate" style="color: var(--mljr-text)">{exp.title}</p>
											<p class="text-xs font-medium mt-0.5" style="color: var(--mljr-primary-600)">{exp.company}</p>
											<p class="text-xs mt-0.5" style="color: var(--mljr-text-muted)">
												{formatDate(exp.start_date)} – {formatDate(exp.end_date)}
												{#if exp.duration}<span class="opacity-70"> · {exp.duration}</span>{/if}
											</p>
											{#if exp.location}
												<p class="text-xs mt-0.5 flex items-center gap-0.5" style="color: var(--mljr-text-muted)">
													<Icon icon="mdi:map-marker" size={11} />{exp.location}
												</p>
											{/if}
										</div>
									</div>
									{#if exp.description}
										<p class="exp-description">{exp.description}</p>
									{/if}
								</div>
							{/each}
						</div>
					</section>
				{/if}
			</div>

			<!-- RIGHT COLUMN: Skills + Education -->
			<div class="sidebar-col">
				<!-- Skills -->
				{#if linkedIn?.skills && linkedIn.skills.length > 0}
					<section id="skills">
						<h2 class="section-heading">
							<span style="color: var(--mljr-primary-500)"><Icon icon="mdi:code-tags" size={28} /></span>
							Skills
						</h2>
						<Card shadow="md" class="transparent-card">
							{#snippet children()}
								<div class="p-4">
									<div class="flex flex-wrap gap-1.5">
										{#each linkedIn?.skills ?? [] as skill, i}
											{@const info = getSkillInfo(skill, i)}
											<Chip variant={info.variant} icon={info.icon} size="sm">
												{skill}
											</Chip>
										{/each}
									</div>
								</div>
							{/snippet}
						</Card>
					</section>
				{/if}

				<!-- Education -->
				{#if linkedIn?.education && linkedIn.education.length > 0}
					<section id="education" class="mt-8">
						<h2 class="section-heading">
							<span style="color: var(--mljr-secondary-500)"><Icon icon="mdi:school" size={28} /></span>
							Education
						</h2>
						<div class="flex flex-col gap-3">
							{#each linkedIn.education as edu}
								<div class="edu-card transparent-card">
									<div class="edu-card-inner">
										{#if getSchoolLogo(edu)}
											<img src={getSchoolLogo(edu)} alt={edu.school} class="school-logo" loading="lazy" />
										{:else}
											<div class="school-logo logo-placeholder secondary">
												<Icon icon="mdi:school" size={20} />
											</div>
										{/if}
										<div class="flex-1 min-w-0">
											<p class="font-semibold text-sm" style="color: var(--mljr-text)">{edu.school}</p>
											<p class="text-xs font-medium mt-0.5" style="color: var(--mljr-secondary-600)">
												{edu.degree}{edu.field ? `, ${edu.field}` : ''}
											</p>
											<p class="text-xs mt-0.5" style="color: var(--mljr-text-muted)">
												{formatDate(edu.start_date)} – {formatDate(edu.end_date)}
											</p>
											{#if edu.description}
												<p class="text-xs mt-1 leading-relaxed line-clamp-2" style="color: var(--mljr-text-secondary)">{edu.description}</p>
											{/if}
										</div>
									</div>
								</div>
							{/each}
						</div>
					</section>
				{/if}
			</div>
		</div>

		<!-- ═══ PROJECTS ═══ -->
		{#if projects.length > 0}
			<section id="projects" class="py-10">
				<h2 class="section-heading mb-8">
					<span style="color: var(--mljr-accent-500)"><Icon icon="mdi:folder-multiple" size={28} /></span>
					Projects
				</h2>

				<div class="projects-grid">
					{#each pagedProjects as project, idx}
						<div class="project-card transparent-card">
							<!-- Image / Carousel -->
							<div class="project-media">
								{#if project.images && project.images.length > 0}
									<Carousel autoplay interval={3000} variant="default" class="project-carousel">
										{#snippet children()}
											{#each project.images as image, i}
												<li class="mljr-carousel-item">
													<img
														src={optimizeImageUrl(image)}
														alt="{project.name} - Image {i + 1}"
														class="w-full h-full object-cover"
														width="400"
														height="225"
														loading="lazy"
													/>
												</li>
											{/each}
										{/snippet}
									</Carousel>
								{:else}
									<div
										class="h-full flex items-center justify-center"
										style="background: linear-gradient(135deg, hsl({idx * 47}, 65%, 55%), hsl({idx * 47 + 45}, 65%, 45%));"
									>
										<Icon icon="mdi:folder-open" size={48} class="text-white opacity-40" />
									</div>
								{/if}
								{#if project.featured}
									<div class="project-featured-badge">
										<Badge variant="primary">Featured</Badge>
									</div>
								{/if}
							</div>

							<!-- Content -->
							<div class="project-content">
								<h3 class="font-semibold text-base mb-2 leading-tight" style="color: var(--mljr-text)">{project.name}</h3>
								<p class="text-sm leading-relaxed mb-3 flex-1" style="color: var(--mljr-text-secondary)">
									{project.description || 'No description available'}
								</p>

								{#if project.badges && project.badges.length > 0}
									<div class="flex flex-wrap gap-1 mb-3">
										{#each project.badges as badge}
											<img
												src={badge}
												alt="Badge"
												class="project-badge"
												loading="lazy"
												onerror={(e) => {
													const t = e.currentTarget;
													if (t instanceof HTMLImageElement) t.style.display = 'none';
												}}
											/>
										{/each}
									</div>
								{/if}

								<div class="flex items-center justify-between mb-3">
									{#if project.language}
										<span class="flex items-center gap-1.5 text-xs" style="color: var(--mljr-text-secondary)">
											<span class="language-dot" style="background: {getLanguageColor(project.language)}"></span>
											{project.language}
										</span>
									{:else}
										<span></span>
									{/if}
									<span class="flex items-center gap-1 text-xs" style="color: var(--mljr-text-muted)">
										<Icon icon="mdi:star" size={13} />
										{project.stars}
									</span>
								</div>

								{#if project.topics && project.topics.length > 0}
									<div class="flex flex-wrap gap-1 mb-3">
										{#each project.topics.slice(0, 4) as topic}
											<Chip variant="secondary" size="sm">{topic}</Chip>
										{/each}
									</div>
								{/if}
							</div>

							<!-- Links footer -->
							<div class="project-footer">
								<a href={project.url} target="_blank" rel="noopener noreferrer" class="project-link-btn primary">
									<Icon icon="mdi:github" size={14} />
									GitHub
								</a>
								{#if project.links}
									{#each project.links as link}
										<a href={link.url} target="_blank" rel="noopener noreferrer" class="project-link-btn">
											<Icon icon={getLinkIcon(link.name, link.icon)} size={14} />
											{link.name}
										</a>
									{/each}
								{/if}
							</div>
						</div>
					{/each}
				</div>

				{#if totalPages > 1}
					<div class="pagination-wrapper">
						<Pagination
							{totalPages}
							bind:currentPage
							size="lg"
							onchange={(p: number) => { currentPage = p; }}
						/>
					</div>
				{/if}
			</section>
		{/if}

		<!-- ═══ ACTIVITIES ═══ -->
		{#if strava}
			<section id="running" class="py-10">
				<h2 class="section-heading mb-8">
					<span style="color: var(--mljr-primary-500)"><Icon icon="mdi:lightning-bolt" size={28} /></span>
					Activities
				</h2>

				<Card shadow="md" class="transparent-card">
					{#snippet children()}
						{@const actTabs = [
							{ id: 'running', label: '🏃 Running', badge: strava!.total_stats.count },
							...(strava!.disciplines ?? []).map((d: StravaDiscipline) => ({
								id: d.type,
								label: `${DISCIPLINE_ICONS[d.type] ?? '⚡'} ${d.label}`,
								badge: d.count
							}))
						]}
						<div class="p-6">
							<Tabs tabs={actTabs} variant="pills">
								{#snippet children({ activeTab }: { activeTab: string })}
									{@const s = strava!}

									<!-- ── Running tab ── -->
									{#if activeTab === 'running'}
										<!-- 4 main stats -->
										<div class="strava-stats-grid" style="margin-bottom: 1.5rem; margin-top: 1rem;">
											<div class="stat-block">
												<span style="color: var(--mljr-primary-500)"><Icon icon="mdi:run-fast" size={26} /></span>
												<span class="stat-value">{s.total_stats.count}</span>
												<span class="stat-label">Total Runs</span>
											</div>
											<div class="stat-block">
												<span style="color: var(--mljr-secondary-500)"><Icon icon="mdi:map-marker-distance" size={26} /></span>
												<span class="stat-value">{formatDistance(s.total_stats.distance)} km</span>
												<span class="stat-label">Distance</span>
											</div>
											<div class="stat-block">
												<span style="color: var(--mljr-text-secondary)"><Icon icon="mdi:clock-outline" size={26} /></span>
												<span class="stat-value">{formatTime(s.total_stats.moving_time)}</span>
												<span class="stat-label">Time</span>
											</div>
											<div class="stat-block">
												<span style="color: var(--mljr-accent-500)"><Icon icon="mdi:image-filter-hdr" size={26} /></span>
												<span class="stat-value">{Math.round(s.total_stats.elevation_gain)} m</span>
												<span class="stat-label">Elevation</span>
											</div>
										</div>

										<!-- Year to Date -->
										<div class="ytd-card" style="margin-bottom: 1.5rem;">
											<p style="font-size: 0.7rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.08em; color: var(--mljr-text-muted); text-align: center; margin-bottom: 0.75rem;">Year to Date</p>
											<div style="display: flex; justify-content: center; gap: 2rem; flex-wrap: wrap;">
												<span style="display: flex; align-items: center; gap: 0.375rem; font-size: 0.875rem; color: var(--mljr-text-secondary);">
													<Icon icon="mdi:run" size={16} />{s.year_to_date_stats.count} runs
												</span>
												<span style="display: flex; align-items: center; gap: 0.375rem; font-size: 0.875rem; color: var(--mljr-text-secondary);">
													<Icon icon="mdi:map-marker-distance" size={16} />{formatDistance(s.year_to_date_stats.distance)} km
												</span>
												<span style="display: flex; align-items: center; gap: 0.375rem; font-size: 0.875rem; color: var(--mljr-text-secondary);">
													<Icon icon="mdi:clock-outline" size={16} />{formatTime(s.year_to_date_stats.moving_time)}
												</span>
											</div>
										</div>

										<!-- Personal Records -->
										{#if s.personal_records && s.personal_records.length > 0}
											<div style="margin-bottom: 1.5rem;">
												<p style="font-size: 0.7rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.08em; color: var(--mljr-text-muted); display: flex; align-items: center; gap: 0.5rem; margin-bottom: 0.75rem;">
													<span style="color: var(--mljr-warning)"><Icon icon="mdi:trophy" size={16} /></span>
													Personal Records
												</p>
												<div style="display: flex; flex-wrap: wrap; gap: 0.75rem;">
													{#each s.personal_records as record}
														<div class="pr-block">
															<span style="font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.06em; color: var(--mljr-text-muted);">{record.type.replace('_', ' ')}</span>
															<span style="font-size: 1.15rem; font-weight: 700; color: var(--mljr-warning-dark);">{formatTime(record.time)}</span>
														</div>
													{/each}
												</div>
											</div>
										{/if}

										<!-- Recent Runs -->
										{#if s.recent_activities && s.recent_activities.length > 0}
											<div>
												<p style="font-size: 0.7rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.08em; color: var(--mljr-text-muted); display: flex; align-items: center; gap: 0.5rem; margin-bottom: 0.75rem;">
													<Icon icon="mdi:history" size={16} />
													Recent Runs
												</p>
												<div style="display: flex; flex-direction: column; gap: 0.5rem;">
													{#each s.recent_activities.slice(0, 5) as activity}
														<div class="run-row">
															<div style="min-width: 0;">
																<p style="font-size: 0.875rem; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--mljr-text);">{activity.name}</p>
																<p style="font-size: 0.75rem; color: var(--mljr-text-muted);">
																	{new Date(activity.start_date).toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' })}
																</p>
															</div>
															<div style="display: flex; gap: 1rem; font-size: 0.875rem; flex-shrink: 0;">
																<span style="font-weight: 600; color: var(--mljr-primary-600);">{formatDistance(activity.distance)} km</span>
																<span style="color: var(--mljr-text-secondary);">{formatTime(activity.moving_time)}</span>
																<span style="color: var(--mljr-text-muted);">{formatPace(activity.average_pace)}</span>
															</div>
														</div>
													{/each}
												</div>
											</div>
										{/if}
									{/if}

									<!-- ── Cycling / Training discipline tabs ── -->
									{#each s.disciplines ?? [] as disc (disc.type)}
										{#if activeTab === disc.type}
											{@const hasDist = disc.total_distance > 0}
											{@const hasHR = disc.avg_heartrate > 0}

											<!-- Summary stats -->
											<div class="strava-stats-grid" style="margin-bottom: 1.5rem; margin-top: 1rem;">
												<div class="stat-block">
													<span style="color: var(--mljr-primary-500)"><Icon icon={disc.type === 'cycling' ? 'mdi:bike' : 'mdi:weight-lifter'} size={26} /></span>
													<span class="stat-value">{disc.count}</span>
													<span class="stat-label">Sessions</span>
												</div>
												<div class="stat-block">
													<span style="color: var(--mljr-secondary-500)"><Icon icon="mdi:clock-outline" size={26} /></span>
													<span class="stat-value">{formatTime(disc.total_time)}</span>
													<span class="stat-label">Total Time</span>
												</div>
												{#if hasDist}
													<div class="stat-block">
														<span style="color: var(--mljr-accent-500)"><Icon icon="mdi:map-marker-distance" size={26} /></span>
														<span class="stat-value">{formatDistance(disc.total_distance)} km</span>
														<span class="stat-label">Distance</span>
													</div>
												{/if}
												{#if hasHR}
													<div class="stat-block">
														<span style="color: var(--mljr-error-500)"><Icon icon="mdi:heart-pulse" size={26} /></span>
														<span class="stat-value">{Math.round(disc.avg_heartrate)} bpm</span>
														<span class="stat-label">Avg HR</span>
													</div>
												{/if}
											</div>

											<!-- Recent activities -->
											{#if disc.activities && disc.activities.length > 0}
												<div>
													<p style="font-size: 0.7rem; font-weight: 700; text-transform: uppercase; letter-spacing: 0.08em; color: var(--mljr-text-muted); display: flex; align-items: center; gap: 0.5rem; margin-bottom: 0.75rem;">
														<Icon icon="mdi:history" size={16} />
														Recent Sessions
													</p>
													<div style="display: flex; flex-direction: column; gap: 0.5rem;">
														{#each disc.activities as activity}
															<div class="run-row">
																<div style="min-width: 0;">
																	<p style="font-size: 0.875rem; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--mljr-text);">{activity.name}</p>
																	<p style="font-size: 0.75rem; color: var(--mljr-text-muted);">
																		{new Date(activity.start_date).toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' })}
																		{#if activity.type}<span style="margin-left: 0.375rem; opacity: 0.7;">· {activity.type}</span>{/if}
																	</p>
																</div>
																<div style="display: flex; gap: 0.75rem; font-size: 0.875rem; flex-shrink: 0; flex-wrap: wrap; justify-content: flex-end;">
																	{#if activity.distance > 100}
																		<span style="font-weight: 600; color: var(--mljr-primary-600);">{formatDistance(activity.distance)} km</span>
																	{/if}
																	<span style="color: var(--mljr-text-secondary);">{formatTime(activity.moving_time)}</span>
																	{#if activity.average_heartrate && activity.average_heartrate > 0}
																		<span style="color: var(--mljr-error-400); display: flex; align-items: center; gap: 0.2rem;">
																			<Icon icon="mdi:heart" size={12} />{Math.round(activity.average_heartrate)}
																		</span>
																	{/if}
																	{#if activity.calories && activity.calories > 0}
																		<span style="color: var(--mljr-warning);">{Math.round(activity.calories)} kcal</span>
																	{/if}
																</div>
															</div>
														{/each}
													</div>
												</div>
											{/if}
										{/if}
									{/each}

								{/snippet}
							</Tabs>
						</div>
					{/snippet}
				</Card>
			</section>
		{/if}

		<!-- Footer -->
		<Footer
			description="Software Engineer & CS Master Student · Built with SvelteKit \n Data from LinkedIn, GitHub & Strava — updated automatically every 48 hours."
			sections={[
				{
					title: 'Pages',
					links: [
						{ label: 'About', href: '/about' },
						{ label: 'GitHub', href: 'https://github.com/MrCodeEU', external: true }
					]
				},
				{
					title: 'Legal',
					links: [
						{ label: 'Impressum', href: '/impressum' },
						{ label: 'Datenschutz', href: '/datenschutz' }
					]
				}
			]}
			socials={[{ icon: 'github', href: 'https://github.com/MrCodeEU', label: 'GitHub' }]}
			legalLinks={[
				{ label: 'Impressum', href: '/impressum' },
				{ label: 'Datenschutz', href: '/datenschutz' }
			]}
			copyright={`\u00A9 ${new Date().getFullYear()} Michael Reinegger. All rights reserved.`}
		>
			{#snippet logo()}
				<div style="display: flex; align-items: center; gap: 0.5rem; color: var(--mljr-primary-500);">
					<Icon icon="mdi:code-braces" size={24} />
					<span style="font-weight: 700; font-size: 1.1rem;">Michael Reinegger</span>
				</div>
			{/snippet}
		</Footer>
</main>
