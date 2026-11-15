<!--
	Target overview (DESIGN.md §11.8): its state (and, when failed, the error with what to check and
	Reconfigure), the connection and exposure as fact cards, then the applications that deploy to it.
	Polls the targets list, so a reconfiguration shows up without a reload.
-->
<script lang="ts">
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import Globe02Icon from '@hugeicons/core-free-icons/Globe02Icon';
	import InformationCircleIcon from '@hugeicons/core-free-icons/InformationCircleIcon';
	import Key01Icon from '@hugeicons/core-free-icons/Key01Icon';
	import Loading03Icon from '@hugeicons/core-free-icons/Loading03Icon';
	import RefreshIcon from '@hugeicons/core-free-icons/RefreshIcon';
	import Settings02Icon from '@hugeicons/core-free-icons/Settings02Icon';
	import { toast } from 'svelte-sonner';
	import CheckmarkCircle02Icon from '@hugeicons/core-free-icons/CheckmarkCircle02Icon';
	import Clock01Icon from '@hugeicons/core-free-icons/Clock01Icon';
	import CommandLineIcon from '@hugeicons/core-free-icons/CommandLineIcon';
	import Layers01Icon from '@hugeicons/core-free-icons/Layers01Icon';
	import TextIcon from '@hugeicons/core-free-icons/TextIcon';
	import ViewIcon from '@hugeicons/core-free-icons/ViewIcon';
	import EnvironmentChip from '#lib/components/environment-chip.svelte';
	import Icon from '#lib/components/icon.svelte';
	import RowMenuGroup from '#lib/components/table/row-menu-group.svelte';
	import RowMenuItem from '#lib/components/table/row-menu-item.svelte';
	import RowMenu from '#lib/components/table/row-menu.svelte';
	import * as Table from '#lib/components/ui/table/index.js';
	import { ACTIONS_CELL, ACTIONS_HEAD, TableColumns } from '#lib/table-columns.svelte.ts';
	import PageHeader from '#lib/components/page-header.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import appsService from '#lib/resources/apps.ts';
	import service, { TargetStatus } from '#lib/resources/targets.ts';
	import { deploymentBadge, deploymentMoment } from '#lib/status.ts';
	import {
		DOCKER_DEFAULT_PORT,
		DOCKER_DEFAULT_USER,
		connectionLabel,
		isLocal,
		usesOf
	} from '#lib/target.ts';
	import { cn } from '#lib/utils.js';
	import TargetTabs from '../target-tabs.svelte';

	let { data } = $props();

	const { data: targets } = service.queryAll();
	const { data: apps } = appsService.queryAll();

	const target = $derived($targets?.find((t) => t.id === data.target.id) ?? data.target);
	const local = $derived(isLocal(target));
	const deleting = $derived(!!target.cleanup_requested_at);
	const targetStatus = $derived(target.state.status);
	const uses = $derived($apps ? usesOf($apps, target.id) : undefined);
	const usedApps = $derived(
		uses ? [...new Map(uses.map((u) => [u.app.id, u.app])).values()] : undefined
	);

	const appColumns = new TableColumns([
		{ id: 'name', label: l.translate('column.name'), icon: TextIcon, width: 'w-[26%]' },
		{
			id: 'environment',
			label: l.translate('column.environment'),
			icon: Layers01Icon,
			width: 'w-[26%]',
			priority: 2
		},
		{
			id: 'latest',
			label: l.translate('column.latest'),
			icon: CheckmarkCircle02Icon,
			width: 'w-[28%]'
		},
		{
			id: 'started',
			label: l.translate('column.started'),
			icon: Clock01Icon,
			width: 'w-[20%]',
			priority: 3
		}
	]);

	let reconfiguring = $state(false);

	async function reconfigure() {
		reconfiguring = true;
		try {
			await service.reconfigure(target.id);
			await service.fetchAll(undefined, { cache: 'no-store' });
		} catch (ex) {
			toast.error(ex instanceof Error ? ex.message : l.translate('unexpected_error'));
		} finally {
			reconfiguring = false;
		}
	}

	const urlHost = $derived.by(() => {
		try {
			return target.url ? new URL(target.url) : undefined;
		} catch {
			return undefined;
		}
	});
