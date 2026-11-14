<!--
	A bento tile (DESIGN.md §11.10): one sunken block, title with an icon, optional actions on the
	right. Width and height come from the caller's grid spans (`class`). States: loading keeps the
	tile's size with skeletons, an error says so in the tile and offers Try again, so one failing
	tile never blanks the page.
-->
<script lang="ts">
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import RefreshIcon from '@hugeicons/core-free-icons/RefreshIcon';
	import type { Snippet } from 'svelte';
	import Icon, { type IconData } from '#lib/components/icon.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import l from '#lib/localization/index.ts';
	import { cn } from '#lib/utils.js';

	type Props = {
		title: string;
		icon?: IconData;
		class?: string;
		id?: string;
		loading?: boolean;
		error?: unknown;
		onretry?: () => void;
		actions?: Snippet;
		children: Snippet;
	};

	let {
		title,
		icon,
		class: className,
		id,
		loading,
		error,
		onretry,
		actions,
		children
	}: Props = $props();

	const headingId = $props.id();
</script>

<section
	{id}
	aria-labelledby={headingId}
	class={cn('flex min-w-0 flex-col gap-3.5 rounded-lg bg-sunken p-5', className)}
>
	<header class="flex min-h-7 flex-wrap items-center justify-between gap-x-3 gap-y-2">
		<h2 id={headingId} class="flex items-center gap-2 text-sm font-semibold">
			{#if icon}<Icon {icon} class="size-4 text-muted-foreground" />{/if}
			{title}
		</h2>
		{#if actions}
			<div class="flex flex-wrap items-center gap-1.5">{@render actions()}</div>
		{/if}
	</header>

	{#if error}
		<div class="flex flex-1 flex-col items-start justify-center gap-2.5 py-4">
			<p class="flex items-center gap-2 text-sm text-danger">
				<Icon icon={Alert02Icon} class="size-4" />
				{l.translate('dashboard.tile.error')}
			</p>
			<p class="text-[13px] text-muted-foreground">{l.translate('dashboard.tile.error.help')}</p>
			{#if onretry}
				<Button variant="outline" size="sm" onclick={onretry}>
					<Icon icon={RefreshIcon} />
					{l.translate('dashboard.retry')}
				</Button>
			{/if}
		</div>
	{:else if loading}
		<div class="flex flex-1 flex-col gap-3" aria-busy="true">
			<Skeleton class="h-3 w-2/5 bg-raised" />
			<Skeleton class="h-8 w-3/5 bg-raised" />
			<Skeleton class="min-h-10 w-full flex-1 bg-raised" />
		</div>
	{:else}
		{@render children()}
	{/if}
</section>
