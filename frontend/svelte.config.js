import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),

	kit: {
		adapter: adapter({
			pages: 'build',
			assets: 'build',
			fallback: '200.html', // separate SPA shell — avoids overwriting prerendered index.html
			precompress: true,
			strict: true
		}),
		prerender: {
			handleHttpError: ({ path, message }) => {
				// Ignore missing static assets (e.g. favicon) during prerender
				if (path.startsWith('/favicon') || path.startsWith('/_app')) return;
				throw new Error(message);
			}
		}
	}
};

export default config;
