<!--
	Backup storage (canvas artboard 4; DESIGN.md §11.10 Storage): the status first (where copies
	go, whether it works, when the last copy left the server, how many are only on their target),
	then an "On this page" index and the settings. Where archives are copied besides the data folder. The
	built-in versitygw on a target (checkship sets it up, shows its state and endpoint), an S3
	server you run (presets fill the endpoint and region, Test connection writes, reads and deletes
	a small file), or the data folder only. Then whether the local copy is kept once stored.
-->
<script lang="ts">
	import CheckmarkCircle02Icon from '@hugeicons/core-free-icons/CheckmarkCircle02Icon';
	import ServerStack01Icon from '@hugeicons/core-free-icons/ServerStack01Icon';
	import TestTube01Icon from '@hugeicons/core-free-icons/TestTube01Icon';
	import Tick02Icon from '@hugeicons/core-free-icons/Tick02Icon';
	import { toast } from 'svelte-sonner';
	import FieldMessage from '#lib/components/field-message.svelte';
	import Icon from '#lib/components/icon.svelte';
	import SettingRow from '#lib/components/setting-row.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { Switch } from '#lib/components/ui/switch/index.js';
	import { errorText, messageOf } from '#lib/backup.ts';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service, {
		type BackupStorage,
		type ExternalStorage,
		type StorageMode
	} from '#lib/resources/backups.ts';
	import targetsService from '#lib/resources/targets.ts';
	import { storageSummary } from '#lib/storage.ts';
	import { appName, formatBytes } from '#lib/backup.ts';
	import Metric from '#lib/components/charts/metric.svelte';
	import SectionIndex from '#lib/components/section-index.svelte';
	import { cn } from '#lib/utils.js';

	const { data: storage } = service.queryStorage();
	const { data: targets } = targetsService.queryAll();
	const { data: backups } = service.queryAll();

	const lastStored = $derived(
		($backups ?? [])
			.filter((b) => b.status === 'succeeded' && b.stored)
			.sort((a, b) => b.requested_at.localeCompare(a.requested_at))[0]
	);
	const localOnly = $derived(
		($backups ?? []).filter((b) => b.status === 'succeeded' && !b.stored && !b.app.deleted)
	);

	let mode = $state<StorageMode>('local');
	let targetId = $state('');
	let keepLocal = $state(true);
	let external = $state<ExternalStorage>({
		endpoint: '',
		region: '',
		bucket: 'checkship-backups',
		prefix: '',
		access_key: '',
		secret_key: '',
		use_ssl: true,
		path_style: true
	});
	let loaded = false;
	let saving = $state(false);
	let testing = $state(false);
	let tested = $state(false);
	let error = $state<string>();

	// Fill the form once from the saved settings.
	$effect(() => {
		const s = $storage;
		if (!s || loaded) return;
		loaded = true;
		fill(s);
	});

	function fill(s: BackupStorage) {
		mode = s.mode;
		targetId = s.target_id ?? $targets?.[0]?.id ?? '';
		keepLocal = s.keep_local;
		if (s.mode === 'external') {
			external = {
				endpoint: s.endpoint,
				region: s.region,
				bucket: s.bucket,
				prefix: s.prefix,
				access_key: s.access_key,
				secret_key: '',
				use_ssl: s.use_ssl,
				path_style: s.path_style
			};
		}
	}

	$effect(() => {
		if (!targetId && $targets?.length) targetId = $targets[0].id;
	});

	const modes: { value: StorageMode; label: string; help: string }[] = [
		{
			value: 'builtin',
			label: l.translate('storage.mode.builtin'),
			help: l.translate('storage.mode.builtin.help')
		},
		{
			value: 'external',
			label: l.translate('storage.mode.external'),
			help: l.translate('storage.mode.external.help')
		},
		{
			value: 'local',
			label: l.translate('storage.mode.local'),
			help: l.translate('storage.mode.local.help')
		}
	];

	// Endpoint and region hints, from each project's documentation.
	const presets: { name: string; endpoint: string; region: string; ssl: boolean; path: boolean }[] =
		[
			{
				name: 'Garage',
				endpoint: 'garage.example.com:3900',
				region: 'garage',
				ssl: true,
				path: true
			},
			{
				name: 'SeaweedFS',
				endpoint: 'seaweed.example.com:8333',
				region: 'us-east-1',
				ssl: true,
				path: true
			},
			{
				name: 'RustFS',
				endpoint: 'rustfs.example.com:9000',
				region: 'us-east-1',
				ssl: true,
				path: true
			},
			{
				name: 'Ceph RGW',
				endpoint: 'rgw.example.com:7480',
				region: 'default',
				ssl: true,
				path: true
			},
			{
				name: 'AWS S3',
				endpoint: 's3.eu-west-3.amazonaws.com',
				region: 'eu-west-3',
				ssl: true,
				path: false
			},
			{
				name: 'Cloudflare R2',
				endpoint: '<account>.r2.cloudflarestorage.com',
				region: 'auto',
				ssl: true,
				path: true
			},
			{
				name: 'Backblaze B2',
				endpoint: 's3.eu-central-003.backblazeb2.com',
				region: 'eu-central-003',
				ssl: true,
				path: true
			}
		];

	function usePreset(p: (typeof presets)[number]) {
		external = {
			...external,
			endpoint: p.endpoint,
			region: p.region,
			use_ssl: p.ssl,
			path_style: p.path
		};
		tested = false;
	}

	const builtinState = $derived(
		$storage?.mode === 'builtin'
			? $storage.status === 'ready'
				? ('ready' as const)
				: $storage.status === 'failed'
					? ('failed' as const)
					: ('configuring' as const)
			: undefined
	);

	async function test() {
		testing = true;
		tested = false;
		error = undefined;
		try {
			await service.testStorage(external);
			tested = true;
		} catch (ex) {
			error = messageOf(ex);
		} finally {
			testing = false;
		}
	}

	async function save(event?: SubmitEvent) {
		event?.preventDefault();
		if (saving) return;
		saving = true;
		error = undefined;
		try {
			const saved = await service.updateStorage({
				mode,
				target_id: mode === 'builtin' ? targetId : undefined,
				external: mode === 'external' ? external : undefined,
				keep_local: keepLocal
			});
			await service.queryStorage();
			toast.success(l.translate('storage.saved'));
			loaded = false;
			fill(saved);
			loaded = true;
		} catch (ex) {
			error = messageOf(ex);
		} finally {
			saving = false;
		}
	}
