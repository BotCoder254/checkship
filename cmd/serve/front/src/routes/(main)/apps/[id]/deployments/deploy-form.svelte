<!--
	Queue a deployment (DESIGN.md §11.6): environment and source as segmented controls, then the
	fields for that source. Used inside the New deployment drawer on the application page and on
	/apps/:id/deployments/new, so the form has an id and its submit button can live outside it (the
	drawer footer, the page header).
-->
<script lang="ts">
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import ArrowDown01Icon from '@hugeicons/core-free-icons/ArrowDown01Icon';
	import Cancel01Icon from '@hugeicons/core-free-icons/Cancel01Icon';
	import CheckmarkCircle02Icon from '@hugeicons/core-free-icons/CheckmarkCircle02Icon';
	import CommandLineIcon from '@hugeicons/core-free-icons/CommandLineIcon';
	import File01Icon from '@hugeicons/core-free-icons/File01Icon';
	import FileUploadIcon from '@hugeicons/core-free-icons/FileUploadIcon';
	import GitBranchIcon from '@hugeicons/core-free-icons/GitBranchIcon';
	import Upload04Icon from '@hugeicons/core-free-icons/Upload04Icon';
	import CodeBlock from '#lib/components/code-block.svelte';
	import CodeEditor from '#lib/components/code-editor.svelte';
	import DropZone from '#lib/components/drop-zone.svelte';
	import FieldMessage from '#lib/components/field-message.svelte';
	import Icon from '#lib/components/icon.svelte';
	import Segmented from '#lib/components/segmented.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Collapsible from '#lib/components/ui/collapsible/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { buildCommand, type CurlPayload } from '#lib/curl.ts';
	import { buildFormData, GLOBAL_ERROR_NAME, submitter } from '#lib/form.ts';
	import l, { type AppTranslationsString } from '#lib/localization/index.ts';
	import type { AppDetail } from '#lib/resources/apps.ts';
	import service, {
		type Deployment,
		type Environment,
		type QueueDeployment,
		type SourceDataDiscriminator
	} from '#lib/resources/deployments.ts';
	import select from '#lib/select.ts';
	import { cn } from '#lib/utils.js';

	type Props = {
		app: AppDetail;
		apiKey: string;
		id: string;
		environment?: Environment;
		/** Source to start on; defaults to Git when the application has a repository. */
		kind?: SourceDataDiscriminator;
		loading?: boolean;
		onqueued: (deployment: Deployment) => unknown;
	};

	let {
		app,
		apiKey,
		id,
		environment: initialEnvironment = 'production',
		kind: initialKind,
		loading = $bindable(false),
		onqueued
	}: Props = $props();

	// Set once when the form opens; switching environments below doesn't reset what was typed.
	// svelte-ignore state_referenced_locally
	let environment = $state<Environment>(initialEnvironment);
	// svelte-ignore state_referenced_locally
	let kind = $state<SourceDataDiscriminator>(
		initialKind && (initialKind !== 'git' || app.version_control)
			? initialKind
			: app.version_control
				? 'git'
				: 'raw'
	);
	let branch = $state(defaultBranch());
	let hash = $state('');
	let raw = $state('');
	let rawFile = $state<string>();
	let archive = $state<File>();
	let archiveError = $state<string>();
	let curlOpen = $state(false);

	function defaultBranch() {
		const source = app.latest_deployments[initialEnvironment]?.source;
		return source?.discriminator === 'git' ? source.data.branch : '';
	}

	const target = $derived(app[environment].target);
	const kinds = $derived(
		[
			{ value: 'git' as const, label: l.translate('deploy.source.git'), icon: GitBranchIcon },
			{ value: 'raw' as const, label: l.translate('deploy.source.raw'), icon: File01Icon },
			{ value: 'archive' as const, label: l.translate('deploy.source.archive'), icon: Upload04Icon }
		].filter((k) => k.value !== 'git' || !!app.version_control)
	);
	const environments = [
		{ value: 'production' as const, label: l.translate('deploy.environment.production') },
		{ value: 'staging' as const, label: l.translate('deploy.environment.staging') }
	];

	async function readCompose(file: File) {
		raw = await file.text();
		rawFile = file.name;
	}

	function pickArchive(file: File) {
		if (/\.(tar\.gz|tgz)$/i.test(file.name)) {
			archive = file;
			archiveError = undefined;
		} else {
			archive = undefined;
			archiveError = file.name;
		}
	}

	const sizeOf = (bytes: number) =>
		bytes < 1024 * 1024
			? `${Math.max(1, Math.round(bytes / 1024))} KB`
			: `${(bytes / 1024 / 1024).toFixed(1)} MB`;

	const {
		submit,
		loading: submitting,
		errors
	} = submitter(async () => {
		const payload = select<SourceDataDiscriminator, () => QueueDeployment>(kind, {
			archive: () => buildFormData({ environment, archive: archive! }),
			git: () => ({ environment, git: { branch, hash: hash || undefined } }),
			raw: () => ({ environment, raw })
		})!();

		return onqueued(await service.queue(app.id, payload));
	});

	$effect(() => {
		loading = $submitting;
	});

	const fieldError = (key: string) => $errors?.[key] ?? undefined;
	const globalError = $derived($errors?.[GLOBAL_ERROR_NAME]);
	let missingArchive = $state(false);

	function onsubmit(event: SubmitEvent) {
		event.preventDefault();
		if ($submitting) return;
		if (kind === 'archive' && !archive) {
			missingArchive = true;
			return;
		}
		missingArchive = false;
		submit().catch(() => {});
	}

	const curl = $derived(
		curlOpen
			? buildCommand({
					apiKey,
					appId: app.id,
					environment,
					origin: location.origin,
					payload: select<SourceDataDiscriminator, CurlPayload>(kind, {
						raw: { kind: 'raw', raw },
						git: { kind: 'git', branch, hash: hash || undefined },
						archive: { kind: 'archive', filename: archive?.name }
					})!
				})
			: ''
	);
	const lines = $derived(raw ? raw.split('\n').length : 0);
