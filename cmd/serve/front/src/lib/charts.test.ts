import { describe, expect, it } from 'vitest';
import {
	calendarWeeks,
	countChange,
	durationChange,
	heatLevel,
	monotonePath,
	niceBytesCeil,
	niceCeil,
	niceSecondsCeil,
	rateChange,
	runs,
	streaks,
	tickIndexes
} from '#lib/charts.ts';

describe('niceCeil', () => {
	it('rounds up to 1, 2, 2.5 or 5 times a power of ten', () => {
		expect(niceCeil(0)).toBe(1);
		expect(niceCeil(7)).toBe(10);
		expect(niceCeil(17)).toBe(20);
		expect(niceCeil(21)).toBe(25);
		expect(niceCeil(437)).toBe(500);
		expect(niceCeil(1000)).toBe(1000);
	});
});

describe('nice scales in their unit', () => {
	it('rounds bytes in the unit they are shown in', () => {
		expect(niceBytesCeil(700)).toBe(1000);
		expect(niceBytesCeil(18.6 * 1024 ** 3)).toBe(20 * 1024 ** 3);
	});

	it('rounds seconds, then minutes, then hours', () => {
		expect(niceSecondsCeil(47)).toBe(50);
		expect(niceSecondsCeil(260)).toBe(300);
		expect(niceSecondsCeil(3 * 3600 + 5)).toBe(5 * 3600);
	});
});

describe('monotonePath', () => {
	it('never leaves the range of its points', () => {
		const path = monotonePath([
			[0, 100],
			[10, 0],
			[20, 0],
			[30, 100]
		]);
		const ys = [...path.matchAll(/(-?\d+(?:\.\d+)?) (-?\d+(?:\.\d+)?)/g)].map((m) => Number(m[2]));
		expect(Math.min(...ys)).toBeGreaterThanOrEqual(0);
		expect(Math.max(...ys)).toBeLessThanOrEqual(100);
	});

	it('handles one point and none', () => {
		expect(monotonePath([])).toBe('');
		expect(monotonePath([[4, 5]])).toBe('M4 5');
	});
});

describe('runs', () => {
	it('splits a series at missing days', () => {
		expect(runs([1, 2, null, 3, undefined, null, 4])).toEqual([
			{ start: 0, values: [1, 2] },
			{ start: 3, values: [3] },
			{ start: 6, values: [4] }
		]);
	});
});

describe('tickIndexes', () => {
	it('labels every day in a week and every 7th or 14th day back from today', () => {
		expect(tickIndexes(7)).toEqual([0, 1, 2, 3, 4, 5, 6]);
		expect(tickIndexes(30)).toEqual([1, 8, 15, 22, 29]);
		expect(tickIndexes(90).at(-1)).toBe(89);
		expect(tickIndexes(90)).toHaveLength(7);
	});
});

describe('calendar', () => {
	it('sorts days into five levels', () => {
		expect([0, 1, 3, 5, 8].map((v) => heatLevel(v, 8))).toEqual([0, 1, 2, 3, 4]);
	});

	it('lays out weeks Monday to Sunday ending this week', () => {
		const weeks = calendarWeeks('2026-11-17', 13); // a Tuesday
		expect(weeks).toHaveLength(13);
		expect(weeks[12][0]).toEqual({ date: '2026-11-16', future: false });
		expect(weeks[12][1]).toEqual({ date: '2026-11-17', future: false });
		expect(weeks[12][2].future).toBe(true);
		expect(weeks[0][0].date).toBe('2026-08-24');
	});

	it('keeps the current run alive while today is still empty', () => {
		expect(streaks([1, 1, 0, 1, 1, 1, 0])).toEqual({ current: 3, longest: 3 });
		expect(streaks([1, 0, 0])).toEqual({ current: 0, longest: 1 });
	});
});

describe('changes in words', () => {
	it('compares counts', () => {
		expect(countChange(12, 8, 30)).toMatchObject({ direction: 'up', good: true });
		expect(countChange(12, 8, 30).text).toBe('4 more than the previous 30 days');
		expect(countChange(5, 0, 7).direction).toBe('none');
		expect(countChange(3, 3, 7).text).toBe('Same as the previous 7 days');
	});

	it('compares rates in points and durations by speed', () => {
		expect(rateChange(0.93, 0.92, 30).text).toBe('1 point higher than the previous 30 days');
		expect(rateChange(null, 0.9, 30).direction).toBe('none');
		expect(durationChange(52, 58, 30)).toMatchObject({ direction: 'down', good: true });
		expect(durationChange(52, 58, 30).text).toBe('6 s faster than the previous 30 days');
	});
});
