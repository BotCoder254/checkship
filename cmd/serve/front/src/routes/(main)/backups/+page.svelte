<!--
	Backups (canvas artboard 3): every backup of every application, including applications deleted
	since (their backups stay restorable into another one). Totals on top, filters on the
	application and the trigger, Upload backup asks which application it belongs to. Storage opens
	the storage settings.
-->
<script lang="ts">
	import Archive02Icon from '@hugeicons/core-free-icons/Archive02Icon';
	import DashboardSquare01Icon from '@hugeicons/core-free-icons/DashboardSquare01Icon';
	import HardDriveIcon from '@hugeicons/core-free-icons/HardDriveIcon';
	import Layers01Icon from '@hugeicons/core-free-icons/Layers01Icon';
	import Tag01Icon from '@hugeicons/core-free-icons/Tag01Icon';
	import Upload04Icon from '@hugeicons/core-free-icons/Upload04Icon';
	import BackupsTable from '#lib/components/backups/backups-table.svelte';
	import DetailsDrawer from '#lib/components/backups/details-drawer.svelte';
	import RestoreDrawer from '#lib/components/backups/restore-drawer.svelte';
	import UploadDrawer from '#lib/components/backups/upload-drawer.svelte';
	import EmptyState from '#lib/components/empty-state.svelte';
	import Icon from '#lib/components/icon.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import FilterMenu, { type FilterOption } from '#lib/components/table/filter-menu.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { appName, formatBytes, triggerLabel } from '#lib/backup.ts';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service, { type Backup, type BackupTrigger } from '#lib/resources/backups.ts';
	import targetsService from '#lib/resources/targets.ts';
	import { storageSummary } from '#lib/storage.ts';

	const { data: backups } = service.queryAll();
	const { data: storage } = service.queryStorage();
	const { data: targets } = targetsService.queryAll();

	let appFilter = $state('any');
	let triggerFilter = $state<'any' | BackupTrigger>('any');

	const appOptions = $derived<FilterOption<string>[]>([
		{ value: 'any', label: l.translate('backups.filter.any'), icon: Layers01Icon },
		...[...new Map(($backups ?? []).map((b) => [b.app.id, b])).values()].map((b) => ({
			value: b.app.id,
			label: appName(b),
			icon: DashboardSquare01Icon
		}))
	]);

	const triggers: BackupTrigger[] = [
		'manual',
		'upload',
		'before_delete',
		'before_target_change',
		'before_restore'
	];
	const triggerOptions: FilterOption<'any' | BackupTrigger>[] = [
		{ value: 'any', label: l.translate('backups.filter.any'), icon: Layers01Icon },
		...triggers.map((t) => ({ value: t, label: triggerLabel(t), icon: Tag01Icon }))
	];

	const shown = $derived(
		($backups ?? []).filter(
			(b) =>
				(appFilter === 'any' || b.app.id === appFilter) &&
				(triggerFilter === 'any' || b.trigger === triggerFilter)
		)
	);

	const stats = $derived.by(() => {
		const all = $backups ?? [];
		let local = 0;
		let stored = 0;
		let attention = 0;
		for (const b of all) {
			if (b.result && b.local) local += b.result.size;
			if (b.result && b.stored) stored += b.result.size;
			if (b.status === 'failed' || b.store_error) attention++;
		}
		return { count: all.length, local, stored, attention };
	});

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
</script>

<PageHeader title={l.translate('breadcrumb.backups')}>
	{#snippet actions()}
		<Button variant="outline" href={routes.backupStorage}>
			<Icon icon={HardDriveIcon} />
			{l.translate('storage.open')}
		</Button>
		<Button onclick={() => (uploadOpen = true)}>
			<Icon icon={Upload04Icon} />
			{l.translate('backups.upload')}
		</Button>
	{/snippet}
	{#snippet status()}
		{#if $storage}
			<span>{storageSummary($storage, $targets ?? [])}</span>
		{/if}
	{/snippet}
</PageHeader>

{#if $backups && $backups.length > 0}
	<div class="flex flex-col gap-6">
		<dl class="grid grid-cols-2 gap-3 md:grid-cols-4">
			<div class="flex flex-col gap-1 rounded-lg bg-sunken px-4 py-3.5">
				<dt class="text-[12.5px] text-muted-foreground">{l.translate('backups.list.title')}</dt>
				<dd class="text-xl font-semibold tabular-nums">{stats.count}</dd>
			</div>
			<div class="flex flex-col gap-1 rounded-lg bg-sunken px-4 py-3.5">
				<dt class="text-[12.5px] text-muted-foreground">{l.translate('backups.stored.here')}</dt>
				<dd class="text-xl font-semibold tabular-nums">{formatBytes(stats.local)}</dd>
			</div>
			<div class="flex flex-col gap-1 rounded-lg bg-sunken px-4 py-3.5">
				<dt class="text-[12.5px] text-muted-foreground">{l.translate('storage.open')}</dt>
				<dd class="text-xl font-semibold tabular-nums">{formatBytes(stats.stored)}</dd>
			</div>
			<div class="flex flex-col gap-1 rounded-lg bg-sunken px-4 py-3.5">
				<dt class="text-[12.5px] text-muted-foreground">
					{l.translate('backups.status.failed')} / {l.translate('backups.status.local_only')}
				</dt>
				<dd class="text-xl font-semibold tabular-nums {stats.attention ? 'text-warning' : ''}">
					{stats.attention}
				</dd>
			</div>
		</dl>

		<div class="flex flex-wrap items-center gap-2">
			<FilterMenu
				label={l.translate('column.application')}
				icon={DashboardSquare01Icon}
				value={appFilter}
				options={appOptions}
				onValueChange={(v) => (appFilter = v)}
			/>
			<FilterMenu
				label={l.translate('column.trigger')}
				icon={Tag01Icon}
				value={triggerFilter}
				options={triggerOptions}
				onValueChange={(v) => (triggerFilter = v)}
			/>
		</div>

		<BackupsTable
			backups={shown}
			showApp
			label={l.translate('breadcrumb.backups')}
			onrestore={openRestore}
			ondetails={(b) => {
				detailsId = b.id;
				detailsOpen = true;
			}}
			selected={detailsOpen ? detailsId : undefined}
		/>
	</div>
{:else if $backups}
	<EmptyState
		icon={Archive02Icon}
		title={l.translate('backups.empty.title')}
		description={l.translate('backups.empty_all.description')}
	>
		{#snippet action()}
			<Button onclick={() => (uploadOpen = true)}>
				<Icon icon={Upload04Icon} />
				{l.translate('backups.upload')}
			</Button>
		{/snippet}
	</EmptyState>
{:else}
	<Skeleton class="h-60 w-full" />
{/if}

<UploadDrawer bind:open={uploadOpen} />
<RestoreDrawer bind:open={restoreOpen} backup={restoring} />
<DetailsDrawer bind:open={detailsOpen} backupId={detailsId} onrestore={openRestore} />
