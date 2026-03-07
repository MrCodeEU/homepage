import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	resolve: {
		alias: {
			// Stub out zxcvbn-typescript (809KB) — it's pulled in by mljr-svelte's
			// Password component which we don't use. This replaces it with a no-op
			// that returns a minimal result, saving ~90% of the main JS bundle.
			'zxcvbn-typescript': '/src/lib/stubs/zxcvbn.ts'
		}
	},
	server: {
		port: 5173
	}
});
