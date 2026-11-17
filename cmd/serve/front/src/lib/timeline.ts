/**
 * Lists grouped by day (DESIGN.md §5 Lists and tables): lists read by time (deployments, activity,
 * backups, failures) show their rows under one heading per day instead of a date column. Items
 * come newest first; a group starts each time the day changes, in the viewer's time zone.
 *
 * Labels follow the reader: Today, Yesterday, the weekday within the last week, then "12 November",
 * with the year only when it isn't this year. Words come from Intl in the page's locale, so every
 * language the dashboard has gets them without a translation key.
 */

export type DayGroup<T> = {
	/** `YYYY-MM-DD` in the viewer's time zone: a stable key for `{#each}`. */
	key: string;
	label: string;
	items: T[];
};

/** The calendar day of a date in the viewer's time zone. */
export function dayKey(value: DateValue): string {
	const d = new Date(value);
	const month = String(d.getMonth() + 1).padStart(2, '0');
	const day = String(d.getDate()).padStart(2, '0');
	return `${d.getFullYear()}-${month}-${day}`;
}

function startOfDay(value: DateValue): number {
	const d = new Date(value);
	return new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
}

const capitalise = (text: string, locale: string) =>
	text.charAt(0).toLocaleUpperCase(locale) + text.slice(1);

/** "Today", "Yesterday", "Monday", "12 November", "12 November 2025". */
export function dayLabel(value: DateValue, locale: string, now: DateValue = new Date()): string {
	// Calendar days apart; rounding absorbs a daylight-saving hour.
	const days = Math.round((startOfDay(now) - startOfDay(value)) / 86_400_000);

	if (days === 0 || days === 1) {
		const relative = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' });
		return capitalise(relative.format(-days, 'day'), locale);
	}

	const date = new Date(value);
	if (days > 1 && days < 7) {
		return capitalise(date.toLocaleDateString(locale, { weekday: 'long' }), locale);
	}

	const sameYear = date.getFullYear() === new Date(now).getFullYear();
	return date.toLocaleDateString(locale, {
		day: 'numeric',
		month: 'long',
		year: sameYear ? undefined : 'numeric'
	});
}

/** Groups items that are already in time order, one group per day. */
export function groupByDay<T>(
	items: readonly T[],
	at: (item: T) => DateValue,
	locale: string,
	now: DateValue = new Date()
): DayGroup<T>[] {
	const groups: DayGroup<T>[] = [];
	for (const item of items) {
		const key = dayKey(at(item));
		const last = groups.at(-1);
		if (last?.key === key) last.items.push(item);
		else groups.push({ key, label: dayLabel(at(item), locale, now), items: [item] });
	}
	return groups;
}

/** Groups items by a key, keeping the order of `order` (keys not in it are dropped). */
export function groupBy<T, K extends string>(
	items: readonly T[],
	keyOf: (item: T) => K,
	order: readonly K[]
): { key: K; items: T[] }[] {
	return order
		.map((key) => ({ key, items: items.filter((item) => keyOf(item) === key) }))
		.filter((group) => group.items.length > 0);
}
