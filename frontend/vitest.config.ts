import { defineConfig } from 'vitest/config';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
	plugins: [svelte({ hot: !process.env.VITEST })],
	esbuild: {
		tsconfigRaw: '{}'
	},
	resolve: {
		conditions: ['browser']
	},
	test: {
		globals: true,
		environment: 'jsdom',
		setupFiles: ['./src/setupTests.ts'],
		alias: {
			$lib: '/src/lib'
		},
		coverage: {
			provider: 'v8',
			reporter: ['text', 'json', 'html'],
			exclude: [
				'node_modules/',
				'src/setupTests.ts',
				'**/*.config.{js,ts}',
				'**/*.d.ts'
			]
		}
	}
});
