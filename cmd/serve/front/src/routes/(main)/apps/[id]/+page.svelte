<!--
	Application overview (DESIGN.md §11.7): staging and production as a pipeline, each card led by
	what is live (URL, state, source), with Promote between them; then the services of both
	environments and the recent deployments. A running deployment gets a banner. Logs and previews
	open in drawers (app-workspace.svelte). Polls the application and the recent list.
-->
<script lang="ts">
	import ArrowDown02Icon from '@hugeicons/core-free-icons/ArrowDown02Icon';
	import ArrowRight01Icon from '@hugeicons/core-free-icons/ArrowRight01Icon';
	import ArrowRight02Icon from '@hugeicons/core-free-icons/ArrowRight02Icon';
	import CancelCircleIcon from '@hugeicons/core-free-icons/CancelCircleIcon';
	import CheckmarkCircle02Icon from '@hugeicons/core-free-icons/CheckmarkCircle02Icon';
	import Clock01Icon from '@hugeicons/core-free-icons/Clock01Icon';
	import CommandLineIcon from '@hugeicons/core-free-icons/CommandLineIcon';
	import ContainerIcon from '@hugeicons/core-free-icons/ContainerIcon';
	import LinkSquare02Icon from '@hugeicons/core-free-icons/LinkSquare02Icon';
	import Loading03Icon from '@hugeicons/core-free-icons/Loading03Icon';
	import RepeatIcon from '@hugeicons/core-free-icons/RepeatIcon';
	import Rocket01Icon from '@hugeicons/core-free-icons/Rocket01Icon';
	import ServerStack01Icon from '@hugeicons/core-free-icons/ServerStack01Icon';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import type { IconData } from '#lib/components/icon.svelte';
	import Icon from '#lib/components/icon.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import appsService from '#lib/resources/apps.ts';
	import service, {
		DeploymentStatus,
		type Deployment,
		type DeploymentDetail,
		type Environment
	} from '#lib/resources/deployments.ts';
	import { deploymentBadge, deploymentMoment } from '#lib/status.ts';
	import { cn } from '#lib/utils.js';
	import AppWorkspace, { type AppActions } from './app-workspace.svelte';
	import { liveUrl } from './app-state.ts';

	let { data } = $props();

	const appQuery = $derived(appsService.queryById(data.app.id));
	const appStore = $derived(appQuery.data);
	const app = $derived($appStore ?? data.app);

	const recentQuery = $derived(service.queryAllByApp(data.app.id, { page: 1 }));
	const recentStore = $derived(recentQuery.data);
	const recent = $derived($recentStore?.data ?? []);

	const deleting = $derived(!!app.cleanup_requested_at);
	const staging = $derived(app.latest_deployments.staging);
	const production = $derived(app.latest_deployments.production);
	const running = $derived(
		(['staging', 'production'] as const)
			.map((environment) => app.latest_deployments[environment])
			.find((d) => d && !d.state.finished_at)
	);

	const isGit = (d?: Deployment) => d?.source.discriminator === 'git';
	const hashOf = (d?: Deployment) =>
		d?.source.discriminator === 'git' ? d.source.data.hash : undefined;

	// Promote is offered once staging has a successful deployment, and not while it runs.
	const canPromote = $derived(!deleting && staging?.state.status === DeploymentStatus.Succeeded);
	const newer = $derived(!!staging && isGit(staging) && hashOf(staging) !== hashOf(production));

	let promoteOpen = $state(false);
	let redeployOpen = $state(false);
	let redeployNumber = $state(0);

	function sourceLabel(d: Deployment) {
		const kind = l.translate(`deployment.source.${d.source.discriminator}`);
		return d.source.discriminator === 'git' ? `${kind} · ${d.source.data.branch}` : kind;
	}

	function timing(d: Deployment) {
		const { started_at, finished_at } = d.state;
		if (started_at && finished_at) {
			return `${l.relative(finished_at)} · ${l.translate('preview.took', [l.duration(started_at, finished_at)])}`;
		}
		if (started_at) return l.translate('preview.started', [l.relative(started_at)]);
		return l.translate('preview.queued', [l.relative(d.requested_at)]);
	}

	const rowIcon: Record<DeploymentStatus, { icon: IconData; tone: string; spin?: boolean }> = {
		[DeploymentStatus.Succeeded]: { icon: CheckmarkCircle02Icon, tone: 'text-success' },
		[DeploymentStatus.Failed]: { icon: CancelCircleIcon, tone: 'text-danger' },
		[DeploymentStatus.Running]: { icon: Loading03Icon, tone: 'text-warning', spin: true },
		[DeploymentStatus.Pending]: { icon: Clock01Icon, tone: 'text-muted-foreground' }
	};

	type EnvService = { url?: string; image: string };
	type ServiceRow = { name: string; production?: EnvService; staging?: EnvService };
	const services = $derived.by(() => {
		const rows: Record<string, ServiceRow> = {};
		for (const environment of ['production', 'staging'] as const) {
			for (const s of app.latest_deployments[environment]?.state.services ?? []) {
				rows[s.name] ??= { name: s.name };
				rows[s.name][environment] = { url: s.entrypoints.find((e) => e.url)?.url, image: s.image };
			}
		}
		return Object.values(rows).sort((a, b) => a.name.localeCompare(b.name));
	});

	function cardState(latest: DeploymentDetail) {
		if (!latest.state.finished_at) return 'deploying';
		return latest.state.status === DeploymentStatus.Succeeded ? 'live' : 'failed';
	}
