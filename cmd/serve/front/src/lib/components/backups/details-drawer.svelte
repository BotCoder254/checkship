<!--
	A backup's details (DESIGN.md §11.10 Backups): its steps (requested, copied, stored or kept on
	its target), where it came from, what it holds (a share bar of volume sizes), what it left out
	and why, and every restore made from it as list rows with their outcome. Refreshes while something runs.
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
	import service, { type Backup, type BackupDetail, type Restore } from '#lib/resources/backups.ts';
	import CancelCircleIcon from '@hugeicons/core-free-icons/CancelCircleIcon';
	import CheckmarkCircle02Icon from '@hugeicons/core-free-icons/CheckmarkCircle02Icon';
	import Clock01Icon from '@hugeicons/core-free-icons/Clock01Icon';
	import Loading03Icon from '@hugeicons/core-free-icons/Loading03Icon';
	import ShareBar, { type ShareItem } from '#lib/components/charts/share-bar.svelte';
	import type { IconData } from '#lib/components/icon.svelte';
	import type { BadgeState } from '#lib/components/status-badge.svelte';
	import Steps, { type Step } from '#lib/components/steps.svelte';
	import TimelineList from '#lib/components/timeline/timeline-list.svelte';
	import TimelineRow, { type TimelineTone } from '#lib/components/timeline/timeline-row.svelte';
	import EmptyState from '#lib/components/empty-state.svelte';

	type Props = { open: boolean; backupId?: string; onrestore: (backup: Backup) => void };

	let { open = $bindable(false), backupId, onrestore }: Props = $props();

	const query = $derived(backupId ? service.queryOne(backupId) : undefined);
	const store = $derived(query?.data);
	const detail = $derived<BackupDetail | undefined>(store ? $store : undefined);

	/** A backup's steps (DESIGN.md §11.10 Backups): requested, copied, then stored or kept here. */
	function stepsOf(b: BackupDetail): Step[] {
		const copied: Step = {
			key: 'copied',
			label: l.translate('backups.steps.copied'),
			time: b.finished_at ? l.time(b.finished_at) : undefined,
			note:
				b.status === 'succeeded' && b.finished_at
					? [
							l.translate('list.took', [l.duration(b.requested_at, b.finished_at)]),
							b.result ? formatBytes(b.result.size) : ''
						]
							.filter(Boolean)
							.join(' · ')
					: b.status === 'failed' && b.error_code
						? errorText(b.error_code)
						: undefined,
			state:
				b.status === 'succeeded'
					? 'done'
					: b.status === 'failed'
						? 'failed'
						: b.status === 'running'
							? 'current'
							: 'todo'
		};
		const stored: Step =
			b.status !== 'succeeded'
				? { key: 'stored', label: l.translate('backups.steps.stored'), state: 'todo' }
				: b.store_error
					? {
							key: 'stored',
							label: l.translate('backups.steps.not_stored'),
							note: errorText(b.store_error),
							state: 'failed'
						}
					: b.stored
						? {
								key: 'stored',
								label: l.translate('backups.steps.stored'),
								note: l.translate('backups.steps.on_storage'),
								state: 'done'
							}
						: {
								key: 'stored',
								label: l.translate('backups.steps.kept'),
								note: l.translate('backups.steps.only_on', [b.target.name ?? b.target.id]),
								state: 'done'
							};
		return [
			{
				key: 'requested',
				label: l.translate('backups.steps.requested'),
				time: l.time(b.requested_at),
				state: 'done'
			},
			copied,
			stored
		];
	}

	const shades = [
		'var(--accent)',
		'color-mix(in oklab, var(--accent) 55%, var(--ink))',
		'color-mix(in oklab, var(--accent) 35%, var(--surface))',
		'var(--faint)'
	];
	function volumeShare(b: BackupDetail): ShareItem[] {
		const volumes = [...(b.result?.volumes ?? [])].sort((x, y) => y.size - x.size);
		const total = volumes.reduce((sum, v) => sum + v.size, 0);
		return volumes.map((v, i) => ({
			key: v.key,
			label: v.key,
			mono: true,
			value: v.size,
			color: shades[Math.min(i, shades.length - 1)],
			cells: [
				formatBytes(v.size),
				`${total ? Math.round((v.size / total) * 100) : 0}%`,
				l.translate('backups.files', [v.files])
			]
		}));
	}

	function restoreTile(status: Restore['status']): {
		icon: IconData;
		tone: TimelineTone;
		spin: boolean;
		badge: BadgeState;
	} {
		if (status === 'succeeded')
			return { icon: CheckmarkCircle02Icon, tone: 'success', spin: false, badge: 'succeeded' };
		if (status === 'failed')
			return { icon: CancelCircleIcon, tone: 'danger', spin: false, badge: 'failed' };
		if (status === 'running')
			return { icon: Loading03Icon, tone: 'warning', spin: true, badge: 'running' };
		return { icon: Clock01Icon, tone: 'muted', spin: false, badge: 'pending' };
	}
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

		<Steps label={l.translate('backups.steps.label')} steps={stepsOf(detail)} />

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
			<section class="flex flex-col gap-3">
				<h3 class="text-xs font-semibold tracking-wide text-muted-foreground">
					{l.translate('backups.details.holds', [
						detail.result.volumes.length,
						formatBytes(detail.result.size)
					])}
				</h3>
				<div class="track p-4">
					<ShareBar
						items={volumeShare(detail)}
						headers={[
							l.translate('column.volumes'),
							l.translate('backups.share.size'),
							l.translate('charts.share'),
							l.translate('backups.details.files')
						]}
						label={detail.result.volumes.map((v) => `${v.key} ${formatBytes(v.size)}`).join(', ')}
					/>
				</div>
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
			<div class="min-w-0 track p-2">
				{#if detail.restores.length}
					<TimelineList label={l.translate('backups.details.restores')}>
						{#each detail.restores as restore (restore.id)}
							{@const tile = restoreTile(restore.status)}
							{@const line = [
								l.relative(restore.requested_at),
								restore.restored.length ? restore.restored.join(', ') : undefined,
								restore.snapshot_id ? l.translate('backups.details.snapshot') : undefined,
								restore.error_code ? errorText(restore.error_code) : undefined
							]
								.filter(Boolean)
								.join(' · ')}
							<TimelineRow
								icon={tile.icon}
								tone={tile.tone}
								spin={tile.spin}
								label={`${l.translate('backups.details.restore_line', [appName(restore), restore.environment])}, ${l.translate(`backups.status.${restore.status}`)}, ${line}`}
							>
								{#snippet title()}{l.translate('backups.details.restore_line', [
										appName(restore),
										restore.environment
									])}{/snippet}
								{#snippet badge()}<StatusBadge
										state={tile.badge}
										label={l.translate(`backups.status.${restore.status}`)}
									/>{/snippet}
								{#snippet details()}{line}{/snippet}
							</TimelineRow>
						{/each}
					</TimelineList>
				{:else}
					<EmptyState
						illustration="backups"
						size="compact"
						level={3}
						title={l.translate('backups.details.no_restores.title')}
					/>
				{/if}
			</div>
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
