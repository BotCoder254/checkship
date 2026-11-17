<!--
	Backups tab ("checkship backups" canvas, artboard 1): the named volumes of an environment, read
	live from its target, with the mounts a backup leaves out and why; then every backup of the app.
	Back up now and Upload backup open drawers; a row opens its details, its menu restores,
	downloads, uploads again, retries or deletes.
-->
<script lang="ts">
	import Archive02Icon from '@hugeicons/core-free-icons/Archive02Icon';
	import DatabaseIcon from '@hugeicons/core-free-icons/DatabaseIcon';
	import Upload04Icon from '@hugeicons/core-free-icons/Upload04Icon';
	import BackupDrawer from '#lib/components/backups/backup-drawer.svelte';
	import BackupsList from '#lib/components/backups/backups-list.svelte';
	import DetailsDrawer from '#lib/components/backups/details-drawer.svelte';
	import RestoreDrawer from '#lib/components/backups/restore-drawer.svelte';
	import UploadDrawer from '#lib/components/backups/upload-drawer.svelte';
	import EmptyState from '#lib/components/empty-state.svelte';
	import Notice from '#lib/components/notice.svelte';
	import Icon from '#lib/components/icon.svelte';
	import Segmented from '#lib/components/segmented.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import TimelineList from '#lib/components/timeline/timeline-list.svelte';
	import TimelineRow from '#lib/components/timeline/timeline-row.svelte';
	import { formatBytes, messageOf } from '#lib/backup.ts';
	import l from '#lib/localization/index.ts';
	import appsService from '#lib/resources/apps.ts';
	import service, { type AppVolumes, type Backup } from '#lib/resources/backups.ts';
	import type { Environment } from '#lib/resources/deployments.ts';
	import targetsService from '#lib/resources/targets.ts';
	import { storageSummary } from '#lib/storage.ts';
	import AppWorkspace from '../app-workspace.svelte';

	let { data } = $props();

	const appQuery = $derived(appsService.queryById(data.app.id));
	const appStore = $derived(appQuery.data);
	const app = $derived($appStore ?? data.app);

	const backupsQuery = $derived(service.queryByApp(data.app.id));
	const backupsStore = $derived(backupsQuery.data);
	const backups = $derived($backupsStore);

	const { data: storage } = service.queryStorage();
	const { data: targets } = targetsService.queryAll();
	const storageLabel = $derived($storage ? storageSummary($storage, $targets ?? []) : undefined);

	let environment = $state<Environment>('production');
	let volumes = $state<AppVolumes>();
	let volumesError = $state<string>();

	const environments = [
		{ value: 'production' as const, label: l.translate('deploy.environment.production') },
		{ value: 'staging' as const, label: l.translate('deploy.environment.staging') }
	];

	// The live volumes of the chosen environment, read again after each backup or restore.
	$effect(() => {
		const current = environment;
		void backups;
		service
			.fetchVolumes(data.app.id, current, { cache: 'no-store' })
			.then((v) => {
				if (current === environment) {
					volumes = v;
					volumesError = undefined;
				}
			})
			.catch((ex) => {
				if (current === environment) volumesError = messageOf(ex);
			});
	});

	const total = $derived((backups ?? []).reduce((sum, b) => sum + (b.result?.size ?? 0), 0));

	let backupOpen = $state(false);
	let uploadOpen = $state(false);
	let restoreOpen = $state(false);
	let restoring = $state<Backup>();
	let detailsOpen = $state(false);
	let detailsId = $state<string>();

	function openRestore(backup: Backup) {
		detailsOpen = false;
		restoring = backup;
		restoreOpen = true;
	}

	function openDetails(backup: Backup) {
		detailsId = backup.id;
		detailsOpen = true;
	}

	const targetName = (id: string) => ($targets ?? []).find((t) => t.id === id)?.name ?? id;
</script>

