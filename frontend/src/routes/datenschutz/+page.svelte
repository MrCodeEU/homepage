<script lang="ts">
	import { Background, Datenschutz, Icon, Navbar, ThemeToggle } from 'mljr-svelte';

	const navItems = [{ label: '← Back', href: '/', icon: 'mdi:arrow-left' }];

	const datenschutzData = {
		controller: {
			name: 'Michael Reinegger',
			address: '4020 Linz, Österreich',
			email: 'contact@michael-reinegger.at' // TODO: Replace with actual contact email
		},
		usesCookies: true,
		lastUpdated: '2025-01-01'
	};
</script>

<svelte:head>
	<title>Datenschutz – Michael Reinegger</title>
	<meta name="description" content="Datenschutzerklärung / Privacy Policy for michael-reinegger.at" />
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

<main class="page-container" style="padding-bottom: 4rem;">
	<Datenschutz data={datenschutzData} />
</main>
