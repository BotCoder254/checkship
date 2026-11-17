<!--
	Restore (canvas artboard 2): the backup it comes from, where it goes (any application that is not
	being deleted, either environment), which volumes, and whether the current data is backed up
	first (on by default). A destructive dialog names the volumes before anything is replaced.
-->
<script lang="ts">
	import DatabaseRestoreIcon from '@hugeicons/core-free-icons/DatabaseRestoreIcon';
	import DashboardSquare01Icon from '@hugeicons/core-free-icons/DashboardSquare01Icon';
	import InformationCircleIcon from '@hugeicons/core-free-icons/InformationCircleIcon';
	import { toast } from 'svelte-sonner';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import Drawer from '#lib/components/drawer.svelte';
	import Icon from '#lib/components/icon.svelte';
	import Segmented from '#lib/components/segmented.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Checkbox } from '#lib/components/ui/checkbox/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { Switch } from '#lib/components/ui/switch/index.js';
	import { appName, formatBytes, messageOf } from '#lib/backup.ts';
	import l from '#lib/localization/index.ts';
	import appsService from '#lib/resources/apps.ts';
	import service, { type Backup, type Volume } from '#lib/resources/backups.ts';
	import type { Environment } from '#lib/resources/deployments.ts';

	type Props = { open: boolean; backup?: Backup };

	let { open = $bindable(false), backup }: Props = $props();

	const { data: apps } = appsService.queryAll();
	const candidates = $derived(($apps ?? []).filter((a) => !a.cleanup_requested_at));

	let appId = $state('');
	let env = $state<Environment>('production');
	let destination = $state<Volume[]>();
	let unticked = $state<string[]>([]);
	let snapshot = $state(true);
	let confirmOpen = $state(false);

	const environments = [
		{ value: 'production' as const, label: l.translate('deploy.environment.production') },
		{ value: 'staging' as const, label: l.translate('deploy.environment.staging') }
	];

	$effect(() => {
		if (open && backup) {
			// A deleted app cannot receive a restore: start from the first one available.
			appId = backup.app.deleted ? (candidates[0]?.id ?? '') : backup.app.id;
			env = backup.environment;
			unticked = [];
			snapshot = true;
		}
	});

	// What the destination declares, to say which volumes are replaced and which are created.
	$effect(() => {
		if (!open || !appId) return;
		const [id, current] = [appId, env];
		destination = undefined;
		service
			.fetchVolumes(id, current, { cache: 'no-store' })
			.then((v) => {
				if (id === appId && current === env) destination = v.volumes;
			})
			.catch(() => {
				if (id === appId && current === env) destination = [];
			});
	});

	const keys = $derived(backup?.result?.volumes ?? []);
	const declared = (key: string) => destination?.find((v) => v.key === key);
	const selected = $derived(keys.filter((v) => !unticked.includes(v.key)).map((v) => v.key));
	const target = $derived(candidates.find((a) => a.id === appId));
	const samePlace = $derived(!!backup && backup.app.id === appId && backup.environment === env);

	function toggle(key: string, on: boolean) {
		unticked = on ? unticked.filter((k) => k !== key) : [...unticked, key];
	}

	async function restore() {
		if (!backup) return;
		await service.restore(backup.id, {
			app_id: appId,
			environment: env,
			// Every volume of the backup that is ticked; none means the ones the destination declares.
			volumes: selected,
			snapshot_first: snapshot
		});
		toast.success(l.translate('restore.queued'));
		open = false;
	}
</script>

<Drawer
	bind:open
	title={l.translate('restore.title')}
	description={l.translate('restore.description')}
