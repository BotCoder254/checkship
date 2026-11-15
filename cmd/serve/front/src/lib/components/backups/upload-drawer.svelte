<!--
	Upload a backup (canvas artboard 3): a .tar.gz made by checkship, here or on another server. The
	server checks the manifest and the size; a refused file shows why in its row.
-->
<script lang="ts">
	import Cancel01Icon from '@hugeicons/core-free-icons/Cancel01Icon';
	import DashboardSquare01Icon from '@hugeicons/core-free-icons/DashboardSquare01Icon';
	import File01Icon from '@hugeicons/core-free-icons/File01Icon';
	import Upload04Icon from '@hugeicons/core-free-icons/Upload04Icon';
	import { toast } from 'svelte-sonner';
	import Drawer from '#lib/components/drawer.svelte';
	import DropZone from '#lib/components/drop-zone.svelte';
	import Icon from '#lib/components/icon.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { formatBytes, messageOf } from '#lib/backup.ts';
	import l from '#lib/localization/index.ts';
	import appsService from '#lib/resources/apps.ts';
	import service from '#lib/resources/backups.ts';
	import type { Environment } from '#lib/resources/deployments.ts';
	import { cn } from '#lib/utils.js';

	/** With `appId`, the backup goes to that application; without, the drawer asks which one. */
	type Props = { open: boolean; appId?: string };

	let { open = $bindable(false), appId }: Props = $props();

	const { data: apps } = appsService.queryAll();
	const candidates = $derived(($apps ?? []).filter((a) => !a.cleanup_requested_at));

	let chosenApp = $state('');
	let environment = $state<'' | Environment>('');
	let file = $state<File>();
	let error = $state<string>();
	let submitting = $state(false);

	$effect(() => {
		if (open) {
			chosenApp = appId ?? candidates[0]?.id ?? '';
			environment = '';
			file = undefined;
			error = undefined;
		}
	});

	const environments = $derived([
		{ value: '', label: l.translate('backups.upload.environment.manifest') },
		{ value: 'production', label: l.translate('deploy.environment.production') },
		{ value: 'staging', label: l.translate('deploy.environment.staging') }
	]);

	async function submit() {
		if (!file || !chosenApp || submitting) return;
		submitting = true;
		error = undefined;
		try {
			await service.upload(chosenApp, file, environment || undefined);
			toast.success(l.translate('backups.uploaded'));
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
	title={l.translate('backups.upload.title')}
	description={l.translate('backups.upload.description')}
>
	{#if !appId}
		<div class="flex flex-col gap-1.5">
			<span id="upload-app-label" class="text-sm font-medium"
				>{l.translate('backups.upload.app')}</span
			>
			<Select.Root type="single" bind:value={chosenApp}>
				<Select.Trigger aria-labelledby="upload-app-label" class="w-full min-w-0">
					<span class="truncate">{candidates.find((a) => a.id === chosenApp)?.name ?? ''}</span>
				</Select.Trigger>
				<Select.Content>
					{#each candidates as app (app.id)}
						<Select.Item value={app.id} label={app.name}>
							<Icon icon={DashboardSquare01Icon} class="text-muted-foreground" />
							{app.name}
						</Select.Item>
					{/each}
				</Select.Content>
			</Select.Root>
		</div>
	{/if}

	<div class="flex flex-col gap-1.5">
		<span id="upload-env-label" class="text-sm font-medium"
			>{l.translate('backups.upload.environment')}</span
		>
		<Select.Root type="single" bind:value={environment}>
			<Select.Trigger aria-labelledby="upload-env-label" class="w-full min-w-0">
				<span class="truncate">{environments.find((e) => e.value === environment)?.label}</span>
			</Select.Trigger>
			<Select.Content>
				{#each environments as env (env.value)}
					<Select.Item value={env.value} label={env.label}>
						<Icon icon={File01Icon} class="text-muted-foreground" />
						{env.label}
					</Select.Item>
				{/each}
			</Select.Content>
		</Select.Root>
	</div>

	{#if !file}
		<DropZone
			id="upload-backup"
			what={l.translate('backups.upload.what')}
			hint={l.translate('backups.upload.hint')}
			accept=".tar.gz,.tgz,application/gzip"
			onfile={(f) => {
				file = f;
				error = undefined;
			}}
		/>
	{:else}
		<div
			class={cn(
				'grid grid-cols-[36px_minmax(0,1fr)_auto] items-center gap-3 rounded-md py-2.5 pr-2.5 pl-3',
				error ? 'bg-danger-soft' : 'bg-sunken'
			)}
		>
			<span
				class="flex size-9 items-center justify-center rounded-md bg-raised text-muted-foreground"
			>
				<Icon icon={File01Icon} class="size-4.5" />
			</span>
			<span class="flex min-w-0 flex-col">
				<span class="truncate font-mono text-sm">{file.name}</span>
				<span class="text-xs" role={error ? 'alert' : undefined}>
					{#if error}{error}{:else}<span class="text-muted-foreground"
							>{formatBytes(file.size)}</span
						>{/if}
				</span>
			</span>
			<Button
				variant="ghost"
				size="icon"
				aria-label={l.translate('deploy.archive.remove')}
				onclick={() => {
					file = undefined;
					error = undefined;
				}}
			>
				<Icon icon={Cancel01Icon} />
			</Button>
		</div>
	{/if}

	{#snippet footer()}
		<Button variant="outline" onclick={() => (open = false)}>{l.translate('cancel')}</Button>
		<Button loading={submitting} disabled={!file || !chosenApp} onclick={submit}>
			<Icon icon={Upload04Icon} />
			{l.translate('backups.upload.submit')}
		</Button>
	{/snippet}
</Drawer>
