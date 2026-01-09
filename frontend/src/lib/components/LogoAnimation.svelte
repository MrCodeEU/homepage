<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { browser } from '$app/environment';
	import gsap from 'gsap';
	import { MotionPathPlugin } from 'gsap/MotionPathPlugin';

	if (browser) {
		gsap.registerPlugin(MotionPathPlugin);
	}

	// Props
	export let svgContent: string = '';
	export let animationDuration: number = 2;
	export let reverseDuration: number = 0.5;
	export let perspective: number = 1200;
	export let zIndex: number = -1;
	export let logoScale: number = 3;
	export let scrollThreshold: number = 15;
	export let windIntensity: number = 1;
	export let pathCurviness: number = 1.2;

	// Internal state
	let container: HTMLDivElement;
	let svgContainer: HTMLDivElement;
	let elements: Element[] = [];
	let mainTimeline: gsap.core.Timeline | null = null;
	let idleTweens: gsap.core.Tween[] = [];
	let prefersReducedMotion = false;
	let viewportWidth = 0;
	let viewportHeight = 0;
	let pageHeight = 0;
	let isAnimatedOut = false;
	let hasInitialized = false;
	let isAbsoluteMode = false;

	// Store initial positions for reverse animation
	let initialContainerRect: DOMRect | null = null;

	interface ElementData {
		el: Element;
		startX: number;
		startY: number;
		targetX: number;
		targetY: number;
		path: { x: number; y: number }[];
		targetRotation: number;
		targetRotateX: number;
		targetRotateY: number;
		targetScale: number;
	}
	let elementData: ElementData[] = [];

	// Generate random end position - distributed across FULL page height
	function generateTargetPosition(index: number, totalElements: number): { x: number; y: number } {
		// Y position: distribute across full page height
		// Divide page into segments and place elements somewhat evenly
		const segmentHeight = pageHeight / totalElements;
		const segmentStart = index * segmentHeight;
		const segmentEnd = (index + 1) * segmentHeight;
		// Random position within this segment, with some padding
		const padding = segmentHeight * 0.1;
		const y = segmentStart + padding + Math.random() * (segmentEnd - segmentStart - padding * 2);

		// X position: mostly left and right, ~10% in middle
		// Keep elements well within viewport bounds
		let x: number;
		const roll = Math.random();
		const edgePadding = 100; // Increased padding from edge
		const elementSize = 80; // Approximate element size to keep fully visible
		const safeLeft = edgePadding;
		const safeRight = viewportWidth - edgePadding - elementSize;
		const leftZoneEnd = viewportWidth * 0.25;
		const rightZoneStart = viewportWidth * 0.75;

		if (roll < 0.45) {
			// Left side (45%)
			x = safeLeft + Math.random() * (leftZoneEnd - safeLeft);
		} else if (roll < 0.90) {
			// Right side (45%)
			x = rightZoneStart + Math.random() * (safeRight - rightZoneStart);
		} else {
			// Middle (10%)
			x = leftZoneEnd + Math.random() * (rightZoneStart - leftZoneEnd);
		}

		// Clamp to safe bounds
		x = Math.max(safeLeft, Math.min(safeRight, x));

		return { x, y };
	}

	// Generate a curved path from start to target (in absolute page coordinates)
	function generatePath(
		startX: number,
		startY: number,
		targetX: number,
		targetY: number
	): { x: number; y: number }[] {
		const points: { x: number; y: number }[] = [];

		// Start at origin (0,0 relative to element's current position)
		points.push({ x: 0, y: 0 });

		// Calculate the offset needed to reach target
		const dx = targetX - startX;
		const dy = targetY - startY;

		// Add 2-3 control points for a nice curve
		const numControlPoints = 2 + Math.floor(Math.random() * 2);

		for (let i = 1; i <= numControlPoints; i++) {
			const t = i / (numControlPoints + 1);
			const baseX = dx * t;
			const baseY = dy * t;

			// Add perpendicular offset for curve
			const perpOffset = (Math.random() - 0.5) * Math.min(viewportWidth, viewportHeight) * 0.4;
			const angle = Math.atan2(dy, dx) + Math.PI / 2;

			points.push({
				x: baseX + Math.cos(angle) * perpOffset,
				y: baseY + Math.sin(angle) * perpOffset
			});
		}

		// End at target (relative offset from start)
		points.push({ x: dx, y: dy });

		return points;
	}

	function initAnimations() {
		if (!browser || !svgContainer || hasInitialized) return;

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

		// Store container's initial position
		initialContainerRect = svgContainer.getBoundingClientRect();

		// Calculate the center of the logo in page coordinates (accounting for scroll)
		const logoCenterX = initialContainerRect.left + initialContainerRect.width / 2;
		const logoCenterY = initialContainerRect.top + initialContainerRect.height / 2 + window.scrollY;

		elementData = elements.map((el, index) => {
			const rect = el.getBoundingClientRect();
			// Element's current position in page coordinates
			const elCenterX = rect.left + rect.width / 2;
			const elCenterY = rect.top + rect.height / 2 + window.scrollY;

			// Generate random target position on the page (absolute coordinates)
			const target = generateTargetPosition(index, elements.length);

			// Generate curved path (offsets from current position)
			const path = generatePath(elCenterX, elCenterY, target.x, target.y);

			// Random 3D rotations and scale - keep elements larger and visible
			const targetRotation = (Math.random() - 0.5) * 30;
			const targetRotateX = (Math.random() - 0.5) * 20;
			const targetRotateY = (Math.random() - 0.5) * 20;
			const targetScale = 2.5 + Math.random() * 1.5; // Much larger scale (2.5x to 4x)

			// Initialize element
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

			return {
				el,
				startX: elCenterX,
				startY: elCenterY,
				targetX: target.x,
				targetY: target.y,
				path,
				targetRotation,
				targetRotateX,
				targetRotateY,
				targetScale
			};
		});

		if (prefersReducedMotion) {
			initReducedMotionAnimations();
		} else {
			initFullAnimations();
		}

		hasInitialized = true;
	}

	function initReducedMotionAnimations() {
		mainTimeline = gsap.timeline({ paused: true });

		elementData.forEach(({ el, path }) => {
			const endPoint = path[path.length - 1];
			mainTimeline!.to(el, {
				x: endPoint.x,
				y: endPoint.y,
				opacity: 0.7,
				duration: 0.5,
				ease: 'power2.out'
			}, 0);
		});
	}

	function initFullAnimations() {
		mainTimeline = gsap.timeline({
			paused: true,
			onComplete: () => {
				// Switch to absolute positioning so elements are fixed to the page
				switchToAbsoluteMode();
				// Start idle wind animation
				initIdleWind();
			},
			onReverseComplete: () => {
				// Reset all elements to initial state
				resetElementsToInitial();
			}
		});

		elementData.forEach(({ el, path, targetRotation, targetRotateX, targetRotateY, targetScale }, index) => {
			const staggerDelay = index * 0.05;

			mainTimeline!.to(el, {
				motionPath: {
					path: path,
					curviness: pathCurviness,
					autoRotate: false
				},
				rotation: targetRotation,
				rotateX: targetRotateX,
				rotateY: targetRotateY,
				scale: targetScale,
				opacity: 0.85,
				duration: animationDuration,
				ease: 'power2.inOut'
			}, staggerDelay);
		});

		window.addEventListener('scroll', handleScroll, { passive: true });
	}

	function resetElementsToInitial() {
		// Ensure we're in fixed mode
		if (container) {
			container.style.position = 'fixed';
			container.style.top = '';
			container.style.left = '';
			container.style.width = '100vw';
			container.style.height = '100vh';
			container.style.transform = '';
		}
		if (svgContainer) {
			svgContainer.style.position = 'absolute';
			svgContainer.style.top = '50%';
			svgContainer.style.left = '50%';
			svgContainer.style.transform = `translate(-50%, -50%) scale(${logoScale})`;
		}

		// Reset all element transforms
		elementData.forEach(({ el }) => {
			gsap.set(el, {
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
		});

		isAbsoluteMode = false;
	}

	function switchToAbsoluteMode() {
		if (isAbsoluteMode || !container) return;

		// Calculate where each element should be in absolute page coordinates
		elementData.forEach(({ el, targetX, targetY }) => {
			// Set absolute position directly
			gsap.set(el, {
				x: targetX,
				y: targetY
			});
		});

		// Switch container to absolute positioning at page origin
		container.style.position = 'absolute';
		container.style.top = '0';
		container.style.left = '0';
		container.style.width = '100%';
		container.style.height = `${pageHeight}px`;
		container.style.transform = 'none';

		// Update the wrapper to not be centered anymore
		svgContainer.style.position = 'absolute';
		svgContainer.style.top = '0';
		svgContainer.style.left = '0';
		svgContainer.style.transform = 'none';

		isAbsoluteMode = true;
	}

	function handleScroll() {
		const scrollY = window.scrollY;

		if (scrollY > scrollThreshold && !isAnimatedOut) {
			isAnimatedOut = true;
			// Stop any idle wind animation
			stopIdleWind();
			// If we were in absolute mode, switch back to fixed for animation
			if (isAbsoluteMode) {
				prepareForAnimation();
			}
			// Play forward at normal speed
			mainTimeline?.timeScale(1);
			mainTimeline?.play();
		} else if (scrollY <= scrollThreshold && isAnimatedOut) {
			isAnimatedOut = false;
			// Stop idle wind
			stopIdleWind();
			// If in absolute mode, prepare for reverse animation
			if (isAbsoluteMode) {
				prepareForAnimation();
			}
			// Reverse at faster speed
			const reverseSpeed = animationDuration / reverseDuration;
			mainTimeline?.timeScale(reverseSpeed);
			mainTimeline?.reverse();
		}
	}

	function prepareForAnimation() {
		if (!isAbsoluteMode || !container) return;

		// Reset container to fixed centered positioning
		container.style.position = 'fixed';
		container.style.top = '';
		container.style.left = '';
		container.style.width = '100vw';
		container.style.height = '100vh';
		container.style.transform = '';

		// Reset wrapper to centered
		svgContainer.style.position = 'absolute';
		svgContainer.style.top = '50%';
		svgContainer.style.left = '50%';
		svgContainer.style.transform = `translate(-50%, -50%) scale(${logoScale})`;

		// Set elements to their animated end positions (path offsets, not absolute positions)
		elementData.forEach(({ el, path, targetRotation, targetRotateX, targetRotateY, targetScale }) => {
			const endPoint = path[path.length - 1];
			gsap.set(el, {
				x: endPoint.x,
				y: endPoint.y,
				rotation: targetRotation,
				rotateX: targetRotateX,
				rotateY: targetRotateY,
				scale: targetScale,
				opacity: 0.85
			});
		});

		isAbsoluteMode = false;
	}

	function initIdleWind() {
		stopIdleWind();

		elementData.forEach(({ el }) => {
			const duration = 3 + Math.random() * 3;
			const xAmp = (15 + Math.random() * 25) * windIntensity;
			const yAmp = (10 + Math.random() * 20) * windIntensity;
			const delay = Math.random() * 2;

			// Use actual pixel offsets since we're now in absolute mode
			const currentX = gsap.getProperty(el, 'x') as number;
			const currentY = gsap.getProperty(el, 'y') as number;

			const tweenX = gsap.to(el, {
				x: currentX + xAmp,
				duration: duration,
				ease: 'sine.inOut',
				yoyo: true,
				repeat: -1,
				delay: delay
			});

			const tweenY = gsap.to(el, {
				y: currentY + yAmp,
				duration: duration * 1.2,
				ease: 'sine.inOut',
				yoyo: true,
				repeat: -1,
				delay: delay + 0.3
			});

			// Subtle 3D rotation wobble
			const currentRotX = gsap.getProperty(el, 'rotateX') as number;
			const currentRotY = gsap.getProperty(el, 'rotateY') as number;

			const tweenRotX = gsap.to(el, {
				rotateX: currentRotX + (2 + Math.random() * 3) * windIntensity,
				duration: duration * 0.9,
				ease: 'sine.inOut',
				yoyo: true,
				repeat: -1,
				delay: delay + 0.5
			});

			const tweenRotY = gsap.to(el, {
				rotateY: currentRotY + (2 + Math.random() * 3) * windIntensity,
				duration: duration * 1.1,
				ease: 'sine.inOut',
				yoyo: true,
				repeat: -1,
				delay: delay + 0.7
			});

			idleTweens.push(tweenX, tweenY, tweenRotX, tweenRotY);
		});
	}

	function stopIdleWind() {
		idleTweens.forEach((tween) => tween.kill());
		idleTweens = [];
	}

	function handleResize() {
		cleanup();
		hasInitialized = false;
		isAnimatedOut = false;
		isAbsoluteMode = false;
		requestAnimationFrame(() => {
			initAnimations();
		});
	}

	function cleanup() {
		if (!browser) return;

		window.removeEventListener('scroll', handleScroll);
		window.removeEventListener('resize', handleResize);

		if (mainTimeline) {
			mainTimeline.kill();
			mainTimeline = null;
		}

		stopIdleWind();

		// Reset to fixed mode
		if (container) {
			container.style.position = '';
			container.style.top = '';
			container.style.left = '';
			container.style.width = '';
			container.style.height = '';
			container.style.transform = '';
		}
		if (svgContainer) {
			svgContainer.style.position = '';
			svgContainer.style.top = '';
			svgContainer.style.left = '';
			svgContainer.style.transform = '';
		}

		elementData = [];
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
				hasInitialized = false;
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
		overflow: visible;
		perspective: var(--perspective, 1200px);
		perspective-origin: 50% 50%;
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
