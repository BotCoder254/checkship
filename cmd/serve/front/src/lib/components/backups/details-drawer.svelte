<!--
	A backup's details: where it came from, what it holds, what it left out and why, where it is
	stored, and every restore made from it with its outcome. Refreshes while something runs.
-->
<script lang="ts">
	import DatabaseRestoreIcon from '@hugeicons/core-free-icons/DatabaseRestoreIcon';
	import Download04Icon from '@hugeicons/core-free-icons/Download04Icon';
	import Drawer from '#lib/components/drawer.svelte';
	import EnvironmentChip from '#lib/components/environment-chip.svelte';
	import Icon from '#lib/components/icon.svelte';
	import Notice from '#lib/components/notice.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import {
		appName,
		backupBadge,
		errorText,
		formatBytes,
		storedLabel,
		triggerLabel
	} from '#lib/backup.ts';
	import l from '#lib/localization/index.ts';
	import service, { type Backup, type BackupDetail } from '#lib/resources/backups.ts';

	type Props = { open: boolean; backupId?: string; onrestore: (backup: Backup) => void };

	let { open = $bindable(false), backupId, onrestore }: Props = $props();

	const query = $derived(backupId ? service.queryOne(backupId) : undefined);
	const store = $derived(query?.data);
	const detail = $derived<BackupDetail | undefined>(store ? $store : undefined);
</script>

<Drawer bind:open title={l.translate('backups.details.title')} size="preview">
	{#if detail}
		{@const badge = backupBadge(detail)}
		{@const stored = storedLabel(detail)}
		<div class="flex flex-wrap items-center gap-2">
			<StatusBadge state={badge.state} label={badge.label} />
			<EnvironmentChip environment={detail.environment} />
			<span class="text-sm text-muted-foreground">{triggerLabel(detail.trigger)}</span>
		</div>

		<dl class="grid grid-cols-[120px_minmax(0,1fr)] gap-x-3 gap-y-2 text-sm">
			<dt class="text-muted-foreground">{l.translate('backups.details.from')}</dt>
			<dd class="truncate">{appName(detail)} · {detail.target.name ?? detail.target.id}</dd>
			<dt class="text-muted-foreground">{l.translate('backups.details.created')}</dt>
			<dd class="truncate">{l.datetime(detail.requested_at)} · {detail.requested_by.email}</dd>
			<dt class="text-muted-foreground">{l.translate('backups.details.stored')}</dt>
			<dd class={stored.tone}>
				{stored.label}{detail.store_error ? ` · ${errorText(detail.store_error)}` : ''}
			</dd>
			{#if detail.result}
				<dt class="text-muted-foreground">{l.translate('backups.details.size')}</dt>
				<dd>{formatBytes(detail.result.size)}</dd>
				<dt class="text-muted-foreground">{l.translate('backups.details.checksum')}</dt>
				<dd class="font-mono text-xs break-all">{detail.result.sha256}</dd>
			{/if}
			{#if detail.error_code}
				<dt class="text-muted-foreground">{l.translate('backups.details.error')}</dt>
				<dd class="text-danger">
					<span class="font-mono text-xs">{detail.error_code}</span> · {errorText(
						detail.error_code
					)}
				</dd>
			{/if}
		</dl>

		{#if detail.result}
			<section class="flex flex-col gap-2">
				<h3 class="text-xs font-semibold tracking-wide text-muted-foreground">
					{l.translate('column.volumes')}
				</h3>
				{#each detail.result.volumes as volume (volume.key)}
					<div class="flex items-center justify-between gap-3 py-1.5 text-sm">
						<span class="font-mono text-label">{volume.key}</span>
						<span class="text-muted-foreground"
							>{formatBytes(volume.size)} · {l.translate('backups.files', [volume.files])}</span
						>
					</div>
				{/each}
			</section>

			{#if detail.result.skipped?.length}
				<section class="flex flex-col gap-2">
					<h3 class="text-xs font-semibold tracking-wide text-muted-foreground">
						{l.translate('backups.details.skipped')}
					</h3>
					{#each detail.result.skipped as mount (mount.service + mount.target)}
						<Notice tone="warning">
							<strong class="font-semibold">
								{l.translate('backups.skipped.title', [
									mount.service,
									mount.source || mount.target
								])}
							</strong>
							<p class="text-muted-foreground">{l.translate(`backups.skipped.${mount.kind}`)}</p>
						</Notice>
					{/each}
				</section>
			{/if}
		{/if}

		<section class="flex flex-col gap-2">
			<h3 class="text-xs font-semibold tracking-wide text-muted-foreground">
				{l.translate('backups.details.restores')}
			</h3>
			{#each detail.restores as restore (restore.id)}
				{@const state =
					restore.status === 'succeeded'
						? 'succeeded'
						: restore.status === 'failed'
							? 'failed'
							: restore.status === 'running'
								? 'running'
								: 'pending'}
				<div class="flex flex-col gap-1 py-1.5 text-sm">
					<div class="flex flex-wrap items-center gap-2">
						<StatusBadge {state} label={l.translate(`backups.status.${restore.status}`)} />
						<span
							>{l.translate('backups.details.restore_line', [
								appName(restore),
								restore.environment
							])}</span
						>
						<span class="text-muted-foreground">{l.relative(restore.requested_at)}</span>
					</div>
					{#if restore.restored.length}
						<span class="font-mono text-xs text-muted-foreground"
							>{restore.restored.join(', ')}</span
						>
					{/if}
					{#if restore.error_code}
						<span class="text-xs text-danger">{errorText(restore.error_code)}</span>
					{/if}
				</div>
			{:else}
				<p class="text-sm text-muted-foreground">{l.translate('backups.details.no_restores')}</p>
			{/each}
		</section>
	{/if}

	{#snippet footer()}
		{#if detail?.status === 'succeeded'}
			<Button variant="outline" href={service.downloadUrl(detail.id)} download="">
				<Icon icon={Download04Icon} />
				{l.translate('backups.menu.download')}
			</Button>
			<Button onclick={() => detail && onrestore(detail)}>
				<Icon icon={DatabaseRestoreIcon} />
				{l.translate('backups.menu.restore')}
			</Button>
		{:else}
			<Button variant="outline" onclick={() => (open = false)}>{l.translate('close')}</Button>
		{/if}
	{/snippet}
</Drawer>
