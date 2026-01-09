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
	export let logoSize: number = 400; // Target size in pixels for the logo
	export let scrollThreshold: number = 15;
	export let windIntensity: number = 1;
	export let pathCurviness: number = 1.2;

	// Internal state
	let container: HTMLDivElement;
	let svgContainer: HTMLDivElement;
	let animatedElements: HTMLDivElement[] = [];
	let mainTimeline: gsap.core.Timeline | null = null;
	let idleTweens: gsap.core.Tween[] = [];
	let prefersReducedMotion = false;
	let viewportWidth = 0;
	let viewportHeight = 0;
	let pageHeight = 0;
	let isAnimatedOut = false;
	let hasInitialized = false;

	interface ElementData {
		wrapper: HTMLDivElement;
		startX: number;
		startY: number;
		endX: number;
		endY: number;
		path: { x: number; y: number }[];
		endRotation: number;
		endRotateX: number;
		endRotateY: number;
		endScale: number;
	}
	let elementData: ElementData[] = [];

	// Extract gradient colors from SVG defs
	function extractGradientColors(svgString: string): Map<string, { start: string; end: string; angle: number }> {
		const gradientMap = new Map<string, { start: string; end: string; angle: number }>();
		const parser = new DOMParser();
		const doc = parser.parseFromString(svgString, 'image/svg+xml');

		const gradients = doc.querySelectorAll('linearGradient');
		gradients.forEach((gradient) => {
			const id = gradient.getAttribute('id');
			if (!id) return;

			const stops = gradient.querySelectorAll('stop');
			if (stops.length < 2) return;

			// Extract colors from stop elements
			const startStyle = stops[0].getAttribute('style') || '';
			const endStyle = stops[stops.length - 1].getAttribute('style') || '';

			const startMatch = startStyle.match(/stop-color:\s*([^;\s]+)/);
			const endMatch = endStyle.match(/stop-color:\s*([^;\s]+)/);

			const startColor = startMatch ? startMatch[1] : '#f28d1d';
			const endColor = endMatch ? endMatch[1] : '#a40054';

			// Get transform to determine angle (simplified - use default diagonal)
			gradientMap.set(id, {
				start: startColor,
				end: endColor,
				angle: 135 // Default diagonal gradient
			});
		});

		return gradientMap;
	}

	// Create individual SVG element with inline gradient and proper viewBox
	function createElementSvg(
		pathElement: Element,
		gradientMap: Map<string, { start: string; end: string; angle: number }>,
		bounds: { x: number; y: number; width: number; height: number }
	): string {
		const pathD = pathElement.getAttribute('d') || '';
		const transform = pathElement.getAttribute('transform') || '';
		const style = pathElement.getAttribute('style') || '';

		// Extract gradient ID from fill
		const fillMatch = style.match(/fill:\s*url\(#([^)]+)\)/);
		let fillStyle = 'fill: #f28d1d;'; // Default fallback
		let gradientDef = '';

		if (fillMatch) {
			const gradientId = fillMatch[1];
			const gradient = gradientMap.get(gradientId);
			if (gradient) {
				// Create inline gradient definition
				const newGradientId = `gradient-${Math.random().toString(36).substr(2, 9)}`;
				gradientDef = `
					<defs>
						<linearGradient id="${newGradientId}" x1="0%" y1="0%" x2="100%" y2="100%">
							<stop offset="0%" style="stop-color:${gradient.start};stop-opacity:1" />
							<stop offset="100%" style="stop-color:${gradient.end};stop-opacity:1" />
						</linearGradient>
					</defs>
				`;
				fillStyle = `fill: url(#${newGradientId});`;
			}
		}

		// Remove the old fill from style and add new one
		const cleanedStyle = style.replace(/fill:\s*url\([^)]+\);?/, '').replace(/stroke:\s*[^;]+;?/g, '').trim();
		const newStyle = (cleanedStyle ? cleanedStyle + ';' : '') + fillStyle + 'stroke:none;';

		// Add padding to viewBox to prevent clipping
		const padding = 5;
		const vbX = bounds.x - padding;
		const vbY = bounds.y - padding;
		const vbW = bounds.width + padding * 2;
		const vbH = bounds.height + padding * 2;

		return `
			<svg viewBox="${vbX} ${vbY} ${vbW} ${vbH}" preserveAspectRatio="xMidYMid meet" style="overflow:visible;">
				${gradientDef}
				<path d="${pathD}" transform="${transform}" style="${newStyle}" />
			</svg>
		`;
	}

	// Generate final position - distributed across FULL page height
	function generateEndPosition(index: number, totalElements: number): { x: number; y: number } {
		const segmentHeight = pageHeight / totalElements;
		const segmentStart = index * segmentHeight;
		const segmentEnd = (index + 1) * segmentHeight;
		const padding = segmentHeight * 0.1;
		const y = segmentStart + padding + Math.random() * (segmentEnd - segmentStart - padding * 2);

		const roll = Math.random();
		const edgePadding = 100;
		const safeLeft = edgePadding;
		const safeRight = viewportWidth - edgePadding;
		const leftZoneEnd = viewportWidth * 0.25;
		const rightZoneStart = viewportWidth * 0.75;

		let x: number;
		if (roll < 0.45) {
			x = safeLeft + Math.random() * (leftZoneEnd - safeLeft);
		} else if (roll < 0.90) {
			x = rightZoneStart + Math.random() * (safeRight - rightZoneStart);
		} else {
			x = leftZoneEnd + Math.random() * (rightZoneStart - leftZoneEnd);
		}
		x = Math.max(safeLeft, Math.min(safeRight, x));

		return { x, y };
	}

	// Generate a curved path from start to end
	function generatePath(
		startX: number,
		startY: number,
		endX: number,
		endY: number
	): { x: number; y: number }[] {
		const points: { x: number; y: number }[] = [];
		points.push({ x: startX, y: startY });

		const dx = endX - startX;
		const dy = endY - startY;
		const numControlPoints = 2 + Math.floor(Math.random() * 2);

		for (let i = 1; i <= numControlPoints; i++) {
			const t = i / (numControlPoints + 1);
			const baseX = startX + dx * t;
			const baseY = startY + dy * t;
			const perpOffset = (Math.random() - 0.5) * Math.min(viewportWidth, viewportHeight) * 0.3;
			const angle = Math.atan2(dy, dx) + Math.PI / 2;

			points.push({
				x: baseX + Math.cos(angle) * perpOffset,
				y: baseY + Math.sin(angle) * perpOffset
			});
		}

		points.push({ x: endX, y: endY });
		return points;
	}

	// Parse path bounding box from d attribute and transform
	function getPathBounds(pathElement: Element): { x: number; y: number; width: number; height: number } {
		const d = pathElement.getAttribute('d') || '';
		const transform = pathElement.getAttribute('transform') || '';

		// Parse SVG path commands properly
		let currentX = 0;
		let currentY = 0;
		let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;

		// Tokenize path: split by commands while keeping the command letter
		const commandRegex = /([MmLlHhVvCcSsQqTtAaZz])([^MmLlHhVvCcSsQqTtAaZz]*)/g;
		let match;

		while ((match = commandRegex.exec(d)) !== null) {
			const cmd = match[1];
			const args = match[2].trim();
			const numbers = args.match(/-?\d+\.?\d*/g)?.map(Number) || [];

			switch (cmd) {
				case 'M': // Move to (absolute)
					if (numbers.length >= 2) {
						currentX = numbers[0];
						currentY = numbers[1];
						minX = Math.min(minX, currentX);
						minY = Math.min(minY, currentY);
						maxX = Math.max(maxX, currentX);
						maxY = Math.max(maxY, currentY);
					}
					break;
				case 'm': // Move to (relative)
					if (numbers.length >= 2) {
						currentX += numbers[0];
						currentY += numbers[1];
						minX = Math.min(minX, currentX);
						minY = Math.min(minY, currentY);
						maxX = Math.max(maxX, currentX);
						maxY = Math.max(maxY, currentY);
					}
					break;
				case 'L': // Line to (absolute)
					for (let i = 0; i < numbers.length; i += 2) {
						currentX = numbers[i];
						currentY = numbers[i + 1];
						minX = Math.min(minX, currentX);
						minY = Math.min(minY, currentY);
						maxX = Math.max(maxX, currentX);
						maxY = Math.max(maxY, currentY);
					}
					break;
				case 'l': // Line to (relative)
					for (let i = 0; i < numbers.length; i += 2) {
						currentX += numbers[i];
						currentY += numbers[i + 1];
						minX = Math.min(minX, currentX);
						minY = Math.min(minY, currentY);
						maxX = Math.max(maxX, currentX);
						maxY = Math.max(maxY, currentY);
					}
					break;
				case 'H': // Horizontal line (absolute)
					for (const n of numbers) {
						currentX = n;
						minX = Math.min(minX, currentX);
						maxX = Math.max(maxX, currentX);
					}
					break;
				case 'h': // Horizontal line (relative)
					for (const n of numbers) {
						currentX += n;
						minX = Math.min(minX, currentX);
						maxX = Math.max(maxX, currentX);
					}
					break;
				case 'V': // Vertical line (absolute)
					for (const n of numbers) {
						currentY = n;
						minY = Math.min(minY, currentY);
						maxY = Math.max(maxY, currentY);
					}
					break;
				case 'v': // Vertical line (relative)
					for (const n of numbers) {
						currentY += n;
						minY = Math.min(minY, currentY);
						maxY = Math.max(maxY, currentY);
					}
					break;
				case 'C': // Cubic bezier (absolute) - 3 pairs of coords
					for (let i = 0; i < numbers.length; i += 6) {
						// Control point 1
						minX = Math.min(minX, numbers[i]);
						minY = Math.min(minY, numbers[i + 1]);
						maxX = Math.max(maxX, numbers[i]);
						maxY = Math.max(maxY, numbers[i + 1]);
						// Control point 2
						minX = Math.min(minX, numbers[i + 2]);
						minY = Math.min(minY, numbers[i + 3]);
						maxX = Math.max(maxX, numbers[i + 2]);
						maxY = Math.max(maxY, numbers[i + 3]);
						// End point
						currentX = numbers[i + 4];
						currentY = numbers[i + 5];
						minX = Math.min(minX, currentX);
						minY = Math.min(minY, currentY);
						maxX = Math.max(maxX, currentX);
						maxY = Math.max(maxY, currentY);
					}
					break;
				case 'c': // Cubic bezier (relative)
					for (let i = 0; i < numbers.length; i += 6) {
						// Control points and end point are relative
						const cp1x = currentX + numbers[i];
						const cp1y = currentY + numbers[i + 1];
						const cp2x = currentX + numbers[i + 2];
						const cp2y = currentY + numbers[i + 3];
						const endX = currentX + numbers[i + 4];
						const endY = currentY + numbers[i + 5];

						minX = Math.min(minX, cp1x, cp2x, endX);
						minY = Math.min(minY, cp1y, cp2y, endY);
						maxX = Math.max(maxX, cp1x, cp2x, endX);
						maxY = Math.max(maxY, cp1y, cp2y, endY);

						currentX = endX;
						currentY = endY;
					}
					break;
				case 'Z':
				case 'z':
					// Close path - no coordinates
					break;
				default:
					// For other commands, just update bounds with any numbers found as pairs
					for (let i = 0; i < numbers.length; i += 2) {
						if (i + 1 < numbers.length) {
							minX = Math.min(minX, numbers[i]);
							minY = Math.min(minY, numbers[i + 1]);
							maxX = Math.max(maxX, numbers[i]);
							maxY = Math.max(maxY, numbers[i + 1]);
						}
					}
			}
		}

		// Fallback if parsing failed
		if (!isFinite(minX) || !isFinite(minY) || !isFinite(maxX) || !isFinite(maxY)) {
			return { x: 0, y: 0, width: 100, height: 100 };
		}

		// Apply transform if present (handle matrix transform)
		// matrix(a,b,c,d,e,f) transforms: x' = a*x + c*y + e, y' = b*x + d*y + f
		const matrixMatch = transform.match(/matrix\(([^)]+)\)/);
		if (matrixMatch) {
			const [a, b, c, dd, e, f] = matrixMatch[1].split(',').map(Number);

			// Transform the bounding box corners
			const corners = [
				{ x: minX, y: minY },
				{ x: maxX, y: minY },
				{ x: minX, y: maxY },
				{ x: maxX, y: maxY }
			];

			const transformedCorners = corners.map(corner => ({
				x: a * corner.x + c * corner.y + e,
				y: b * corner.x + dd * corner.y + f
			}));

			minX = Math.min(...transformedCorners.map(c => c.x));
			minY = Math.min(...transformedCorners.map(c => c.y));
			maxX = Math.max(...transformedCorners.map(c => c.x));
			maxY = Math.max(...transformedCorners.map(c => c.y));
		}

		return {
			x: minX,
			y: minY,
			width: maxX - minX,
			height: maxY - minY
		};
	}

	function initAnimations() {
		if (!browser || !svgContainer || hasInitialized || !svgContent) return;

		prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
		viewportWidth = window.innerWidth;
		viewportHeight = window.innerHeight;
		pageHeight = document.documentElement.scrollHeight;

		// Parse SVG content
		const parser = new DOMParser();
		const svgDoc = parser.parseFromString(svgContent, 'image/svg+xml');
		const originalSvg = svgDoc.querySelector('svg');
		if (!originalSvg) return;

		// Get viewBox for proper scaling
		const viewBox = originalSvg.getAttribute('viewBox') || '0 0 2666.6667 2666.6667';
		const viewBoxParts = viewBox.split(' ').map(Number);
		const svgWidth = viewBoxParts[2] || 2666.6667;
		const svgHeight = viewBoxParts[3] || 2666.6667;

		// Extract gradient definitions
		const gradientMap = extractGradientColors(svgContent);

		// Find all path elements
		const paths = originalSvg.querySelectorAll('path');
		if (paths.length === 0) return;

		// Clear any existing animated elements
		animatedElements.forEach(el => el.remove());
		animatedElements = [];

		// Calculate logo center position
		const logoCenterX = viewportWidth / 2;
		const logoCenterY = viewportHeight / 2;

		// Calculate scale factor to fit logo to desired size
		const scaleFactor = logoSize / Math.max(svgWidth, svgHeight);

		elementData = [];

		paths.forEach((path, index) => {
			// Get path bounding box in SVG coordinates (with transform applied)
			const pathBBox = getPathBounds(path);

			// Calculate element center in SVG coordinates
			const elCenterX = pathBBox.x + pathBBox.width / 2;
			const elCenterY = pathBBox.y + pathBBox.height / 2;

			// Calculate offset from SVG center
			const offsetX = (elCenterX - svgWidth / 2) * scaleFactor;
			const offsetY = (elCenterY - svgHeight / 2) * scaleFactor;

			// Starting position in page coordinates
			const startX = logoCenterX + offsetX;
			const startY = logoCenterY + offsetY;

			// Calculate element size (use a reasonable minimum)
			const elWidth = Math.max(pathBBox.width * scaleFactor, 20);
			const elHeight = Math.max(pathBBox.height * scaleFactor, 20);

			// Create wrapper div for this element
			const wrapper = document.createElement('div');
			wrapper.className = 'animated-element';
			wrapper.style.cssText = `
				position: absolute;
				top: 0;
				left: 0;
				width: ${elWidth}px;
				height: ${elHeight}px;
				transform-style: preserve-3d;
				will-change: transform, opacity;
			`;

			// Create SVG with inline gradient and element-specific viewBox
			wrapper.innerHTML = createElementSvg(path, gradientMap, pathBBox);

			// Style the inner SVG
			const innerSvg = wrapper.querySelector('svg');
			if (innerSvg) {
				innerSvg.style.width = '100%';
				innerSvg.style.height = '100%';
			}

			svgContainer.appendChild(wrapper);
			animatedElements.push(wrapper);

			// Generate end position
			const endPos = generateEndPosition(index, paths.length);
			const pathPoints = generatePath(startX, startY, endPos.x, endPos.y);

			// Random 3D rotations and scale for end state
			const endRotation = (Math.random() - 0.5) * 30;
			const endRotateX = (Math.random() - 0.5) * 20;
			const endRotateY = (Math.random() - 0.5) * 20;
			const endScale = 1.5 + Math.random() * 1.5; // 1.5x to 3x at end

			// Initialize element at starting position
			gsap.set(wrapper, {
				x: startX,
				y: startY,
				xPercent: -50,
				yPercent: -50,
				transformOrigin: '50% 50%',
				force3D: true,
				rotation: 0,
				rotateX: 0,
				rotateY: 0,
				scale: 1,
				opacity: 1
			});

			elementData.push({
				wrapper,
				startX,
				startY,
				endX: endPos.x,
				endY: endPos.y,
				path: pathPoints,
				endRotation,
				endRotateX,
				endRotateY,
				endScale
			});
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

		elementData.forEach(({ wrapper, endX, endY, endScale }) => {
			mainTimeline!.to(wrapper, {
				x: endX,
				y: endY,
				scale: endScale,
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
				initIdleWind();
			},
			onReverseComplete: () => {
				stopIdleWind();
			}
		});

		elementData.forEach(({ wrapper, path, endRotation, endRotateX, endRotateY, endScale }, index) => {
			const staggerDelay = index * 0.05;

			mainTimeline!.to(wrapper, {
				motionPath: {
					path: path,
					curviness: pathCurviness,
					autoRotate: false
				},
				rotation: endRotation,
				rotateX: endRotateX,
				rotateY: endRotateY,
				scale: endScale,
				opacity: 0.85,
				duration: animationDuration,
				ease: 'power2.inOut'
			}, staggerDelay);
		});

		window.addEventListener('scroll', handleScroll, { passive: true });
	}

	function handleScroll() {
		const scrollY = window.scrollY;

		if (scrollY > scrollThreshold && !isAnimatedOut) {
			isAnimatedOut = true;
			stopIdleWind();
			mainTimeline?.timeScale(1);
			mainTimeline?.play();
		} else if (scrollY <= scrollThreshold && isAnimatedOut) {
			isAnimatedOut = false;
			stopIdleWind();
			const reverseSpeed = animationDuration / reverseDuration;
			mainTimeline?.timeScale(reverseSpeed);
			mainTimeline?.reverse();
		}
	}

	function initIdleWind() {
		stopIdleWind();

		elementData.forEach(({ wrapper }) => {
			const duration = 3 + Math.random() * 3;
			const xAmp = (15 + Math.random() * 25) * windIntensity;
			const yAmp = (10 + Math.random() * 20) * windIntensity;
			const delay = Math.random() * 2;

			const currentX = gsap.getProperty(wrapper, 'x') as number;
			const currentY = gsap.getProperty(wrapper, 'y') as number;

			const tweenX = gsap.to(wrapper, {
				x: currentX + xAmp,
				duration: duration,
				ease: 'sine.inOut',
				yoyo: true,
				repeat: -1,
				delay: delay
			});

			const tweenY = gsap.to(wrapper, {
				y: currentY + yAmp,
				duration: duration * 1.2,
				ease: 'sine.inOut',
				yoyo: true,
				repeat: -1,
				delay: delay + 0.3
			});

			const currentRotX = gsap.getProperty(wrapper, 'rotateX') as number;
			const currentRotY = gsap.getProperty(wrapper, 'rotateY') as number;

			const tweenRotX = gsap.to(wrapper, {
				rotateX: currentRotX + (2 + Math.random() * 3) * windIntensity,
				duration: duration * 0.9,
				ease: 'sine.inOut',
				yoyo: true,
				repeat: -1,
				delay: delay + 0.5
			});

			const tweenRotY = gsap.to(wrapper, {
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

		// Remove animated elements
		animatedElements.forEach(el => el.remove());
		animatedElements = [];
		elementData = [];
	}

	onMount(() => {
		if (browser) {
			requestAnimationFrame(() => {
				requestAnimationFrame(() => {
					initAnimations();

					// Handle page reload while scrolled
					if (window.scrollY > scrollThreshold) {
						isAnimatedOut = true;
						mainTimeline?.play();
					}
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
>
	<div class="logo-animation-wrapper" bind:this={svgContainer}>
		<!-- Elements are created dynamically -->
	</div>
</div>

<style>
	.logo-animation-container {
		position: absolute;
		top: 0;
		left: 0;
		width: 100%;
		height: 100%;
		min-height: 100vh;
		z-index: var(--z-index, -1);
		pointer-events: none;
		overflow: visible;
		perspective: var(--perspective, 1200px);
		perspective-origin: 50% 50%;
	}

	.logo-animation-wrapper {
		position: absolute;
		top: 0;
		left: 0;
		width: 100%;
		height: 100%;
		transform-style: preserve-3d;
	}

	.logo-animation-wrapper :global(.animated-element) {
		transform-style: preserve-3d;
		backface-visibility: visible;
	}

	.logo-animation-wrapper :global(.animated-element svg) {
		overflow: visible;
	}

	@media (prefers-reduced-motion: reduce) {
		.logo-animation-wrapper :global(.animated-element) {
			transition: transform 0.5s ease-out, opacity 0.5s ease-out !important;
		}
	}
</style>
