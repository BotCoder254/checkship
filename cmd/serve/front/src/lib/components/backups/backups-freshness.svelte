<!--
	Freshness (DESIGN.md §11.10 Backups): for each application environment, how old the newest
	backup you could restore is, and whether a copy is off the server. Backup age per workload shows
	real exposure better than whether the last job passed, and catches backups that silently
	stopped. Tones are fixed until schedules exist: under a day `success`, under a week quiet, older
	a `warning`, never a `danger`. Deployed environments without a backup are listed when they have
	volumes (read live, only for those). Each row opens the application's Backups tab.
-->
<script lang="ts">
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import CancelCircleIcon from '@hugeicons/core-free-icons/CancelCircleIcon';
	import CheckmarkCircle02Icon from '@hugeicons/core-free-icons/CheckmarkCircle02Icon';
	import Clock01Icon from '@hugeicons/core-free-icons/Clock01Icon';
	import EnvironmentChip from '#lib/components/environment-chip.svelte';
	import SectionHeading from '#lib/components/section-heading.svelte';
	import TimelineList from '#lib/components/timeline/timeline-list.svelte';
	import TimelineRow, { type TimelineTone } from '#lib/components/timeline/timeline-row.svelte';
	import type { IconData } from '#lib/components/icon.svelte';
	import { formatBytes, storedLabel } from '#lib/backup.ts';
	import {
		freshnessByEnvironment,
		type EnvironmentFreshness,
		type Freshness
	} from '#lib/backup-stats.ts';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import type { App } from '#lib/resources/apps.ts';
	import service, { type Backup } from '#lib/resources/backups.ts';
	import type { Environment } from '#lib/resources/deployments.ts';

	type Props = { backups: Backup[]; apps: App[] | undefined };

	let { backups, apps }: Props = $props();

	const known = $derived(freshnessByEnvironment(backups));

	// Deployed environments with no backup at all: listed only when they have volumes.
	let never = $state<EnvironmentFreshness[]>([]);
	$effect(() => {
		const has: Record<string, true> = {};
		for (const r of known) has[`${r.app.id}:${r.environment}`] = true;
		const candidates = (apps ?? []).flatMap((app) =>
			(['production', 'staging'] as Environment[])
				.filter((env) => app.latest_deployments[env] && !has[`${app.id}:${env}`])
				.map((env) => ({ app, env }))
		);
		let cancelled = false;
		Promise.all(
			candidates.map(({ app, env }) =>
				service
					.fetchVolumes(app.id, env)
					.then((v) =>
						v.volumes.length
							? ({
									app: { id: app.id, name: app.name, deleted: false },
									environment: env,
									freshness: 'never'
								} satisfies EnvironmentFreshness)
							: undefined
					)
					.catch(() => undefined)
			)
		).then((rows) => {
			if (!cancelled) never = rows.filter((r) => r !== undefined);
		});
		return () => {
			cancelled = true;
		};
	});

	const rows = $derived([...never, ...known]);

	const tiles: Record<Freshness, { icon: IconData; tone: TimelineTone }> = {
		fresh: { icon: CheckmarkCircle02Icon, tone: 'success' },
		quiet: { icon: Clock01Icon, tone: 'muted' },
		stale: { icon: Alert02Icon, tone: 'warning' },
		never: { icon: CancelCircleIcon, tone: 'danger' }
	};

	function detailsOf(row: EnvironmentFreshness): string {
		if (!row.lastGood) return l.translate('backups.fresh.never');
		const parts = [
			l.relative(row.lastGood.requested_at),
			row.lastGood.result ? formatBytes(row.lastGood.result.size) : undefined,
			storedLabel(row.lastGood).label,
			row.freshness === 'stale' ? l.translate('backups.fresh.stale') : undefined
		];
		return parts.filter(Boolean).join(' · ');
	}
</script>

{#if rows.length}
	<section class="flex flex-col gap-3" aria-labelledby="freshness-title">
		<SectionHeading id="freshness-title">
			{l.translate('backups.fresh.title')}
			{#snippet aside()}{l.translate('backups.fresh.note')}{/snippet}
		</SectionHeading>
		<TimelineList labelledby="freshness-title">
			{#each rows as row (`${row.app.id}:${row.environment}`)}
				{@const tile = tiles[row.freshness]}
				<TimelineRow
					icon={tile.icon}
					tone={tile.tone}
					label={[
						row.app.name,
						l.translate(`deploy.environment.${row.environment}`),
						detailsOf(row)
					].join(', ')}
					href={routes.appBackups(row.app.id)}
				>
					{#snippet title()}{row.app.name}{/snippet}
					{#snippet badge()}<EnvironmentChip environment={row.environment} />{/snippet}
					{#snippet details()}<span
							class={row.freshness === 'stale'
								? 'text-warning'
								: row.freshness === 'never'
									? 'text-danger'
									: undefined}>{detailsOf(row)}</span
						>{/snippet}
				</TimelineRow>
			{/each}
		</TimelineList>
		<p class="text-xs text-muted-foreground">{l.translate('backups.fresh.tones')}</p>
	</section>
{/if}
