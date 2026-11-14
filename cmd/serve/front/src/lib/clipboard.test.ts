import { describe, expect, it } from 'vitest';
import { copyText, type ClipboardEnv } from './clipboard.ts';

type FakeArea = { value: string; selected: boolean; removed: boolean };

function fakeDocument(execResult: boolean | 'throw') {
	const areas: FakeArea[] = [];
	const doc = {
		activeElement: null,
		body: { appendChild: () => undefined },
		createElement: () => {
			const area = {
				value: '',
				selected: false,
				removed: false,
				style: {},
				setAttribute: () => undefined,
				select() {
					area.selected = true;
				},
				remove() {
					area.removed = true;
				}
			};
			areas.push(area);
			return area;
		},
		execCommand: () => {
			if (execResult === 'throw') throw new Error('not allowed');
			return execResult;
		}
	};
	return { doc: doc as unknown as ClipboardEnv['document'], areas };
}

describe('copyText', () => {
	it('uses the Clipboard API in a secure context', async () => {
		let written = '';
		const { doc, areas } = fakeDocument(true);

		const ok = await copyText('line', {
			isSecureContext: true,
			clipboard: { writeText: async (t) => void (written = t) },
			document: doc
		});

		expect(ok).toBe(true);
		expect(written).toBe('line');
		expect(areas).toHaveLength(0);
	});

	it('falls back to a selected textarea over plain HTTP', async () => {
		const { doc, areas } = fakeDocument(true);

		const ok = await copyText('line', { isSecureContext: false, document: doc });

		expect(ok).toBe(true);
		expect(areas[0].value).toBe('line');
		expect(areas[0].selected).toBe(true);
		expect(areas[0].removed).toBe(true);
	});

	it('falls back when the Clipboard API refuses', async () => {
		const { doc, areas } = fakeDocument(true);

		const ok = await copyText('line', {
			isSecureContext: true,
			clipboard: { writeText: () => Promise.reject(new Error('denied')) },
			document: doc
		});

		expect(ok).toBe(true);
		expect(areas).toHaveLength(1);
	});

	it('says so when nothing works', async () => {
		const refused = fakeDocument(false);
		const throwing = fakeDocument('throw');

		expect(await copyText('line', { isSecureContext: false, document: refused.doc })).toBe(false);
		expect(await copyText('line', { isSecureContext: false, document: throwing.doc })).toBe(false);
		expect(throwing.areas[0].removed).toBe(true);
		expect(await copyText('line', { isSecureContext: false })).toBe(false);
	});
});
