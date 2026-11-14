<!--
	Target form (DESIGN.md §5): a 760px column, sections separated by headings and space, switch
	rows on a sunken fill with the control on the right. The page header (title, actions) is inside
	the form so its submit button submits.
-->
<script lang="ts">
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import CheckmarkCircle02Icon from '@hugeicons/core-free-icons/CheckmarkCircle02Icon';
	import Loading03Icon from '@hugeicons/core-free-icons/Loading03Icon';
	import type { Snippet } from 'svelte';
	import FieldMessage from '#lib/components/field-message.svelte';
	import Icon from '#lib/components/icon.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { Switch } from '#lib/components/ui/switch/index.js';
	import { Textarea } from '#lib/components/ui/textarea/index.js';
	import { GLOBAL_ERROR_NAME, submitter } from '#lib/form.ts';
	import l, { type AppTranslationsString } from '#lib/localization/index.ts';
	import {
		TargetStatus,
		type CreateTarget,
		type Target,
		type UpdateTarget
	} from '#lib/resources/targets.ts';

	type Props = {
		title: string;
		crumbs: { href: string; label: string }[];
		/** Header actions; gets whether the form is submitting. */
		actions: Snippet<[boolean]>;
		disabled?: boolean;
	} & (
		| { initialData: Target; handler: (data: UpdateTarget) => Promise<unknown> }
		| { initialData?: undefined; handler: (data: CreateTarget) => Promise<unknown> }
	);

	let props: Props = $props();
	// The form is created once per page; later prop changes don't reset what the person typed.
	// svelte-ignore state_referenced_locally
	const initialData = props.initialData;

	let name = $state(initialData?.name ?? '');
	let url = $state(initialData?.url ?? '');
	let exposeAutomatically = $state(initialData ? !!initialData.url : true);
	let isRemote = $state(!!initialData?.provider.data.host);
	let host = $state(initialData?.provider.data.host ?? '');
	let user = $state(initialData?.provider.data.user ?? '');
	let port = $state(initialData?.provider.data.port?.toString() ?? '');
	let privateKey = $state(initialData?.provider.data.private_key ?? '');

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
			docker: isRemote
				? {
						...remote,
						private_key:
							privateKey !== props.initialData.provider.data.private_key
								? privateKey || null
								: undefined
					}
				: undefined
		});
	});

	const fieldError = (key: string) => $errors?.[key] ?? undefined;
	const globalError = $derived($errors?.[GLOBAL_ERROR_NAME]);

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
		{#snippet status()}
			<span>{l.translate('target.page.description')}</span>
		{/snippet}
	</PageHeader>

	<fieldset class="flex flex-col gap-9" disabled={props.disabled}>
		{#if globalError}
			<Alert.Root variant="destructive">
				<Icon icon={Alert02Icon} />
				<Alert.Title class="font-semibold">
					{l.translate(globalError as AppTranslationsString)}
				</Alert.Title>
			</Alert.Root>
		{/if}

		{#if initialData?.state.status === TargetStatus.Failed}
			<Alert.Root variant="destructive">
				<Icon icon={Alert02Icon} />
				<Alert.Title class="font-semibold">{l.translate('target.failed')}</Alert.Title>
				<Alert.Description class="font-mono text-xs"
					>{initialData.state.error_code}</Alert.Description
				>
			</Alert.Root>
		{:else if initialData?.state.status === TargetStatus.Configuring}
			<Alert.Root variant="warning">
				<Icon icon={Loading03Icon} class="animate-spin" />
				<Alert.Title class="font-semibold">{l.translate('target.configuring')}</Alert.Title>
				<Alert.Description>{l.translate('target.configuring.description')}</Alert.Description>
			</Alert.Root>
		{:else if initialData?.state.status === TargetStatus.Ready && initialData.state.last_ready_version}
			<Alert.Root variant="success">
				<Icon icon={CheckmarkCircle02Icon} />
				<Alert.Title class="font-semibold">{l.translate('target.ready')}</Alert.Title>
				<Alert.Description>
					{l.translate('target.ready.description', [initialData.state.last_ready_version])}
				</Alert.Description>
			</Alert.Root>
		{/if}

		<SettingsSection
			title={l.translate('settings.general')}
			description={l.translate('targets.section.general')}
		>
			<div class="flex flex-col gap-1.5">
				<Label for="target-name">{l.translate('name')}</Label>
				<Input
					id="target-name"
					bind:value={name}
					required
					aria-invalid={!!fieldError('name') || undefined}
					aria-describedby="target-name-message"
				/>
				<FieldMessage id="target-name-message" error={fieldError('name')}>
					{#snippet help()}{l.translate('target.name.help')}{/snippet}
				</FieldMessage>
			</div>

			<div class="flex items-center gap-6 rounded-md bg-sunken px-4 py-3.5">
				<div class="flex min-w-0 flex-1 flex-col gap-0.5">
					<Label for="target-expose">{l.translate('target.automatic_proxy_configuration')}</Label>
					<p class="text-sm text-muted-foreground [&_a]:text-primary [&_a]:underline">
						<!-- eslint-disable-next-line svelte/no-at-html-tags -- our own translation string -->
						{@html l.translate('target.automatic_proxy_configuration.help')}
					</p>
				</div>
				<Switch id="target-expose" bind:checked={exposeAutomatically} />
			</div>

			{#if exposeAutomatically}
				<div class="flex flex-col gap-1.5">
					<Label for="target-url">{l.translate('url')}</Label>
					<Input
						id="target-url"
						type="url"
						bind:value={url}
						placeholder="https://apps.example.com"
						aria-invalid={!!fieldError('url') || undefined}
						aria-describedby="target-url-message"
					/>
					<FieldMessage id="target-url-message" error={fieldError('url')}>
						<!-- eslint-disable-next-line svelte/no-at-html-tags -- our own translation string -->
						{#snippet help()}{@html l.translate('target.url.help')}{/snippet}
					</FieldMessage>
				</div>
			{/if}
		</SettingsSection>

		<SettingsSection
			title="{l.translate('target.provider')} · Docker"
			description={l.translate('targets.section.provider')}
		>
			<div class="flex items-center gap-6 rounded-md bg-sunken px-4 py-3.5">
				<div class="flex min-w-0 flex-1 flex-col gap-0.5">
					<Label for="target-remote">{l.translate('target.remote.label')}</Label>
					<p class="text-sm text-muted-foreground">{l.translate('target.remote.description')}</p>
				</div>
				<Switch id="target-remote" bind:checked={isRemote} disabled={!!initialData} />
			</div>

			{#if isRemote}
				<div class="grid gap-4.5 md:grid-cols-[minmax(0,2fr)_minmax(0,1fr)_minmax(0,1fr)]">
					<div class="flex flex-col gap-1.5">
						<Label for="target-host">{l.translate('target.docker.host')}</Label>
						<Input
							id="target-host"
							bind:value={host}
							disabled={!!initialData}
							aria-invalid={!!(fieldError('docker') ?? fieldError('docker.host')) || undefined}
							aria-describedby="target-host-message"
						/>
						<FieldMessage
							id="target-host-message"
							error={fieldError('docker') ?? fieldError('docker.host')}
						/>
					</div>
					<div class="flex flex-col gap-1.5">
						<Label for="target-user">{l.translate('target.docker.user')}</Label>
						<Input
							id="target-user"
							bind:value={user}
							placeholder="docker"
							aria-invalid={!!fieldError('docker.user') || undefined}
						/>
						<FieldMessage id="target-user-message" error={fieldError('docker.user')} />
					</div>
					<div class="flex flex-col gap-1.5">
						<Label for="target-port">{l.translate('target.docker.port')}</Label>
						<Input
							id="target-port"
							type="number"
							bind:value={port}
							placeholder="22"
							aria-invalid={!!fieldError('docker.port') || undefined}
						/>
						<FieldMessage id="target-port-message" error={fieldError('docker.port')} />
					</div>
				</div>
				<div class="flex flex-col gap-1.5">
					<Label for="target-key">{l.translate('target.docker.private_key')}</Label>
					<Textarea
						id="target-key"
						bind:value={privateKey}
						rows={7}
						class="font-mono text-xs"
						aria-invalid={!!fieldError('docker.private_key') || undefined}
						aria-describedby="target-key-message"
					/>
					<FieldMessage id="target-key-message" error={fieldError('docker.private_key')}>
						<!-- eslint-disable-next-line svelte/no-at-html-tags -- our own translation string -->
						{#snippet help()}{@html l.translate('target.docker.private_key.help')}{/snippet}
					</FieldMessage>
				</div>
			{/if}
		</SettingsSection>
	</fieldset>
</form>
