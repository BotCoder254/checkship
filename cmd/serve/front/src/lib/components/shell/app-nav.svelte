<!--
	Sidebar (DESIGN.md §5). Workspace links start 24px below the title bar; Recent lists the last
	five applications opened, each with its production state; Documentation, the theme switch and
	the version sit at the bottom. No lines: groups are spacing and small headings.
-->
<script lang="ts">
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import Archive02Icon from '@hugeicons/core-free-icons/Archive02Icon';
	import Book02Icon from '@hugeicons/core-free-icons/Book02Icon';
	import CancelCircleIcon from '@hugeicons/core-free-icons/CancelCircleIcon';
	import CheckmarkCircle02Icon from '@hugeicons/core-free-icons/CheckmarkCircle02Icon';
	import Clock01Icon from '@hugeicons/core-free-icons/Clock01Icon';
	import DashboardSpeed01Icon from '@hugeicons/core-free-icons/DashboardSpeed01Icon';
	import DashboardSquare01Icon from '@hugeicons/core-free-icons/DashboardSquare01Icon';
	import LinkSquare02Icon from '@hugeicons/core-free-icons/LinkSquare02Icon';
	import Loading03Icon from '@hugeicons/core-free-icons/Loading03Icon';
	import PackageIcon from '@hugeicons/core-free-icons/PackageIcon';
	import ServerStack01Icon from '@hugeicons/core-free-icons/ServerStack01Icon';
	import TaskDone01Icon from '@hugeicons/core-free-icons/TaskDone01Icon';
	import { page } from '$app/state';
	import Icon from '#lib/components/icon.svelte';
	import NavLink from '#lib/components/shell/nav-link.svelte';
	import ThemeSwitch from '#lib/components/shell/theme-switch.svelte';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import { preferences } from '#lib/preferences.svelte.ts';
	import appsService from '#lib/resources/apps.ts';
	import { DeploymentStatus } from '#lib/resources/deployments.ts';
	import jobsService from '#lib/resources/jobs.ts';
	import targetsService, { TargetStatus } from '#lib/resources/targets.ts';
	import { cn } from '#lib/utils.js';

	const DOCS_URL = 'https://checkship-deploy.github.io/checkship/guide/quickstart.html';
	const SOURCE_URL = 'https://github.com/checkship-deploy/checkship/';
	const CREDITS_URL = 'https://checkship-deploy.github.io/checkship/contributing/credits.html';

	type Props = {
		collapsed?: boolean;
		version: string;
		/** Called after a link is followed, so the mobile sheet can close. */
		onnavigate?: () => void;
	};

	let { collapsed = false, version, onnavigate }: Props = $props();

	const { data: apps } = appsService.queryAll();
	const { data: targets } = targetsService.queryAll();
	const { data: jobs } = jobsService.queryAll({ page: 1 });

	const path = $derived(page.url.pathname);
	const failingTargets = $derived(
		($targets ?? []).filter((t) => t.state.status === TargetStatus.Failed).length
	);
	const queuedJobs = $derived($jobs?.total ?? 0);
	const recent = $derived(
		preferences.recentApps
			.map((id) => ($apps ?? []).find((app) => app.id === id))
			.filter((app) => app !== undefined)
	);

	function recentIcon(status: DeploymentStatus | undefined) {
		switch (status) {
			case DeploymentStatus.Succeeded:
				return { icon: CheckmarkCircle02Icon, tone: 'text-success' };
			case DeploymentStatus.Failed:
				return { icon: CancelCircleIcon, tone: 'text-danger' };
			case DeploymentStatus.Running:
				return { icon: Loading03Icon, tone: 'text-warning' };
			default:
				return { icon: Clock01Icon, tone: 'text-muted-foreground' };
		}
	}
</script>

<nav
	aria-label={l.translate('nav.main')}
	class={cn(
		'flex h-full flex-col justify-between gap-7.5 overflow-y-auto pt-6 pb-4.5',
		collapsed ? 'items-center px-2' : 'px-3'
	)}
