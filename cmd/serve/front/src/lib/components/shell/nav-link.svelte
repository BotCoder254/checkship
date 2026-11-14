<!--
	One sidebar link (DESIGN.md §5): 40px, 6px corner, Hugeicons at 20px. The current page is an
	accent-soft fill with accent text, never a coloured strip. Collapsed, only the icon shows and a
	tooltip carries the label.
-->
<script lang="ts">
	import type { IconData } from '#lib/components/icon.svelte';
	import type { Snippet } from 'svelte';
	import Icon from '#lib/components/icon.svelte';
	import * as Tooltip from '#lib/components/ui/tooltip/index.js';
	import { cn } from '#lib/utils.js';

	type Props = {
		href: string;
		label: string;
		icon: IconData;
		active?: boolean;
		collapsed?: boolean;
		external?: boolean;
		/** Extra words for assistive technology, such as "1 target failing". */
		description?: string;
		/** Shown after the label when expanded: a count, a warning icon. */
		trailing?: Snippet;
		onclick?: () => void;
	};

	let {
		href,
		label,
		icon,
		active = false,
		collapsed = false,
		external = false,
		description,
		trailing,
		onclick
	}: Props = $props();

	const accessibleName = $derived(description ? `${label}, ${description}` : label);
</script>

{#snippet link(props: Record<string, unknown> = {})}
	<a
		{...props}
		{href}
		{onclick}
		aria-current={active ? 'page' : undefined}
		aria-label={collapsed || description ? accessibleName : undefined}
		target={external ? '_blank' : undefined}
		rel={external ? 'noopener noreferrer' : undefined}
		class={cn(
			'flex h-10 items-center gap-3 rounded-md text-sm font-medium text-ink transition-colors outline-none hover:bg-hover focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset',
			collapsed ? 'w-10 justify-center' : 'px-3',
			active && 'bg-accent-soft font-semibold text-primary hover:bg-accent-soft'
		)}
	>
		<Icon {icon} class="size-5" />
		{#if !collapsed}
			<span class="min-w-0 flex-1 truncate">{label}</span>
			{@render trailing?.()}
		{/if}
	</a>
{/snippet}

{#if collapsed}
	<Tooltip.Root>
		<Tooltip.Trigger>
			{#snippet child({ props })}
				{@render link(props)}
			{/snippet}
		</Tooltip.Trigger>
		<Tooltip.Content side="right">{accessibleName}</Tooltip.Content>
	</Tooltip.Root>
{:else}
	{@render link()}
{/if}
