<!--
	One row of a list (DESIGN.md §5 Lists), inside a TimelineGroup or a TimelineList:

	  [tile]  Title  [badge]                      [actions] [⋯]
	          14:32 · main · a41c9e2 · took 48 s

	- The tile carries the state as an icon and a colour, drawn bare: no well behind it (owner's call
	  2026-11-20). The title line or the badge says it in words too, so meaning never rests on colour
	  alone.
	- The title is the row's one link or button, stretched over the whole row, so clicking anywhere
	  opens the item. Its accessible name is `label`, which says everything the row shows. With
	  neither `href` nor `onclick` (an internal service) the row is plain text.
	- Actions (a quick button, the ⋯ menu) sit above the stretched link and are always visible.
	- At rest a row has no fill (§1 Cardless); the fill answers you. Hover: the `hover` wash; an
	  open ⋯ menu keeps `raised`. Selected (its drawer is open): `accent-soft`.
-->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import Icon, { type IconData } from '#lib/components/icon.svelte';
	import { cn } from '#lib/utils.js';

	export type TimelineTone = 'success' | 'danger' | 'warning' | 'muted';

	type Props = {
		/** The tile's icon; `tile` replaces it with a picture (a registry's logo). */
		icon?: IconData;
		tile?: Snippet;
		tone?: TimelineTone;
		/** Spin the icon (running work). */
		spin?: boolean;
		/** Everything the row says, for assistive technology ("Deployment 5, production, failed, …"). */
		label: string;
		href?: string;
		/** `href` leaves checkship (a service's URL): opens in a new tab. */
		external?: boolean;
		onclick?: () => void;
		selected?: boolean;
		/** Just arrived (a live list): `accent-soft` for a moment, without meaning "selected". */
		highlight?: boolean;
		/** Dim the row (a deleted application's backup). */
		muted?: boolean;
		title: Snippet;
		/** After the title: a badge, an environment chip. */
		badge?: Snippet;
		details?: Snippet;
		/** Under the details line, clickable on its own (a deploy strip). */
		extra?: Snippet;
		actions?: Snippet;
	};

	let {
		icon,
		tile,
		tone = 'muted',
		spin = false,
		label,
		href,
		external = false,
		onclick,
		selected = false,
		highlight = false,
		muted = false,
		title,
		badge,
		details,
		extra,
		actions
	}: Props = $props();

	/** A row that opens something gets a hover; a plain row (a volume) does not. */
	const interactive = $derived(!!href || !!onclick);

	const toneClass: Record<TimelineTone, string> = {
		success: 'text-success',
		danger: 'text-danger',
		warning: 'text-warning',
		muted: 'text-muted-foreground'
	};

	const stretch =
		'min-w-0 truncate rounded-sm text-left outline-none after:absolute after:inset-0 after:rounded-md';
</script>

<li
	aria-current={selected ? 'true' : undefined}
	class={cn(
		'group/row relative grid grid-cols-[32px_minmax(0,1fr)_auto] items-center gap-3 rounded-md p-2 transition-colors has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-ring has-[:focus-visible]:ring-inset',
		highlight && 'duration-700 motion-reduce:transition-none',
		selected || highlight
			? 'bg-accent-soft'
			: interactive && 'hover:bg-hover has-[[data-state=open]]:bg-raised'
	)}
>
	<span aria-hidden="true" class={cn('flex size-8 items-center justify-center', toneClass[tone])}>
		{#if tile}
			{@render tile()}
		{:else if icon}
			<Icon {icon} class={cn('size-5', spin && 'animate-spin')} />
		{/if}
	</span>

	<div class="flex min-w-0 flex-col">
		<div class="flex min-w-0 items-center gap-2 text-sm font-semibold">
			{#if href}
				<a
					{href}
					aria-label={label}
					target={external ? '_blank' : undefined}
					rel={external ? 'noopener noreferrer' : undefined}
					class={cn(stretch, muted && 'text-muted-foreground')}>{@render title()}</a
				>
			{:else if onclick}
				<button
					type="button"
					aria-label={label}
					{onclick}
					class={cn(stretch, 'font-semibold', muted && 'text-muted-foreground')}
					>{@render title()}</button
				>
			{:else}
				<span class={cn('min-w-0 truncate', muted && 'text-muted-foreground')}
					>{@render title()}</span
				>
			{/if}
			{@render badge?.()}
		</div>
		{#if details}
			<div class="flex min-w-0 items-center gap-x-1.5 text-xs text-muted-foreground tabular-nums">
				<span class="min-w-0 truncate">{@render details()}</span>
			</div>
		{/if}
		{#if extra}
			<div class="relative z-10 mt-1.5 self-start">{@render extra()}</div>
		{/if}
	</div>

	{#if actions}
		<div class="relative z-10 flex items-center gap-1">{@render actions()}</div>
	{:else}
		<span></span>
	{/if}
</li>