</script>

<form class="flex flex-col" novalidate onsubmit={save}>
	<PageHeader
		title={l.translate('storage.title')}
		crumbs={[{ href: routes.backups, label: l.translate('breadcrumb.backups') }]}
	>
		{#snippet actions()}
			<Button type="submit" loading={saving}>
				<Icon icon={Tick02Icon} />
				{l.translate('save')}
			</Button>
		{/snippet}
		{#snippet status()}
			{#if $storage}<span>{storageSummary($storage, $targets ?? [])}</span>{/if}
		{/snippet}
	</PageHeader>

	{#if error}
		<p
			role="alert"
			class="mb-6 rounded-md bg-danger-soft px-3.5 py-3 text-sm font-semibold text-ink"
		>
			{error}
		</p>
	{/if}

	<!-- The status first (DESIGN.md §11.10 Storage): where copies go, whether it works, when the last
	     copy left the server, and how many backups are still only on their target. -->
	{#if $storage}
		<section
			aria-label={l.translate('storage.figures.label')}
			class="mb-12 grid grid-cols-2 gap-6 track p-4 sm:gap-x-10 sm:p-5 xl:grid-cols-4"
		>
			<Metric
				label={l.translate('storage.kpi.where')}
				value={l.translate(`storage.kpi.mode.${$storage.mode}`)}
				change={storageSummary($storage, $targets ?? [])}
			/>
			<Metric
				label={l.translate('storage.kpi.state')}
				value={$storage.mode === 'local' ? '—' : l.translate(`storage.state.${$storage.status}`)}
				change={$storage.mode === 'local'
					? l.translate('storage.kpi.state.local')
					: $storage.error_code
						? $storage.error_code
						: $storage.updated_at
							? l.translate('storage.kpi.state.since', [l.date($storage.updated_at)])
							: undefined}
			/>
			<Metric
				label={l.translate('storage.kpi.last')}
				value={$storage.mode !== 'local' && lastStored ? l.relative(lastStored.requested_at) : '—'}
				change={$storage.mode === 'local'
					? l.translate('storage.kpi.state.local')
					: lastStored
						? `${appName(lastStored)} · ${l.translate(`deploy.environment.${lastStored.environment}`)} · ${formatBytes(lastStored.result?.size ?? 0)}`
						: l.translate('storage.kpi.last.none')}
			/>
			<Metric
				label={l.translate('storage.kpi.local')}
				value={String(localOnly.length)}
				change={$storage.mode === 'local'
					? l.translate('storage.kpi.local.pick')
					: localOnly.length
						? l.translate('storage.kpi.local.which', [
								[...new Set(localOnly.map((b) => `${appName(b)} ${b.environment}`))]
									.slice(0, 3)
									.join(' · ')
							])
						: l.translate('storage.kpi.local.none')}
			/>
		</section>
	{/if}

	<div class="grid gap-10 xl:grid-cols-[180px_minmax(0,1fr)]">
		<SectionIndex
			items={[
				{ id: 'storage-where', label: l.translate('storage.where.title') },
				...(mode === 'builtin'
					? [{ id: 'storage-builtin', label: l.translate('storage.builtin.title') }]
					: []),
				...(mode === 'external'
					? [{ id: 'storage-external', label: l.translate('storage.external.title') }]
					: []),
				...(mode !== 'local'
					? [{ id: 'storage-copies', label: l.translate('storage.copies.title') }]
					: []),
				{ id: 'storage-images', label: l.translate('storage.images.title') }
			]}
		/>
		<div class="flex min-w-0 flex-col gap-9">
			<SettingsSection
				id="storage-where"
				title={l.translate('storage.where.title')}
				description={l.translate('storage.where.description')}
			>
				<div
					role="radiogroup"
					aria-label={l.translate('storage.where.title')}
					class="flex flex-col gap-2"
				>
					{#each modes as option (option.value)}
						{@const on = mode === option.value}
						<button
							type="button"
							role="radio"
							aria-checked={on}
							aria-label={option.label}
							aria-describedby="mode-{option.value}-help"
							onclick={() => {
								mode = option.value;
								tested = false;
							}}
							class={cn(
								'flex items-start gap-3 rounded-md bg-sunken px-4 py-3.5 text-left outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset',
								on && 'bg-accent-soft'
							)}
						>
							<span
								class={cn(
									'mt-0.5 size-4 shrink-0 rounded-full border border-input',
									on && 'border-[5px] border-primary'
								)}
							></span>
							<span class="flex flex-col gap-1">
								<span class="text-sm font-semibold">{option.label}</span>
								<span id="mode-{option.value}-help" class="text-xs/[18px] text-muted-foreground"
									>{option.help}</span
								>
							</span>
						</button>
					{/each}
				</div>
			</SettingsSection>

			{#if mode === 'builtin'}
				<SettingsSection
					id="storage-builtin"
					title={l.translate('storage.builtin.title')}
					description={l.translate('storage.builtin.description')}
				>
					{#if !$targets?.length}
						<p class="text-sm text-muted-foreground">
							{l.translate('storage.builtin.no_target')}
							<a href={routes.createTarget} class="text-primary hover:underline"
								>{l.translate('settings.create_target')}</a
							>
						</p>
					{:else}
						<div class="flex flex-col gap-1.5">
							<span id="storage-target-label" class="text-sm font-medium"
								>{l.translate('storage.builtin.target')}</span
							>
							<Select.Root type="single" bind:value={targetId}>
								<Select.Trigger aria-labelledby="storage-target-label" class="w-full min-w-0">
									<span class="truncate">{$targets.find((t) => t.id === targetId)?.name ?? ''}</span
									>
								</Select.Trigger>
								<Select.Content>
									{#each $targets as target (target.id)}
										<Select.Item value={target.id} label={target.name}>
											<Icon icon={ServerStack01Icon} class="text-muted-foreground" />
											{target.name}
										</Select.Item>
									{/each}
								</Select.Content>
							</Select.Root>
							<p class="text-xs text-muted-foreground">
								{l.translate('storage.builtin.target.help')}
							</p>
						</div>

						{#if $storage?.mode === 'builtin'}
							<dl
								class="grid grid-cols-[120px_minmax(0,1fr)] items-center gap-x-3 gap-y-2.5 text-sm"
							>
								<dt class="text-muted-foreground">{l.translate('storage.state')}</dt>
								<dd class="flex flex-wrap items-center gap-2">
									{#if builtinState}
										<StatusBadge
											state={builtinState}
											label={l.translate(`storage.state.${builtinState}`)}
										/>
									{/if}
									{#if $storage.error_code}
										<span class="text-xs text-danger">{errorText($storage.error_code)}</span>
									{/if}
								</dd>
								<dt class="text-muted-foreground">{l.translate('storage.endpoint')}</dt>
								<dd class="truncate font-mono text-xs">{$storage.endpoint || '—'}</dd>
								<dt class="text-muted-foreground">{l.translate('storage.bucket')}</dt>
								<dd class="font-mono text-xs">{$storage.bucket}</dd>
								<dt class="text-muted-foreground">{l.translate('storage.access_key')}</dt>
								<dd class="truncate font-mono text-xs">{$storage.access_key}</dd>
							</dl>
						{/if}
					{/if}
				</SettingsSection>
			{/if}

			{#if mode === 'external'}
				<SettingsSection
					id="storage-external"
					title={l.translate('storage.external.title')}
					description={l.translate('storage.external.description')}
				>
					<div class="flex flex-wrap gap-1.5" role="group" aria-label="Presets">
						{#each presets as preset (preset.name)}
							<Button variant="outline" size="sm" onclick={() => usePreset(preset)}
								>{preset.name}</Button
							>
						{/each}
					</div>
					<div class="grid gap-4 sm:grid-cols-2">
						<div class="flex flex-col gap-1.5">
							<label for="s3-endpoint" class="text-sm font-medium"
								>{l.translate('storage.endpoint')}</label
							>
							<Input
								id="s3-endpoint"
								class="font-mono"
								bind:value={external.endpoint}
								autocomplete="off"
							/>
							<FieldMessage id="s3-endpoint-help"
								>{#snippet help()}{l.translate('storage.endpoint.help')}{/snippet}</FieldMessage
							>
						</div>
						<div class="flex flex-col gap-1.5">
							<label for="s3-region" class="text-sm font-medium"
								>{l.translate('storage.region')}</label
							>
							<Input
								id="s3-region"
								class="font-mono"
								bind:value={external.region}
								autocomplete="off"
							/>
						</div>
						<div class="flex flex-col gap-1.5">
							<label for="s3-bucket" class="text-sm font-medium"
								>{l.translate('storage.bucket')}</label
							>
							<Input
								id="s3-bucket"
								class="font-mono"
								bind:value={external.bucket}
								autocomplete="off"
							/>
							<FieldMessage id="s3-bucket-help"
								>{#snippet help()}{l.translate('storage.bucket.help')}{/snippet}</FieldMessage
							>
						</div>
						<div class="flex flex-col gap-1.5">
							<label for="s3-prefix" class="text-sm font-medium"
								>{l.translate('storage.prefix')}</label
							>
							<Input
								id="s3-prefix"
								class="font-mono"
								bind:value={external.prefix}
								autocomplete="off"
							/>
							<FieldMessage id="s3-prefix-help"
								>{#snippet help()}{l.translate('storage.prefix.help')}{/snippet}</FieldMessage
							>
						</div>
						<div class="flex flex-col gap-1.5">
							<label for="s3-access" class="text-sm font-medium"
								>{l.translate('storage.access_key')}</label
							>
							<Input
								id="s3-access"
								class="font-mono"
								bind:value={external.access_key}
								autocomplete="off"
							/>
						</div>
						<div class="flex flex-col gap-1.5">
							<label for="s3-secret" class="text-sm font-medium"
								>{l.translate('storage.secret_key')}</label
							>
							<Input
								id="s3-secret"
								type="password"
								class="font-mono"
								bind:value={external.secret_key}
								autocomplete="new-password"
							/>
							{#if $storage?.mode === 'external'}
								<FieldMessage id="s3-secret-help"
									>{#snippet help()}{l.translate('storage.secret_key.keep')}{/snippet}</FieldMessage
								>
							{/if}
						</div>
					</div>
					<SettingRow for="s3-ssl" label={l.translate('storage.use_ssl')}>
						<Switch id="s3-ssl" bind:checked={external.use_ssl} />
					</SettingRow>
					<SettingRow
						for="s3-path"
						label={l.translate('storage.path_style')}
						description={l.translate('storage.path_style.help')}
					>
						<Switch id="s3-path" bind:checked={external.path_style} />
					</SettingRow>
					<div class="flex flex-wrap items-center gap-3">
						<Button variant="outline" loading={testing} onclick={test}>
							<Icon icon={TestTube01Icon} />
							{l.translate('storage.test')}
						</Button>
						{#if tested}
							<span role="status" class="flex items-center gap-2 text-sm">
								<Icon icon={CheckmarkCircle02Icon} class="size-4 text-success" />
								{l.translate('storage.test.ok')}
							</span>
						{/if}
					</div>
				</SettingsSection>
			{/if}

			{#if mode !== 'local'}
				<SettingsSection
					id="storage-copies"
					title={l.translate('storage.copies.title')}
					description={l.translate('storage.copies.description')}
				>
					<SettingRow
						for="keep-local"
						label={l.translate('storage.keep_local')}
						description={l.translate('storage.keep_local.help')}
					>
						<Switch id="keep-local" bind:checked={keepLocal} />
					</SettingRow>
				</SettingsSection>
			{/if}

			{#if $storage}
				<SettingsSection
					id="storage-images"
					title={l.translate('storage.images.title')}
					description={l.translate('storage.images.description')}
				>
					<dl class="flex flex-col gap-4">
						{#if mode === 'builtin'}
							<div class="flex flex-col gap-1">
								<dt class="text-sm font-medium">{l.translate('storage.images.storage')}</dt>
								<dd class="font-mono text-xs">{$storage.image}</dd>
								<dd class="text-xs/[18px] text-muted-foreground">
									{l.translate('storage.images.storage.help')}
								</dd>
							</div>
						{/if}
						<div class="flex flex-col gap-1">
							<dt class="text-sm font-medium">{l.translate('storage.images.helper')}</dt>
							<dd class="font-mono text-xs">{$storage.helper_image}</dd>
							<dd class="text-xs/[18px] text-muted-foreground">
								{l.translate('storage.images.helper.help')}
							</dd>
						</div>
					</dl>
				</SettingsSection>
			{/if}
		</div>
	</div>
</form>
