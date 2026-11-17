<!--
	A calendar of the last weeks (DESIGN.md §11.10 Charts): a column per week, a row per weekday
	(Monday on top), the current week on the right. Square cells that grow with the width; five
	levels on the accent against the busiest day, an empty day is a `sunken` cell with an inner line
	(empty is not missing), today has an accent outline, days still to come keep their space but are
	not drawn. Mon, Wed and Fri label the rows; a month names the first week that starts in it. The
	grid is one image with a written summary; each cell has a tooltip with the date and count.
-->
<script lang="ts">
	import { calendarWeeks, dayDate, dayString, heatLevel, longDay, streaks } from '#lib/charts.ts';
	import l from '#lib/localization/index.ts';
	import { cn } from '#lib/utils.js';

	type Props = {
		/** One entry per day, oldest first, the last is today. */
		days: { date: string; value: number }[];
		label: string;
		/** "3 deployments" */
		count: (n: number) => string;
		weeks?: number;
	};

	let { days, label, count, weeks = 13 }: Props = $props();

	const byDay = $derived(new Map(days.map((d) => [d.date, d.value])));
	const today = $derived(days.at(-1)?.date ?? dayString(new Date()));
	const max = $derived(Math.max(1, ...days.map((d) => d.value)));

	const grid = $derived(
		calendarWeeks(today, weeks).map((week) =>
			week.map((cell) => ({ ...cell, value: byDay.get(cell.date) ?? 0 }))
		)
	);

	const months = $derived.by(() => {
		const out: { col: number; text: string }[] = [];
		const fmt = new Intl.DateTimeFormat(l.locale(), { month: 'short' });
		let last = -1;
		let lastCol = -9;
		grid.forEach((week, c) => {
			const first = dayDate(week[0].date);
			if (first.getMonth() !== last && c - lastCol > 2) {
				out.push({ col: c, text: fmt.format(first) });
				last = first.getMonth();
				lastCol = c;
			}
		});
		return out;
	});

	const run = $derived(streaks(days.map((d) => d.value)));
	const weekday = (i: number) =>
		new Intl.DateTimeFormat(l.locale(), { weekday: 'short' }).format(
			new Date(2026, 10, 16 + i, 12)
		);

	const levels = [
		'bg-sunken shadow-[inset_0_0_0_1px_var(--line)]',
		'bg-accent-soft',
		'bg-[color-mix(in_oklab,var(--accent)_45%,var(--surface))]',
		'bg-[color-mix(in_oklab,var(--accent)_70%,var(--surface))]',
		'bg-primary'
	];
</script>

<div class="flex flex-col gap-2">
	<div class="grid grid-cols-[28px_minmax(0,1fr)] gap-x-1.5 gap-y-1 text-xs text-muted-foreground">
		<span></span>
		<div class="relative h-4">
			{#each months as m (m.col)}
				<span class="absolute top-0 whitespace-nowrap" style:left="{(m.col / weeks) * 100}%"
					>{m.text}</span
				>
			{/each}
		</div>
		<div class="grid grid-rows-7 gap-[3px]" aria-hidden="true">
			{#each [0, 1, 2, 3, 4, 5, 6] as r (r)}
				<span class="self-center leading-none">{r % 2 === 0 && r < 6 ? weekday(r) : ''}</span>
			{/each}
		</div>
		<div
			role="img"
			aria-label={l.translate('charts.calendar.summary', [label, run.current, run.longest])}
			class="grid grid-flow-col grid-rows-7 gap-[3px]"
		>
			{#each grid as week, c (c)}
				{#each week as cell (cell.date)}
					<span
						class={cn(
							'aspect-square rounded-xs',
							levels[heatLevel(cell.value, max)],
							cell.date === today && 'outline outline-1 outline-offset-1 outline-primary',
							cell.future && 'invisible'
						)}
						title={cell.future ? undefined : `${longDay(cell.date)}: ${count(cell.value)}`}
					></span>
				{/each}
			{/each}
		</div>
	</div>
	<div
		class="flex flex-wrap items-center justify-between gap-x-4 gap-y-1 text-xs text-muted-foreground"
	>
		<span>{l.translate('charts.calendar.streak', [run.current, run.longest])}</span>
		<span class="flex items-center gap-[3px]" aria-hidden="true">
			{l.translate('charts.calendar.less')}
			{#each levels as level, i (i)}<i class={cn('size-2.5 rounded-xs', level)}></i>{/each}
			{l.translate('charts.calendar.more')}
		</span>
	</div>
</div>
