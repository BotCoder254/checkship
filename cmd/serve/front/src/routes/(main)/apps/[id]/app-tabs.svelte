<!--
	Application tabs (DESIGN.md §11.7): Overview, Deployments (with the total), Logs, Settings. Links, so
	each tab has its own address; the current one has the raised fill. They scroll sideways on phones.
-->
<script lang="ts">
	import ComputerTerminal01Icon from '@hugeicons/core-free-icons/ComputerTerminal01Icon';
	import DashboardSquare01Icon from '@hugeicons/core-free-icons/DashboardSquare01Icon';
	import Layers01Icon from '@hugeicons/core-free-icons/Layers01Icon';
	import Settings02Icon from '@hugeicons/core-free-icons/Settings02Icon';
	import Icon from '#lib/components/icon.svelte';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service from '#lib/resources/deployments.ts';
	import { cn } from '#lib/utils.js';

	type Props = { appId: string; active: 'overview' | 'deployments' | 'logs' | 'settings' };

	let { appId, active }: Props = $props();

	// Page 1 of the whole history is enough to know the total.
	const totalQuery = $derived(service.queryAllByApp(appId, { page: 1 }));
	const totalStore = $derived(totalQuery.data);
	const total = $derived($totalStore?.total);

	const tabs = $derived([
		{
			id: 'overview',
			href: routes.app(appId),
			icon: DashboardSquare01Icon,
			label: l.translate('detail.tab.overview')
		},
		{
			id: 'deployments',
			href: routes.appDeployments(appId),
			icon: Layers01Icon,
			label: l.translate('detail.tab.deployments'),
			count: total
		},
		{
			id: 'logs',
			href: routes.appLogs(appId),
			icon: ComputerTerminal01Icon,
			label: l.translate('detail.tab.logs')
		},
		{
			id: 'settings',
			href: routes.editApp(appId),
			icon: Settings02Icon,
			label: l.translate('detail.tab.settings')
		}
	]);
</script>

<nav aria-label={l.translate('detail.tabs')} class="-mx-1 flex gap-1 overflow-x-auto px-1">
	{#each tabs as tab (tab.id)}
		{@const current = tab.id === active}
		<a
			href={tab.href}
			aria-current={current ? 'page' : undefined}
			class={cn(
				'flex h-9 shrink-0 items-center gap-2 rounded-md px-3 text-sm font-medium whitespace-nowrap text-muted-foreground outline-none hover:bg-hover hover:text-ink focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset md:h-8.5',
				current && 'bg-raised font-semibold text-ink hover:bg-raised'
			)}
		>
			<Icon icon={tab.icon} class="size-4" />
			{tab.label}
			{#if tab.count !== undefined}
				<span
					class={cn(
						'min-w-5 rounded-full px-1.5 text-center text-[11.5px]/[18px] tabular-nums',
						current ? 'bg-line-strong' : 'bg-raised'
					)}>{tab.count}</span
				>
			{/if}
		</a>
	{/each}
</nav>
