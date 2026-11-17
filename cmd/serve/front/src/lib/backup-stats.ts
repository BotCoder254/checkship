/**
 * What the Backups page counts (DESIGN.md §11.10, Backups): per day, per application and per
 * trigger, from the backups list the page already loads (no server summary). Days are the viewer's
 * local days, oldest first, the last is today.
 */
import { dayString } from '#lib/charts.ts';
import type { Backup, BackupTrigger } from '#lib/resources/backups.ts';

export type BackupDay = {
	date: string;
	succeeded: number;
	failed: number;
	other: number;
	/** Size of the backups kept now that were made by the end of this day: on the storage... */
	stored: number;
	/** ...and only on their target. */
	local: number;
};

const localDay = (iso: string) => dayString(new Date(iso));

/** The last `count` days ending today, as "YYYY-MM-DD". */
export function lastDays(count: number, today = new Date()): string[] {
	return Array.from({ length: count }, (_, i) => {
		const d = new Date(
			today.getFullYear(),
			today.getMonth(),
			today.getDate() - (count - 1 - i),
			12
		);
		return dayString(d);
	});
}

/** Size kept on the storage and only on targets, for a backup that finished with a result. */
function kept(b: Backup): { stored: number; local: number } {
	if (b.status !== 'succeeded' || !b.result) return { stored: 0, local: 0 };
	if (b.stored) return { stored: b.result.size, local: 0 };
	return { stored: 0, local: b.local ? b.result.size : 0 };
}

export function backupDays(backups: Backup[], count: number, today = new Date()): BackupDay[] {
	const days = lastDays(count, today);
	const index = new Map(days.map((d, i) => [d, i]));
	const out: BackupDay[] = days.map((date) => ({
		date,
		succeeded: 0,
		failed: 0,
		other: 0,
		stored: 0,
		local: 0
	}));
	let storedBefore = 0;
	let localBefore = 0;
	for (const b of backups) {
		const day = localDay(b.requested_at);
		const size = kept(b);
		const i = index.get(day);
		if (i === undefined) {
			if (day < days[0]) {
				storedBefore += size.stored;
				localBefore += size.local;
			}
			continue;
		}
		if (b.status === 'succeeded') out[i].succeeded++;
		else if (b.status === 'failed') out[i].failed++;
		else out[i].other++;
		out[i].stored += size.stored;
		out[i].local += size.local;
	}
	// Running totals: what was kept by the end of each day.
	let stored = storedBefore;
	let local = localBefore;
	for (const d of out) {
		stored += d.stored;
		local += d.local;
		d.stored = stored;
		d.local = local;
	}
	return out;
}

export type BackupFigures = {
	count: number;
	previousCount: number;
	rate: number | null;
	previousRate: number | null;
	stored: number;
	local: number;
	lastGood?: Backup;
};

export function backupFigures(backups: Backup[], count: number, today = new Date()): BackupFigures {
	const days = lastDays(count * 2, today);
	const start = days[count];
	const previousStart = days[0];
	let n = 0;
	let ok = 0;
	let finished = 0;
	let pn = 0;
	let pok = 0;
	let pfinished = 0;
	let stored = 0;
	let local = 0;
	let lastGood: Backup | undefined;
	for (const b of backups) {
		const day = localDay(b.requested_at);
		const size = kept(b);
		stored += size.stored;
		local += size.local;
		if (b.status === 'succeeded' && (!lastGood || b.requested_at > lastGood.requested_at))
			lastGood = b;
		const done = b.status === 'succeeded' || b.status === 'failed';
		if (day >= start) {
			n++;
			if (done) finished++;
			if (b.status === 'succeeded') ok++;
		} else if (day >= previousStart) {
			pn++;
			if (done) pfinished++;
			if (b.status === 'succeeded') pok++;
		}
	}
	return {
		count: n,
		previousCount: pn,
		rate: finished ? ok / finished : null,
		previousRate: pfinished ? pok / pfinished : null,
		stored,
		local,
		lastGood
	};
}

