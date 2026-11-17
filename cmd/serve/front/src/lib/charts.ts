/**
 * The maths behind checkship's charts (DESIGN.md §11.10 Charts): a rounded scale, a curve that
 * never invents a value, day labels and the written comparison under each figure. No drawing here,
 * so it is unit-tested; the parts in `#lib/components/charts/` draw with it.
 */
import l from '#lib/localization/index.ts';
import { formatSeconds } from '#lib/dashboard.ts';

/** The top of a scale: 1, 2, 2.5 or 5 × 10ⁿ, so gridlines read 20 and 10, never 17 and 8.5. */
export function niceCeil(value: number): number {
	if (!(value > 0)) return 1;
	const power = 10 ** Math.floor(Math.log10(value));
	for (const m of [1, 2, 2.5, 5, 10]) if (m * power >= value - 1e-9) return m * power;
	return 10 * power;
}

/** A nice top for bytes, in the unit it will be shown in (1024-based, as formatBytes). */
export function niceBytesCeil(bytes: number): number {
	if (!(bytes > 0)) return 1024;
	let unit = 1;
	while (bytes / unit >= 1024 && unit < 1024 ** 4) unit *= 1024;
	return niceCeil(bytes / unit) * unit;
}

/** A nice top for seconds: seconds up to two minutes, then minutes, then hours. */
export function niceSecondsCeil(seconds: number): number {
	if (!(seconds > 0)) return 1;
	if (seconds <= 120) return niceCeil(seconds);
	if (seconds <= 7200) return niceCeil(seconds / 60) * 60;
	return niceCeil(seconds / 3600) * 3600;
}

export type Point = [x: number, y: number];

/**
 * A smooth path through points sorted by x that never overshoots them (Fritsch–Carlson monotone
 * cubic, d3's curveMonotoneX): between two days the curve stays between their values, so it never
 * shows a number that did not happen or dips below zero.
 */
export function monotonePath(points: Point[]): string {
	const n = points.length;
	if (n === 0) return '';
	const f = (v: number) => Math.round(v * 100) / 100;
	if (n === 1) return `M${f(points[0][0])} ${f(points[0][1])}`;
	const slope: number[] = [];
	for (let i = 0; i < n - 1; i++)
		slope.push((points[i + 1][1] - points[i][1]) / (points[i + 1][0] - points[i][0]));
	const tangent: number[] = new Array(n);
	tangent[0] = slope[0];
	tangent[n - 1] = slope[n - 2];
	for (let i = 1; i < n - 1; i++)
		tangent[i] = slope[i - 1] * slope[i] <= 0 ? 0 : (slope[i - 1] + slope[i]) / 2;
	for (let i = 0; i < n - 1; i++) {
		if (slope[i] === 0) {
			tangent[i] = 0;
			tangent[i + 1] = 0;
			continue;
		}
		const a = tangent[i] / slope[i];
		const b = tangent[i + 1] / slope[i];
		const s = a * a + b * b;
		if (s > 9) {
			const k = 3 / Math.sqrt(s);
			tangent[i] = k * a * slope[i];
			tangent[i + 1] = k * b * slope[i];
		}
	}
	let d = `M${f(points[0][0])} ${f(points[0][1])}`;
	for (let i = 0; i < n - 1; i++) {
		const [x0, y0] = points[i];
		const [x1, y1] = points[i + 1];
		const h = (x1 - x0) / 3;
		d += `C${f(x0 + h)} ${f(y0 + tangent[i] * h)} ${f(x1 - h)} ${f(y1 - tangent[i + 1] * h)} ${f(x1)} ${f(y1)}`;
	}
	return d;
}

/** Split a series into runs of days that have a value, so a missing day leaves a gap. */
export function runs<T>(values: (T | null | undefined)[]): { start: number; values: T[] }[] {
	const out: { start: number; values: T[] }[] = [];
	values.forEach((v, i) => {
		if (v == null) return;
		const last = out.at(-1);
		if (last && last.start + last.values.length === i) last.values.push(v);
		else out.push({ start: i, values: [v] });
	});
	return out;
}

/** Which days get a label: every day up to a week, then every 7th (30 days) or 14th, from today back. */
export function tickIndexes(count: number): number[] {
	if (count <= 7) return Array.from({ length: count }, (_, i) => i);
	const step = count <= 31 ? 7 : 14;
	const out: number[] = [];
	for (let i = count - 1; i >= 0; i -= step) out.unshift(i);
	return out;
}

/** A server day ("2026-11-17", already in the viewer's time zone) as a local date at noon. */
export const dayDate = (day: string) => new Date(`${day}T12:00:00`);