</script>

<form {id} class="flex flex-col gap-6" novalidate {onsubmit}>
	{#if globalError}
		<Alert.Root variant="destructive">
			<Icon icon={Alert02Icon} />
			<Alert.Title class="font-semibold">
				{l.translate(globalError as AppTranslationsString)}
			</Alert.Title>
		</Alert.Root>
	{/if}

	<div class="flex flex-col gap-1.5">
		<span id="{id}-env-label" class="text-sm font-medium"
			>{l.translate('deployment.environment')}</span
		>
		<Segmented
			bind:value={environment}
			items={environments}
			labelledby="{id}-env-label"
			describedby="{id}-env-help"
		/>
		<p id="{id}-env-help" class="text-sm text-muted-foreground">
			{l.translate('deploy.environment.help', [target.name])}
			{#if target.url}
				· <span class="font-mono text-label">{target.url}</span>
			{/if}
		</p>
	</div>

	<div class="flex flex-col gap-1.5">
		<span id="{id}-kind-label" class="text-sm font-medium">{l.translate('deployment.source')}</span>
		<Segmented bind:value={kind} items={kinds} labelledby="{id}-kind-label" />
	</div>

	{#if kind === 'git'}
		<div class="grid gap-4 sm:grid-cols-2">
			<div class="flex flex-col gap-1.5">
				<Label for="{id}-branch">{l.translate('deployment.payload.vcs.branch')}</Label>
				<Input
					id="{id}-branch"
					bind:value={branch}
					required
					class="font-mono text-label"
					aria-invalid={!!fieldError('git.branch') || undefined}
					aria-describedby="{id}-branch-message"
				/>
				<FieldMessage id="{id}-branch-message" error={fieldError('git.branch')}>
					{#snippet help()}{l.translate('deploy.branch.help', [
							app.version_control?.url.replace(/^https?:\/\//, '') ?? ''
						])}{/snippet}
				</FieldMessage>
			</div>
			<div class="flex flex-col gap-1.5">
				<Label for="{id}-hash">{l.translate('deployment.payload.vcs.commit')}</Label>
				<Input
					id="{id}-hash"
					bind:value={hash}
					placeholder={l.translate('deploy.commit.placeholder')}
					class="font-mono text-label"
					aria-invalid={!!fieldError('git.hash') || undefined}
					aria-describedby="{id}-hash-message"
				/>
				<FieldMessage id="{id}-hash-message" error={fieldError('git.hash')}>
					{#snippet help()}{l.translate('deploy.commit.help')}{/snippet}
				</FieldMessage>
			</div>
		</div>
	{:else if kind === 'raw'}
		<div class="flex flex-col gap-2.5">
			<Label for="{id}-raw">{l.translate('deploy.source.raw')}</Label>
			<DropZone
				compact
				icon={FileUploadIcon}
				what={l.translate('deploy.compose.what')}
				hint={l.translate('deploy.compose.hint')}
				accept=".yml,.yaml,application/yaml,text/yaml"
				onfile={readCompose}
			/>
			{#if rawFile}
				<div class="flex items-center justify-between gap-3 text-label/[18px]">
					<span class="flex min-w-0 items-center gap-1.5 text-muted-foreground">
						<Icon icon={CheckmarkCircle02Icon} class="size-3.5 text-success" />
						{l.translate('deploy.compose.loaded_from')}
						<span class="truncate font-mono text-ink">{rawFile}</span>
						· {l.translate('deploy.compose.lines', [lines])}
					</span>
					<Button
						variant="ghost"
						size="sm"
						onclick={() => {
							raw = '';
							rawFile = undefined;
						}}
					>
						<Icon icon={Cancel01Icon} />
						{l.translate('deploy.compose.clear')}
					</Button>
				</div>
			{/if}
			<CodeEditor
				id="{id}-raw"
				lang="yaml"
				rows={12}
				bind:value={raw}
				required
				placeholder={l.translate('deploy.compose.placeholder')}
				aria-invalid={!!fieldError('raw') || !!fieldError('raw.content') || undefined}
				aria-describedby="{id}-raw-message"
			/>
			<FieldMessage id="{id}-raw-message" error={fieldError('raw') ?? fieldError('raw.content')}>
				{#snippet help()}{l.translate('deploy.compose.help')}{/snippet}
			</FieldMessage>
		</div>
	{:else}
		<div class="flex flex-col gap-2.5">
			<span id="{id}-archive-label" class="text-sm font-medium"
				>{l.translate('deploy.source.archive')}</span
			>
			{#if !archive}
				<DropZone
					what={l.translate('deploy.archive.what')}
					hint={l.translate('deploy.archive.hint')}
					accept=".tar.gz,.tgz,application/gzip,application/x-gzip"
					onfile={pickArchive}
					describedby="{id}-archive-message"
				/>
			{/if}
			{#if archive || archiveError}
				<div
					class={cn(
						'grid grid-cols-[36px_minmax(0,1fr)_auto] items-center gap-3 rounded-md py-2.5 pr-2.5 pl-3',
						archiveError ? 'bg-danger-soft' : 'bg-sunken'
					)}
				>
					<span
						class={cn(
							'flex size-9 items-center justify-center rounded-md',
							archiveError ? 'bg-sunken text-danger' : 'bg-raised text-muted-foreground'
						)}
					>
						<Icon icon={File01Icon} class="size-4.5" />
					</span>
					<span class="flex min-w-0 flex-col">
						<span class="truncate font-mono text-sm">{archive?.name ?? archiveError}</span>
						<span class="text-xs" role={archiveError ? 'alert' : undefined}>
							{#if archive}
								<span class="text-muted-foreground">{sizeOf(archive.size)}</span>
							{:else}
								{l.translate('deploy.archive.invalid')}
							{/if}
						</span>
					</span>
					<Button
						variant="ghost"
						size="icon"
						aria-label={l.translate('deploy.archive.remove')}
						onclick={() => {
							archive = undefined;
							archiveError = undefined;
						}}
					>
						<Icon icon={Cancel01Icon} />
					</Button>
				</div>
			{/if}
			<FieldMessage
				id="{id}-archive-message"
				error={missingArchive ? 'deploy.archive.missing' : fieldError('archive')}
			/>
		</div>
	{/if}

	<Collapsible.Root bind:open={curlOpen} class="flex flex-col gap-2.5">
		<Collapsible.Trigger
			class="-ml-2 flex h-8 items-center gap-2 self-start rounded-md px-2 text-sm font-semibold text-primary outline-none hover:bg-hover focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset"
		>
			<Icon icon={CommandLineIcon} class="size-4" />
			{l.translate('deploy.curl')}
			<Icon
				icon={ArrowDown01Icon}
				class={cn('size-4 transition-transform', curlOpen && 'rotate-180')}
			/>
		</Collapsible.Trigger>
		<Collapsible.Content class="flex flex-col gap-2.5">
			<p class="text-label text-muted-foreground">{l.translate('deploy.curl.help')}</p>
			<CodeBlock code={curl} lang="shellscript" copy label={l.translate('deploy.curl')} />
		</Collapsible.Content>
	</Collapsible.Root>
</form>
