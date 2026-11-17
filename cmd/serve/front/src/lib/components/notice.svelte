<!--
	A note on the page (DESIGN.md §5 Notices): an icon and text. Neutral notes have no fill, they are
	just words (§1 Cardless: a fill means something). Only a real state gets a tint: `warning`
	(something is left out or waiting) and `danger` (something failed), always with the icon and the
	words, so meaning never rests on colour alone.
-->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import InformationCircleIcon from '@hugeicons/core-free-icons/InformationCircleIcon';
	import Icon, { type IconData } from '#lib/components/icon.svelte';
	import { cn } from '#lib/utils.js';

	type Props = {
		tone?: 'neutral' | 'warning' | 'danger';
		icon?: IconData;
		role?: 'alert' | 'status';
		class?: string;
		children: Snippet;
	};

	let { tone = 'neutral', icon, role, class: className, children }: Props = $props();

	const toneIcon = $derived(icon ?? (tone === 'neutral' ? InformationCircleIcon : Alert02Icon));
</script>

<div
	{role}
	class={cn(
		'grid grid-cols-[18px_minmax(0,1fr)] items-start gap-2.5 text-label/5',
		tone === 'neutral' && 'text-muted-foreground',
		tone === 'warning' && 'rounded-md bg-warning-soft px-3.5 py-3 text-ink',
		tone === 'danger' && 'rounded-md bg-danger-soft px-3.5 py-3 text-ink',
		className
	)}
>
	<Icon
		icon={toneIcon}
		class={cn(
			'mt-px size-4.5',
			tone === 'warning' && 'text-warning',
			tone === 'danger' && 'text-danger'
		)}
	/>
	<div class="flex min-w-0 flex-col gap-1">{@render children()}</div>
</div>
