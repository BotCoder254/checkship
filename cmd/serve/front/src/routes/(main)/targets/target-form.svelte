<!--
	Target form (DESIGN.md §11.8), in two columns, following the server's rules
	(internal/deployment/infra/provider/docker/data.go, domain/target.go):
	- one target per Docker engine: "This machine" is offered only when no local target exists;
	- a remote target's host identifies it and can't change, so Settings shows it read-only;
	- user defaults to docker and port to 22; the private key is optional and masked once saved;
	- a URL means automatic exposure (root only, unique among targets), none means manual.
	The page header is inside the form so its submit button submits.
-->
<script lang="ts">
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import Globe02Icon from '@hugeicons/core-free-icons/Globe02Icon';
	import Key01Icon from '@hugeicons/core-free-icons/Key01Icon';
	import ServerStack01Icon from '@hugeicons/core-free-icons/ServerStack01Icon';
	import type { Snippet } from 'svelte';
	import CodeEditor from '#lib/components/code-editor.svelte';
	import DropZone from '#lib/components/drop-zone.svelte';
	import FieldMessage from '#lib/components/field-message.svelte';
	import Icon from '#lib/components/icon.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { Switch } from '#lib/components/ui/switch/index.js';
	import { GLOBAL_ERROR_NAME, submitter } from '#lib/form.ts';
	import l, { type AppTranslationsString } from '#lib/localization/index.ts';
	import type { CreateTarget, Target, UpdateTarget } from '#lib/resources/targets.ts';
	import { DOCKER_DEFAULT_PORT, DOCKER_DEFAULT_USER } from '#lib/target.ts';
	import { cn } from '#lib/utils.js';

	type Props = {
		title: string;
		crumbs: { href: string; label: string }[];
		/** Header actions; gets whether the form is submitting. */
		actions: Snippet<[boolean]>;
		/** Shown under the header (the target tabs in Settings). */
		tabs?: Snippet;
		disabled?: boolean;
	} & (
		| { initialData: Target; handler: (data: UpdateTarget) => Promise<unknown> }
		| {
				initialData?: undefined;
				handler: (data: CreateTarget) => Promise<unknown>;
				/** Name of the existing local target: this machine can't be added twice. */
				localTarget?: string;
		  }
	);

	let props: Props = $props();
	// The form is created once per page; later prop changes don't reset what the person typed.
	// svelte-ignore state_referenced_locally
	const initialData = props.initialData;
	// svelte-ignore state_referenced_locally
	const localTarget = 'localTarget' in props ? props.localTarget : undefined;

	let name = $state(initialData?.name ?? '');
	let url = $state(initialData?.url ?? '');
	let exposeAutomatically = $state(initialData ? !!initialData.url : true);
	let isRemote = $state(initialData ? !!initialData.provider.data.host : !!localTarget);
	let host = $state(initialData?.provider.data.host ?? '');
	let user = $state(initialData?.provider.data.user ?? '');
	let port = $state(initialData?.provider.data.port?.toString() ?? '');
	// Never sent back by the server (masked); typing a new key replaces it.
	let privateKey = $state('');
	const hasSavedKey = !!initialData?.provider.data.private_key;
	let keyFile = $state<string>();

	const { submit, loading, errors } = submitter(() => {
		const remote = {
			host: host || undefined,
			user: user || undefined,
			port: parseInt(port) || undefined
		};

		if (!props.initialData) {
			return props.handler({
				name,
				url: exposeAutomatically ? url : undefined,
				docker: isRemote ? { ...remote, private_key: privateKey || undefined } : {}
			});
		}

		return props.handler({
			name,
			url: exposeAutomatically ? (props.initialData.url !== url ? url : undefined) : null,
			docker: isRemote ? { ...remote, private_key: privateKey ? privateKey : undefined } : undefined
		});
	});

	const fieldError = (key: string) => $errors?.[key] ?? undefined;
	const globalError = $derived($errors?.[GLOBAL_ERROR_NAME]);

	async function readKey(file: File) {
		privateKey = (await file.text()).trim();
		keyFile = file.name;
	}

	// The URLs applications will get, from the root URL typed so far.
	const preview = $derived.by(() => {
		try {
			const u = new URL(url);
			return { app: `${u.protocol}//<app>.${u.host}`, dns: `*.${u.hostname}` };
		} catch {
			return undefined;
		}
	});

	function onsubmit(event: SubmitEvent) {
		event.preventDefault();
		if ($loading) return;
		submit().catch(() => {});
	}
</script>

