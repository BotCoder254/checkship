<!--
	Applications (DESIGN.md §5, §11.1): one row per application on a sunken fill, 6px apart, no lines.
	Each row shows the latest production and staging deployments as state badges.
-->
<script lang="ts">
	import Add01Icon from '@hugeicons/core-free-icons/Add01Icon';
	import ArrowRight01Icon from '@hugeicons/core-free-icons/ArrowRight01Icon';
	import DashboardSquare01Icon from '@hugeicons/core-free-icons/DashboardSquare01Icon';
	import EmptyState from '#lib/components/empty-state.svelte';
	import Icon from '#lib/components/icon.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service, { type App } from '#lib/resources/apps.ts';
	import { DeploymentStatus, type Deployment } from '#lib/resources/deployments.ts';
	import jobsService from '#lib/resources/jobs.ts';
	import { deploymentBadge, deploymentMoment } from '#lib/status.ts';

	const { data: apps } = service.queryAll();
	const { data: jobs } = jobsService.queryAll({ page: 1 });

	const failed = $derived(
		($apps ?? []).filter(
			(app) => app.latest_deployments.production?.state.status === DeploymentStatus.Failed
		).length
	);

	/** The default service's URL: <app>.<target host> (how checkship exposes production). */
	function appUrl(app: App): string | undefined {
		if (!app.production_target.url) return undefined;
		try {
			const target = new URL(app.production_target.url);
			return `${target.protocol}//${app.name}.${target.host}`;
		} catch {
			return undefined;
		}
	}
</script>

{#snippet environment(deployment: Deployment | undefined)}
	<span role="cell" class="flex flex-col items-start gap-1">
		{#if deployment}
			{@const badge = deploymentBadge(deployment.state)}
			<StatusBadge state={badge.state} label="{badge.label} · #{deployment.deployment_number}" />
			<span class="text-xs text-muted-foreground">
				{l.relative(deploymentMoment(deployment))}{#if deployment.state.error_code}
					· {deployment.state.error_code}{/if}
			</span>
		{:else}
			<span class="text-sm text-muted-foreground">{l.translate('apps.no_deployment')}</span>
		{/if}
	</span>
{/snippet}

<PageHeader title={l.translate('breadcrumb.applications')}>
	{#snippet actions()}
		<Button href={routes.createApp}>
			<Icon icon={Add01Icon} />
			{l.translate('app.new')}
		</Button>
	{/snippet}
	{#snippet status()}
		{#if $apps}
			<span>{l.translate('apps.summary', [$apps.length, failed, $jobs?.total ?? 0])}</span>
		{/if}
	{/snippet}
</PageHeader>

{#if $apps && $apps.length > 0}
	<div
		role="table"
		aria-label={l.translate('breadcrumb.applications')}
		class="flex flex-col gap-1.5"
	>
		<div
			role="row"
			class="hidden grid-cols-[minmax(0,1.5fr)_minmax(0,1fr)_minmax(0,1fr)_20px] gap-4.5 px-4.5 pb-1.5 text-xs font-semibold text-muted-foreground md:grid"
		>
			<span role="columnheader">{l.translate('apps.column.application')}</span>
			<span role="columnheader">{l.translate('apps.column.production')}</span>
			<span role="columnheader">{l.translate('apps.column.staging')}</span>
			<span role="columnheader"
				><span class="sr-only">{l.translate('breadcrumb.applications')}</span></span
			>
		</div>

		{#each $apps as app (app.id)}
			{@const url = appUrl(app)}
			<a
				role="row"
				href={routes.app(app.id)}
				class="grid grid-cols-1 gap-3 rounded-md bg-sunken px-4 py-3.5 text-ink transition-colors outline-none hover:bg-hover focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset md:grid-cols-[minmax(0,1.5fr)_minmax(0,1fr)_minmax(0,1fr)_20px] md:items-center md:gap-4.5 md:px-4.5"
			>
				<span role="cell" class="flex min-w-0 flex-col gap-0.5">
					<span class="truncate text-[15px]/[22px] font-semibold">{app.name}</span>
					<span class="truncate font-mono text-xs text-muted-foreground">
						{url ?? l.translate('apps.on_target', [app.production_target.name])}
					</span>
				</span>

				{#if app.cleanup_requested_at}
					<span role="cell" class="flex flex-wrap items-center gap-3 md:col-span-2">
						<StatusBadge state="deleting" label={l.translate('apps.deletion_requested')} />
						<span class="text-xs text-muted-foreground">
							{l.translate('apps.deletion_requested.description')}
						</span>
					</span>
				{:else}
					<span class="grid grid-cols-[80px_minmax(0,1fr)] items-center gap-y-1.5 md:contents">
						<span class="text-xs text-muted-foreground md:hidden">
							{l.translate('apps.column.production')}
						</span>
						{@render environment(app.latest_deployments.production)}
						<span class="text-xs text-muted-foreground md:hidden">
							{l.translate('apps.column.staging')}
						</span>
						{@render environment(app.latest_deployments.staging)}
					</span>
				{/if}

				<Icon icon={ArrowRight01Icon} class="hidden size-4.5 text-muted-foreground md:block" />
			</a>
		{/each}
	</div>
{:else if $apps}
	<EmptyState
		icon={DashboardSquare01Icon}
		title={l.translate('apps.empty.title')}
		description={l.translate('apps.empty.description')}
	>
		{#snippet action()}
			<Button href={routes.createApp}>
				<Icon icon={Add01Icon} />
				{l.translate('app.new')}
			</Button>
		{/snippet}
	</EmptyState>
{/if}