>
	{#if backup}
		<dl class="grid grid-cols-[110px_minmax(0,1fr)] gap-x-3 gap-y-2 text-sm">
			<dt class="text-muted-foreground">{l.translate('backups.details.from')}</dt>
			<dd class="truncate">{appName(backup)} · {backup.environment}</dd>
			<dt class="text-muted-foreground">{l.translate('backups.details.created')}</dt>
			<dd class="truncate">{l.datetime(backup.requested_at)} · {backup.requested_by.email}</dd>
			{#if backup.result}
				<dt class="text-muted-foreground">{l.translate('backups.details.size')}</dt>
				<dd>{formatBytes(backup.result.size)}</dd>
				<dt class="text-muted-foreground">{l.translate('backups.details.checksum')}</dt>
				<dd class="truncate font-mono text-xs" title={backup.result.sha256}>
					{backup.result.sha256}
				</dd>
			{/if}
		</dl>

		<div class="grid gap-4 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-end">
			<div class="flex flex-col gap-1.5">
				<span id="restore-app-label" class="text-sm font-medium"
					>{l.translate('restore.into_app')}</span
				>
				<Select.Root type="single" bind:value={appId}>
					<Select.Trigger aria-labelledby="restore-app-label" class="w-full min-w-0">
						<span class="truncate">{target?.name ?? ''}</span>
					</Select.Trigger>
					<Select.Content>
						{#each candidates as app (app.id)}
							<Select.Item value={app.id} label={app.name}>
								<Icon icon={DashboardSquare01Icon} class="text-muted-foreground" />
								{app.name}
							</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</div>
			<div class="flex flex-col gap-1.5">
				<span id="restore-env-label" class="text-sm font-medium"
					>{l.translate('restore.environment')}</span
				>
				<Segmented bind:value={env} items={environments} labelledby="restore-env-label" />
			</div>
		</div>
		<p class="-mt-3 text-xs text-muted-foreground">
			{samePlace ? l.translate('restore.same_place') : l.translate('restore.other_place')}
		</p>

		<fieldset class="flex flex-col gap-1.5">
			<legend class="pb-1.5 text-sm font-medium">{l.translate('restore.volumes')}</legend>
			{#each keys as volume (volume.key)}
				{@const existing = declared(volume.key)}
				<label
					class="-mx-2 flex min-h-12 cursor-pointer items-center gap-3 rounded-md px-2 py-1.5 hover:bg-hover"
				>
					<Checkbox
						checked={!unticked.includes(volume.key)}
						onCheckedChange={(on) => toggle(volume.key, on === true)}
						aria-label={volume.key}
					/>
					<span class="flex min-w-0 flex-col gap-0.5">
						<span class="flex items-baseline gap-2">
							<span class="truncate font-mono text-label">{volume.key}</span>
							<span class="text-xs text-muted-foreground">{formatBytes(volume.size)}</span>
						</span>
						<span
							class="truncate text-xs {existing || !destination
								? 'text-muted-foreground'
								: 'text-warning'}"
						>
							{#if !destination}
								…
							{:else if existing}
								{l.translate('restore.replaces', [existing.docker_name])}
							{:else}
								{l.translate('restore.creates')}
							{/if}
						</span>
					</span>
				</label>
			{/each}
		</fieldset>

		<div class="flex min-h-[54px] items-start gap-4 py-2">
			<span class="flex min-w-0 flex-1 flex-col gap-1">
				<label for="restore-snapshot" class="text-sm font-medium"
					>{l.translate('restore.snapshot')}</label
				>
				<span class="text-xs/[18px] text-muted-foreground"
					>{l.translate('restore.snapshot.help')}</span
				>
			</span>
			<Switch id="restore-snapshot" bind:checked={snapshot} />
		</div>

		<p class="flex gap-2.5 text-xs/[18px] text-muted-foreground">
			<Icon icon={InformationCircleIcon} class="mt-px size-4 shrink-0" />
			{l.translate('restore.stops')}
		</p>
	{/if}

	{#snippet footer()}
		<Button variant="outline" onclick={() => (open = false)}>{l.translate('cancel')}</Button>
		<Button disabled={!selected.length || !appId} onclick={() => (confirmOpen = true)}>
			<Icon icon={DatabaseRestoreIcon} />
			{l.translate('restore.submit', [selected.length])}
		</Button>
	{/snippet}
</Drawer>

<ConfirmDialog
	bind:open={confirmOpen}
	destructive
	icon={DatabaseRestoreIcon}
	title={l.translate('restore.confirm.title', [selected.length, target?.name ?? '', env])}
	description={snapshot
		? l.translate('restore.confirm.snapshot', [selected.join(', ')])
		: l.translate('restore.confirm.no_snapshot', [selected.join(', ')])}
	confirmLabel={l.translate('restore.confirm.action')}
	onconfirm={() => restore().catch((ex) => Promise.reject(new Error(messageOf(ex))))}
/>
