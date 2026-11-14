import { describe, expect, it } from 'vitest';
import { containRect, dither, parseHex, type DitherOptions } from './dither';

const DARK: Pick<DitherOptions, 'ground' | 'light'> = {
	ground: parseHex('#0b0b0d'),
	light: parseHex('#7f95ff')
};

function solid(width: number, height: number, value: number): Uint8ClampedArray {
	const data = new Uint8ClampedArray(width * height * 4);
	for (let i = 0; i < data.length; i += 4) {
		data[i] = data[i + 1] = data[i + 2] = value;
		data[i + 3] = 255;
	}
	return data;
}

function colours(data: Uint8ClampedArray): Set<string> {
	const set = new Set<string>();
	for (let i = 0; i < data.length; i += 4) {
		set.add(`${data[i]},${data[i + 1]},${data[i + 2]},${data[i + 3]}`);
	}
	return set;
}

describe('parseHex', () => {
	it('parses long and short hex', () => {
		expect(parseHex('#7f95ff')).toEqual([127, 149, 255]);
		expect(parseHex(' #fff ')).toEqual([255, 255, 255]);
	});

	it('falls back to black for anything else', () => {
		expect(parseHex('rgb(1, 2, 3)')).toEqual([0, 0, 0]);
	});
});

describe('dither', () => {
	const options = { ...DARK, cell: 2, threshold: 0.45, pattern: 'bayer' as const };

	it('turns black into the ground colour only', () => {
		const target = new Uint8ClampedArray(16 * 16 * 4);
		dither(solid(16, 16, 0), target, 16, 16, options);
		expect(colours(target)).toEqual(new Set(['11,11,13,255']));
	});

	it('turns white into the light colour only', () => {
		const target = new Uint8ClampedArray(16 * 16 * 4);
		dither(solid(16, 16, 255), target, 16, 16, options);
		expect(colours(target)).toEqual(new Set(['127,149,255,255']));
	});

	it('mixes exactly two flat colours for a mid grey, in whole cells', () => {
		const size = 32;
		const target = new Uint8ClampedArray(size * size * 4);
		dither(solid(size, size, 128), target, size, size, options);
		expect(colours(target)).toEqual(new Set(['11,11,13,255', '127,149,255,255']));

		// Every 2x2 cell is a single colour.
		for (let y = 0; y < size; y += 2) {
			for (let x = 0; x < size; x += 2) {
				const at = (cx: number, cy: number) => target[(cy * size + cx) * 4];
				expect(new Set([at(x, y), at(x + 1, y), at(x, y + 1), at(x + 1, y + 1)]).size).toBe(1);
			}
		}
	});

	it('handles sizes that are not a multiple of the cell', () => {
		const target = new Uint8ClampedArray(5 * 3 * 4);
		dither(solid(5, 3, 255), target, 5, 3, options);
		expect(colours(target)).toEqual(new Set(['127,149,255,255']));
	});

	it('supports the halftone pattern', () => {
		const target = new Uint8ClampedArray(32 * 32 * 4);
		dither(solid(32, 32, 128), target, 32, 32, { ...options, pattern: 'halftone' });
		expect(colours(target).size).toBe(2);
	});
});

describe('containRect', () => {
	it('fits a portrait image into a wide box, centred', () => {
		expect(containRect(720, 900, 960, 1080)).toEqual({ x: 48, y: 0, width: 864, height: 1080 });
	});
});
