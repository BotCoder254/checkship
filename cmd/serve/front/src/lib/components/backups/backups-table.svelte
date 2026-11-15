<!--
	Backups table ("checkship backups" canvas, artboards 1 and 3): one row per backup, newest first.
	The ⋯ menu changes with the state: a finished backup can be restored, downloaded, checksummed and
	deleted; one kept here only after a failed upload can be uploaded again; a failed one can be
	retried. Clicking the row opens its details.
-->
<script lang="ts">
	import Archive02Icon from '@hugeicons/core-free-icons/Archive02Icon';
	import Calendar03Icon from '@hugeicons/core-free-icons/Calendar03Icon';
	import CloudUploadIcon from '@hugeicons/core-free-icons/CloudUploadIcon';
	import Copy01Icon from '@hugeicons/core-free-icons/Copy01Icon';
	import DashboardSquare01Icon from '@hugeicons/core-free-icons/DashboardSquare01Icon';
	import DatabaseIcon from '@hugeicons/core-free-icons/DatabaseIcon';
	import DatabaseRestoreIcon from '@hugeicons/core-free-icons/DatabaseRestoreIcon';
	import Delete02Icon from '@hugeicons/core-free-icons/Delete02Icon';
	import Download04Icon from '@hugeicons/core-free-icons/Download04Icon';
	import HardDriveIcon from '@hugeicons/core-free-icons/HardDriveIcon';
	import InformationCircleIcon from '@hugeicons/core-free-icons/InformationCircleIcon';
	import Layers01Icon from '@hugeicons/core-free-icons/Layers01Icon';
	import RepeatIcon from '@hugeicons/core-free-icons/RepeatIcon';
	import Tag01Icon from '@hugeicons/core-free-icons/Tag01Icon';
	import UserIcon from '@hugeicons/core-free-icons/UserIcon';
	import { untrack } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { copyText } from '#lib/clipboard.ts';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import EnvironmentChip from '#lib/components/environment-chip.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import ColumnsMenu from '#lib/components/table/columns-menu.svelte';
	import RowMenuGroup from '#lib/components/table/row-menu-group.svelte';
	import RowMenuItem from '#lib/components/table/row-menu-item.svelte';
	import RowMenu from '#lib/components/table/row-menu.svelte';
	import * as Table from '#lib/components/ui/table/index.js';
	import {
		appName,
		backupBadge,
		messageOf,
		sizeOf,
		storedLabel,
		triggerLabel,
		volumesOf
	} from '#lib/backup.ts';
	import l from '#lib/localization/index.ts';
	import service, { type Backup } from '#lib/resources/backups.ts';
	import { ACTIONS_CELL, ACTIONS_HEAD, TableColumns } from '#lib/table-columns.svelte.ts';

	type Props = {
		backups: Backup[];
		/** The global page shows which application each backup belongs to. */
		showApp?: boolean;
		label: string;
		onrestore: (backup: Backup) => void;
		ondetails: (backup: Backup) => void;
		selected?: string;
	};

	let { backups, showApp = false, label, onrestore, ondetails, selected }: Props = $props();

	// Each table keeps the columns it started with.
	const withApp = untrack(() => showApp);

	const columns = new TableColumns(
		[
			{
				id: 'status',
				label: l.translate('column.status'),
				icon: InformationCircleIcon,
				width: 'w-[15%]',
				locked: true
			},
			...(withApp
				? [
						{
							id: 'app',
							label: l.translate('column.application'),
							icon: DashboardSquare01Icon,
							width: 'w-[14%]'
						}
					]
				: []),
			{
				id: 'created',
				label: l.translate('column.created'),
				icon: Calendar03Icon,
				width: 'w-[12%]'
			},
			{
				id: 'environment',
				label: l.translate('column.environment'),
				icon: Layers01Icon,
				width: 'w-[11%]',
				priority: 2
			},
			{
				id: 'trigger',
				label: l.translate('column.trigger'),
				icon: Tag01Icon,
				width: 'w-[13%]',
				priority: 2
			},
			{
				id: 'volumes',
				label: l.translate('column.volumes'),
				icon: DatabaseIcon,
				width: 'w-[13%]',
				priority: 3
			},
			{ id: 'size', label: l.translate('column.size'), icon: Archive02Icon, width: 'w-[11%]' },
			{
				id: 'stored',
				label: l.translate('column.stored'),
				icon: HardDriveIcon,
				width: 'w-[12%]',
				priority: 2
			},
			...(withApp
				? []
				: [
						{
							id: 'by',
							label: l.translate('column.by'),
							icon: UserIcon,
							width: 'w-[10%]',
							priority: 3 as const
						}
					])
		],
		withApp ? 'backups.all' : 'backups.app'
	);

	let deleting = $state<Backup>();
	let deleteOpen = $state(false);

	function download(backup: Backup) {
		const a = document.createElement('a');
		a.href = service.downloadUrl(backup.id);
		a.download = '';
		document.body.appendChild(a);
		a.click();
		a.remove();
	}

	async function copyChecksum(backup: Backup) {
		if (!backup.result) return;
		await copyText(backup.result.sha256);
		toast.success(l.translate('backups.copied'));
	}

	async function store(backup: Backup) {
		try {
			await service.store(backup.id);
			toast.success(l.translate('backups.store.queued'));
		} catch (ex) {
			toast.error(messageOf(ex));
		}
	}

	async function retry(backup: Backup) {
		try {
			await service.retry(backup.id);
			toast.success(l.translate('backups.retry.queued'));
		} catch (ex) {
			toast.error(messageOf(ex));
		}
	}
</script>

