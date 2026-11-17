/** Formatting and comparisons for the Overview dashboard (DESIGN.md §11.10). */
import type { Deployment } from '#lib/resources/deployments.ts';

/** "48 s", "1 m 12 s", "14 min", "2 h 5 min", "3 d 4 h". */
export function formatSeconds(seconds: number | null | undefined): string {
	if (seconds == null || !Number.isFinite(seconds)) return '—';
	const s = Math.max(0, Math.round(seconds));
	if (s < 60) return `${s} s`;
	if (s < 600)
		return s % 60
			? `${Math.floor(s / 60)} m ${String(s % 60).padStart(2, '0')} s`
			: `${s / 60} min`;
	const minutes = Math.round(s / 60);
	if (minutes < 60) return `${minutes} min`;
	const hours = Math.floor(minutes / 60);
	if (hours < 48) return minutes % 60 ? `${hours} h ${minutes % 60} min` : `${hours} h`;
	const days = Math.floor(hours / 24);
	return hours % 24 ? `${days} d ${hours % 24} h` : `${days} d`;
}

/** Value and unit apart, for the large KPI figures: [52, "s"], [14, "min"], [2.5, "h"]. */
export function splitSeconds(seconds: number | null | undefined): [string, string] {
	if (seconds == null || !Number.isFinite(seconds)) return ['—', ''];
	const s = Math.max(0, seconds);
	if (s < 120) return [String(Math.round(s)), 's'];
	if (s < 7200) return [String(Math.round(s / 60)), 'min'];
	if (s < 172800) return [(s / 3600).toFixed(1).replace(/\.0$/, ''), 'h'];
	return [(s / 86400).toFixed(1).replace(/\.0$/, ''), 'd'];
}

export const formatRate = (rate: number | null | undefined) =>
	rate == null ? '—' : (rate * 100).toFixed(1).replace(/\.0$/, '');

/** Feed rows are unique by app and number. */
export const deploymentKey = (d: Pick<Deployment, 'app_id' | 'deployment_number'>) =>
	`${d.app_id}#${d.deployment_number}`;

/**
 * Merge a freshly fetched first page into the rows already shown: new rows go on top, known rows
 * take their fresh state, older pages stay. Returns the merged rows and how many are new.
 */
export function mergeFeed(
	current: Deployment[],
	fresh: Deployment[]
): { rows: Deployment[]; added: number } {
	const known = new Set(current.map(deploymentKey));
	const latest = new Map(fresh.map((d) => [deploymentKey(d), d]));
	const added = fresh.filter((d) => !known.has(deploymentKey(d)));
	const updated = current.map((d) => latest.get(deploymentKey(d)) ?? d);
	return { rows: [...added, ...updated], added: current.length === 0 ? 0 : added.length };
}

/** Append an older page, skipping anything already shown. */
export function appendFeed(current: Deployment[], older: Deployment[]): Deployment[] {
	const known = new Set(current.map(deploymentKey));
	return [...current, ...older.filter((d) => !known.has(deploymentKey(d)))];
}
