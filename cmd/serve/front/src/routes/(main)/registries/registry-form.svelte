<!--
	Registry (DESIGN.md §11.6): name and URL, then a switch row for credentials. Rendered in the
	drawer on the registries page and on /registries/new and /registries/:id/edit, so the form has an
	id and its submit button can live outside it.
-->
<script lang="ts">
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import FieldMessage from '#lib/components/field-message.svelte';
	import Icon from '#lib/components/icon.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { Switch } from '#lib/components/ui/switch/index.js';
	import { GLOBAL_ERROR_NAME, submitter } from '#lib/form.ts';
	import l, { type AppTranslationsString } from '#lib/localization/index.ts';
	import service, { type Registry } from '#lib/resources/registries.ts';

	type Props = {
		id: string;
		initialData?: Registry;
		loading?: boolean;
		disabled?: boolean;
		onsaved: (registry: Registry) => unknown;
	};

	let { id, initialData, loading = $bindable(false), disabled = false, onsaved }: Props = $props();

	// Set once per form; the drawer remounts it for each registry.
	// svelte-ignore state_referenced_locally
	const initial = initialData;
	let name = $state(initial?.name ?? '');
	let url = $state(initial?.url ?? '');
	let useCredentials = $state(!!initial?.credentials);
	let username = $state(initial?.credentials?.username ?? '');
	let password = $state(initial?.credentials?.password ?? '');

	const {
		submit,
		loading: submitting,
		errors
	} = submitter(async () => {
		const saved = initial
			? await service.update(initial.id, {
					name,
					url,
					credentials: useCredentials
						? {
								username,
								password: password !== initial.credentials?.password ? password : undefined
							}
						: null
				})
			: await service.create({
					name,
					url,
					credentials: useCredentials ? { username, password } : undefined
				});
		return onsaved(saved);
	});

	$effect(() => {
		loading = $submitting;
	});

	const fieldError = (key: string) => $errors?.[key] ?? undefined;
	const globalError = $derived($errors?.[GLOBAL_ERROR_NAME]);

	function onsubmit(event: SubmitEvent) {
		event.preventDefault();
		if ($submitting) return;
		submit().catch(() => {});
	}
</script>

<form {id} class="flex flex-col gap-5" novalidate {onsubmit}>
	<fieldset class="contents" {disabled}>
		{#if globalError}
			<Alert.Root variant="destructive">
				<Icon icon={Alert02Icon} />
				<Alert.Title class="font-semibold">
					{l.translate(globalError as AppTranslationsString)}
				</Alert.Title>
			</Alert.Root>
		{/if}

		<div class="flex flex-col gap-1.5">
			<Label for="{id}-name">{l.translate('name')}</Label>
			<Input
				id="{id}-name"
				bind:value={name}
				required
				aria-invalid={!!fieldError('name') || undefined}
				aria-describedby="{id}-name-message"
			/>
			<FieldMessage id="{id}-name-message" error={fieldError('name')}>
				{#snippet help()}{l.translate('registry.name.help')}{/snippet}
			</FieldMessage>
		</div>

		<div class="flex flex-col gap-1.5">
			<Label for="{id}-url">{l.translate('url')}</Label>
			<Input
				id="{id}-url"
				type="url"
				bind:value={url}
				required
				class="font-mono text-[13px]"
				placeholder="https://index.docker.io/v1/"
				aria-invalid={!!fieldError('url') || undefined}
				aria-describedby="{id}-url-message"
			/>
			<FieldMessage id="{id}-url-message" error={fieldError('url')}>
				{#snippet help()}{l.translate('registries.url.help')}
					<code>https://index.docker.io/v1/</code>{/snippet}
			</FieldMessage>
		</div>

		<div class="flex items-center justify-between gap-4 rounded-md bg-sunken px-4 py-3.5">
			<div class="flex flex-col gap-0.5">
				<Label for="{id}-auth">{l.translate('registries.auth.label')}</Label>
				<p class="text-[13px] text-muted-foreground">{l.translate('registries.auth.help')}</p>
			</div>
			<Switch id="{id}-auth" bind:checked={useCredentials} />
		</div>

		{#if useCredentials}
			<div class="grid gap-4 sm:grid-cols-2">
				<div class="flex flex-col gap-1.5">
					<Label for="{id}-username">{l.translate('registry.username')}</Label>
					<Input
						id="{id}-username"
						bind:value={username}
						required
						autocomplete="off"
						aria-invalid={!!fieldError('credentials.username') || undefined}
						aria-describedby="{id}-username-message"
					/>
					<FieldMessage id="{id}-username-message" error={fieldError('credentials.username')} />
				</div>
				<div class="flex flex-col gap-1.5">
					<Label for="{id}-password">{l.translate('registry.password')}</Label>
					<Input
						id="{id}-password"
						type="password"
						autocomplete="new-password"
						bind:value={password}
						required
						aria-invalid={!!fieldError('credentials.password') || undefined}
						aria-describedby="{id}-password-message"
					/>
					<FieldMessage id="{id}-password-message" error={fieldError('credentials.password')} />
				</div>
			</div>
		{/if}
	</fieldset>
</form>