</script>

<PageHeader
	title={target.name}
	crumbs={[{ href: routes.targets, label: l.translate('breadcrumb.targets') }]}
>
	{#snippet actions()}
		{#if !deleting}
			<Button
				variant="outline"
				loading={reconfiguring}
				disabled={targetStatus === TargetStatus.Configuring}
				onclick={reconfigure}
			>
				<Icon icon={RefreshIcon} />
				{l.translate('target.reconfigure')}
			</Button>
			<Button variant="outline" href={routes.editTarget(target.id)}>
				<Icon icon={Settings02Icon} />
				{l.translate('app.settings')}
			</Button>
		{/if}
	{/snippet}
	{#snippet status()}
		<span class="flex items-center gap-2 text-ink">
			{#if deleting}
				<StatusBadge state="deleting" label={l.translate('targets.state.deleting')} />
			{:else if targetStatus === TargetStatus.Configuring}
				<Icon icon={Loading03Icon} class="size-3.5 animate-spin text-warning" />
				{l.translate('targets.state.configuring')}
			{:else if targetStatus === TargetStatus.Failed}
				<span class="size-2 rounded-full bg-danger"></span>
				{l.translate('targets.state.failed')}
			{:else}
				<span class="size-2 rounded-full bg-success"></span>
				{target.state.last_ready_version
					? l.translate('targets.state.ready_at', [l.datetime(target.state.last_ready_version)])
					: l.translate('status.ready')}
			{/if}
		</span>
		<span class={cn(!local && 'font-mono text-[13px]')}>{connectionLabel(target)}</span>
		{#if targetStatus === TargetStatus.Failed && !target.state.last_ready_version}
			<span>{l.translate('targets.never_ready')}</span>
		{/if}
		<span>{l.translate('app.created', [l.date(target.created_at), target.created_by.email])}</span>
	{/snippet}
</PageHeader>

<div class="-mt-3 mb-7"><TargetTabs targetId={target.id} active="overview" /></div>

<div class="flex flex-col gap-8">
	{#if deleting}
		<Alert.Root variant="destructive">
			<Icon icon={Loading03Icon} class="animate-spin" />
			<Alert.Title class="font-semibold">{l.translate('targets.deleting.title')}</Alert.Title>
			<Alert.Description>{l.translate('targets.deleting.description')}</Alert.Description>
		</Alert.Root>
	{:else if targetStatus === TargetStatus.Failed}
		<div
			role="alert"
			class="grid grid-cols-[20px_minmax(0,1fr)] items-start gap-3 rounded-lg bg-danger-soft p-4 sm:grid-cols-[20px_minmax(0,1fr)_auto]"
		>
			<Icon icon={Alert02Icon} class="size-5 text-danger" />
			<div class="flex min-w-0 flex-col gap-2 text-[13px]/5">
				<strong class="text-sm font-semibold">{l.translate('targets.failed.title')}</strong>
				<span>{l.translate('targets.failed.consequence')}</span>
				{#if target.state.error_code}
					<pre
						aria-label={l.translate('targets.failed.error')}
						class="max-h-28 overflow-auto rounded-md bg-sunken px-3 py-2 font-mono text-xs/[18px] break-words whitespace-pre-wrap text-ink">{target
							.state.error_code}</pre>
				{/if}
				<ul class="list-disc pl-4.5">
					{#if local}
						<li>{l.translate('targets.check.local_docker')}</li>
					{:else}
						<li>
							{l.translate('targets.check.reach', [
								target.provider.data.host ?? '',
								target.provider.data.port || DOCKER_DEFAULT_PORT
							])}
						</li>
						<li>
							{l.translate('targets.check.key', [target.provider.data.user || DOCKER_DEFAULT_USER])}
						</li>
						<li>
							{l.translate('targets.check.docker', [
								target.provider.data.user || DOCKER_DEFAULT_USER
							])}
						</li>
					{/if}
				</ul>
			</div>
			<Button class="max-sm:col-span-2" loading={reconfiguring} onclick={reconfigure}>
				<Icon icon={RefreshIcon} />
				{l.translate('target.reconfigure')}
			</Button>
		</div>
	{:else if targetStatus === TargetStatus.Configuring}
		<Alert.Root variant="warning">
			<Icon icon={Loading03Icon} class="animate-spin" />
			<Alert.Title class="font-semibold">{l.translate('target.configuring')}</Alert.Title>
			<Alert.Description>{l.translate('targets.configuring.description')}</Alert.Description>
		</Alert.Root>
	{/if}

	<div class="grid gap-4 lg:grid-cols-2">
		<section
			class="flex flex-col gap-3.5 rounded-lg bg-sunken p-5"
			aria-labelledby="connection-title"
		>
			<div class="flex items-center gap-2.5">
				<span
					class="flex size-9 items-center justify-center rounded-lg bg-raised text-muted-foreground"
				>
					<Icon icon={Key01Icon} class="size-4.5" />
				</span>
				<h2 id="connection-title" class="text-base font-semibold">
					{l.translate('targets.connection')}
				</h2>
			</div>
			<dl class="grid grid-cols-[120px_minmax(0,1fr)] gap-x-3 gap-y-2 text-[13px]/5">
				<dt class="text-muted-foreground">{l.translate('targets.provider')}</dt>
				<dd>
					{local ? l.translate('targets.provider.local') : l.translate('targets.provider.ssh')}
				</dd>
				{#if !local}
					<dt class="text-muted-foreground">{l.translate('target.docker.host')}</dt>
					<dd class="truncate font-mono">{target.provider.data.host}</dd>
					<dt class="text-muted-foreground">{l.translate('target.docker.user')}</dt>
					<dd class="font-mono">{target.provider.data.user || DOCKER_DEFAULT_USER}</dd>
					<dt class="text-muted-foreground">{l.translate('target.docker.port')}</dt>
					<dd class="font-mono">{target.provider.data.port || DOCKER_DEFAULT_PORT}</dd>
					<dt class="text-muted-foreground">{l.translate('target.docker.private_key')}</dt>
					<dd>
						{target.provider.data.private_key
							? l.translate('targets.key.saved')
							: l.translate('targets.key.none')}
					</dd>
				{/if}
			</dl>
			{#if !local}
				<p class="text-xs text-faint">{l.translate('targets.host_fixed')}</p>
			{/if}
		</section>

		<section
			class="flex flex-col gap-3.5 rounded-lg bg-sunken p-5"
			aria-labelledby="exposure-title"
		>
			<div class="flex items-center gap-2.5">
				<span
					class="flex size-9 items-center justify-center rounded-lg bg-raised text-muted-foreground"
				>
					<Icon icon={Globe02Icon} class="size-4.5" />
				</span>
				<h2 id="exposure-title" class="text-base font-semibold">
					{l.translate('targets.exposure')}
				</h2>
			</div>
			<dl class="grid grid-cols-[120px_minmax(0,1fr)] gap-x-3 gap-y-2 text-[13px]/5">
				<dt class="text-muted-foreground">{l.translate('targets.proxy')}</dt>
				<dd>
					{target.url
						? l.translate('targets.proxy.automatic')
						: l.translate('targets.proxy.manual')}
				</dd>
				{#if target.url && urlHost}
					<dt class="text-muted-foreground">{l.translate('targets.root_url')}</dt>
					<dd class="truncate">
						<a
							href={target.url}
							target="_blank"
							rel="noopener noreferrer"
							class="font-mono text-primary hover:underline">{target.url}</a
						>
					</dd>
					<dt class="text-muted-foreground">{l.translate('targets.apps_get')}</dt>
					<dd class="truncate font-mono text-muted-foreground">
						{urlHost.protocol}//&lt;app&gt;.{urlHost.host}
					</dd>
					<dt class="text-muted-foreground">DNS</dt>
					<dd class="truncate">
						<span class="font-mono">*.{urlHost.hostname}</span>
						→ {l.translate('targets.dns.this_server')}
					</dd>
				{:else}
					<dt class="text-muted-foreground">{l.translate('targets.apps_get')}</dt>
					<dd>{l.translate('targets.no_url')}</dd>
				{/if}
			</dl>
		</section>
	</div>

	<section class="flex flex-col gap-3" aria-labelledby="apps-title">
		<div class="flex flex-wrap items-baseline justify-between gap-x-3">
			<h2 id="apps-title" class="text-base font-semibold">{l.translate('targets.apps')}</h2>
			{#if uses && usedApps && uses.length > 0}
				<p class="text-[13px] text-muted-foreground">
					{l.translate('targets.apps.count', [usedApps.length, uses.length])}
				</p>
			{/if}
		</div>
		{#if usedApps && usedApps.length > 0}
			<Table.Root aria-labelledby="apps-title">
				<Table.Header>
					<Table.Row>
						{#each appColumns.shown as column (column.id)}
							<Table.Head class={appColumns.head(column.id)}>{column.label}</Table.Head>
						{/each}
						<Table.Head class={ACTIONS_HEAD}>{l.translate('column.actions')}</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each usedApps as app (app.id)}
						{@const environments = (['production', 'staging'] as const).filter(
							(e) => app[`${e}_target`]?.id === target.id
						)}
						{@const latest = environments
							.map((e) => app.latest_deployments[e])
							.filter((d) => !!d)
							.sort((a, b) => b.requested_at.localeCompare(a.requested_at))[0]}
						<Table.Row class="relative">
							<Table.Cell class={appColumns.cell('name')}>
								<!-- The whole row opens the application; the link stretches over it. -->
								<a
									href={routes.app(app.id)}
									title={app.name}
									class="text-sm font-semibold outline-none after:absolute after:inset-0 after:rounded-md focus-visible:after:ring-2 focus-visible:after:ring-ring focus-visible:after:ring-inset"
									>{app.name}</a
								>
							</Table.Cell>
							<Table.Cell class={appColumns.cell('environment')}>
								<span class="flex gap-1.5">
									{#each environments as e (e)}
										<EnvironmentChip environment={e} />
									{/each}
								</span>
							</Table.Cell>
							<Table.Cell class={appColumns.cell('latest')}>
								{#if latest}
									{@const badge = deploymentBadge(latest.state)}
									<StatusBadge
										state={badge.state}
										label="{badge.label} · #{latest.deployment_number}"
									/>
								{:else}
									<span class="text-muted-foreground">{l.translate('apps.no_deployment')}</span>
								{/if}
							</Table.Cell>
							<Table.Cell class={appColumns.cell('started', 'text-muted-foreground')}>
								{latest ? l.relative(deploymentMoment(latest)) : '—'}
							</Table.Cell>
							<Table.Cell class={ACTIONS_CELL}>
								<RowMenu label={l.translate('table.menu', [app.name])}>
									<RowMenuGroup>
										<RowMenuItem icon={ViewIcon} href={routes.app(app.id)}>
											{l.translate('menu.open')}
										</RowMenuItem>
										<RowMenuItem icon={CommandLineIcon} href={routes.appLogs(app.id)}>
											{l.translate('menu.logs')}
										</RowMenuItem>
										<RowMenuItem icon={Settings02Icon} href={routes.editApp(app.id)}>
											{l.translate('menu.settings')}
										</RowMenuItem>
									</RowMenuGroup>
								</RowMenu>
							</Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		{:else if usedApps}
			<div
				class="flex items-center gap-3 rounded-md bg-sunken p-4 text-[13px] text-muted-foreground"
			>
				<Icon icon={InformationCircleIcon} class="size-4.5 shrink-0" />
				{target.state.last_ready_version || targetStatus !== TargetStatus.Failed
					? l.translate('targets.apps.none', [target.name])
					: l.translate('targets.apps.none_never_ready', [target.name])}
			</div>
		{/if}
	</section>
</div>
