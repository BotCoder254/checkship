<!--
	Deployment preview (DESIGN.md §11.6): opens from a history row or the latest deployment, 680px.
	State, the facts, and the last log lines; the full page is one click away. Polls while the
	deployment runs.
-->
<script lang="ts">
	import ArrowUpRight01Icon from '@hugeicons/core-free-icons/ArrowUpRight01Icon';
	import RepeatIcon from '@hugeicons/core-free-icons/RepeatIcon';
	import CodeBlock from '#lib/components/code-block.svelte';
	import Drawer from '#lib/components/drawer.svelte';
	import Icon from '#lib/components/icon.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { submitter } from '#lib/form.ts';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import type { AppDetail } from '#lib/resources/apps.ts';
	import service from '#lib/resources/deployments.ts';
	import { deploymentBadge, isCancelled } from '#lib/status.ts';

	type Props = { app: AppDetail; number: number; open: boolean };

	let { app, number = $bindable(), open = $bindable(false) }: Props = $props();

	const TAIL = 14;

	let poll = $state(true);
	const deploymentQuery = $derived(service.queryByAppAndNumber(app.id, number, poll));
	const logsQuery = $derived(service.queryLogs(app.id, number, poll));
	const deploymentStore = $derived(deploymentQuery.data);
	const logsStore = $derived(logsQuery.data);
	const deployment = $derived($deploymentStore);
	const logsErrorStore = $derived(logsQuery.error);
	const logsFailed = $derived(!!$logsErrorStore);
	const tail = $derived(($logsStore ?? '').trimEnd().split('\n').slice(-TAIL).join('\n'));

	$effect(() => {
		void number;
		poll = true;
	});
	$effect(() => {
		if (poll && deployment?.state.finished_at) poll = false;
	});

	const badge = $derived(deployment ? deploymentBadge(deployment.state) : undefined);

	const { submit: redeploy, loading: redeploying } = submitter(() =>
		service.redeploy(app.id, number).then((d) => (number = d.deployment_number))
	);
</script>

<Drawer
	bind:open
	size="preview"
	title={l.translate('breadcrumb.deployment.name', [number])}
	description="{app.name} · {deployment?.environment ?? ''}"
>
	{#if deployment && badge}
		<div class="flex flex-wrap items-center gap-2.5">
			<StatusBadge state={badge.state} label={badge.label} />
			<span class="text-label text-muted-foreground">
				{#if deployment.state.started_at && deployment.state.finished_at}
					{l.relative(deployment.state.finished_at)} · {l.translate('preview.took', [
						l.duration(deployment.state.started_at, deployment.state.finished_at)
					])}
				{:else if deployment.state.started_at}
					{l.translate('preview.started', [l.relative(deployment.state.started_at)])}
				{:else}
					{l.translate('preview.queued', [l.relative(deployment.requested_at)])}
				{/if}
			</span>
		</div>

		<dl
			class="grid grid-cols-[110px_minmax(0,1fr)] gap-x-4 gap-y-2.5 track p-4 text-label/5 [&_dt]:first-letter:uppercase"
		>
			<dt class="text-muted-foreground">{l.translate('deployment.environment')}</dt>
			<dd>
				{deployment.environment} · {deployment.target.name ??
					l.translate('deployment.target.deleted')}
			</dd>
			<dt class="text-muted-foreground">{l.translate('deployment.source')}</dt>
			<dd class="min-w-0 truncate">
				{l.translate(`deployment.source.${deployment.source.discriminator}`)}
				{#if deployment.source.discriminator === 'git'}
					· {deployment.source.data.branch}
					{#if deployment.source.data.hash}
						· <span class="font-mono">{deployment.source.data.hash.slice(0, 7)}</span>
					{/if}
				{/if}
			</dd>
			<dt class="text-muted-foreground">{l.translate('deployment.queued_by')}</dt>
			<dd class="truncate">{deployment.requested_by.email}</dd>
			<dt class="text-muted-foreground">{l.translate('jobs.queued')}</dt>
			<dd>{l.datetime(deployment.requested_at)}</dd>
			{#if deployment.state.finished_at}
				<dt class="text-muted-foreground">{l.translate('preview.finished')}</dt>
				<dd>{l.datetime(deployment.state.finished_at)}</dd>
			{/if}
			{#if deployment.state.error_code && !isCancelled(deployment.state)}
				<dt class="text-muted-foreground">{l.translate('deployment.error_code')}</dt>
				<dd class="font-mono break-all text-danger">{deployment.state.error_code}</dd>
			{/if}
		</dl>

		<section class="flex flex-col gap-2.5">
			<div class="flex items-center justify-between gap-3">
				<h3 class="text-xs font-semibold tracking-wide text-muted-foreground">
					{l.translate('preview.last_lines')}
				</h3>
				<a
					href={routes.deployment(app.id, number)}
					class="flex items-center gap-1 rounded-sm text-label font-semibold text-primary outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring"
				>
					{l.translate('preview.full_log')}
					<Icon icon={ArrowUpRight01Icon} class="size-3.5" />
				</a>
			</div>
			{#if logsFailed}
				<p class="text-sm text-danger" role="status">{l.translate('deployment.logs.failed')}</p>
			{/if}
			<CodeBlock
				code={tail || l.translate('deployment.waiting_for_logs')}
				lang={tail ? 'log' : 'text'}
				label={l.translate('preview.last_lines')}
				class="text-xs/5"
			/>
		</section>
	{:else}
		<p class="text-sm text-muted-foreground">{l.translate('deployment.waiting_for_logs')}</p>
	{/if}

	{#snippet footer()}
		<Button variant="outline" href={routes.deployment(app.id, number)}>
			<Icon icon={ArrowUpRight01Icon} />
			{l.translate('preview.open_page')}
		</Button>
		{#if !app.cleanup_requested_at}
			<Button loading={$redeploying} onclick={() => redeploy().catch(() => {})}>
				<Icon icon={RepeatIcon} />
				{l.translate('deployment.redeploy')}
			</Button>
		{/if}
	{/snippet}
</Drawer>
