<script lang="ts">
	import '../app.css';
	import { CookieBanner } from 'mljr-svelte';
	import type { Snippet } from 'svelte';

	interface Props {
		children: Snippet;
	}
	let { children }: Props = $props();

	const cookieCategories = [
		{
			id: 'necessary',
			name: { en: 'Necessary', de: 'Notwendig' },
			description: {
				en: 'Essential cookies required for the website to function properly. These cannot be disabled.',
				de: 'Notwendige Cookies für den ordnungsgemäßen Betrieb der Website. Diese können nicht deaktiviert werden.'
			},
			required: true
		}
	];
</script>

{@render children()}

<CookieBanner categories={cookieCategories} showFloatingButton={true} />
