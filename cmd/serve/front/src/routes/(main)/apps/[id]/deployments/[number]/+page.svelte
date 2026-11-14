<!--
	Deployment (DESIGN.md §5, §11.5): breadcrumb title, state in the status line, Promote and
	Redeploy behind a confirmation dialog, then Source, Services and Logs as sections (headings and
	space, no lines). Polls while the deployment runs or its target is being configured.
-->
<script lang="ts">
	import { copyText } from '#lib/clipboard.ts';
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import Copy01Icon from '@hugeicons/core-free-icons/Copy01Icon';
	import RefreshIcon from '@hugeicons/core-free-icons/RefreshIcon';
	import Rocket01Icon from '@hugeicons/core-free-icons/Rocket01Icon';
	import { toast } from 'svelte-sonner';
	import { goto } from '$app/navigation';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import Icon from '#lib/components/icon.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service from '#lib/resources/deployments.ts';
	import { TargetStatus } from '#lib/resources/targets.ts';
	import { deploymentBadge } from '#lib/status.ts';

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
	const logsQuery = $derived(service.queryLogs(app.id, number, poll));
	const deploymentStore = $derived(deploymentQuery.data);
	const logsStore = $derived(logsQuery.data);
	const deployment = $derived($deploymentStore ?? data.deployment);
	const logs = $derived($logsStore);
	const logsErrorStore = $derived(logsQuery.error);
	const logsFailed = $derived(!!$logsErrorStore);

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
	let logsElement = $state<HTMLPreElement>();

	// Keep the newest lines in view while the deployment runs, unless the reader scrolled up.
	let followLogs = true;

	function onLogsScroll() {
		if (!logsElement) return;
		followLogs = logsElement.scrollHeight - logsElement.scrollTop - logsElement.clientHeight < 24;
	}

	$effect(() => {
		void logs;
		if (poll && followLogs && logsElement) logsElement.scrollTop = logsElement.scrollHeight;
	});

	const badge = $derived(deploymentBadge(deployment.state));
	const services = $derived(deployment.state.services ?? []);

	function timing() {
		const { started_at, finished_at } = deployment.state;
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
		{#if deployment.state.error_code}
			<span class="font-mono text-xs">{deployment.state.error_code}</span>
		{/if}
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

	<div class="grid gap-9 md:grid-cols-2">
		<section class="flex flex-col gap-3">
			<h2 class="text-base font-semibold">{l.translate('deployment.source')}</h2>
			<dl class="grid grid-cols-[120px_minmax(0,1fr)] gap-x-3 gap-y-2 text-sm">
				<dt class="text-muted-foreground">{l.translate('deployment.source.kind')}</dt>
				<dd>{l.translate(`deployment.source.${deployment.source.discriminator}`)}</dd>
				{#if deployment.source.discriminator === 'git'}
					<dt class="text-muted-foreground">{l.translate('deployment.payload.vcs.branch')}</dt>
					<dd>{deployment.source.data.branch}</dd>
					<dt class="text-muted-foreground">{l.translate('deployment.payload.vcs.commit')}</dt>
					<dd class="font-mono text-[13px]">{deployment.source.data.hash.slice(0, 7)}</dd>
				{/if}
				<dt class="text-muted-foreground">{l.translate('deployment.queued_by')}</dt>
				<dd class="truncate">{deployment.requested_by.email}</dd>
			</dl>
		</section>

		<section class="flex flex-col gap-3">
			<h2 class="text-base font-semibold">{l.translate('deployment.services.title')}</h2>
			{#if services.length > 0}
				<ul class="flex flex-col gap-1.5">
					{#each services as service (service.name)}
						{@const urls = service.entrypoints.filter((e) => e.url).map((e) => e.url!)}
						<li class="flex flex-col gap-1 rounded-md bg-sunken px-3.5 py-2.5 text-sm">
							<span class="flex items-center justify-between gap-3">
								<span class="font-semibold">{service.name}</span>
								{#if urls.length === 0}
									<span class="text-muted-foreground">{l.translate('deployment.not_exposed')}</span>
								{/if}
							</span>
							{#each urls as url (url)}
								<a
									href={url}
									target="_blank"
									rel="noopener noreferrer"
									class="truncate font-mono text-[13px] text-primary hover:underline"
								>
									{url}
								</a>
							{/each}
							<span class="truncate font-mono text-xs text-muted-foreground">{service.image}</span>
						</li>
					{/each}
				</ul>
			{:else}
				<p class="text-sm text-muted-foreground">{l.translate('deployment.waiting_for_logs')}</p>
			{/if}
		</section>
	</div>

	<section class="flex flex-col gap-3">
		<div class="flex items-center justify-between gap-3">
			<h2 class="text-base font-semibold">{l.translate('deployment.logs.title')}</h2>
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
			class="max-h-[70vh] min-h-40 overflow-auto rounded-md bg-canvas px-4 py-3.5 font-mono text-xs/5 whitespace-pre-wrap text-ink">{logs ||
				l.translate('deployment.waiting_for_logs')}</pre>
	</section>
</div>

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
