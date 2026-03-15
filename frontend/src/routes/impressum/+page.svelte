<script lang="ts">
	import { Background, Impressum, Icon, Navbar, ThemeToggle } from 'mljr-svelte';

	const navItems = [{ label: '← Back', href: '/', icon: 'mdi:arrow-left' }];

	const impressumData = {
		name: 'Michael Reinegger',
		address: {
			street: 'Linz', // TODO: Add street address if desired
			zip: '4020',
			city: 'Linz',
			country: 'Österreich'
		},
		email: 'contact@michael-reinegger.at', // TODO: Replace with actual contact email
		website: 'https://michael-reinegger.at',
		lastUpdated: '2025-01-01'
	};
</script>

<svelte:head>
	<title>Impressum – Michael Reinegger</title>
	<meta name="description" content="Impressum / Legal Notice for michael-reinegger.at" />
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
	<Impressum data={impressumData} />
</main>
