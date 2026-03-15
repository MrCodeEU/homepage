<script lang="ts">
	import { Background, AboutPage, Icon, Navbar, ThemeToggle } from 'mljr-svelte';
	import logoSvg from '$lib/assets/logo.svg?raw';

	const navItems = [{ label: '← Back', href: '/', icon: 'mdi:arrow-left' }];
</script>

<svelte:head>
	<title>About – Michael Reinegger</title>
	<meta name="description" content="About Michael Reinegger – Software Engineer & CS Master Student" />
</svelte:head>

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

<Background pattern="dots" opacity={0.06} size={18} />

<main class="page-container">
	<AboutPage
		companyName="Michael Reinegger"
		tagline="Software Engineer & CS Master Student"
		description="Passionate software engineer based in Linz, Austria. Currently pursuing a Master's degree in Computer Science at Johannes Kepler Universität Linz while working on exciting projects in web development, systems programming, and beyond."
		founded="2024"
		mission="Building high-quality, maintainable software that solves real problems — with a focus on clean architecture, performance, and great developer experience."
		stats={[
			{ value: '10+', label: 'Projects' },
			{ value: '5+', label: 'Years Coding' },
			{ value: 'MSc', label: 'CS at JKU Linz' },
			{ value: 'AT', label: 'Based in Austria' }
		]}
		values={[
			{
				icon: '🛠️',
				name: 'Engineering Excellence',
				description: 'Writing clean, well-tested code with attention to architecture and long-term maintainability.'
			},
			{
				icon: '🌐',
				name: 'Open Source',
				description: 'Contributing to and building open-source software to share knowledge and collaborate with the community.'
			},
			{
				icon: '📚',
				name: 'Continuous Learning',
				description: 'Always exploring new technologies, algorithms, and best practices to grow as an engineer.'
			},
			{
				icon: '🚀',
				name: 'Shipping Fast',
				description: 'Balancing pragmatism with quality to deliver working software iteratively and efficiently.'
			}
		]}
	/>
</main>
