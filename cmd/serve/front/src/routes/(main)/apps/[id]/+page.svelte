<!--
	Application overview (DESIGN.md §11.7): staging and production as a pipeline, each card led by
	what is live (URL, state, source), with Promote between them; then the services of both
	environments and the recent deployments. A running deployment gets a banner. Logs and previews
	open in drawers (app-workspace.svelte). Polls the application and the recent list.
-->
<script lang="ts">
	import ArrowDown02Icon from '@hugeicons/core-free-icons/ArrowDown02Icon';
	import ArrowRight01Icon from '@hugeicons/core-free-icons/ArrowRight01Icon';
	import CheckmarkCircle02Icon from '@hugeicons/core-free-icons/CheckmarkCircle02Icon';
	import Clock01Icon from '@hugeicons/core-free-icons/Clock01Icon';
	import CommandLineIcon from '@hugeicons/core-free-icons/CommandLineIcon';
	import ComputerTerminal01Icon from '@hugeicons/core-free-icons/ComputerTerminal01Icon';
	import ContainerIcon from '@hugeicons/core-free-icons/ContainerIcon';
	import LinkSquare02Icon from '@hugeicons/core-free-icons/LinkSquare02Icon';
	import Loading03Icon from '@hugeicons/core-free-icons/Loading03Icon';
	import RepeatIcon from '@hugeicons/core-free-icons/RepeatIcon';
	import Rocket01Icon from '@hugeicons/core-free-icons/Rocket01Icon';
	import ServerStack01Icon from '@hugeicons/core-free-icons/ServerStack01Icon';
	import Copy01Icon from '@hugeicons/core-free-icons/Copy01Icon';
	import GitCommitIcon from '@hugeicons/core-free-icons/GitCommitIcon';
	import Image01Icon from '@hugeicons/core-free-icons/Image01Icon';
	import Tag01Icon from '@hugeicons/core-free-icons/Tag01Icon';
	import TestTube01Icon from '@hugeicons/core-free-icons/TestTube01Icon';
	import { toast } from 'svelte-sonner';
	import { copyText } from '#lib/clipboard.ts';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import EnvironmentChip from '#lib/components/environment-chip.svelte';
	import RowMenuGroup from '#lib/components/table/row-menu-group.svelte';
	import RowMenuItem from '#lib/components/table/row-menu-item.svelte';
	import RowMenu from '#lib/components/table/row-menu.svelte';
	import * as Table from '#lib/components/ui/table/index.js';
	import { ACTIONS_CELL, ACTIONS_HEAD, TableColumns } from '#lib/table-columns.svelte.ts';
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

	const serviceColumns = new TableColumns([
		{ id: 'service', label: l.translate('column.service'), icon: ContainerIcon, width: 'w-[20%]' },
		{
			id: 'production',
			label: l.translate('column.production'),
			icon: Rocket01Icon,
			width: 'w-[30%]'
		},
		{
			id: 'staging',
			label: l.translate('column.staging'),
			icon: TestTube01Icon,
			width: 'w-[30%]',
			priority: 2
		},
		{
			id: 'image',
			label: l.translate('column.image'),
			icon: Image01Icon,
			width: 'w-[20%]',
			priority: 3
		}
	]);
	const recentColumns = new TableColumns([
		{
			id: 'status',
			label: l.translate('column.status'),
			icon: CheckmarkCircle02Icon,
			width: 'w-[28%]'
		},
		{ id: 'number', label: l.translate('column.number'), icon: Tag01Icon, width: 'w-[14%]' },
		{
			id: 'environment',
			label: l.translate('column.environment'),
			icon: ServerStack01Icon,
			width: 'w-[22%]',
			priority: 2
		},
		{
			id: 'commit',
			label: l.translate('column.commit'),
			icon: GitCommitIcon,
			width: 'w-[16%]',
			priority: 3
		},
		{ id: 'started', label: l.translate('column.started'), icon: Clock01Icon, width: 'w-[20%]' }
	]);

	async function copy(text: string) {
		if (await copyText(text)) toast.success(l.translate('copied'));
		else toast.error(l.translate('copy_blocked'));
	}

	const copyLink = (number: number) =>
		copy(new URL(routes.deployment(app.id, number), location.origin).href);

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

				<!-- Stacked, each card the full width and the same height (owner's call, 2026-11-14):
				     staging, then Promote, then production. -->
				<div class="flex flex-col gap-3">
					{#each ['staging', 'connector', 'production'] as const as slot (slot)}
						{#if slot === 'connector'}
							<div class="flex flex-wrap items-center gap-x-3.5 gap-y-2 px-1 md:px-6">
								<span
									class={cn(
										'flex size-9 shrink-0 items-center justify-center rounded-full bg-raised',
										canPromote ? 'text-primary' : 'text-faint'
									)}
								>
									<Icon icon={ArrowDown02Icon} class="size-4.5" />
								</span>
								{#if staging}
									<Button
										variant="outline"
										size="sm"
										disabled={!canPromote}
										onclick={() => (promoteOpen = true)}
									>
										<Icon icon={Rocket01Icon} />
										{l.translate('detail.promote.to_production', [staging.deployment_number])}
									</Button>
									<span class="text-[12.5px] text-muted-foreground">
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
									<span class="text-[12.5px] text-muted-foreground">
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
									'grid min-w-0 items-center gap-x-8 gap-y-4 rounded-lg bg-sunken px-5 py-5 md:h-39 md:grid-cols-[minmax(0,1fr)_minmax(0,1.15fr)_150px] md:px-6',
									state === 'deploying' && 'ring-1 ring-warning ring-inset'
								)}
							>
								<div class="flex min-w-0 flex-col gap-2.5">
									<div class="flex flex-wrap items-center gap-2.5">
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
											class="flex min-w-0 items-center gap-1.5 self-start rounded-sm font-mono text-[13.5px] text-primary outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring"
										>
											<span class="truncate">{url}</span>
											<Icon icon={LinkSquare02Icon} class="size-3.5 shrink-0" />
										</a>
									{:else}
										<p class="truncate text-[13px] text-muted-foreground">
											{l.translate('app.no_url')}
										</p>
									{/if}
									{#if latest}
										{@const badge = deploymentBadge(latest.state)}
										<div class="flex flex-wrap items-center gap-3">
											<StatusBadge
												state={badge.state}
												label="{badge.label} · #{latest.deployment_number}"
											/>
											<span class="text-[12.5px] whitespace-nowrap text-muted-foreground"
												>{timing(latest)}</span
											>
										</div>
									{/if}
								</div>

								{#if latest}
									<dl
										class="grid min-w-0 grid-cols-[96px_minmax(0,1fr)] gap-x-3 gap-y-2 text-[13px]/5"
									>
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
									<div class="flex gap-1.5 md:flex-col">
										<Button
											variant="outline"
											size="sm"
											class="max-md:flex-1"
											onclick={() => actions.openPreview(latest.deployment_number)}
										>
											<Icon icon={CommandLineIcon} />
											{state === 'deploying'
												? l.translate('detail.follow_logs')
												: l.translate('detail.logs')}
										</Button>
										{#if state === 'live'}
											<Button
												variant="outline"
												size="sm"
												class="max-md:flex-1"
												href={routes.appLogs(app.id, environment)}
											>
												<Icon icon={ComputerTerminal01Icon} />
												{l.translate('detail.app_logs')}
											</Button>
										{/if}
										{#if state !== 'deploying' && !deleting}
											<Button
												variant="outline"
												size="sm"
												class="max-md:flex-1"
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
									<p class="text-sm text-muted-foreground">
										{l.translate(`app.env.empty.${environment}`)}
									</p>
									<div class="flex md:flex-col">
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
						<Table.Root aria-labelledby="services-title">
							<Table.Header>
								<Table.Row>
									{#each serviceColumns.shown as column (column.id)}
										<Table.Head class={serviceColumns.head(column.id)}>{column.label}</Table.Head>
									{/each}
									<Table.Head class={ACTIONS_HEAD}>{l.translate('column.actions')}</Table.Head>
								</Table.Row>
							</Table.Header>
							<Table.Body>
								{#each services as s (s.name)}
									{@const url = s.production?.url ?? s.staging?.url}
									<Table.Row>
										<Table.Cell class={serviceColumns.cell('service', 'text-sm font-semibold')}>
											{s.name}
										</Table.Cell>
										{#each [['production', s.production], ['staging', s.staging]] as const as [id, env] (id)}
											<Table.Cell class={serviceColumns.cell(id)} title={env?.url}>
												{#if env?.url}
													<a
														href={env.url}
														target="_blank"
														rel="noopener noreferrer"
														class="font-mono text-[12.5px] text-primary hover:underline"
														>{env.url}</a
													>
												{:else}
													<span class="text-muted-foreground">
														{env ? l.translate('detail.services.internal') : '—'}
													</span>
												{/if}
											</Table.Cell>
										{/each}
										<Table.Cell
											class={serviceColumns.cell('image', 'font-mono text-[12.5px] text-faint')}
											title={(s.production ?? s.staging)?.image}
										>
											{(s.production ?? s.staging)?.image ?? '—'}
										</Table.Cell>
										<Table.Cell class={ACTIONS_CELL}>
											<RowMenu label={l.translate('table.menu', [s.name])}>
												<RowMenuGroup>
													{#if s.production?.url}
														<RowMenuItem icon={LinkSquare02Icon} href={s.production.url} external>
															{l.translate('menu.open_production')}
														</RowMenuItem>
													{/if}
													{#if s.staging?.url}
														<RowMenuItem icon={LinkSquare02Icon} href={s.staging.url} external>
															{l.translate('menu.open_staging')}
														</RowMenuItem>
													{/if}
													{#if url}
														<RowMenuItem icon={Copy01Icon} onSelect={() => copy(url)}>
															{l.translate('menu.copy_url')}
														</RowMenuItem>
													{/if}
													<RowMenuItem icon={CommandLineIcon} href={routes.appLogs(app.id)}>
														{l.translate('menu.logs')}
													</RowMenuItem>
												</RowMenuGroup>
											</RowMenu>
										</Table.Cell>
									</Table.Row>
								{/each}
							</Table.Body>
						</Table.Root>
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
						<Table.Root aria-labelledby="recent-title">
							<Table.Header>
								<Table.Row>
									{#each recentColumns.shown as column (column.id)}
										<Table.Head class={recentColumns.head(column.id)}>{column.label}</Table.Head>
									{/each}
									<Table.Head class={ACTIONS_HEAD}>{l.translate('column.actions')}</Table.Head>
								</Table.Row>
							</Table.Header>
							<Table.Body>
								{#each recent as d (d.deployment_number)}
									{@const badge = deploymentBadge(d.state)}
									<Table.Row
										class="relative"
										data-state={actions.previewNumber() === d.deployment_number
											? 'selected'
											: undefined}
									>
										<Table.Cell class={recentColumns.cell('status')}>
											<StatusBadge state={badge.state} label={badge.label} />
										</Table.Cell>
										<Table.Cell class={recentColumns.cell('number', 'font-semibold tabular-nums')}>
											<!-- The whole row opens the preview; the button stretches over it. -->
											<button
												type="button"
												class="outline-none after:absolute after:inset-0 after:rounded-md focus-visible:after:ring-2 focus-visible:after:ring-ring focus-visible:after:ring-inset"
												onclick={() => actions.openPreview(d.deployment_number)}
												>#{d.deployment_number}</button
											>
										</Table.Cell>
										<Table.Cell class={recentColumns.cell('environment')}>
											<EnvironmentChip environment={d.environment} />
										</Table.Cell>
										<Table.Cell
											class={recentColumns.cell(
												'commit',
												d.state.error_code
													? 'font-mono text-[12.5px] text-danger'
													: 'font-mono text-[12.5px] text-muted-foreground'
											)}
											title={d.state.error_code ?? hashOf(d)}
										>
											{d.state.error_code ?? hashOf(d)?.slice(0, 7) ?? '—'}
										</Table.Cell>
										<Table.Cell
											class={recentColumns.cell('started', 'text-muted-foreground tabular-nums')}
										>
											{l.relative(deploymentMoment(d))}
										</Table.Cell>
										<Table.Cell class={ACTIONS_CELL}>
											<RowMenu label={l.translate('detail.row.menu', [d.deployment_number])}>
												<RowMenuGroup>
													<RowMenuItem
														icon={CommandLineIcon}
														onSelect={() => actions.openPreview(d.deployment_number)}
													>
														{l.translate('detail.row.logs')}
													</RowMenuItem>
													{#if !deleting && d.state.finished_at}
														<RowMenuItem
															icon={RepeatIcon}
															onSelect={() => {
																redeployNumber = d.deployment_number;
																redeployOpen = true;
															}}
														>
															{l.translate('deployment.redeploy')}
														</RowMenuItem>
													{/if}
													<RowMenuItem
														icon={Copy01Icon}
														onSelect={() => copyLink(d.deployment_number)}
													>
														{l.translate('detail.row.copy')}
													</RowMenuItem>
												</RowMenuGroup>
											</RowMenu>
										</Table.Cell>
									</Table.Row>
								{/each}
							</Table.Body>
						</Table.Root>
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
