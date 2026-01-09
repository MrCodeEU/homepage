<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { browser } from '$app/environment';
	import gsap from 'gsap';
	import { ScrollTrigger } from 'gsap/ScrollTrigger';
	import { MotionPathPlugin } from 'gsap/MotionPathPlugin';

	if (browser) {
		gsap.registerPlugin(ScrollTrigger, MotionPathPlugin);
	}

	// Props
	export let svgContent: string = '';
	export let animationSpeed: number = 1;
	export let perspective: number = 1200;
	export let zIndex: number = -1;
	export let logoScale: number = 3;
	export let edgePadding: number = 120;
	export let pathCurviness: number = 0.5;
	export let windIntensity: number = 1; // Multiplier for wind wobble in path

	// Internal state
	let container: HTMLDivElement;
	let svgContainer: HTMLDivElement;
	let elements: Element[] = [];
	let scrollTriggerInstance: ScrollTrigger | null = null;
	let mainTimeline: gsap.core.Timeline | null = null;
	let idleTweens: gsap.core.Tween[] = [];
	let prefersReducedMotion = false;
	let viewportWidth = 0;
	let viewportHeight = 0;
	let pageHeight = 0;

	interface ElementData {
		el: Element;
		fullPath: { x: number; y: number }[];
		scaleKeyframes: number[];
		rotationKeyframes: number[];
		maxRotateX: number;
		maxRotateY: number;
	}
	let elementData: ElementData[] = [];

	// Generate path with wind wobble baked in
	function generateFullPath(
		el: Element,
		centerX: number,
		centerY: number,
		index: number,
		totalElements: number
	): { path: { x: number; y: number }[]; scales: number[]; rotations: number[] } {
		const rect = el.getBoundingClientRect();
		const elCenterX = rect.left + rect.width / 2;
		const elCenterY = rect.top + rect.height / 2;

		// Direction from logo center
		let dx = elCenterX - centerX;
		let dy = elCenterY - centerY;
		const distance = Math.sqrt(dx * dx + dy * dy) || 1;
		const normalizedDx = dx / distance;
		const normalizedDy = dy / distance;

		// Safe bounds
		const safeWidth = (viewportWidth / 2) - edgePadding;
		const safeHeight = (viewportHeight / 2) - edgePadding;

		const points: { x: number; y: number }[] = [{ x: 0, y: 0 }];
		const scales: number[] = [1];
		const rotations: number[] = [0];

		// Calculate total points based on page height
		const scrollableHeight = pageHeight - viewportHeight;
		// More points = more opportunity for wind wobble
		const totalPoints = Math.max(8, Math.floor(scrollableHeight / 150));

		// Initial spread
		const spreadFactor = 0.25 + Math.random() * 0.15;
		const baseSpreadX = normalizedDx * safeWidth * spreadFactor;
		const baseSpreadY = normalizedDy * safeHeight * spreadFactor;

		// Wind parameters unique to this element
		const windFreqX = 2 + Math.random() * 2; // How many oscillations
		const windFreqY = 2 + Math.random() * 2;
		const windAmpX = (3 + Math.random() * 4) * windIntensity; // Amplitude in pixels
		const windAmpY = (2 + Math.random() * 3) * windIntensity;
		const windPhaseX = Math.random() * Math.PI * 2;
		const windPhaseY = Math.random() * Math.PI * 2;

		// Rotation wind
		const rotWindFreq = 1.5 + Math.random() * 2;
		const rotWindAmp = (2 + Math.random() * 3) * windIntensity;
		const rotWindPhase = Math.random() * Math.PI * 2;

		// Scale wind (depth bobbing)
		const scaleWindFreq = 1 + Math.random() * 1.5;
		const scaleWindAmp = 0.03 * windIntensity;
		const scaleWindPhase = Math.random() * Math.PI * 2;

		// Generate points along the main path with wind wobble
		for (let i = 1; i <= totalPoints; i++) {
			const t = i / totalPoints; // Progress 0 to 1

			// Main drift path - gradual movement outward then slight return
			const driftProgress = t < 0.7 ? t / 0.7 : 1 - (t - 0.7) / 0.3 * 0.2;
			const mainX = baseSpreadX * driftProgress;
			const mainY = baseSpreadY * driftProgress;

			// Add wind wobble as sine waves
			const windX = Math.sin(t * Math.PI * 2 * windFreqX + windPhaseX) * windAmpX;
			const windY = Math.sin(t * Math.PI * 2 * windFreqY + windPhaseY) * windAmpY;

			// Small perpendicular drift for variety
			const perpAngle = Math.atan2(normalizedDy, normalizedDx) + Math.PI / 2;
			const perpDrift = Math.sin(t * Math.PI * 3 + index) * safeWidth * 0.03;
			const perpX = Math.cos(perpAngle) * perpDrift;
			const perpY = Math.sin(perpAngle) * perpDrift;

			const finalX = mainX + windX + perpX;
			const finalY = mainY + windY + perpY;

			points.push({
				x: clamp(finalX, -safeWidth, safeWidth),
				y: clamp(finalY, -safeHeight, safeHeight)
			});

			// Scale with wind bobbing for depth
			const baseScale = 0.95 - t * 0.05; // Slight shrink as moves out
			const scaleWind = Math.sin(t * Math.PI * 2 * scaleWindFreq + scaleWindPhase) * scaleWindAmp;
			scales.push(baseScale + scaleWind);

			// Rotation with wind
			const baseRotation = (Math.random() - 0.5) * 30 * t; // Gradual rotation
			const rotWind = Math.sin(t * Math.PI * 2 * rotWindFreq + rotWindPhase) * rotWindAmp;
			rotations.push(baseRotation + rotWind);
		}

		return { path: points, scales, rotations };
	}

	function clamp(value: number, min: number, max: number): number {
		return Math.max(min, Math.min(max, value));
	}

	function initAnimations() {
		if (!browser || !svgContainer) return;

		prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
		viewportWidth = window.innerWidth;
		viewportHeight = window.innerHeight;
		pageHeight = document.documentElement.scrollHeight;

		elements = Array.from(svgContainer.querySelectorAll('.logo-element'));
		if (elements.length === 0) {
			elements = Array.from(
				svgContainer.querySelectorAll('rect, circle, ellipse, path, polygon, polyline')
			);
		}

		if (elements.length === 0) return;

		const svgRect = svgContainer.getBoundingClientRect();
		const centerX = svgRect.left + svgRect.width / 2;
		const centerY = svgRect.top + svgRect.height / 2;

		elementData = elements.map((el, index) => {
			gsap.set(el, {
				transformOrigin: '50% 50%',
				force3D: true,
				x: 0,
				y: 0,
				xPercent: 0,
				yPercent: 0,
				rotation: 0,
				rotateX: 0,
				rotateY: 0,
				scale: 1,
				opacity: 1
			});

			const { path, scales, rotations } = generateFullPath(el, centerX, centerY, index, elements.length);

			return {
				el,
				fullPath: path,
				scaleKeyframes: scales,
				rotationKeyframes: rotations,
				maxRotateX: (Math.random() - 0.5) * 15,
				maxRotateY: (Math.random() - 0.5) * 15
			};
		});

		if (prefersReducedMotion) {
			initReducedMotionAnimations();
		} else {
			initFullAnimations();
		}
	}

	function initReducedMotionAnimations() {
		elementData.forEach(({ el, fullPath }) => {
			const target = fullPath[Math.floor(fullPath.length / 2)] || { x: 0, y: 0 };
			gsap.to(el, {
				x: target.x * 0.5,
				y: target.y * 0.5,
				opacity: 0.6,
				scrollTrigger: {
					trigger: document.body,
					start: 'top top',
					end: 'bottom bottom',
					scrub: 1
				}
			});
		});
	}

	// Idle wind animation - uses xPercent/yPercent which don't conflict with motionPath's x/y
	function initIdleWind() {
		elementData.forEach(({ el }) => {
			const duration = 3 + Math.random() * 3;
			const xAmp = (1.5 + Math.random() * 2) * windIntensity;
			const yAmp = (1 + Math.random() * 1.5) * windIntensity;
			const delay = Math.random() * 2;

			// Floating X movement (xPercent is separate from x used by motionPath)
			const tweenX = gsap.to(el, {
				xPercent: xAmp,
				duration: duration,
				ease: 'sine.inOut',
				yoyo: true,
				repeat: -1,
				delay: delay
			});

			// Floating Y movement (yPercent is separate from y used by motionPath)
			const tweenY = gsap.to(el, {
				yPercent: yAmp,
				duration: duration * 1.3,
				ease: 'sine.inOut',
				yoyo: true,
				repeat: -1,
				delay: delay + 0.5
			});

			idleTweens.push(tweenX, tweenY);
		});
	}

	function initFullAnimations() {
		// Single unified timeline - no competing animations
		mainTimeline = gsap.timeline({ paused: true });

		elementData.forEach(({ el, fullPath, scaleKeyframes, rotationKeyframes, maxRotateX, maxRotateY }, index) => {
			const staggerOffset = index * 0.008;
			const elementTl = gsap.timeline();

			// Path animation with wind baked in
			elementTl.to(el, {
				motionPath: {
					path: fullPath,
					curviness: pathCurviness,
					autoRotate: false
				},
				rotateX: maxRotateX,
				rotateY: maxRotateY,
				opacity: 0.8,
				duration: 1,
				ease: 'none'
			}, 0);

			// Scale keyframes (includes wind bobbing)
			const numScales = scaleKeyframes.length;
			if (numScales > 1) {
				const scaleTl = gsap.timeline();
				for (let i = 1; i < numScales; i++) {
					const duration = 1 / (numScales - 1);
					scaleTl.to(el, {
						scale: scaleKeyframes[i],
						duration: duration,
						ease: 'none' // Linear for smooth wind feel
					}, (i - 1) * duration);
				}
				elementTl.add(scaleTl, 0);
			}

			// Rotation keyframes (includes wind wobble)
			const numRotations = rotationKeyframes.length;
			if (numRotations > 1) {
				const rotTl = gsap.timeline();
				for (let i = 1; i < numRotations; i++) {
					const duration = 1 / (numRotations - 1);
					rotTl.to(el, {
						rotation: rotationKeyframes[i],
						duration: duration,
						ease: 'none'
					}, (i - 1) * duration);
				}
				elementTl.add(rotTl, 0);
			}

			mainTimeline!.add(elementTl, staggerOffset);
		});

		// Single ScrollTrigger controls everything
		scrollTriggerInstance = ScrollTrigger.create({
			trigger: document.body,
			start: 'top top',
			end: 'bottom bottom',
			scrub: 1.2 / animationSpeed,
			onUpdate: (self) => {
				if (mainTimeline) {
					mainTimeline.progress(self.progress);
				}
			}
		});

		// Start idle wind animation (runs continuously, uses separate properties)
		initIdleWind();
	}

	function handleResize() {
		viewportWidth = window.innerWidth;
		viewportHeight = window.innerHeight;
		pageHeight = document.documentElement.scrollHeight;
		cleanup();
		requestAnimationFrame(() => {
			initAnimations();
		});
	}

	function cleanup() {
		if (!browser) return;

		if (scrollTriggerInstance) {
			scrollTriggerInstance.kill();
			scrollTriggerInstance = null;
		}

		if (mainTimeline) {
			mainTimeline.kill();
			mainTimeline = null;
		}

		// Kill all idle wind tweens
		idleTweens.forEach((tween) => tween.kill());
		idleTweens = [];

		elementData = [];

		window.removeEventListener('resize', handleResize);
	}

	onMount(() => {
		if (browser) {
			requestAnimationFrame(() => {
				requestAnimationFrame(() => {
					initAnimations();
				});
			});
			window.addEventListener('resize', handleResize, { passive: true });

			const mediaQuery = window.matchMedia('(prefers-reduced-motion: reduce)');
			mediaQuery.addEventListener('change', () => {
				prefersReducedMotion = mediaQuery.matches;
				cleanup();
				initAnimations();
			});
		}
	});

	onDestroy(() => {
		cleanup();
	});
