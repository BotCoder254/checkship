/**
 * Ordered (Bayer) and halftone dithering into two flat colours.
 *
 * The sign-in art panel (DESIGN.md §5) uses it to turn a luminance image into the palette's
 * `canvas` (ground) and `accent` (lit) tokens, so the result stays flat and solid even though the
 * source has soft light in it. Pure and DOM-free so it can be unit-tested.
 */

export type DitherPattern = 'bayer' | 'halftone';

export type Rgb = [number, number, number];

export type DitherOptions = {
	/** Colour for pixels darker than the threshold. */
	ground: Rgb;
	/** Colour for pixels lighter than the threshold. */
	light: Rgb;
	/** Side of one dither cell, in pixels of the given buffer. */
	cell: number;
	/** Bias of the threshold, 0 to 1. Lower lights more cells. */
	threshold: number;
	pattern: DitherPattern;
};

// 8x8 Bayer matrix, values 0..63.
const BAYER_8 = [
	0, 32, 8, 40, 2, 34, 10, 42, 48, 16, 56, 24, 50, 18, 58, 26, 12, 44, 4, 36, 14, 46, 6, 38, 60, 28,
	52, 20, 62, 30, 54, 22, 3, 35, 11, 43, 1, 33, 9, 41, 51, 19, 59, 27, 49, 17, 57, 25, 15, 47, 7,
	39, 13, 45, 5, 37, 63, 31, 55, 23, 61, 29, 53, 21
];

/** Parses `#rgb` or `#rrggbb` (the palette tokens are hex). Anything else is black. */
export function parseHex(value: string): Rgb {
	const hex = value.trim().replace(/^#/, '');
	const full =
		hex.length === 3
			? hex
					.split('')
					.map((c) => c + c)
					.join('')
			: hex;

	if (!/^[0-9a-f]{6}$/i.test(full)) {
		return [0, 0, 0];
	}

	return [0, 2, 4].map((i) => parseInt(full.slice(i, i + 2), 16)) as Rgb;
}

function thresholdAt(pattern: DitherPattern, x: number, y: number, cell: number): number {
	if (pattern === 'halftone') {
		const angle = Math.PI / 4;
		const scale = cell * 2;
		const rx = x * Math.cos(angle) + y * Math.sin(angle);
		const ry = -x * Math.sin(angle) + y * Math.cos(angle);
		return (Math.sin(rx / scale) + Math.sin(ry / scale) + 2) / 4;
	}

	return BAYER_8[((y / cell) % 8) * 8 + ((x / cell) % 8)] / 64;
}

/**
 * Dithers `source` (RGBA, `width` x `height`) into `target` (same size). Each cell takes the
 * luminance of its top-left source pixel and becomes all `ground` or all `light`.
 */
export function dither(
	source: Uint8ClampedArray,
	target: Uint8ClampedArray,
	width: number,
	height: number,
	options: DitherOptions
): void {
	const cell = Math.max(1, Math.round(options.cell));
	const { ground, light, threshold, pattern } = options;

	for (let y = 0; y < height; y += cell) {
		for (let x = 0; x < width; x += cell) {
			const i = (y * width + x) * 4;
			const luminance = (0.299 * source[i] + 0.587 * source[i + 1] + 0.114 * source[i + 2]) / 255;
			const t = thresholdAt(pattern, x, y, cell) * (1 - threshold) + threshold * 0.5;
			const [r, g, b] = luminance < t ? ground : light;

			const yEnd = Math.min(y + cell, height);
			const xEnd = Math.min(x + cell, width);

			for (let cy = y; cy < yEnd; cy++) {
				for (let cx = x; cx < xEnd; cx++) {
					const o = (cy * width + cx) * 4;
					target[o] = r;
					target[o + 1] = g;
					target[o + 2] = b;
					target[o + 3] = 255;
				}
			}
		}
	}
}

/** Where to draw an image of `iw` x `ih` so it fits inside `w` x `h`, centred (object-fit: contain). */
export function containRect(iw: number, ih: number, w: number, h: number) {
	const scale = Math.min(w / iw, h / ih);
	const dw = iw * scale;
	const dh = ih * scale;

	return { x: (w - dw) / 2, y: (h - dh) / 2, width: dw, height: dh };
}
