<!--
	Draws an image dithered into two palette tokens (DESIGN.md §5, sign-in art panel).

	Ground pixels take `--canvas`, lit pixels take `--accent`, both read from CSS when drawing, so
	the art follows light and dark mode. It is static: drawn once per size, theme or image change,
	never animated (DESIGN.md §9). Decorative, hidden from assistive technology.
-->
<script lang="ts">
	import { mode } from 'mode-watcher';
	import { cn } from '#lib/utils.js';
	import { containRect, dither, parseHex, type DitherPattern } from '#lib/dither.ts';

	type Props = {
		/** Same-origin image (bundled under static/), drawn with object-fit: contain. */
		src: string;
		/** Cell size in CSS pixels. */
		cell?: number;
		/** Threshold bias, 0 to 1. */
		threshold?: number;
		pattern?: DitherPattern;
		class?: string;
	};

	let { src, cell = 2, threshold = 0.45, pattern = 'bayer', class: className }: Props = $props();

	let container = $state<HTMLDivElement>();
	let canvas = $state<HTMLCanvasElement>();
	let image = $state<HTMLImageElement>();
	let size = $state({ width: 0, height: 0 });

	$effect(() => {
		const img = new Image();
		img.decoding = 'async';
		img.onload = () => (image = img);
		img.src = src;

		return () => {
			img.onload = null;
		};
	});

	$effect(() => {
		if (!container) return;

		const observer = new ResizeObserver(([entry]) => {
			const { width, height } = entry.contentRect;
			size = { width: Math.round(width), height: Math.round(height) };
		});
		observer.observe(container);

		return () => observer.disconnect();
	});

	$effect(() => {
		// Redraw when the theme changes: mode-watcher swaps the `.dark` class, which changes the tokens.
		void mode.current;

		const img = image;
		const target = canvas;
		const { width, height } = size;
		const options = { cell, threshold, pattern };

		if (!img || !target || width === 0 || height === 0) return;

		const frame = requestAnimationFrame(() => draw(target, img, width, height, options));
		return () => cancelAnimationFrame(frame);
	});

	function draw(
		target: HTMLCanvasElement,
		img: HTMLImageElement,
		width: number,
		height: number,
		options: { cell: number; threshold: number; pattern: DitherPattern }
	) {
		const ratio = window.devicePixelRatio || 1;
		const w = Math.round(width * ratio);
		const h = Math.round(height * ratio);
		const styles = getComputedStyle(document.documentElement);

		const source = document.createElement('canvas');
		source.width = w;
		source.height = h;
		const sourceContext = source.getContext('2d', { willReadFrequently: true });
		if (!sourceContext) return;

		sourceContext.fillStyle = '#000';
		sourceContext.fillRect(0, 0, w, h);
		const rect = containRect(img.naturalWidth || w, img.naturalHeight || h, w, h);
		sourceContext.drawImage(img, rect.x, rect.y, rect.width, rect.height);

		target.width = w;
		target.height = h;
		const context = target.getContext('2d');
		if (!context) return;

		const output = context.createImageData(w, h);
		dither(sourceContext.getImageData(0, 0, w, h).data, output.data, w, h, {
			ground: parseHex(styles.getPropertyValue('--canvas')),
			light: parseHex(styles.getPropertyValue('--accent')),
			cell: options.cell * ratio,
			threshold: options.threshold,
			pattern: options.pattern
		});
		context.putImageData(output, 0, 0);
	}
</script>

<div
	bind:this={container}
	class={cn('relative overflow-hidden bg-canvas', className)}
	aria-hidden="true"
>
	<canvas bind:this={canvas} class="absolute inset-0 size-full [image-rendering:pixelated]"
	></canvas>
</div>
