<!--
	Backups list ("checkship backups" canvas, artboards 1 and 3): one row per backup, newest first,
	grouped by day (DESIGN.md §5 Lists and tables). The details line says when, how big, where it
	is kept and which volumes. The ⋯ menu changes with the state: a finished backup can be restored, downloaded, checksummed and
	deleted; one kept here only after a failed upload can be uploaded again; a failed one can be
	retried. Clicking the row opens its details.
-->
<script lang="ts">
	import CloudUploadIcon from '@hugeicons/core-free-icons/CloudUploadIcon';
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import CancelCircleIcon from '@hugeicons/core-free-icons/CancelCircleIcon';
	import CheckmarkCircle02Icon from '@hugeicons/core-free-icons/CheckmarkCircle02Icon';
	import Clock01Icon from '@hugeicons/core-free-icons/Clock01Icon';
	import Copy01Icon from '@hugeicons/core-free-icons/Copy01Icon';
	import DatabaseRestoreIcon from '@hugeicons/core-free-icons/DatabaseRestoreIcon';
	import Delete02Icon from '@hugeicons/core-free-icons/Delete02Icon';
	import Download04Icon from '@hugeicons/core-free-icons/Download04Icon';
	import InformationCircleIcon from '@hugeicons/core-free-icons/InformationCircleIcon';
	import Loading03Icon from '@hugeicons/core-free-icons/Loading03Icon';
	import RepeatIcon from '@hugeicons/core-free-icons/RepeatIcon';
	import { toast } from 'svelte-sonner';
	import { copyText } from '#lib/clipboard.ts';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import EnvironmentChip from '#lib/components/environment-chip.svelte';
	import type { IconData } from '#lib/components/icon.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import TimelineGroup from '#lib/components/timeline/timeline-group.svelte';
	import TimelineRow, { type TimelineTone } from '#lib/components/timeline/timeline-row.svelte';
	import { clock } from '#lib/deployment-row.ts';
	import { groupByDay } from '#lib/timeline.ts';
	import RowMenuGroup from '#lib/components/table/row-menu-group.svelte';
	import RowMenuItem from '#lib/components/table/row-menu-item.svelte';
	import RowMenu from '#lib/components/table/row-menu.svelte';
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

	/** The tile: a stored backup is a check, one kept only here (upload failed) a warning. */
	function tileOf(backup: Backup): { icon: IconData; tone: TimelineTone; spin: boolean } {
		if (backup.status === 'succeeded')
			return backup.store_error
				? { icon: Alert02Icon, tone: 'warning', spin: false }
				: { icon: CheckmarkCircle02Icon, tone: 'success', spin: false };
		if (backup.status === 'failed') return { icon: CancelCircleIcon, tone: 'danger', spin: false };
		if (backup.status === 'running') return { icon: Loading03Icon, tone: 'warning', spin: true };
		return { icon: Clock01Icon, tone: 'muted', spin: false };
	}

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

<div class="flex flex-col gap-4" role="group" aria-label={label}>
	{#each groupByDay(backups, (b) => b.requested_at, l.locale()) as group (group.key)}
		<TimelineGroup label={group.label} count={group.items.length}>
			{#each group.items as backup (backup.id)}
				{@const status = backupBadge(backup)}
				{@const stored = storedLabel(backup)}
				{@const done = backup.status === 'succeeded'}
				{@const tile = tileOf(backup)}
				{@const keys = volumesOf(backup)}
				{@const name = showApp ? appName(backup) : triggerLabel(backup.trigger)}
				<TimelineRow
					icon={tile.icon}
					tone={tile.tone}
					spin={tile.spin}
					muted={showApp && backup.app.deleted}
					label={[
						name,
						status.label,
						l.datetime(backup.requested_at),
						sizeOf(backup),
						stored.label
					].join(', ')}
					selected={selected === backup.id}
					onclick={() => ondetails(backup)}
				>
					{#snippet title()}{name}{/snippet}
					{#snippet badge()}
						<EnvironmentChip environment={backup.environment} />
						{#if !done || backup.store_error}
							<StatusBadge state={status.state} label={status.label} />
						{/if}
					{/snippet}
					{#snippet details()}
						{clock(backup.requested_at)}
						{#if showApp}· {triggerLabel(backup.trigger)}{/if}
						·
						<span class={backup.status === 'failed' ? 'font-mono text-danger' : ''}
							>{sizeOf(backup)}</span
						>
						· <span class={stored.tone}>{stored.label}</span>
						{#if keys.length}· <span class="font-mono">{keys.join(', ')}</span>{/if}
						{#if !showApp}· {backup.requested_by.email}{/if}
					{/snippet}
					{#snippet actions()}
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
					{/snippet}
				</TimelineRow>
			{/each}
		</TimelineGroup>
	{/each}
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
