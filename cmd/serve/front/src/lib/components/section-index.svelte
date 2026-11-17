<!--
	"On this page" (DESIGN.md §5 Section index): a sticky list of a long form's sections from xl,
	the one in view highlighted in `accent-soft`, so where you are stays visible as you scroll.
	Hidden below xl, where the page is one column and the tabs stay the one navigation.
-->
<script lang="ts">
	import { onMount } from 'svelte';
	import l from '#lib/localization/index.ts';
	import { cn } from '#lib/utils.js';

	type Props = { items: { id: string; label: string; danger?: boolean }[]; class?: string };

	let { items, class: className }: Props = $props();

	let active = $state('');

	onMount(() => {
		active = items[0]?.id ?? '';
		const visible: Record<string, boolean> = {};
		const observer = new IntersectionObserver(
			(entries) => {
				for (const entry of entries) {
					visible[entry.target.id] = entry.isIntersecting;
				}
				const first = items.find((i) => visible[i.id]);
				if (first) active = first.id;
			},
			{ rootMargin: '-10% 0px -55% 0px' }
		);
		for (const item of items) {
			const el = document.getElementById(item.id);
			if (el) observer.observe(el);
		}
		// The last sections may never reach the band above: at the bottom, the last one is current.
		const scroller = document.querySelector('main');
		const atBottom = () => {
			if (scroller && scroller.scrollTop + scroller.clientHeight >= scroller.scrollHeight - 4)
				active = items.at(-1)?.id ?? active;
		};
		scroller?.addEventListener('scroll', atBottom, { passive: true });
		return () => {
			observer.disconnect();
			scroller?.removeEventListener('scroll', atBottom);
		};
	});
</script>

<nav
	aria-label={l.translate('settings.index')}
	class={cn('sticky top-6 hidden self-start xl:flex xl:flex-col xl:gap-0.5', className)}
>
	<span class="mb-1.5 px-2.5 text-xs font-semibold tracking-wide text-muted-foreground"
		>{l.translate('settings.index')}</span
	>
	{#each items as item (item.id)}
		<a
			href="#{item.id}"
			aria-current={active === item.id ? 'location' : undefined}
			class={cn(
				'rounded-md px-2.5 py-1.5 text-label outline-none hover:bg-hover focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset',
				active === item.id
					? 'bg-accent-soft font-semibold text-primary hover:bg-accent-soft'
					: item.danger
						? 'text-danger'
						: 'text-muted-foreground'
			)}
			onclick={() => (active = item.id)}>{item.label}</a
		>
	{/each}
</nav>
