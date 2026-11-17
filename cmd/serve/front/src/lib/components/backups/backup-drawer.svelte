<!--
	Back up now (canvas artboard 1): the environment, its volumes read live from the target (all
	ticked), and whether containers stop during the copy (on by default). Says where the archive
	goes, with a link to the storage settings.
-->
<script lang="ts">
	import Archive02Icon from '@hugeicons/core-free-icons/Archive02Icon';
	import { toast } from 'svelte-sonner';
	import Drawer from '#lib/components/drawer.svelte';
	import Icon from '#lib/components/icon.svelte';
	import Segmented from '#lib/components/segmented.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Checkbox } from '#lib/components/ui/checkbox/index.js';
	import { Switch } from '#lib/components/ui/switch/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { messageOf } from '#lib/backup.ts';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service, { type BackupStorage, type Volume } from '#lib/resources/backups.ts';
	import type { Environment } from '#lib/resources/deployments.ts';
	import EmptyState from '#lib/components/empty-state.svelte';

	type Props = {
		open: boolean;
		app: { id: string; name: string };
		environment?: Environment;
		storage?: BackupStorage;
		storageLabel?: string;
	};

	let {
		open = $bindable(false),
		app,
		environment = 'production',
		storage,
		storageLabel
	}: Props = $props();

	let env = $state<Environment>('production');
	let volumes = $state<Volume[]>();
	let loadError = $state<string>();
	let unticked = $state<string[]>([]);
	let stop = $state(true);
	let submitting = $state(false);
	let error = $state<string>();

	const environments = [
		{ value: 'production' as const, label: l.translate('deploy.environment.production') },
		{ value: 'staging' as const, label: l.translate('deploy.environment.staging') }
	];

	// Start from the environment asked for each time the drawer opens.
	$effect(() => {
		if (open) {
			env = environment;
			unticked = [];
			stop = true;
			error = undefined;
		}
	});

	// Read the volumes of the chosen environment.
	$effect(() => {
		if (!open) return;
		const current = env;
		volumes = undefined;
		loadError = undefined;
		service
			.fetchVolumes(app.id, current, { cache: 'no-store' })
			.then((v) => {
				if (current === env) volumes = v.volumes;
			})
			.catch((ex) => {
				if (current === env) loadError = messageOf(ex);
			});
	});

	const selected = $derived(
		(volumes ?? []).filter((v) => !unticked.includes(v.key)).map((v) => v.key)
	);

	function toggle(key: string, on: boolean) {
		unticked = on ? unticked.filter((k) => k !== key) : [...unticked, key];
	}

	async function submit(event?: SubmitEvent) {
		event?.preventDefault();
		if (submitting || !selected.length) return;
		submitting = true;
		error = undefined;
		try {
			// Every volume: send none, so volumes added later are included too.
			const all = selected.length === volumes?.length;
			await service.request(app.id, {
				environment: env,
				volumes: all ? [] : selected,
				stop_containers: stop
			});
			toast.success(l.translate('backups.requested'));
			open = false;
		} catch (ex) {
			error = messageOf(ex);
		} finally {
			submitting = false;
		}
	}
</script>

<Drawer
	bind:open
	title={l.translate('backups.drawer.title', [app.name])}
	description={l.translate('backups.drawer.description')}
>
	<form id="backup-form" class="flex flex-col gap-6" novalidate onsubmit={submit}>
		<div class="flex flex-col gap-1.5">
			<span id="backup-env-label" class="text-sm font-medium"
				>{l.translate('backups.drawer.environment')}</span
			>
			<Segmented bind:value={env} items={environments} labelledby="backup-env-label" />
		</div>

		<fieldset class="flex flex-col gap-1.5">
			<legend class="pb-1.5 text-sm font-medium">{l.translate('backups.drawer.volumes')}</legend>
			{#if loadError}
				<p role="alert" class="rounded-md bg-danger-soft px-3 py-2.5 text-sm text-ink">
					{loadError}
				</p>
			{:else if !volumes}
				<Skeleton class="h-14 w-full" />
				<Skeleton class="h-14 w-full" />
			{:else if volumes.length === 0}
				<EmptyState
					illustration="volumes"
					size="compact"
					level={3}
					title={l.translate('backups.volumes.empty.title')}
					description={l.translate('backups.drawer.no_volumes')}
				/>
			{:else}
				{#each volumes as volume (volume.key)}
					<label
						class="-mx-2 flex min-h-12 cursor-pointer items-center gap-3 rounded-md px-2 py-1.5 hover:bg-hover"
					>
						<Checkbox
							checked={!unticked.includes(volume.key)}
							onCheckedChange={(on) => toggle(volume.key, on === true)}
							aria-label={volume.key}
						/>
						<span class="flex min-w-0 flex-col gap-0.5">
							<span class="truncate font-mono text-label">{volume.key}</span>
							<span class="text-xs text-muted-foreground">
								{l.translate('backups.drawer.used_by', [volume.services.join(', ') || '—'])}
							</span>
						</span>
					</label>
				{/each}
			{/if}
		</fieldset>

		<div class="flex min-h-[54px] items-start gap-4 py-2">
			<span class="flex min-w-0 flex-1 flex-col gap-1">
				<label for="backup-stop" class="text-sm font-medium"
					>{l.translate('backups.drawer.stop')}</label
				>
				<span class="text-xs/[18px] text-muted-foreground">
					{stop ? l.translate('backups.drawer.stop.on') : l.translate('backups.drawer.stop.off')}
				</span>
			</span>
			<Switch id="backup-stop" bind:checked={stop} />
		</div>

		<div class="flex flex-col gap-1">
			<span class="text-sm font-medium">{l.translate('backups.drawer.saved_to')}</span>
			<p class="text-sm text-muted-foreground">
				{storage && storage.mode !== 'local' && storageLabel
					? l.translate('backups.drawer.saved_to.storage', [storageLabel])
					: l.translate('backups.drawer.saved_to.local')}
				<a href={routes.backupStorage} class="text-primary hover:underline"
					>{l.translate('backups.drawer.change')}</a
				>
			</p>
		</div>

		{#if error}
			<p role="alert" class="rounded-md bg-danger-soft px-3 py-2.5 text-sm font-semibold text-ink">
				{error}
			</p>
		{/if}
	</form>

	{#snippet footer()}
		<Button variant="outline" onclick={() => (open = false)}>{l.translate('cancel')}</Button>
		<Button type="submit" form="backup-form" loading={submitting} disabled={!selected.length}>
			<Icon icon={Archive02Icon} />
			{l.translate('backups.drawer.submit', [selected.length])}
		</Button>
	{/snippet}
</Drawer>