>
	<div class="flex flex-col gap-7.5">
		<div class="flex flex-col gap-1">
			{#if !collapsed}
				<p class="mb-1.5 px-3 text-xs font-semibold tracking-wide text-muted-foreground">
					{l.translate('nav.workspace')}
				</p>
			{/if}
			<NavLink
				href={routes.dashboard}
				label={l.translate('breadcrumb.overview')}
				icon={DashboardSpeed01Icon}
				active={path === '/'}
				{collapsed}
				onclick={onnavigate}
			/>
			<NavLink
				href={routes.apps}
				label={l.translate('breadcrumb.applications')}
				icon={DashboardSquare01Icon}
				active={path.startsWith('/apps')}
				{collapsed}
				onclick={onnavigate}
			/>
			<NavLink
				href={routes.targets}
				label={l.translate('breadcrumb.targets')}
				icon={ServerStack01Icon}
				active={path.startsWith(routes.targets)}
				description={failingTargets > 0
					? l.translate('nav.targets_failing', [failingTargets])
					: undefined}
				{collapsed}
				onclick={onnavigate}
			>
				{#snippet trailing()}
					{#if failingTargets > 0}
						<Icon icon={Alert02Icon} class="size-4 text-danger" />
					{/if}
				{/snippet}
			</NavLink>
			<NavLink
				href={routes.registries}
				label={l.translate('breadcrumb.registries')}
				icon={PackageIcon}
				active={path.startsWith(routes.registries)}
				{collapsed}
				onclick={onnavigate}
			/>
			<NavLink
				href={routes.backups}
				label={l.translate('breadcrumb.backups')}
				icon={Archive02Icon}
				active={path.startsWith(routes.backups)}
				{collapsed}
				onclick={onnavigate}
			/>
			<NavLink
				href={routes.jobs}
				label={l.translate('breadcrumb.jobs')}
				icon={TaskDone01Icon}
				active={path.startsWith(routes.jobs)}
				description={queuedJobs > 0 ? l.translate('nav.jobs_queued', [queuedJobs]) : undefined}
				{collapsed}
				onclick={onnavigate}
			>
				{#snippet trailing()}
					{#if queuedJobs > 0}
						<span
							class="inline-flex h-5 min-w-5 items-center justify-center rounded-sm bg-raised px-1.5 text-xs font-semibold text-ink tabular-nums"
						>
							{queuedJobs}
						</span>
					{/if}
				{/snippet}
			</NavLink>
		</div>

		{#if !collapsed && recent.length > 0}
			<div class="flex flex-col gap-0.5">
				<p class="mb-1.5 px-3 text-xs font-semibold tracking-wide text-muted-foreground">
					{l.translate('nav.recent')}
				</p>
				{#each recent as app (app.id)}
					{@const state = recentIcon(app.latest_deployments.production?.state.status)}
					<a
						href={routes.app(app.id)}
						onclick={onnavigate}
						aria-current={path.startsWith(routes.app(app.id)) ? 'page' : undefined}
						class="flex h-9 items-center gap-3 rounded-md px-3 text-sm text-ink outline-none hover:bg-hover focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset aria-[current=page]:font-semibold"
					>
						<Icon icon={state.icon} class={cn('size-4', state.tone)} />
						<span class="min-w-0 flex-1 truncate">{app.name}</span>
					</a>
				{/each}
			</div>
		{/if}
	</div>

	<div class={cn('flex flex-col gap-3', collapsed && 'items-center')}>
		<NavLink
			href={DOCS_URL}
			external
			label={l.translate('nav.documentation')}
			icon={Book02Icon}
			{collapsed}
		>
			{#snippet trailing()}
				<Icon icon={LinkSquare02Icon} class="size-3.5 text-muted-foreground" />
			{/snippet}
		</NavLink>

		{#if collapsed}
			<ThemeSwitch collapsed />
		{:else}
			<div class="flex flex-col gap-3 px-3">
				<ThemeSwitch />
				<p class="flex flex-wrap gap-x-2 pt-1.5 text-xs text-muted-foreground">
					<span title={version}
						>{l.translate('nav.version', [
							version.replace(/-([0-9a-f]{7})[0-9a-f]+$/, '-$1')
						])}</span
					>
					<a href={SOURCE_URL} target="_blank" rel="noopener noreferrer" class="hover:text-ink">
						{l.translate('footer.source')}
					</a>
					<a href={CREDITS_URL} target="_blank" rel="noopener noreferrer" class="hover:text-ink">
						{l.translate('footer.credits')}
					</a>
				</p>
			</div>
		{/if}
	</div>
</nav>
