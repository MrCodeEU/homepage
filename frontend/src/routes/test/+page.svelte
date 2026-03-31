<script lang="ts">
	import { LogoAnimation } from '$lib/components';
	import logoSvg from '$lib/assets/logo.svg?raw';
</script>

<svelte:head>
	<title>Animation Test</title>
</svelte:head>

<!-- Background Logo Animation - Full viewport, behind everything -->
<LogoAnimation
	svgContent={logoSvg}
	animationDuration={2}
	reverseDuration={0.5}
	perspective={1200}
	zIndex={-1}
	logoSize={400}
	scrollThreshold={15}
	windIntensity={1}
	pathCurviness={1.2}
/>

<!-- Page Content -->
<main class="page">
	<!-- Hero Section -->
	<section class="hero">
		<div class="hero-content">
			<h1>Animation Test Page</h1>
			<p>Scroll down to see the logo elements explode and float around</p>
			<div class="scroll-indicator">
				<span>Scroll</span>
				<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M12 5v14M5 12l7 7 7-7"/>
				</svg>
			</div>
		</div>
	</section>

	<!-- Cards Section -->
	<section class="cards-section">
		<h2>Featured Content</h2>
		<div class="cards-grid">
			{#each Array(12) as _, i}
				<article class="card">
					<div class="card-image" style="background: linear-gradient(135deg, hsl({i * 30}, 70%, 60%), hsl({i * 30 + 40}, 70%, 50%));">
						<span class="card-number">{i + 1}</span>
					</div>
					<div class="card-content">
						<h3>Card Title {i + 1}</h3>
						<p>This is some sample content for card number {i + 1}. The animation should be visible behind this card.</p>
						<div class="card-tags">
							<span class="tag">Tag A</span>
							<span class="tag">Tag B</span>
						</div>
					</div>
				</article>
			{/each}
		</div>
	</section>

	<!-- Another Section for more scroll distance -->
	<section class="info-section">
		<h2>More Content</h2>
		<p>Keep scrolling to see the floating animation continue in the background.</p>
		<div class="info-grid">
			{#each Array(6) as _, i}
				<div class="info-box">
					<h4>Info Box {i + 1}</h4>
					<p>Additional content to extend the page and allow more scrolling for testing the animation behavior.</p>
				</div>
			{/each}
		</div>
	</section>

	<!-- Footer -->
	<footer class="footer">
		<p>End of test page - scroll back up to see the animation reverse</p>
	</footer>
</main>

<style>
	:global(html, body) {
		margin: 0;
		padding: 0;
		min-height: 100%;
	}

	:global(body) {
		font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
		background: linear-gradient(180deg, #0f0f1a 0%, #1a1a2e 50%, #16213e 100%);
		background-attachment: fixed;
		color: #e0e0e0;
	}

	.page {
		position: relative;
		z-index: 1;
	}

	/* Hero Section */
	.hero {
		min-height: 100vh;
		display: flex;
		align-items: center;
		justify-content: center;
		text-align: center;
		padding: 2rem;
	}

	.hero-content {
		max-width: 600px;
	}

	.hero h1 {
		font-size: clamp(2.5rem, 8vw, 4rem);
		margin: 0 0 1rem;
		background: linear-gradient(135deg, #667eea 0%, #764ba2 50%, #f093fb 100%);
		-webkit-background-clip: text;
		-webkit-text-fill-color: transparent;
		background-clip: text;
	}

	.hero p {
		font-size: 1.25rem;
		color: #a0a0a0;
		margin: 0 0 3rem;
	}

	.scroll-indicator {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.5rem;
		color: #667eea;
		animation: bounce 2s infinite;
	}

	@keyframes bounce {
		0%, 100% { transform: translateY(0); }
		50% { transform: translateY(10px); }
	}

	/* Cards Section */
	.cards-section {
		padding: 4rem 2rem;
		max-width: 1400px;
		margin: 0 auto;
	}

	.cards-section h2 {
		text-align: center;
		font-size: 2rem;
		margin: 0 0 3rem;
		color: #fff;
	}

	.cards-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
		gap: 2rem;
	}

	.card {
		background: rgba(255, 255, 255, 0.05);
		border-radius: 16px;
		overflow: hidden;
		backdrop-filter: blur(10px);
		border: 1px solid rgba(255, 255, 255, 0.1);
		transition: transform 0.3s ease, box-shadow 0.3s ease;
	}

	.card:hover {
		transform: translateY(-8px);
		box-shadow: 0 20px 40px rgba(0, 0, 0, 0.3);
	}

	.card-image {
		height: 180px;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.card-number {
		font-size: 4rem;
		font-weight: bold;
		color: rgba(255, 255, 255, 0.3);
	}

	.card-content {
		padding: 1.5rem;
	}

	.card-content h3 {
		margin: 0 0 0.75rem;
		color: #fff;
		font-size: 1.25rem;
	}

	.card-content p {
		margin: 0 0 1rem;
		color: #a0a0a0;
		font-size: 0.9rem;
		line-height: 1.6;
	}

	.card-tags {
		display: flex;
		gap: 0.5rem;
	}

	.tag {
		background: rgba(102, 126, 234, 0.2);
		color: #667eea;
		padding: 0.25rem 0.75rem;
		border-radius: 20px;
		font-size: 0.8rem;
	}

	/* Info Section */
	.info-section {
		padding: 4rem 2rem;
		max-width: 1200px;
		margin: 0 auto;
	}

	.info-section h2 {
		text-align: center;
		font-size: 2rem;
		margin: 0 0 1rem;
		color: #fff;
	}

	.info-section > p {
		text-align: center;
		color: #a0a0a0;
		margin: 0 0 3rem;
	}

	.info-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
		gap: 1.5rem;
	}

	.info-box {
		background: rgba(255, 255, 255, 0.03);
		border-radius: 12px;
		padding: 1.5rem;
		border: 1px solid rgba(255, 255, 255, 0.05);
	}

	.info-box h4 {
		margin: 0 0 0.75rem;
		color: #667eea;
	}

	.info-box p {
		margin: 0;
		color: #808080;
		font-size: 0.9rem;
		line-height: 1.5;
	}

	/* Footer */
	.footer {
		padding: 4rem 2rem;
		text-align: center;
		color: #606060;
		border-top: 1px solid rgba(255, 255, 255, 0.05);
		margin-top: 4rem;
	}

	/* Mobile adjustments */
	@media (max-width: 768px) {
		.cards-grid {
			grid-template-columns: 1fr;
		}

		.hero {
			min-height: 80vh;
		}
	}
</style>
