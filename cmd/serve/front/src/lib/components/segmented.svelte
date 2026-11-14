<!--
	Segmented control (DESIGN.md §11.6): 2 to 4 choices that change what follows (environment,
	source). A radio group: one tab stop, arrow keys move and select. The track is sunken, the
	selected item raised.
-->
<script lang="ts" generics="T extends string">
	import type { IconData } from '#lib/components/icon.svelte';
	import Icon from '#lib/components/icon.svelte';
	import { cn } from '#lib/utils.js';

	type Item = { value: T; label: string; icon?: IconData };
	type Props = {
		value: T;
		items: Item[];
		/** id of the visible label */
		labelledby: string;
		describedby?: string;
		disabled?: boolean;
		class?: string;
	};

	let {
		value = $bindable(),
		items,
		labelledby,
		describedby,
		disabled = false,
		class: className
	}: Props = $props();

	let group = $state<HTMLDivElement>();

	function keydown(event: KeyboardEvent, index: number) {
		const step = { ArrowRight: 1, ArrowDown: 1, ArrowLeft: -1, ArrowUp: -1 }[event.key];
		if (!step) return;
		event.preventDefault();
		const next = (index + step + items.length) % items.length;
		value = items[next].value;
		group?.querySelectorAll<HTMLButtonElement>('[role="radio"]')[next]?.focus();
	}
</script>

<div
	bind:this={group}
	role="radiogroup"
	aria-labelledby={labelledby}
	aria-describedby={describedby}
	class={cn('grid auto-cols-fr grid-flow-col gap-1 rounded-lg bg-sunken p-1', className)}
>
	{#each items as item, index (item.value)}
		{@const selected = item.value === value}
		<button
			type="button"
			role="radio"
			aria-checked={selected}
			tabindex={selected ? 0 : -1}
			{disabled}
			class={cn(
				'flex h-9 min-w-0 items-center justify-center gap-2 rounded-[5px] px-3 text-sm font-medium whitespace-nowrap text-muted-foreground outline-none hover:text-ink focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset disabled:opacity-50 md:h-7',
				selected && 'bg-raised font-semibold text-ink'
			)}
			onclick={() => (value = item.value)}
			onkeydown={(event) => keydown(event, index)}
		>
			{#if item.icon}<Icon icon={item.icon} class="size-4" />{/if}
			<span class="truncate">{item.label}</span>
		</button>
	{/each}
</div>