<AppWorkspace {app} apiKey={data.user.api_key} tab="backups">
	<div class="flex flex-col gap-9">
		<section class="flex flex-col gap-3" aria-labelledby="volumes-title">
			<div class="flex flex-wrap items-end justify-between gap-3">
				<div class="flex flex-col gap-1">
					<h2 id="volumes-title" class="text-xs font-semibold tracking-wide text-muted-foreground">
						{l.translate('backups.volumes.title')}
					</h2>
					<p class="text-label text-muted-foreground">
						{l.translate('backups.volumes.description')}
					</p>
				</div>
				<div class="flex flex-wrap items-center gap-2">
					<span id="volumes-env" class="sr-only">{l.translate('backups.drawer.environment')}</span>
					<Segmented bind:value={environment} items={environments} labelledby="volumes-env" />
					<Button variant="outline" onclick={() => (uploadOpen = true)}>
						<Icon icon={Upload04Icon} />
						{l.translate('backups.upload')}
					</Button>
					<Button onclick={() => (backupOpen = true)} disabled={!!app.cleanup_requested_at}>
						<Icon icon={Archive02Icon} />
						{l.translate('backups.back_up')}
					</Button>
				</div>
			</div>

			{#if volumesError}
				<Notice tone="danger" role="alert">
					<p>{l.translate('backups.volumes.unreachable')} {volumesError}</p>
				</Notice>
			{:else if !volumes}
				<Skeleton class="h-24 w-full" />
			{:else if volumes.volumes.length === 0}
				<Notice>
					<p>
						{l.translate('backups.volumes.empty', [
							l.translate(`deploy.environment.${environment}`)
						])}
					</p>
				</Notice>
			{:else}
				{@const where = volumes.target}
				<TimelineList label={l.translate('backups.volumes.title')}>
					{#each volumes.volumes as volume (volume.key)}
						{@const parts = [
							volume.services.length
								? l.translate('backups.volume.used_by', [volume.services.join(', ')])
								: undefined,
							targetName(where),
							volume.docker_name
						].filter((part): part is string => !!part)}
						<TimelineRow icon={DatabaseIcon} label={[volume.key, ...parts].join(', ')}>
							{#snippet title()}<span class="font-mono">{volume.key}</span>{/snippet}
							{#snippet details()}{parts.join(' · ')}{/snippet}
						</TimelineRow>
					{/each}
				</TimelineList>
			{/if}

			{#each volumes?.skipped ?? [] as mount (mount.service + mount.target)}
				<Notice tone="warning">
					<strong class="font-semibold">
						{l.translate('backups.skipped.title', [mount.service, mount.source || mount.target])}
					</strong>
					<p class="text-muted-foreground">{l.translate(`backups.skipped.${mount.kind}`)}</p>
				</Notice>
			{/each}
		</section>

		<section class="flex flex-col gap-3" aria-labelledby="backups-title">
			<div class="flex flex-col gap-1">
				<h2 id="backups-title" class="text-xs font-semibold tracking-wide text-muted-foreground">
					{l.translate('backups.list.title')}
				</h2>
				<p class="text-label text-muted-foreground">
					{backups?.length
						? l.translate('backups.summary', [backups.length, formatBytes(total)])
						: l.translate('backups.list.description')}
				</p>
			</div>
			{#if backups && backups.length > 0}
				<BackupsList
					{backups}
					label={l.translate('backups.list.title')}
					onrestore={openRestore}
					ondetails={openDetails}
					selected={detailsOpen ? detailsId : undefined}
				/>
			{:else if backups}
				<EmptyState
					icon={Archive02Icon}
					title={l.translate('backups.empty.title')}
					description={l.translate('backups.empty.description')}
				>
					{#snippet action()}
						<Button onclick={() => (backupOpen = true)} disabled={!!app.cleanup_requested_at}>
							<Icon icon={Archive02Icon} />
							{l.translate('backups.back_up')}
						</Button>
					{/snippet}
				</EmptyState>
			{:else}
				<Skeleton class="h-40 w-full" />
			{/if}
		</section>
	</div>
</AppWorkspace>

<BackupDrawer
	bind:open={backupOpen}
	app={{ id: app.id, name: app.name }}
	{environment}
	storage={$storage}
	{storageLabel}
/>
<UploadDrawer bind:open={uploadOpen} appId={app.id} />
<RestoreDrawer bind:open={restoreOpen} backup={restoring} />
<DetailsDrawer bind:open={detailsOpen} backupId={detailsId} onrestore={openRestore} />