<div class="flex flex-col gap-2.5">
	<div class="flex justify-end">
		<ColumnsMenu {columns} />
	</div>
	<Table.Root aria-label={label}>
		<Table.Header>
			<Table.Row>
				{#each columns.shown as column (column.id)}
					<Table.Head class={columns.head(column.id)}>{column.label}</Table.Head>
				{/each}
				<Table.Head class={ACTIONS_HEAD}>{l.translate('column.actions')}</Table.Head>
			</Table.Row>
		</Table.Header>
		<Table.Body>
			{#each backups as backup (backup.id)}
				{@const badge = backupBadge(backup)}
				{@const stored = storedLabel(backup)}
				{@const done = backup.status === 'succeeded'}
				<Table.Row class="relative" data-state={selected === backup.id ? 'selected' : undefined}>
					{#each columns.shown as column (column.id)}
						{#if column.id === 'status'}
							<Table.Cell class={columns.cell('status')}>
								<!-- The whole row opens the details; the button stretches over it. -->
								<button
									type="button"
									class="outline-none after:absolute after:inset-0 after:rounded-md focus-visible:after:ring-2 focus-visible:after:ring-ring focus-visible:after:ring-inset"
									aria-label={l.translate('backups.menu.details')}
									onclick={() => ondetails(backup)}
								>
									<StatusBadge state={badge.state} label={badge.label} />
								</button>
							</Table.Cell>
						{:else if column.id === 'app'}
							<Table.Cell
								class={columns.cell(
									'app',
									backup.app.deleted ? 'font-semibold text-faint' : 'font-semibold'
								)}
								title={appName(backup)}>{appName(backup)}</Table.Cell
							>
						{:else if column.id === 'created'}
							<Table.Cell
								class={columns.cell('created', 'text-muted-foreground')}
								title={l.datetime(backup.requested_at)}
							>
								{l.relative(backup.requested_at)}
							</Table.Cell>
						{:else if column.id === 'environment'}
							<Table.Cell class={columns.cell('environment')}>
								<EnvironmentChip environment={backup.environment} />
							</Table.Cell>
						{:else if column.id === 'trigger'}
							<Table.Cell class={columns.cell('trigger')}>{triggerLabel(backup.trigger)}</Table.Cell
							>
						{:else if column.id === 'volumes'}
							{@const keys = volumesOf(backup)}
							<Table.Cell
								class={columns.cell(
									'volumes',
									keys.length ? 'font-mono text-[12.5px]' : 'text-muted-foreground'
								)}
								title={keys.join(', ')}
							>
								{keys.length ? keys.join(', ') : l.translate('backups.all_volumes')}
							</Table.Cell>
						{:else if column.id === 'size'}
							<Table.Cell
								class={columns.cell(
									'size',
									backup.status === 'failed'
										? 'font-mono text-[12.5px] text-danger'
										: 'text-muted-foreground'
								)}
								title={sizeOf(backup)}>{sizeOf(backup)}</Table.Cell
							>
						{:else if column.id === 'stored'}
							<Table.Cell class={columns.cell('stored', stored.tone)}>{stored.label}</Table.Cell>
						{:else if column.id === 'by'}
							<Table.Cell
								class={columns.cell('by', 'text-muted-foreground')}
								title={backup.requested_by.email}
							>
								{backup.requested_by.email}
							</Table.Cell>
						{/if}
					{/each}
					<Table.Cell class={ACTIONS_CELL}>
						<RowMenu label={l.translate('table.menu', [l.datetime(backup.requested_at)])}>
							<RowMenuGroup>
								{#if done}
									<RowMenuItem icon={DatabaseRestoreIcon} onSelect={() => onrestore(backup)}>
										{l.translate('backups.menu.restore')}
									</RowMenuItem>
									<RowMenuItem icon={Download04Icon} onSelect={() => download(backup)}>
										{l.translate('backups.menu.download')}
									</RowMenuItem>
									{#if backup.store_error && backup.local}
										<RowMenuItem icon={CloudUploadIcon} onSelect={() => store(backup)}>
											{l.translate('backups.menu.store')}
										</RowMenuItem>
									{/if}
									<RowMenuItem icon={Copy01Icon} onSelect={() => copyChecksum(backup)}>
										{l.translate('backups.menu.copy_checksum')}
									</RowMenuItem>
								{/if}
								{#if backup.status === 'failed'}
									<RowMenuItem icon={RepeatIcon} onSelect={() => retry(backup)}>
										{l.translate('backups.menu.retry')}
									</RowMenuItem>
								{/if}
								<RowMenuItem icon={InformationCircleIcon} onSelect={() => ondetails(backup)}>
									{l.translate('backups.menu.details')}
								</RowMenuItem>
							</RowMenuGroup>
							{#if backup.status === 'succeeded' || backup.status === 'failed'}
								<RowMenuGroup>
									<RowMenuItem
										icon={Delete02Icon}
										variant="destructive"
										onSelect={() => {
											deleting = backup;
											deleteOpen = true;
										}}
									>
										{l.translate('menu.delete')}
									</RowMenuItem>
								</RowMenuGroup>
							{/if}
						</RowMenu>
					</Table.Cell>
				</Table.Row>
			{/each}
		</Table.Body>
	</Table.Root>
</div>

{#if deleting}
	<ConfirmDialog
		bind:open={deleteOpen}
		destructive
		icon={Delete02Icon}
		title={l.translate('backups.delete.title')}
		description={l.translate('backups.delete.description')}
		confirmLabel={l.translate('backups.delete.confirm')}
		onconfirm={() => service.delete(deleting!.id)}
	/>
{/if}
