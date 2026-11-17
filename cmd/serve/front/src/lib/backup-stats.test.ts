import { describe, expect, it } from 'vitest';
import {
	backupDays,
	backupFigures,
	byTrigger,
	freshnessByEnvironment,
	freshnessOf,
	lastDays,
	lastVolumeBackup,
	sizeByApp
} from '#lib/backup-stats.ts';
import type { Backup } from '#lib/resources/backups.ts';

const today = new Date(2026, 10, 17, 15);
const at = (day: number, hour = 10) => new Date(2026, 10, day, hour).toISOString();

function backup(over: Partial<Backup> & { day: number; size?: number }): Backup {
	const { day, size = 100, ...rest } = over;
	return {
		id: `b${day}${rest.status ?? ''}`,
		app: { id: 'a1', name: 'shop', deleted: false },
		environment: 'production',
		target: { id: 't1' },
		trigger: 'manual',
		status: 'succeeded',
		stop_containers: false,
		volumes: [],
		result: { size, sha256: '', volumes: [] },
		local: true,
		stored: true,
		requested_at: at(day),
		requested_by: { id: 'u', email: 'u@example.com' },
		...rest
	};
}

const list = [
	backup({ day: 1 }),
	backup({ day: 15, size: 50, stored: false }),
	backup({ day: 16, status: 'failed', result: null }),
	backup({ day: 17, status: 'running', result: null, trigger: 'upload' }),
	backup({ day: 17, size: 20, app: { id: 'a2', name: 'blog', deleted: false } })
];

describe('backup stats', () => {
	it('lists the last days ending today', () => {
		expect(lastDays(3, today)).toEqual(['2026-11-15', '2026-11-16', '2026-11-17']);
	});

	it('counts each day and keeps a running total of space', () => {
		const days = backupDays(list, 3, today);
		expect(days.map((d) => [d.succeeded, d.failed, d.other])).toEqual([
			[1, 0, 0],
			[0, 1, 0],
			[1, 0, 1]
		]);
		// Day 1's 100 bytes were kept before the range; day 15 adds 50 only on its target.
		expect(days.map((d) => [d.stored, d.local])).toEqual([
			[100, 50],
			[100, 50],
			[120, 50]
		]);
	});

	it('compares the period with the one before', () => {
		const f = backupFigures(list, 3, today);
		expect(f.count).toBe(4);
		expect(f.rate).toBeCloseTo(2 / 3);
		expect([f.stored, f.local]).toEqual([120, 50]);
		expect(f.lastGood?.app.name).toBe('blog');
	});

	it('splits space by application and starts by trigger', () => {
		expect(sizeByApp(list).map((a) => [a.name, a.size, a.count])).toEqual([
			['shop', 150, 4],
			['blog', 20, 1]
		]);
		expect(byTrigger(list, 3, today)).toEqual([
			{ trigger: 'manual', count: 3 },
			{ trigger: 'upload', count: 1 }
		]);
	});

	it('tells how fresh the last good backup is', () => {
		expect(freshnessOf(new Date(2026, 10, 17, 3).toISOString(), today)).toBe('fresh');
		expect(freshnessOf(new Date(2026, 10, 13).toISOString(), today)).toBe('quiet');
		expect(freshnessOf(new Date(2026, 10, 1).toISOString(), today)).toBe('stale');
		expect(freshnessOf(undefined, today)).toBe('never');
	});

	it('lists each environment with its last good backup, stalest first', () => {
		const rows = freshnessByEnvironment(
			[...list, backup({ day: 2, environment: 'staging', status: 'failed', result: null })],
			today
		);
		expect(rows.map((r) => [r.app.name, r.environment, r.freshness])).toEqual([
			['shop', 'staging', 'never'],
			['shop', 'production', 'quiet'],
			['blog', 'production', 'fresh']
		]);
	});

	it('finds when a volume was last backed up', () => {
		const withVolume = backup({
			day: 15,
			result: { size: 50, sha256: '', volumes: [{ key: 'data', size: 42, files: 1 }] }
		});
		expect(lastVolumeBackup([withVolume], 'a1', 'production', 'data')?.size).toBe(42);
		expect(lastVolumeBackup([withVolume], 'a1', 'production', 'other')).toBeUndefined();
	});
});
