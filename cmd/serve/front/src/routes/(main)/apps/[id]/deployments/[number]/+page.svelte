<!--
	Deployment (DESIGN.md §11.7 Deployment): breadcrumb title, state in the status line, Promote and
	Redeploy behind a confirmation dialog, then what happened in order (a timeline: queued, started,
	finished, with the wait and the run time), the error first when it failed, then Source, Services
	and Logs as sections (headings and
	space, no lines). Polls the details while the deployment runs or its target is being configured;
	the log is streamed live (#lib/deployment-logs.svelte.ts) and only polled when the stream keeps
	failing. Cancel (seelf#7) shows while it is pending or running.
-->
<script lang="ts">
	import { copyText } from '#lib/clipboard.ts';
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import Copy01Icon from '@hugeicons/core-free-icons/Copy01Icon';
	import RefreshIcon from '@hugeicons/core-free-icons/RefreshIcon';
	import Rocket01Icon from '@hugeicons/core-free-icons/Rocket01Icon';
	import StopCircleIcon from '@hugeicons/core-free-icons/StopCircleIcon';
	import { toast } from 'svelte-sonner';
	import { goto } from '$app/navigation';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import Icon from '#lib/components/icon.svelte';
	import Notice from '#lib/components/notice.svelte';
	import DeploymentTimeline from './deployment-timeline.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service, { DeploymentStatus } from '#lib/resources/deployments.ts';
	import { TargetStatus } from '#lib/resources/targets.ts';
	import { compactProgress } from '#lib/deployment-logs.ts';
	import { DeploymentLogStream } from '#lib/deployment-logs.svelte.ts';
	import { deploymentBadge, isCancellable, isCancelled } from '#lib/status.ts';

	let { data } = $props();

	const app = $derived(data.app);
	const number = $derived(data.deployment.deployment_number);
	const latestNumber = $derived(
		app.latest_deployments[data.deployment.environment]?.deployment_number
	);
	const isStale = $derived(latestNumber !== undefined && latestNumber !== number);

	// Recomputed for each deployment shown (promote and redeploy navigate to a new one on this
	// page); set to false below once it has finished.
	let poll = $derived(
		!data.deployment.state.finished_at || data.deployment.target.status === TargetStatus.Configuring
	);

	const deploymentQuery = $derived(service.queryByAppAndNumber(app.id, number, poll));
	const deploymentStore = $derived(deploymentQuery.data);
	const deployment = $derived($deploymentStore ?? data.deployment);

	// The log, live. One stream per deployment shown (promote and redeploy navigate to another).
	let stream = $state<DeploymentLogStream>();

	$effect(() => {
		const live = new DeploymentLogStream(service.logsStreamUrl(app.id, number));
		stream = live;
		live.start();
		return () => live.stop();
	});

	// The stream keeps failing (a proxy buffering it, for example): poll the whole log as before.
	const fallback = $derived(stream?.status === 'fallback');
	const logsQuery = $derived(fallback ? service.queryLogs(app.id, number, poll) : undefined);
	const logsStore = $derived(logsQuery?.data);
	const logsErrorStore = $derived(logsQuery?.error);
	const logs = $derived(fallback ? $logsStore : stream?.text);
	const logsFailed = $derived(fallback && !!$logsErrorStore);
	// Downloads collapse to one line per layer; Copy still copies everything.
	const shownLogs = $derived(compactProgress(logs ?? ''));
	const logsLive = $derived(
		!deployment.state.finished_at && (stream?.status === 'live' || stream?.status === 'connecting')
	);
	const logsReconnecting = $derived(
		!deployment.state.finished_at && stream?.status === 'reconnecting'
	);

	// Stop polling once this deployment has finished and its target is no longer configuring.
	$effect(() => {
		if (
			poll &&
			deployment.state.finished_at &&
			deployment.target.status !== TargetStatus.Configuring
		) {
			poll = false;
		}
	});

	let promoteOpen = $state(false);
	let redeployOpen = $state(false);
	let cancelOpen = $state(false);
	// The deployment a cancel was asked for: its button says "Cancelling…" until it has ended.
	let cancelRequestedFor = $state<number>();

	const cancelled = $derived(isCancelled(deployment.state));
	const cancellable = $derived(!app.cleanup_requested_at && isCancellable(deployment.state));
	const cancelling = $derived(cancelRequestedFor === number && cancellable);
	const running = $derived(deployment.state.status === DeploymentStatus.Running);

	async function cancelDeployment() {
		const result = await service.cancel(app.id, number);
		if (isCancelled(result.state)) {
			toast.success(l.translate('deployment.cancel.done', [number]));
		} else {
			cancelRequestedFor = number;
			toast.success(l.translate('deployment.cancel.requested', [number]));
		}
	}
	let logsElement = $state<HTMLPreElement>();

	// Keep the newest lines in view while the deployment runs, unless the reader scrolled up.
	let followLogs = true;

	function onLogsScroll() {
		if (!logsElement) return;
		followLogs = logsElement.scrollHeight - logsElement.scrollTop - logsElement.clientHeight < 24;
	}

	$effect(() => {
		void shownLogs;
		// Opened while it ran: keep following until the last lines, which may arrive after the end.
		if ((poll || !data.deployment.state.finished_at) && followLogs && logsElement)
			logsElement.scrollTop = logsElement.scrollHeight;
	});

	const badge = $derived(deploymentBadge(deployment.state));
	const services = $derived(deployment.state.services ?? []);

	function timing() {
		const { started_at, finished_at } = deployment.state;
		if (cancelled && finished_at) {
			return started_at
				? l.translate('deployment.cancelled_ran_line', [
						l.time(finished_at),
						l.duration(started_at, finished_at)
					])
				: l.translate('deployment.cancelled_line', [l.time(finished_at)]);
		}
		if (started_at && finished_at) {
			return l.translate('deployment.finished_line', [
				l.time(finished_at),
				l.duration(started_at, finished_at)
			]);
		}
		if (started_at) return l.translate('deployment.started_line', [l.time(started_at)]);
		return l.translate('deployment.queued_line', [l.time(deployment.requested_at)]);
	}

	async function copyLogs() {
		if (await copyText(logs ?? '')) toast.success(l.translate('copied'));
		else toast.error(l.translate('copy_blocked'));
	}
</script>

<PageHeader
	title={l.translate('breadcrumb.deployment.name', [number])}
	crumbs={[
		{ href: routes.apps, label: l.translate('breadcrumb.applications') },
		{ href: routes.app(app.id), label: app.name }
	]}
>
	{#snippet actions()}
		{#if cancellable}
			<Button
				variant="outline"
				loading={cancelling}
				onclick={() => {
					if (!cancelling) cancelOpen = true;
				}}
			>
				<Icon icon={StopCircleIcon} />
				{l.translate(cancelling ? 'deployment.cancelling' : 'deployment.cancel')}
			</Button>
		{/if}
		{#if !app.cleanup_requested_at}
			{#if deployment.environment !== 'production'}
				<Button variant="outline" onclick={() => (promoteOpen = true)}>
					<Icon icon={Rocket01Icon} />
					{l.translate('deployment.promote.title')}
				</Button>
			{/if}
			<Button variant="outline" onclick={() => (redeployOpen = true)}>
				<Icon icon={RefreshIcon} />
				{l.translate('deployment.redeploy')}
			</Button>
		{/if}
	{/snippet}
	{#snippet status()}
		<StatusBadge state={badge.state} label={badge.label} />
		<span>
			{l.translate('deployment.status_line', [
				deployment.environment,
				deployment.target.name ?? l.translate('deployment.target.deleted')
			])} · {timing()}
		</span>
	{/snippet}
</PageHeader>

<div class="flex flex-col gap-9">
	{#if app.cleanup_requested_at}
		<Alert.Root variant="destructive">
			<Icon icon={Alert02Icon} />
			<Alert.Title class="font-semibold">{l.translate('cleanup_requested')}</Alert.Title>
		</Alert.Root>
	{:else if isStale && latestNumber !== undefined}
		<Alert.Root variant="warning">
			<Icon icon={Alert02Icon} />
			<Alert.Title class="font-semibold">{l.translate('deployment.outdated')}</Alert.Title>
			<Alert.Description>
				<a href={routes.deployment(app.id, latestNumber)} class="font-semibold underline">
					{l.translate('deployment.outdated.link', [latestNumber])}
				</a>
			</Alert.Description>
		</Alert.Root>
	{/if}

	<DeploymentTimeline requestedAt={deployment.requested_at} state={deployment.state} />

	{#if cancelled}
		<p class="text-sm text-muted-foreground">
			{l.translate(
				deployment.state.started_at ? 'deployment.cancelled.ran' : 'deployment.cancelled.queued'
			)}
		</p>
	{:else if deployment.state.error_code}
		<Notice tone="danger" role="alert">
			<strong class="font-mono font-semibold">{deployment.state.error_code}</strong>
			<p class="text-muted-foreground">{l.translate('deployment.failed.help')}</p>
		</Notice>
	{/if}

	<div class="grid gap-9 md:grid-cols-2">
		<section class="flex flex-col gap-3">
			<h2 class="text-xs font-semibold tracking-wide text-muted-foreground">
				{l.translate('deployment.source')}
			</h2>
			<dl class="grid grid-cols-[120px_minmax(0,1fr)] gap-x-3 gap-y-2 track p-4 text-sm sm:p-5">
				<dt class="text-muted-foreground">{l.translate('deployment.source.kind')}</dt>
				<dd>{l.translate(`deployment.source.${deployment.source.discriminator}`)}</dd>
				{#if deployment.source.discriminator === 'git'}
					<dt class="text-muted-foreground">{l.translate('deployment.payload.vcs.branch')}</dt>
					<dd>{deployment.source.data.branch}</dd>
					<dt class="text-muted-foreground">{l.translate('deployment.payload.vcs.commit')}</dt>
					<dd class="font-mono text-label">{deployment.source.data.hash.slice(0, 7)}</dd>
				{/if}
				<dt class="text-muted-foreground">{l.translate('deployment.queued_by')}</dt>
				<dd class="truncate">{deployment.requested_by.email}</dd>
			</dl>
		</section>

		<section class="flex flex-col gap-3">
			<h2 class="text-xs font-semibold tracking-wide text-muted-foreground">
				{l.translate('deployment.services.title')}
			</h2>
			<div class="min-w-0 track p-4 sm:p-5">
				{#if services.length > 0}
					<ul class="flex flex-col gap-1.5">
						{#each services as service (service.name)}
							{@const entrypointUrls = service.entrypoints.filter((e) => e.url).map((e) => e.url!)}
							<!-- Custom domains first (seelf#5), then the default address and other ports -->
							{@const urls = [...new Set([...(service.urls ?? []), ...entrypointUrls])]}
							{@const defaultUrl = service.domains?.length ? service.urls?.at(-1) : undefined}
							<li class="flex flex-col gap-1 py-1.5 text-sm">
								<span class="flex items-center justify-between gap-3">
									<span class="font-semibold">{service.name}</span>
									{#if urls.length === 0}
										<span class="text-muted-foreground"
											>{l.translate('deployment.not_exposed')}</span
										>
									{/if}
								</span>
								{#each urls as url (url)}
									<a
										href={url}
										target="_blank"
										rel="noopener noreferrer"
										class="truncate font-mono text-label text-primary hover:underline"
									>
										{url}{#if url === defaultUrl}<span
												class="ml-1.5 font-sans text-xs font-semibold text-muted-foreground"
												>{l.translate('domains.url.default')}</span
											>{/if}
									</a>
								{/each}
								<span class="truncate font-mono text-xs text-muted-foreground">{service.image}</span
								>
							</li>
						{/each}
					</ul>
				{:else}
					<p class="text-sm text-muted-foreground">{l.translate('deployment.waiting_for_logs')}</p>
				{/if}
			</div>
		</section>
	</div>

	<section class="flex flex-col gap-3">
		<div class="flex items-center justify-between gap-3">
			<div class="flex items-center gap-3.5">
				<h2 class="text-xs font-semibold tracking-wide text-muted-foreground">
					{l.translate('deployment.logs.title')}
				</h2>
				{#if logsLive}
					<span class="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
						<span
							aria-hidden="true"
							class="size-2 animate-pulse rounded-full bg-success motion-reduce:animate-none"
						></span>
						{l.translate('deployment.logs.live')}
					</span>
				{:else if logsReconnecting}
					<span class="text-xs font-medium text-warning" role="status">
						{l.translate('deployment.logs.reconnecting')}
					</span>
				{/if}
			</div>
			<Button variant="outline" size="sm" onclick={copyLogs} disabled={!logs}>
				<Icon icon={Copy01Icon} />
				{l.translate('copy')}
			</Button>
		</div>
		{#if logsFailed}
			<p class="text-sm text-danger" role="status">{l.translate('deployment.logs.failed')}</p>
		{/if}
		<pre
			bind:this={logsElement}
			onscroll={onLogsScroll}
			aria-label={l.translate('deployment.logs.title')}
			class="max-h-[70vh] min-h-40 overflow-auto rounded-md bg-canvas px-4 py-3.5 font-mono text-xs/5 whitespace-pre-wrap text-ink">{shownLogs.text ||
				l.translate(
					cancelled && !deployment.state.started_at
						? 'deployment.logs.nothing_cancelled'
						: 'deployment.waiting_for_logs'
				)}</pre>
		{#if shownLogs.collapsed > 0}
			<p class="text-xs text-muted-foreground">{l.translate('deployment.logs.compacted')}</p>
		{/if}
	</section>
</div>

<ConfirmDialog
	bind:open={cancelOpen}
	title={l.translate('deployment.cancel.title', [number])}
	description={l.translate(
		running ? 'deployment.cancel.description.running' : 'deployment.cancel.description.pending',
		[number]
	)}
	confirmLabel={l.translate('deployment.cancel.confirm')}
	cancelLabel={l.translate('deployment.cancel.keep')}
	icon={StopCircleIcon}
	destructive
	onconfirm={cancelDeployment}
>
	{#snippet extra()}
		{#if running}
			<Notice tone="warning">{l.translate('deployment.cancel.warning')}</Notice>
		{/if}
	{/snippet}
</ConfirmDialog>

<ConfirmDialog
	bind:open={promoteOpen}
	title={l.translate('deployment.promote.title')}
	description={l.translate('deployment.promote.description', [number])}
	confirmLabel={l.translate('deployment.promote.title')}
	icon={Rocket01Icon}
	onconfirm={() =>
		service
			.promote(app.id, number)
			.then((d) => goto(routes.deployment(app.id, d.deployment_number)))}
/>

<ConfirmDialog
	bind:open={redeployOpen}
	title={l.translate('deployment.redeploy')}
	description={l.translate('deployment.redeploy.description', [number, deployment.environment])}
	confirmLabel={l.translate('deployment.redeploy')}
	icon={RefreshIcon}
	onconfirm={() =>
		service
			.redeploy(app.id, number)
			.then((d) => goto(routes.deployment(app.id, d.deployment_number)))}
/>