</script>

<AppWorkspace {app} apiKey={data.user.api_key} tab="overview">
	{#snippet children(actions: AppActions)}
		<div class="flex flex-col gap-8">
			{#if deleting}
				<Alert.Root variant="destructive">
					<Icon icon={Loading03Icon} class="animate-spin" />
					<Alert.Title class="font-semibold">{l.translate('apps.deletion_requested')}</Alert.Title>
					<Alert.Description>{l.translate('app.deleting.description')}</Alert.Description>
				</Alert.Root>
			{:else if running}
				<div
					role="status"
					class="grid grid-cols-[20px_minmax(0,1fr)] items-center gap-3 rounded-lg bg-warning-soft py-3 pr-3 pl-4 sm:grid-cols-[20px_minmax(0,1fr)_auto]"
				>
					<Icon icon={Loading03Icon} class="size-5 animate-spin text-warning" />
					<span class="flex min-w-0 flex-col gap-0.5">
						<strong class="text-sm font-semibold">
							{l.translate('detail.running.title', [
								running.deployment_number,
								running.environment
							])}
						</strong>
						<span class="text-[13px]">
							{timing(running)}{#if running.environment === 'staging' && production}
								· {l.translate('detail.running.serving', [production.deployment_number])}{/if}
						</span>
					</span>
					<Button
						variant="outline"
						size="sm"
						class="max-sm:col-span-2"
						onclick={() => actions.openPreview(running.deployment_number)}
					>
						<Icon icon={CommandLineIcon} />
						{l.translate('detail.follow_logs')}
					</Button>
				</div>
			{/if}

			<section class="flex flex-col gap-3" aria-labelledby="environments-title">
				<div class="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1">
					<h2 id="environments-title" class="text-base font-semibold">
						{l.translate('detail.environments')}
					</h2>
					<p class="text-[13px] text-muted-foreground">{l.translate('detail.environments.help')}</p>
				</div>

				<div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_176px_minmax(0,1fr)]">
					{#each ['staging', 'connector', 'production'] as const as slot (slot)}
						{#if slot === 'connector'}
							<div class="flex items-center justify-center gap-3 text-center lg:flex-col">
								<span
									class={cn(
										'flex size-10 items-center justify-center rounded-full bg-raised',
										canPromote ? 'text-primary' : 'text-faint'
									)}
								>
									<Icon icon={ArrowRight02Icon} class="hidden size-5 lg:block" />
									<Icon icon={ArrowDown02Icon} class="size-5 lg:hidden" />
								</span>
								{#if staging}
									<Button
										variant="outline"
										size="sm"
										disabled={!canPromote}
										onclick={() => (promoteOpen = true)}
									>
										<Icon icon={Rocket01Icon} />
										{l.translate('detail.promote', [staging.deployment_number])}
									</Button>
									<span class="hidden max-w-40 text-xs text-muted-foreground lg:block">
										{#if !staging.state.finished_at}
											{l.translate('detail.promote.wait', [staging.deployment_number])}
										{:else if !canPromote}
											{l.translate('detail.promote.failed')}
										{:else if newer}
											{l.translate('detail.promote.newer', [staging.deployment_number])}
										{:else}
											{l.translate('detail.promote.same')}
										{/if}
									</span>
								{:else}
									<span class="max-w-40 text-xs text-muted-foreground">
										{l.translate('detail.promote.none')}
									</span>
								{/if}
							</div>
						{:else}
							{@const environment = slot as Environment}
							{@const latest = app.latest_deployments[environment]}
							{@const target = app[environment].target}
							{@const url = liveUrl(latest)}
							{@const state = latest ? cardState(latest) : undefined}
							<section
								aria-labelledby="card-{environment}"
								class={cn(
									'flex min-w-0 flex-col gap-4 rounded-lg bg-sunken p-5',
									state === 'deploying' && 'ring-1 ring-warning ring-inset'
								)}
							>
								<div class="flex items-center justify-between gap-3">
									<div class="flex items-center gap-2.5">
										<h3 id="card-{environment}" class="text-lg/[26px] font-semibold">
											{l.translate(`deploy.environment.${environment}`)}
										</h3>
										{#if state}
											<span
												class={cn(
													'flex items-center gap-1.5 text-xs font-semibold',
													state === 'live' && 'text-success',
													state === 'deploying' && 'text-warning',
													state === 'failed' && 'text-danger'
												)}
											>
												{#if state === 'deploying'}
													<Icon icon={Loading03Icon} class="size-3 animate-spin" />
												{:else}
													<span class="size-2 rounded-full bg-current"></span>
												{/if}
												{l.translate(`detail.card.${state}`)}
											</span>
										{/if}
									</div>
									<a
										href={routes.editTarget(target.id)}
										class="flex h-6 items-center gap-1.5 rounded-sm bg-raised px-2 text-xs font-medium text-ink outline-none hover:bg-hover focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset"
									>
										<Icon icon={ServerStack01Icon} class="size-3.5" />
										{l.translate('app.on_target', [target.name])}
									</a>
								</div>

								{#if url}
									<a
										href={url}
										target="_blank"
										rel="noopener noreferrer"
										class="flex items-center gap-1.5 self-start truncate rounded-sm font-mono text-[13.5px] text-primary outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring"
									>
										{url}
										<Icon icon={LinkSquare02Icon} class="size-3.5 shrink-0" />
									</a>
								{:else}
									<p class="text-[13px] text-muted-foreground">{l.translate('app.no_url')}</p>
								{/if}

								{#if latest}
									{@const badge = deploymentBadge(latest.state)}
									<div class="flex flex-wrap items-center justify-between gap-3">
										<StatusBadge
											state={badge.state}
											label="{badge.label} · #{latest.deployment_number}"
										/>
										<span class="text-[12.5px] text-muted-foreground">{timing(latest)}</span>
									</div>
									<dl class="grid grid-cols-[96px_minmax(0,1fr)] gap-x-3 gap-y-2 text-[13px]/5">
										<dt class="text-muted-foreground">{l.translate('deployment.source')}</dt>
										<dd class="truncate">
											{sourceLabel(latest)}
											{#if hashOf(latest)}
												· <span class="font-mono">{hashOf(latest)?.slice(0, 7)}</span>
											{/if}
										</dd>
										{#if latest.state.error_code}
											<dt class="text-muted-foreground first-letter:uppercase">
												{l.translate('deployment.error_code')}
											</dt>
											<dd class="truncate font-mono text-danger">{latest.state.error_code}</dd>
										{:else if latest.state.services?.length}
											<dt class="text-muted-foreground">{l.translate('deployment.services')}</dt>
											<dd class="truncate">
												{latest.state.services.map((s) => s.name).join(' · ')}
											</dd>
										{/if}
										<dt class="text-muted-foreground">{l.translate('deployment.queued_by')}</dt>
										<dd class="truncate">{latest.requested_by.email}</dd>
									</dl>
									<div class="mt-auto flex flex-wrap gap-1.5">
										<Button
											variant="outline"
											size="sm"
											onclick={() => actions.openPreview(latest.deployment_number)}
										>
											<Icon icon={CommandLineIcon} />
											{state === 'deploying'
												? l.translate('detail.follow_logs')
												: l.translate('detail.logs')}
										</Button>
										{#if state !== 'deploying' && !deleting}
											<Button
												variant="outline"
												size="sm"
												onclick={() => {
													redeployNumber = latest.deployment_number;
													redeployOpen = true;
												}}
											>
												<Icon icon={RepeatIcon} />
												{l.translate('deployment.redeploy')}
											</Button>
										{/if}
									</div>
								{:else}
									<div class="flex flex-col items-start gap-3">
										<p class="text-sm text-muted-foreground">
											{l.translate(`app.env.empty.${environment}`)}
										</p>
										{#if !deleting}
											<Button
												variant="outline"
												size="sm"
												onclick={() => actions.quickDeploy(environment)}
											>
												<Icon icon={Rocket01Icon} />
												{l.translate('app.env.deploy', [environment])}
											</Button>
										{/if}
									</div>
								{/if}
							</section>
						{/if}
					{/each}
				</div>
			</section>

			<div class="grid items-start gap-8 xl:grid-cols-[minmax(0,1.25fr)_minmax(0,1fr)]">
				<section class="flex min-w-0 flex-col gap-3" aria-labelledby="services-title">
					<div class="flex flex-wrap items-baseline justify-between gap-x-3">
						<h2 id="services-title" class="text-base font-semibold">
							{l.translate('detail.services')}
						</h2>
						{#if production || staging}
							<p class="text-[13px] text-muted-foreground">
								{l.translate('detail.services.from', [
									production ? `#${production.deployment_number}` : '—',
									staging ? `#${staging.deployment_number}` : '—'
								])}
							</p>
						{/if}
					</div>
					{#if services.length > 0}
						<div role="table" aria-labelledby="services-title" class="flex flex-col gap-1.5">
							<div
								role="row"
								class="hidden grid-cols-[120px_minmax(0,1fr)_minmax(0,1fr)] gap-4 px-4 pb-0.5 text-xs font-semibold text-muted-foreground md:grid"
							>
								<span role="columnheader">{l.translate('detail.services.service')}</span>
								<span role="columnheader">{l.translate('deploy.environment.production')}</span>
								<span role="columnheader">{l.translate('deploy.environment.staging')}</span>
							</div>
							{#each services as s (s.name)}
								<div
									role="row"
									class="grid min-h-14 items-center gap-x-4 gap-y-1.5 rounded-md bg-sunken px-4 py-2.5 md:grid-cols-[120px_minmax(0,1fr)_minmax(0,1fr)]"
								>
									<span role="cell" class="flex items-center gap-2.5 text-sm font-semibold">
										<Icon icon={ContainerIcon} class="size-4 text-muted-foreground" />
										{s.name}
									</span>
									{#each [s.production, s.staging] as env, i (i)}
										<span role="cell" class="flex min-w-0 flex-col">
											{#if env?.url}
												<a
													href={env.url}
													target="_blank"
													rel="noopener noreferrer"
													class="truncate font-mono text-[12.5px] text-primary hover:underline"
													>{env.url}</a
												>
											{:else}
												<span class="text-[13px] text-muted-foreground">
													{env ? l.translate('detail.services.internal') : '—'}
												</span>
											{/if}
											{#if env}
												<span class="truncate font-mono text-[11.5px] text-faint">{env.image}</span>
											{/if}
										</span>
									{/each}
								</div>
							{/each}
						</div>
					{:else}
						<p class="rounded-md bg-sunken px-4 py-3.5 text-sm text-muted-foreground">
							{l.translate('detail.services.none')}
						</p>
					{/if}
				</section>

				<section class="flex min-w-0 flex-col gap-3" aria-labelledby="recent-title">
					<div class="flex items-baseline justify-between gap-3">
						<h2 id="recent-title" class="text-base font-semibold">
							{l.translate('detail.recent')}
						</h2>
						<a
							href={routes.appDeployments(app.id)}
							class="flex items-center gap-1 rounded-sm text-[13px] font-semibold text-primary outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring"
						>
							{l.translate('detail.recent.all')}
							<Icon icon={ArrowRight01Icon} class="size-3.5" />
						</a>
					</div>
					{#if recent.length > 0}
						<div class="flex flex-col gap-1.5">
							{#each recent as d (d.deployment_number)}
								{@const row = rowIcon[d.state.status]}
								<button
									type="button"
									class={cn(
										'grid min-h-11 grid-cols-[18px_34px_auto_minmax(0,1fr)_auto] items-center gap-2.5 rounded-md bg-sunken px-3.5 text-left text-[13px] text-ink transition-colors outline-none hover:bg-hover focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset',
										actions.previewNumber() === d.deployment_number && 'bg-raised'
									)}
									onclick={() => actions.openPreview(d.deployment_number)}
								>
									<Icon
										icon={row.icon}
										class={cn('size-4', row.tone, row.spin && 'animate-spin')}
									/>
									<span class="font-semibold tabular-nums">#{d.deployment_number}</span>
									<span
										class={cn(
											'rounded-sm px-1.5 text-[11.5px]/5 font-semibold',
											d.environment === 'production'
												? 'bg-accent-soft text-primary'
												: 'bg-raised text-muted-foreground'
										)}>{d.environment}</span
									>
									<span class="truncate">
										{deploymentBadge(d.state).label} · {sourceLabel(d)}
										{#if d.state.error_code}
											· <span class="font-mono text-muted-foreground">{d.state.error_code}</span>
										{:else if hashOf(d)}
											· <span class="font-mono text-muted-foreground">{hashOf(d)?.slice(0, 7)}</span
											>
										{/if}
									</span>
									<span class="text-muted-foreground tabular-nums"
										>{l.relative(deploymentMoment(d))}</span
									>
								</button>
							{/each}
						</div>
					{:else}
						<div class="flex flex-col items-start gap-3 rounded-md bg-sunken px-4 py-4">
							<p class="text-sm text-muted-foreground">{l.translate('app.empty.description')}</p>
							{#if !deleting}
								<Button size="sm" onclick={() => actions.openDeploy('production')}>
									<Icon icon={Rocket01Icon} />
									{l.translate('deployment.new')}
								</Button>
							{/if}
						</div>
					{/if}
				</section>
			</div>
		</div>

		{#if staging}
			<ConfirmDialog
				bind:open={promoteOpen}
				title={l.translate('deployment.promote.title')}
				description={l.translate('deployment.promote.description', [staging.deployment_number])}
				confirmLabel={l.translate('deployment.promote.title')}
				icon={Rocket01Icon}
				onconfirm={() =>
					service
						.promote(app.id, staging.deployment_number)
						.then((d) => actions.openPreview(d.deployment_number))}
			/>
		{/if}
		<ConfirmDialog
			bind:open={redeployOpen}
			title={l.translate('deployment.redeploy')}
			description={l.translate('deployment.redeploy.description', [
				redeployNumber,
				redeployNumber === production?.deployment_number ? 'production' : 'staging'
			])}
			confirmLabel={l.translate('deployment.redeploy')}
			icon={RepeatIcon}
			onconfirm={() =>
				service
					.redeploy(app.id, redeployNumber)
					.then((d) => actions.openPreview(d.deployment_number))}
		/>
	{/snippet}
</AppWorkspace>
