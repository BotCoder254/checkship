import { describe, expect, it } from 'vitest';
import { dayKey, dayLabel, groupBy, groupByDay } from '#lib/timeline.ts';

// Local times, so the tests hold in any time zone.
const now = new Date(2026, 10, 17, 15, 0); // Tuesday 17 November 2026, 15:00

describe('dayLabel', () => {
	it('names today and yesterday', () => {
		expect(dayLabel(new Date(2026, 10, 17, 0, 5), 'en', now)).toBe('Today');
		expect(dayLabel(new Date(2026, 10, 16, 23, 59), 'en', now)).toBe('Yesterday');
		expect(dayLabel(new Date(2026, 10, 16, 9), 'fr', now)).toBe('Hier');
	});

	it('names the weekday within the last week', () => {
		expect(dayLabel(new Date(2026, 10, 15, 12), 'en', now)).toBe('Sunday');
		expect(dayLabel(new Date(2026, 10, 11, 12), 'en', now)).toBe('Wednesday');
		expect(dayLabel(new Date(2026, 10, 11, 12), 'fr', now)).toBe('Mercredi');
	});

	it('gives the date from a week back, and the year only when it differs', () => {
		expect(dayLabel(new Date(2026, 10, 10, 12), 'en', now)).toBe('November 10');
		expect(dayLabel(new Date(2025, 10, 10, 12), 'en', now)).toBe('November 10, 2025');
		expect(dayLabel(new Date(2026, 10, 10, 12), 'fr', now)).toBe('10 novembre');
	});
});

describe('groupByDay', () => {
	it('starts a group each time the day changes, keeping the order', () => {
		const items = [
			new Date(2026, 10, 17, 14),
			new Date(2026, 10, 17, 9),
			new Date(2026, 10, 16, 20),
			new Date(2026, 10, 3, 8)
		];
		const groups = groupByDay(items, (d) => d, 'en', now);
		expect(groups.map((g) => [g.label, g.items.length])).toEqual([
			['Today', 2],
			['Yesterday', 1],
			['November 3', 1]
		]);
		expect(groups[0].key).toBe(dayKey(items[0]));
	});

	it('returns nothing for no items', () => {
		expect(groupByDay([], (d: Date) => d, 'en', now)).toEqual([]);
	});
});

describe('groupBy', () => {
	it('follows the given order and drops empty groups', () => {
		const groups = groupBy(['b', 'a', 'b', 'c'], (x) => x as 'a' | 'b' | 'c', ['c', 'a', 'b']);
		expect(groups).toEqual([
			{ key: 'c', items: ['c'] },
			{ key: 'a', items: ['a'] },
			{ key: 'b', items: ['b', 'b'] }
		]);
	});
});
