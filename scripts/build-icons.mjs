// Renders the checkship app icon from the brand mark.
//
//   assets/brand/logo-mark.svg  ->  assets/brand/icon.png (512x512, transparent)
//                               ->  assets/brand/icon.ico (16..256, PNG-compressed entries)
//
// Rule: the app icon is the same mark at stroke 2.25 on a 512px transparent canvas,
// in the light brand blue (#2f5bea). Run with `npm run build:icons` from the repo root.
import { readFile, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import { Resvg } from '@resvg/resvg-js';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const brandDir = join(root, 'assets', 'brand');

const ICON_SIZE = 512;
const ICON_STROKE = '2.25';
const ICON_COLOR = '#2f5bea';
const ICO_SIZES = [16, 24, 32, 48, 64, 128, 256];

/** Turn the adaptive source mark into a fixed-colour, icon-weight SVG. */
function iconSvg(source) {
	return source
		.replace(/<style>[\s\S]*?<\/style>/, '')
		.replace('stroke-width="2"', `stroke-width="${ICON_STROKE}"`)
		.replace('class="mark"', `stroke="${ICON_COLOR}"`);
}

function render(svg, size) {
	return new Resvg(svg, {
		fitTo: { mode: 'width', value: size },
		background: 'rgba(0,0,0,0)'
	})
		.render()
		.asPng();
}

/** Builds an ICO container whose entries are PNG images (supported since Windows Vista). */
function ico(images) {
	const header = Buffer.alloc(6);
	header.writeUInt16LE(0, 0); // reserved
	header.writeUInt16LE(1, 2); // type: icon
	header.writeUInt16LE(images.length, 4);

	const entries = [];
	let offset = 6 + images.length * 16;

	for (const { size, png } of images) {
		const entry = Buffer.alloc(16);
		entry.writeUInt8(size >= 256 ? 0 : size, 0); // width (0 = 256)
		entry.writeUInt8(size >= 256 ? 0 : size, 1); // height
		entry.writeUInt8(0, 2); // palette
		entry.writeUInt8(0, 3); // reserved
		entry.writeUInt16LE(1, 4); // colour planes
		entry.writeUInt16LE(32, 6); // bits per pixel
		entry.writeUInt32LE(png.length, 8);
		entry.writeUInt32LE(offset, 12);
		entries.push(entry);
		offset += png.length;
	}

	return Buffer.concat([header, ...entries, ...images.map((i) => i.png)]);
}

const source = await readFile(join(brandDir, 'logo-mark.svg'), 'utf8');
const svg = iconSvg(source);

await writeFile(join(brandDir, 'icon.png'), render(svg, ICON_SIZE));
await writeFile(
	join(brandDir, 'icon.ico'),
	ico(ICO_SIZES.map((size) => ({ size, png: render(svg, size) })))
);

console.log(`icons written to ${brandDir}: icon.png (${ICON_SIZE}px), icon.ico (${ICO_SIZES.join(', ')})`);