<form class="flex flex-col" novalidate {onsubmit}>
	<PageHeader title={props.title} crumbs={props.crumbs}>
		{#snippet actions()}
			{@render props.actions($loading)}
		{/snippet}
	</PageHeader>
	{#if props.tabs}
		<div class="-mt-3 mb-7">{@render props.tabs()}</div>
	{/if}

	<fieldset class="flex flex-col gap-10" disabled={props.disabled}>
		{#if globalError}
			<Alert.Root variant="destructive">
				<Icon icon={Alert02Icon} />
				<Alert.Title class="font-semibold">
					{l.translate(globalError as AppTranslationsString)}
				</Alert.Title>
			</Alert.Root>
		{/if}

		<SettingsSection
			title={l.translate('settings.general')}
			description={initialData
				? l.translate('targets.section.general.edit')
				: l.translate('targets.section.general.new')}
		>
			<div class="flex flex-col gap-1.5">
				<Label for="target-name">{l.translate('name')}</Label>
				<Input
					id="target-name"
					bind:value={name}
					required
					autofocus={!initialData}
					aria-invalid={!!fieldError('name') || undefined}
					aria-describedby="target-name-message"
				/>
				<FieldMessage id="target-name-message" error={fieldError('name')} />
			</div>
		</SettingsSection>

		<SettingsSection
			title={initialData ? l.translate('targets.connection') : l.translate('targets.where')}
			description={initialData
				? l.translate('targets.section.connection.edit')
				: l.translate('targets.section.where')}
		>
			{#if !initialData}
				<div
					role="radiogroup"
					aria-label={l.translate('targets.where')}
					class="grid gap-2 sm:grid-cols-2"
				>
					{#each [false, true] as remote (remote)}
						{@const selected = isRemote === remote}
						{@const unavailable = !remote && !!localTarget}
						<button
							type="button"
							role="radio"
							aria-checked={selected}
							disabled={unavailable}
							class={cn(
								'grid grid-cols-[36px_minmax(0,1fr)] items-start gap-3 rounded-lg bg-sunken p-3.5 text-left outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset disabled:opacity-60',
								selected && 'bg-accent-soft ring-2 ring-primary ring-inset'
							)}
							onclick={() => (isRemote = remote)}
						>
							<span
								class={cn(
									'flex size-9 items-center justify-center rounded-lg bg-raised',
									selected ? 'text-primary' : 'text-muted-foreground'
								)}
							>
								<Icon icon={remote ? Globe02Icon : ServerStack01Icon} class="size-4.5" />
							</span>
							<span class="flex flex-col gap-0.5">
								<strong class="text-sm font-semibold">
									{remote ? l.translate('targets.where.ssh') : l.translate('targets.where.local')}
								</strong>
								<span class="text-[12.5px]/[18px] text-muted-foreground">
									{remote
										? l.translate('targets.where.ssh.help')
										: unavailable
											? l.translate('targets.where.local.taken', [localTarget ?? ''])
											: l.translate('targets.where.local.help')}
								</span>
							</span>
						</button>
					{/each}
				</div>
			{:else if !isRemote}
				<p class="rounded-md bg-sunken px-4 py-3.5 text-sm">{l.translate('targets.kind.local')}</p>
			{/if}

			{#if isRemote}
				<div class="grid gap-4 md:grid-cols-[minmax(0,1fr)_150px_110px]">
					<div class="flex flex-col gap-1.5">
						<Label for="target-host">{l.translate('target.docker.host')}</Label>
						<Input
							id="target-host"
							bind:value={host}
							required
							readonly={!!initialData}
							class={cn(
								'font-mono text-[13px]',
								initialData && 'border-transparent bg-raised text-muted-foreground'
							)}
							placeholder="server.example.com"
							aria-invalid={!!(fieldError('docker') ?? fieldError('docker.host')) || undefined}
							aria-describedby="target-host-message"
						/>
						<FieldMessage
							id="target-host-message"
							error={fieldError('docker') ?? fieldError('docker.host')}
						>
							{#snippet help()}{initialData
									? l.translate('targets.host.fixed')
									: l.translate('targets.host.help')}{/snippet}
						</FieldMessage>
					</div>
					<div class="flex flex-col gap-1.5">
						<Label for="target-user">{l.translate('target.docker.user')}</Label>
						<Input
							id="target-user"
							bind:value={user}
							placeholder={DOCKER_DEFAULT_USER}
							class="font-mono text-[13px]"
							aria-invalid={!!fieldError('docker.user') || undefined}
							aria-describedby="target-user-message"
						/>
						<FieldMessage id="target-user-message" error={fieldError('docker.user')}>
							{#snippet help()}{l.translate('targets.default', [DOCKER_DEFAULT_USER])}{/snippet}
						</FieldMessage>
					</div>
					<div class="flex flex-col gap-1.5">
						<Label for="target-port">{l.translate('target.docker.port')}</Label>
						<Input
							id="target-port"
							type="number"
							bind:value={port}
							placeholder={String(DOCKER_DEFAULT_PORT)}
							class="font-mono text-[13px]"
							aria-invalid={!!fieldError('docker.port') || undefined}
							aria-describedby="target-port-message"
						/>
						<FieldMessage id="target-port-message" error={fieldError('docker.port')}>
							{#snippet help()}{l.translate('targets.default', [DOCKER_DEFAULT_PORT])}{/snippet}
						</FieldMessage>
					</div>
				</div>

				<div class="flex flex-col gap-2.5">
					<Label for="target-key">
						{l.translate('target.docker.private_key')}
						<span class="font-normal text-muted-foreground"
							>· {l.translate('targets.optional')}</span
						>
					</Label>
					<DropZone
						compact
						icon={Key01Icon}
						what={l.translate('targets.key.what')}
						hint={hasSavedKey
							? l.translate('targets.key.hint.saved')
							: l.translate('targets.key.hint')}
						accept=""
						onfile={readKey}
					/>
					{#if keyFile}
						<p class="text-[13px] text-muted-foreground">
							{l.translate('deploy.compose.loaded_from')}
							<span class="font-mono text-ink">{keyFile}</span>
						</p>
					{/if}
					<CodeEditor
						id="target-key"
						lang="text"
						rows={4}
						bind:value={privateKey}
						placeholder={hasSavedKey
							? l.translate('targets.key.placeholder.saved')
							: '-----BEGIN OPENSSH PRIVATE KEY-----'}
						aria-invalid={!!fieldError('docker.private_key') || undefined}
						aria-describedby="target-key-message"
					/>
					<FieldMessage id="target-key-message" error={fieldError('docker.private_key')}>
						{#snippet help()}{l.translate('targets.key.help', [
								user || DOCKER_DEFAULT_USER
							])}{/snippet}
					</FieldMessage>
				</div>
			{/if}
		</SettingsSection>

		<SettingsSection
			title={l.translate('targets.exposure')}
			description={initialData
				? l.translate('targets.section.exposure.edit')
				: l.translate('targets.section.exposure')}
		>
			<div class="flex items-center gap-6 rounded-md bg-sunken px-4 py-3.5">
				<div class="flex min-w-0 flex-1 flex-col gap-0.5">
					<Label for="target-expose">{l.translate('target.automatic_proxy_configuration')}</Label>
					<p class="text-[13px] text-muted-foreground">{l.translate('targets.expose.help')}</p>
				</div>
				<Switch id="target-expose" bind:checked={exposeAutomatically} />
			</div>

			{#if exposeAutomatically}
				<div class="flex flex-col gap-1.5">
					<Label for="target-url">{l.translate('targets.root_url')}</Label>
					<Input
						id="target-url"
						type="url"
						bind:value={url}
						required
						placeholder="https://apps.example.com"
						class="font-mono text-[13px]"
						aria-invalid={!!fieldError('url') || undefined}
						aria-describedby="target-url-message"
					/>
					<FieldMessage id="target-url-message" error={fieldError('url')}>
						{#snippet help()}
							{l.translate('targets.url.help')}
							{#if preview}
								{l.translate('targets.url.preview')}
								<code>{preview.app}</code>; {l.translate('targets.url.dns')}
								<code>{preview.dns}</code>.
							{/if}
						{/snippet}
					</FieldMessage>
				</div>
			{:else}
				<p class="text-[13px] text-muted-foreground">{l.translate('targets.expose.manual')}</p>
			{/if}
		</SettingsSection>

		{#if !initialData}
			<div class="grid gap-4 md:grid-cols-[260px_minmax(0,1fr)] md:gap-10">
				<span class="max-md:hidden"></span>
				<div class="flex flex-col gap-2.5 rounded-lg bg-sunken p-4">
					<strong class="text-sm font-semibold">{l.translate('targets.next.title')}</strong>
					{#each ['targets.next.1', 'targets.next.2', 'targets.next.3'] as const as step, i (step)}
						<span class="grid grid-cols-[22px_minmax(0,1fr)] gap-2.5 text-[13px]/5">
							<span
								class="flex size-5.5 items-center justify-center rounded-full bg-raised text-xs font-semibold"
								>{i + 1}</span
							>
							{l.translate(step)}
						</span>
					{/each}
				</div>
			</div>
		{/if}
	</fieldset>
</form>