/** Size kept now per application, biggest first. */
export function sizeByApp(
	backups: Backup[]
): { id: string; name: string; deleted: boolean; size: number; count: number }[] {
	const apps = new Map<
		string,
		{ id: string; name: string; deleted: boolean; size: number; count: number }
	>();
	for (const b of backups) {
		const size = kept(b);
		const entry = apps.get(b.app.id) ?? {
			id: b.app.id,
			name: b.app.name,
			deleted: b.app.deleted,
			size: 0,
			count: 0
		};
		entry.size += size.stored + size.local;
		entry.count++;
		apps.set(b.app.id, entry);
	}
	return [...apps.values()].sort((a, b) => b.size - a.size || b.count - a.count);
}

/** Backups started in the last `count` days, per trigger, most first. */
export function byTrigger(
	backups: Backup[],
	count: number,
	today = new Date()
): { trigger: BackupTrigger; count: number }[] {
	const start = lastDays(count, today)[0];
	const counts = new Map<BackupTrigger, number>();
	for (const b of backups)
		if (localDay(b.requested_at) >= start) counts.set(b.trigger, (counts.get(b.trigger) ?? 0) + 1);
	return [...counts.entries()]
		.map(([trigger, n]) => ({ trigger, count: n }))
		.sort((a, b) => b.count - a.count);
}

/** How fresh the last good backup is (DESIGN.md §11.10 Freshness): fixed until schedules exist. */
export type Freshness = 'fresh' | 'quiet' | 'stale' | 'never';

const DAY = 24 * 3600 * 1000;

export function freshnessOf(lastGood: string | undefined, now = new Date()): Freshness {
	if (!lastGood) return 'never';
	const age = now.getTime() - new Date(lastGood).getTime();
	return age < DAY ? 'fresh' : age < 7 * DAY ? 'quiet' : 'stale';
}

export type EnvironmentFreshness = {
	app: Backup['app'];
	environment: Backup['environment'];
	lastGood?: Backup;
	freshness: Freshness;
};

/**
 * The last good backup of each application environment that has one, stalest first (the ones
 * needing attention on top). Environments never backed up are added by the page, which knows
 * whether they have volumes.
 */
export function freshnessByEnvironment(
	backups: Backup[],
	now = new Date()
): EnvironmentFreshness[] {
	const latest = new Map<string, Backup>();
	const seen = new Map<string, Backup>();
	for (const b of backups) {
		const key = `${b.app.id}:${b.environment}`;
		if (!seen.has(key)) seen.set(key, b);
		if (b.status !== 'succeeded') continue;
		const current = latest.get(key);
		if (!current || b.requested_at > current.requested_at) latest.set(key, b);
	}
	const order: Record<Freshness, number> = { never: 0, stale: 1, quiet: 2, fresh: 3 };
	return [...seen.entries()]
		.filter(([, b]) => !b.app.deleted)
		.map(([key, b]) => {
			const lastGood = latest.get(key);
			return {
				app: b.app,
				environment: b.environment,
				lastGood,
				freshness: freshnessOf(lastGood?.requested_at, now)
			};
		})
		.sort(
			(a, b) =>
				order[a.freshness] - order[b.freshness] ||
				(a.lastGood?.requested_at ?? '').localeCompare(b.lastGood?.requested_at ?? '')
		);
}

/** When a volume was last in a good backup of this application environment, and its size then. */
export function lastVolumeBackup(
	backups: Backup[],
	appId: string,
	environment: Backup['environment'],
	volume: string
): { at: string; size: number } | undefined {
	let best: { at: string; size: number } | undefined;
	for (const b of backups) {
		if (b.app.id !== appId || b.environment !== environment || b.status !== 'succeeded') continue;
		const stats = b.result?.volumes.find((v) => v.key === volume);
		if (stats && (!best || b.requested_at > best.at))
			best = { at: b.requested_at, size: stats.size };
	}
	return best;
}

/** "Last backed up 3 hours ago · 310 MB" or "Never backed up", with its freshness. */
export function volumeLine(
	backups: Backup[],
	appId: string,
	environment: Backup['environment'],
	volume: string,
	describe: (last: { at: string; size: number } | undefined) => string,
	now = new Date()
): { text: string; freshness: Freshness } {
	const last = lastVolumeBackup(backups, appId, environment, volume);
	return { text: describe(last), freshness: freshnessOf(last?.at, now) };
}