/** "2026-11-17" for a local date. */
export function dayString(date: Date): string {
	const pad = (n: number) => String(n).padStart(2, '0');
	return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
}

/** The label under a day: "Today" for the last, a weekday letter in a week, else "Nov 10". */
export function tickLabel(day: string, index: number, count: number, locale = l.locale()): string {
	if (index === count - 1) return l.translate('charts.today');
	const date = dayDate(day);
	return count <= 7
		? new Intl.DateTimeFormat(locale, { weekday: 'narrow' }).format(date)
		: new Intl.DateTimeFormat(locale, { day: 'numeric', month: 'short' }).format(date);
}

/** "Tuesday, November 17" for a tooltip or a table row. */
export const longDay = (day: string, locale = l.locale()) =>
	new Intl.DateTimeFormat(locale, { weekday: 'long', day: 'numeric', month: 'long' }).format(
		dayDate(day)
	);

/** Five calendar levels against the busiest day: 0 is nothing, 1 to 4 are quarters of the most. */
export function heatLevel(value: number, max: number): 0 | 1 | 2 | 3 | 4 {
	if (!value || max <= 0) return 0;
	const r = value / max;
	return r <= 0.25 ? 1 : r <= 0.5 ? 2 : r <= 0.75 ? 3 : 4;
}

/** Current and longest run of consecutive days with something; today may still be empty. */
export function streaks(values: number[]): { current: number; longest: number } {
	let longest = 0;
	let run = 0;
	for (const v of values) {
		run = v ? run + 1 : 0;
		longest = Math.max(longest, run);
	}
	let current = 0;
	for (let i = values.length - 1; i >= 0; i--) {
		if (values[i]) current++;
		else if (i < values.length - 1) break;
	}
	return { current, longest };
}

/** The line under a figure: the comparison in words, an arrow for the direction, a colour for its meaning. */
export type Change = {
	text: string;
	direction: 'up' | 'down' | 'same' | 'none';
	/** Whether the change is good news; undefined when there is nothing to judge. */
	good?: boolean;
};

/** A count against the previous period: "4 more than the previous 30 days". */
export function countChange(current: number, previous: number, days: number): Change {
	if (previous === 0 && current === 0)
		return { text: l.translate('change.same', [days]), direction: 'same' };
	if (previous === 0) return { text: l.translate('change.none', [days]), direction: 'none' };
	const diff = current - previous;
	if (diff === 0) return { text: l.translate('change.same', [days]), direction: 'same' };
	return diff > 0
		? { text: l.translate('change.more', [diff, days]), direction: 'up', good: true }
		: { text: l.translate('change.fewer', [-diff, days]), direction: 'down', good: false };
}

/** A rate (0 to 1) in percentage points: "3 points higher than the previous 30 days". */
export function rateChange(current: number | null, previous: number | null, days: number): Change {
	if (current == null || previous == null)
		return { text: l.translate('change.none', [days]), direction: 'none' };
	const points = Math.round((current - previous) * 100);
	if (points === 0) return { text: l.translate('change.same', [days]), direction: 'same' };
	return points > 0
		? { text: l.translate('change.higher', [points, days]), direction: 'up', good: true }
		: { text: l.translate('change.lower', [-points, days]), direction: 'down', good: false };
}

/** A duration in seconds, shorter is better: "6 s faster than the previous 30 days". */
export function durationChange(
	current: number | null,
	previous: number | null,
	days: number
): Change {
	if (current == null || previous == null)
		return { text: l.translate('change.none', [days]), direction: 'none' };
	const diff = Math.round(current - previous);
	if (diff === 0) return { text: l.translate('change.same', [days]), direction: 'same' };
	const amount = formatSeconds(Math.abs(diff));
	return diff < 0
		? { text: l.translate('change.faster', [amount, days]), direction: 'down', good: true }
		: { text: l.translate('change.slower', [amount, days]), direction: 'up', good: false };
}

/** One series of a chart: which field, its name, its colour (a token, never a literal). */
export type ChartSeries<D> = { key: keyof D & string; label: string; color: string };

/**
 * The days of a calendar ending this week: `weeks` columns of Monday to Sunday, the last holding
 * `today` ("YYYY-MM-DD"). Days after today are marked `future`.
 */
export function calendarWeeks(today: string, weeks: number): { date: string; future: boolean }[][] {
	const end = dayDate(today);
	const start = dayDate(today);
	start.setDate(end.getDate() - ((end.getDay() + 6) % 7) - (weeks - 1) * 7);
	return Array.from({ length: weeks }, (_, c) =>
		Array.from({ length: 7 }, (_, r) => {
			const d = new Date(start);
			d.setDate(start.getDate() + c * 7 + r);
			return { date: dayString(d), future: d > end };
		})
	);
}
