import { describe, expect, test } from 'vitest';
import {
	appendFeed,
	countTrend,
	durationTrend,
	formatRate,
	formatSeconds,
	mergeFeed,
	rateTrend,
	splitSeconds
} from './dashboard';
import type { Deployment } from '#lib/resources/deployments.ts';

const row = (app: string, n: number, status = 0) =>
	({ app_id: app, deployment_number: n, state: { status } }) as unknown as Deployment;

describe('formatting', () => {
	test('durations', () => {
		expect(formatSeconds(null)).toBe('—');
		expect(formatSeconds(48.4)).toBe('48 s');
		expect(formatSeconds(72)).toBe('1 m 12 s');
		expect(formatSeconds(14 * 60)).toBe('14 min');
		expect(formatSeconds(125 * 60)).toBe('2 h 5 min');
		expect(formatSeconds(76 * 3600)).toBe('3 d 4 h');
		expect(splitSeconds(52)).toEqual(['52', 's']);
		expect(splitSeconds(14 * 60)).toEqual(['14', 'min']);
		expect(splitSeconds(3 * 3600)).toEqual(['3', 'h']);
	});

	test('rates', () => {
		expect(formatRate(0.934)).toBe('93.4');
		expect(formatRate(1)).toBe('100');
		expect(formatRate(null)).toBe('—');
	});
});

describe('trends', () => {
	test('counts compare in percent, more is good', () => {
		expect(countTrend(214, 181)).toEqual({ text: '+18%', up: true, good: true });
		expect(countTrend(5, 10)).toEqual({ text: '−50%', up: false, good: false });
		expect(countTrend(5, 0)).toBeNull();
	});

	test('rates compare in points', () => {
		expect(rateTrend(0.934, 0.913)).toEqual({ text: '+2.1 pts', up: true, good: true });
		expect(rateTrend(0.9, null)).toBeNull();
	});

	test('durations: shorter is good', () => {
		expect(durationTrend(52, 58)).toEqual({ text: '−6 s', up: false, good: true });
		expect(durationTrend(14 * 60, 11 * 60)).toEqual({ text: '+3 min', up: true, good: false });
	});
});

describe('activity feed', () => {
	test('new rows go on top, known rows take their fresh state', () => {
		const current = [row('a', 2, 1), row('a', 1, 3)];
		const { rows, added } = mergeFeed(current, [row('b', 1), row('a', 2, 3)]);

		expect(rows.map((d) => `${d.app_id}${d.deployment_number}`)).toEqual(['b1', 'a2', 'a1']);
		expect(rows[1].state.status).toBe(3);
		expect(added).toBe(1);
	});

	test('the first load is not counted as new', () => {
		expect(mergeFeed([], [row('a', 1)]).added).toBe(0);
	});

	test('older pages skip rows already shown', () => {
		expect(appendFeed([row('a', 3), row('a', 2)], [row('a', 2), row('a', 1)]).length).toBe(3);
	});
});
