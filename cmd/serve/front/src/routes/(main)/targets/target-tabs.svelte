<!-- Target tabs (DESIGN.md §11.8): Overview and Settings, as links. -->
<script lang="ts">
	import DashboardSquare01Icon from '@hugeicons/core-free-icons/DashboardSquare01Icon';
	import Settings02Icon from '@hugeicons/core-free-icons/Settings02Icon';
	import Icon from '#lib/components/icon.svelte';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import { cn } from '#lib/utils.js';

	type Props = { targetId: string; active: 'overview' | 'settings' };

	let { targetId, active }: Props = $props();

	const tabs = $derived([
		{
			id: 'overview',
			href: routes.target(targetId),
			icon: DashboardSquare01Icon,
			label: l.translate('detail.tab.overview')
		},
		{
			id: 'settings',
			href: routes.editTarget(targetId),
			icon: Settings02Icon,
			label: l.translate('detail.tab.settings')
		}
	]);
</script>

<nav aria-label={l.translate('targets.tabs')} class="flex gap-1">
	{#each tabs as tab (tab.id)}
		{@const current = tab.id === active}
		<a
			href={tab.href}
			aria-current={current ? 'page' : undefined}
			class={cn(
				'flex h-9 items-center gap-2 rounded-md px-3 text-sm font-medium text-muted-foreground outline-none hover:bg-hover hover:text-ink focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset md:h-8.5',
				current && 'bg-raised font-semibold text-ink hover:bg-raised'
			)}
		>
			<Icon icon={tab.icon} class="size-4" />
			{tab.label}
		</a>
	{/each}
</nav>