</script>

<div
	class="logo-animation-container"
	bind:this={container}
	style:--perspective="{perspective}px"
	style:--z-index={zIndex}
	style:--logo-scale={logoScale}
>
	<div class="logo-animation-wrapper" bind:this={svgContainer}>
		{#if svgContent}
			{@html svgContent}
		{:else}
			<slot />
		{/if}
	</div>
</div>

<style>
	.logo-animation-container {
		position: fixed;
		inset: 0;
		width: 100vw;
		height: 100vh;
		z-index: var(--z-index, -1);
		pointer-events: none;
		overflow: hidden;
		perspective: var(--perspective, 1200px);
		perspective-origin: 50% 50%;
		contain: layout style;
	}

	.logo-animation-wrapper {
		position: absolute;
		top: 50%;
		left: 50%;
		transform: translate(-50%, -50%) scale(var(--logo-scale, 3));
		transform-style: preserve-3d;
	}

	.logo-animation-wrapper :global(svg) {
		display: block;
		transform-style: preserve-3d;
		overflow: visible;
	}

	.logo-animation-wrapper :global(.logo-element),
	.logo-animation-wrapper :global(rect),
	.logo-animation-wrapper :global(circle),
	.logo-animation-wrapper :global(ellipse),
	.logo-animation-wrapper :global(path),
	.logo-animation-wrapper :global(polygon),
	.logo-animation-wrapper :global(polyline) {
		transform-style: preserve-3d;
		will-change: transform, opacity;
		backface-visibility: visible;
	}

	@media (prefers-reduced-motion: reduce) {
		.logo-animation-wrapper :global(*) {
			transition: transform 0.5s ease-out, opacity 0.5s ease-out !important;
			animation: none !important;
		}
	}
</style>
