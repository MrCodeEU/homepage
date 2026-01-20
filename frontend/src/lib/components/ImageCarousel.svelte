<script lang="ts">
	import { onMount, onDestroy } from 'svelte';

	export let images: string[] = [];
	export let alt: string = '';
	export let interval: number = 3000;
	export let height: string = '280px';

	let currentIndex = 0;
	let timer: ReturnType<typeof setInterval> | null = null;
	let isPaused = false;
	let isMounted = false;
	let isFocused = false;

	function nextImage() {
		if (images.length > 1 && isMounted) {
			currentIndex = (currentIndex + 1) % images.length;
		}
	}

	function prevImage() {
		if (images.length > 1 && isMounted) {
			currentIndex = (currentIndex - 1 + images.length) % images.length;
		}
	}

	function goToImage(index: number) {
		if (!isMounted) return;
		currentIndex = index;
		resetTimer();
	}

	function startTimer() {
		if (images.length > 1 && !isPaused && isMounted) {
			timer = setInterval(nextImage, interval);
		}
	}

	function stopTimer() {
		if (timer) {
			clearInterval(timer);
			timer = null;
		}
	}

	function resetTimer() {
		if (!isMounted) return;
		stopTimer();
		startTimer();
	}

	function handleMouseEnter() {
		isPaused = true;
		stopTimer();
	}

	function handleMouseLeave() {
		isPaused = false;
		if (isMounted) startTimer();
	}

	function handleFocusIn() {
		isFocused = true;
		isPaused = true;
		stopTimer();
	}

	function handleFocusOut() {
		isFocused = false;
		isPaused = false;
		if (isMounted) startTimer();
	}

	function handleImageError(e: Event) {
		const target = e.currentTarget;
		if (target instanceof HTMLImageElement) {
			target.style.display = 'none';
		}
	}

	onMount(() => {
		isMounted = true;
		startTimer();
	});

	onDestroy(() => {
		isMounted = false;
		stopTimer();
	});
</script>

<div
	class="carousel-container relative overflow-hidden detroit-clip-lg bg-black/40"
	class:is-focused={isFocused}
	style="height: {height};"
	on:mouseenter={handleMouseEnter}
	on:mouseleave={handleMouseLeave}
	on:focusin={handleFocusIn}
	on:focusout={handleFocusOut}
	role="region"
	aria-label="Image carousel"
	aria-roledescription="carousel"
>
	{#each images as image, index}
		<div
			class="carousel-slide absolute inset-0 transition-all duration-500"
			class:opacity-100={index === currentIndex}
			class:opacity-0={index !== currentIndex}
			class:z-10={index === currentIndex}
			class:z-0={index !== currentIndex}
			class:slide-active={index === currentIndex}
			role="group"
			aria-roledescription="slide"
			aria-label="Slide {index + 1} of {images.length}"
			aria-hidden={index !== currentIndex}
		>
			<img
				src={image}
				alt="{alt} - Image {index + 1}"
				class="carousel-image w-full h-full object-contain transition-all duration-600"
				on:error={handleImageError}
			/>
		</div>
	{/each}

	<!-- Navigation arrows (only show if multiple images) -->
	{#if images.length > 1}
		<button
			class="carousel-nav carousel-nav-prev absolute left-3 top-1/2 -translate-y-1/2 z-20 w-8 h-8 flex items-center justify-center bg-detroit-bg-card/80 border border-detroit-primary/30 detroit-clip-xs text-detroit-primary/70 hover:text-detroit-primary hover:bg-detroit-bg-card hover:border-detroit-primary/50 focus:text-detroit-primary focus:bg-detroit-bg-card focus:border-detroit-primary/50 focus:outline-none focus:ring-2 focus:ring-detroit-primary/50 transition-all duration-300"
			on:click={prevImage}
			aria-label="Previous image"
		>
			<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M15 18l-6-6 6-6"/>
			</svg>
		</button>
		<button
			class="carousel-nav carousel-nav-next absolute right-3 top-1/2 -translate-y-1/2 z-20 w-8 h-8 flex items-center justify-center bg-detroit-bg-card/80 border border-detroit-primary/30 detroit-clip-xs text-detroit-primary/70 hover:text-detroit-primary hover:bg-detroit-bg-card hover:border-detroit-primary/50 focus:text-detroit-primary focus:bg-detroit-bg-card focus:border-detroit-primary/50 focus:outline-none focus:ring-2 focus:ring-detroit-primary/50 transition-all duration-300"
			on:click={nextImage}
			aria-label="Next image"
		>
			<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M9 18l6-6-6-6"/>
			</svg>
		</button>

		<!-- Dot indicators -->
		<div class="carousel-dots absolute bottom-3 left-1/2 -translate-x-1/2 z-20 flex gap-2" role="tablist" aria-label="Slide indicators">
			{#each images as _, index}
				<button
					class="w-2 h-2 rounded-full transition-all duration-300 focus:outline-none focus:ring-2 focus:ring-detroit-primary/50 {index === currentIndex ? 'bg-detroit-primary scale-125' : 'bg-white/30'}"
					on:click={() => goToImage(index)}
					aria-label="Go to slide {index + 1}"
					aria-selected={index === currentIndex}
					role="tab"
				></button>
			{/each}
		</div>

		<!-- Image counter -->
		<div class="absolute top-3 right-3 z-20 py-1 px-3 bg-detroit-bg-card/80 border border-detroit-primary/30 detroit-clip-xs text-xs text-detroit-primary/70" aria-live="polite">
			{currentIndex + 1} / {images.length}
		</div>
	{/if}
</div>

<style>
	.carousel-container {
		clip-path: polygon(0 0, calc(100% - 50px) 0, 100% 50px, 100% 100%, 0 100%);
	}

	.carousel-nav {
		opacity: 0;
		transform: translateY(-50%);
	}

	/* Show nav on hover or when container has focus within */
	.carousel-container:hover .carousel-nav,
	.carousel-container.is-focused .carousel-nav,
	.carousel-nav:focus {
		opacity: 1;
	}

	/* Apply desaturated/dimmed style to inactive slides, full color to active */
	.carousel-image {
		filter: saturate(0.7) brightness(0.9);
	}

	.slide-active .carousel-image {
		filter: saturate(1) brightness(1);
	}
</style>
