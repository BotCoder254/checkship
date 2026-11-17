<!--
	Target overview (DESIGN.md §11.8): its state (and, when failed, the error with what to check and
	Reconfigure), the connection and exposure as fact cards, then the applications that deploy to it.
	The header is shared with Settings (target-header.svelte). Polls the targets list, so a
	reconfiguration shows up without a reload.
-->
<script lang="ts">
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import Globe02Icon from '@hugeicons/core-free-icons/Globe02Icon';
	import Key01Icon from '@hugeicons/core-free-icons/Key01Icon';
	import Loading03Icon from '@hugeicons/core-free-icons/Loading03Icon';
	import RefreshIcon from '@hugeicons/core-free-icons/RefreshIcon';
	import Settings02Icon from '@hugeicons/core-free-icons/Settings02Icon';
	import CommandLineIcon from '@hugeicons/core-free-icons/CommandLineIcon';
	import DashboardSquare01Icon from '@hugeicons/core-free-icons/DashboardSquare01Icon';
	import ViewIcon from '@hugeicons/core-free-icons/ViewIcon';
	import EnvironmentChip from '#lib/components/environment-chip.svelte';
	import Icon from '#lib/components/icon.svelte';
	import Notice from '#lib/components/notice.svelte';
	import SectionHeading from '#lib/components/section-heading.svelte';
	import RowMenuGroup from '#lib/components/table/row-menu-group.svelte';
	import RowMenuItem from '#lib/components/table/row-menu-item.svelte';
	import RowMenu from '#lib/components/table/row-menu.svelte';
	import TimelineList from '#lib/components/timeline/timeline-list.svelte';
	import TimelineRow from '#lib/components/timeline/timeline-row.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import appsService from '#lib/resources/apps.ts';
	import service, { TargetStatus } from '#lib/resources/targets.ts';
	import { deploymentTile } from '#lib/deployment-row.ts';
	import { deploymentBadge, deploymentMoment } from '#lib/status.ts';
	import { DOCKER_DEFAULT_PORT, DOCKER_DEFAULT_USER, isLocal, usesOf } from '#lib/target.ts';
	import TargetHeader from './target-header.svelte';

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

	let reconfiguring = $state(false);
	let header = $state<ReturnType<typeof TargetHeader>>();

	const urlHost = $derived.by(() => {
		try {
			return target.url ? new URL(target.url) : undefined;
		} catch {
			return undefined;
		}
	});
</script>

<TargetHeader bind:this={header} bind:reconfiguring {target} tab="overview" />

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
			<div class="flex min-w-0 flex-col gap-2 text-label/5">
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
			<Button
				class="max-sm:col-span-2"
				loading={reconfiguring}
				onclick={() => header?.reconfigure()}
			>
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

	<div class="grid gap-10 lg:grid-cols-2">
		<section class="flex flex-col gap-3" aria-labelledby="connection-title">
			<SectionHeading id="connection-title">
				<Icon icon={Key01Icon} class="size-4" />
				{l.translate('targets.connection')}
			</SectionHeading>
			<dl class="grid grid-cols-[120px_minmax(0,1fr)] gap-x-3 gap-y-2 text-label/5">
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
				<p class="text-xs text-muted-foreground">{l.translate('targets.host_fixed')}</p>
			{/if}
		</section>

		<section class="flex flex-col gap-3" aria-labelledby="exposure-title">
			<SectionHeading id="exposure-title">
				<Icon icon={Globe02Icon} class="size-4" />
				{l.translate('targets.exposure')}
			</SectionHeading>
			<dl class="grid grid-cols-[120px_minmax(0,1fr)] gap-x-3 gap-y-2 text-label/5">
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
		<SectionHeading id="apps-title">
			{l.translate('targets.apps')}
			{#snippet aside()}
				{#if uses && usedApps && uses.length > 0}
					{l.translate('targets.apps.count', [usedApps.length, uses.length])}
				{/if}
			{/snippet}
		</SectionHeading>
		{#if usedApps && usedApps.length > 0}
			<TimelineList labelledby="apps-title">
				{#each usedApps as app (app.id)}
					{@const environments = (['production', 'staging'] as const).filter(
						(e) => app[`${e}_target`]?.id === target.id
					)}
					{@const latest = environments
						.map((e) => app.latest_deployments[e])
						.filter((d) => !!d)
						.sort((a, b) => b.requested_at.localeCompare(a.requested_at))[0]}
					{@const tile = latest
						? deploymentTile(latest.state)
						: { icon: DashboardSquare01Icon, tone: 'muted' as const, spin: false }}
					{@const status = latest ? deploymentBadge(latest.state) : undefined}
					{@const latestText = latest
						? `${status!.label} · #${latest.deployment_number}`
						: l.translate('apps.no_deployment')}
					<TimelineRow
						icon={tile.icon}
						tone={tile.tone}
						spin={tile.spin}
						label={[
							app.name,
							...environments.map((e) => l.translate(`deploy.environment.${e}`)),
							latestText,
							latest ? l.relative(deploymentMoment(latest)) : undefined
						]
							.filter(Boolean)
							.join(', ')}
						href={routes.app(app.id)}
					>
						{#snippet title()}{app.name}{/snippet}
						{#snippet badge()}
							{#each environments as e (e)}
								<EnvironmentChip environment={e} />
							{/each}
						{/snippet}
						{#snippet details()}
							{[latestText, latest ? l.relative(deploymentMoment(latest)) : undefined]
								.filter(Boolean)
								.join(' · ')}
						{/snippet}
						{#snippet actions()}
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
						{/snippet}
					</TimelineRow>
				{/each}
			</TimelineList>
		{:else if usedApps}
			<Notice>
				<p>
					{target.state.last_ready_version || targetStatus !== TargetStatus.Failed
						? l.translate('targets.apps.none', [target.name])
						: l.translate('targets.apps.none_never_ready', [target.name])}
				</p>
			</Notice>
		{/if}
	</section>
</div>
