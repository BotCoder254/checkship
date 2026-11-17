<!--
	Theme switch (DESIGN.md §5): System, Light, Dark as a radiogroup on a tonal track, no outline.
	Collapsed, one button cycles through the three. Arrow keys move between options.
-->
<script lang="ts">
	import ComputerIcon from '@hugeicons/core-free-icons/ComputerIcon';
	import Moon02Icon from '@hugeicons/core-free-icons/Moon02Icon';
	import Sun03Icon from '@hugeicons/core-free-icons/Sun03Icon';
	import { setMode, userPrefersMode } from 'mode-watcher';
	import Icon from '#lib/components/icon.svelte';
	import * as Tooltip from '#lib/components/ui/tooltip/index.js';
	import l from '#lib/localization/index.ts';
	import { cn } from '#lib/utils.js';

	type Mode = 'system' | 'light' | 'dark';

	let { collapsed = false }: { collapsed?: boolean } = $props();

	const options = [
		{ value: 'system', label: l.translate('theme.system'), icon: ComputerIcon },
		{ value: 'light', label: l.translate('theme.light'), icon: Sun03Icon },
		{ value: 'dark', label: l.translate('theme.dark'), icon: Moon02Icon }
	] as const;

	const current = $derived((userPrefersMode.current ?? 'system') as Mode);
	const currentOption = $derived(options.find((o) => o.value === current) ?? options[0]);
	let buttons: HTMLButtonElement[] = $state([]);

	function pick(index: number) {
		const option = options[(index + options.length) % options.length];
		setMode(option.value);
		return option;
	}

	function onkeydown(event: KeyboardEvent, index: number) {
		const step = { ArrowRight: 1, ArrowDown: 1, ArrowLeft: -1, ArrowUp: -1 }[event.key];
		if (!step) return;
		event.preventDefault();
		const next = (index + step + options.length) % options.length;
		pick(next);
		buttons[next]?.focus();
	}
</script>

{#if collapsed}
	<Tooltip.Root>
		<Tooltip.Trigger>
			{#snippet child({ props })}
				<button
					{...props}
					type="button"
					aria-label={l.translate('theme.switch', [currentOption.label])}
					class="flex size-10 items-center justify-center rounded-md text-ink outline-none hover:bg-hover-frame focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset"
					onclick={() => pick(options.indexOf(currentOption) + 1)}
				>
					<Icon icon={currentOption.icon} class="size-5" />
				</button>
			{/snippet}
		</Tooltip.Trigger>
		<Tooltip.Content side="right"
			>{l.translate('theme.switch', [currentOption.label])}</Tooltip.Content
		>
	</Tooltip.Root>
{:else}
	<div class="flex flex-col gap-1.5">
		<span id="theme-switch-label" class="text-xs font-semibold text-muted-foreground">
			{l.translate('nav.theme')}
		</span>
		<div
			role="radiogroup"
			aria-labelledby="theme-switch-label"
			class="grid grid-cols-3 gap-0.5 rounded-md bg-surface p-0.75"
		>
			{#each options as option, index (option.value)}
				{@const selected = option.value === current}
				<button
					bind:this={buttons[index]}
					type="button"
					role="radio"
					aria-checked={selected}
					aria-label={option.label}
					title={option.label}
					tabindex={selected ? 0 : -1}
					class={cn(
						'flex h-7.5 items-center justify-center rounded-sm text-muted-foreground transition-colors outline-none hover:text-ink focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset',
						selected && 'bg-accent-soft text-primary hover:text-primary'
					)}
					onclick={() => pick(index)}
					onkeydown={(event) => onkeydown(event, index)}
				>
					<Icon icon={option.icon} class="size-4.5" />
				</button>
			{/each}
		</div>
	</div>
{/if}
