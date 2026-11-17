<!--
	Registry (DESIGN.md §11.8). A new registry starts from a preset (Docker Hub, GitHub, GitLab, Quay,
	with their official marks) that fills the name and URL and says which token to create; Other is
	any URL. Server rules (create_registry.go): name required, URL required and unique, username and
	password together. The password is masked once saved; Docker uses the login for images from this
	registry, on every target. Rendered in the registries drawer and on its two pages, so the form has
	an id and its submit button can live outside it.
-->
<script lang="ts">
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import PackageIcon from '@hugeicons/core-free-icons/PackageIcon';
	import FieldMessage from '#lib/components/field-message.svelte';
	import Icon from '#lib/components/icon.svelte';
	import SettingRow from '#lib/components/setting-row.svelte';
	import RegistryLogo from '#lib/components/registry-logo.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { Switch } from '#lib/components/ui/switch/index.js';
	import { GLOBAL_ERROR_NAME, submitter } from '#lib/form.ts';
	import l, { type AppTranslationsString } from '#lib/localization/index.ts';
	import { REGISTRY_PRESETS, presetForUrl, type RegistryPreset } from '#lib/registry-presets.ts';
	import service, { type Registry } from '#lib/resources/registries.ts';
	import { cn } from '#lib/utils.js';

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
	let useCredentials = $state(initial ? !!initial.credentials : true);
	let username = $state(initial?.credentials?.username ?? '');
	// Masked by the server: kept untouched unless a new one is typed.
	let password = $state('');
	const savedPassword = !!initial?.credentials;

	let chosen = $state<RegistryPreset['id'] | 'other' | undefined>(
		initial ? (presetForUrl(initial.url)?.id ?? 'other') : undefined
	);
	const preset = $derived(
		REGISTRY_PRESETS.find((p) => p.id === chosen) ?? (url ? presetForUrl(url) : undefined)
	);

	function choose(next: RegistryPreset | undefined) {
		chosen = next?.id ?? 'other';
		const previous = REGISTRY_PRESETS.find((p) => p.name === name || p.url === url);
		// Fill only what the person hasn't typed themselves.
		if (next) {
			if (!name || previous) name = next.name;
			if (!url || previous) url = next.url;
		} else if (previous) {
			name = '';
			url = '';
		}
	}

	const {
		submit,
		loading: submitting,
		errors
	} = submitter(async () => {
		const saved = initial
			? await service.update(initial.id, {
					name,
					url,
					credentials: useCredentials ? { username, password: password || undefined } : null
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

	const options: (RegistryPreset | undefined)[] = [...REGISTRY_PRESETS, undefined];
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

		{#if initial}
			<div class="flex items-center gap-3">
				<span
					class="flex size-10 items-center justify-center rounded-lg bg-raised text-muted-foreground"
				>
					{#if preset}<RegistryLogo name={preset.id} class="size-6" />{:else}<Icon
							icon={PackageIcon}
							class="size-5"
						/>{/if}
				</span>
				<span class="flex flex-col">
					<strong class="text-base">{initial.name}</strong>
					<span class="text-xs text-muted-foreground">
						{preset
							? l.translate('registries.recognised', [
									l.translate(`registries.preset.${preset.id}`)
								])
							: l.translate('registries.custom')}
					</span>
				</span>
			</div>
		{:else}
			<div class="flex flex-col gap-2">
				<span id="{id}-preset-label" class="text-sm font-medium"
					>{l.translate('registries.preset')}</span
				>
				<div
					role="radiogroup"
					aria-labelledby="{id}-preset-label"
					class="grid grid-cols-3 gap-1.5 sm:grid-cols-5"
				>
					{#each options as option (option?.id ?? 'other')}
						{@const selected = chosen === (option?.id ?? 'other')}
						<button
							type="button"
							role="radio"
							aria-checked={selected}
							class={cn(
								'flex flex-col items-center gap-1.5 rounded-lg bg-sunken px-1 pt-3 pb-2.5 text-xs font-medium text-muted-foreground outline-none hover:bg-hover focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset',
								selected &&
									'bg-accent-soft font-semibold text-ink ring-2 ring-primary ring-inset hover:bg-accent-soft'
							)}
							onclick={() => choose(option)}
						>
							{#if option}
								<RegistryLogo name={option.id} class="size-6" />
								{l.translate(`registries.preset.${option.id}`)}
							{:else}
								<Icon icon={PackageIcon} class="size-6" />
								{l.translate('registries.preset.other')}
							{/if}
						</button>
					{/each}
				</div>
				<p class="text-label text-muted-foreground">{l.translate('registries.preset.help')}</p>
			</div>
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
			<FieldMessage id="{id}-name-message" error={fieldError('name')} />
		</div>

		<div class="flex flex-col gap-1.5">
			<Label for="{id}-url">{l.translate('url')}</Label>
			<Input
				id="{id}-url"
				type="url"
				bind:value={url}
				required
				class="font-mono text-label"
				placeholder="https://registry.example.com"
				aria-invalid={!!fieldError('url') || undefined}
				aria-describedby="{id}-url-message"
			/>
			<FieldMessage id="{id}-url-message" error={fieldError('url')}>
				{#snippet help()}
					{preset?.id === 'docker'
						? l.translate('registries.url.docker')
						: preset?.id === 'gitlab'
							? l.translate('registries.url.gitlab')
							: l.translate('registries.url.unique')}
				{/snippet}
			</FieldMessage>
		</div>

		<SettingRow
			for="{id}-auth"
			label={l.translate('registries.auth.label')}
			description={initial?.credentials
				? l.translate('registries.auth.off_removes')
				: l.translate('registries.auth.help')}
		>
			<Switch id="{id}-auth" bind:checked={useCredentials} />
		</SettingRow>

		{#if useCredentials}
			<div class="grid gap-4 sm:grid-cols-2">
				<div class="flex flex-col gap-1.5">
					<Label for="{id}-username">{l.translate('registry.username')}</Label>
					<Input
						id="{id}-username"
						bind:value={username}
						required
						autocomplete="off"
						placeholder={preset ? l.translate(`registries.user.example.${preset.id}`) : undefined}
						aria-invalid={!!fieldError('credentials.username') || undefined}
						aria-describedby="{id}-username-message"
					/>
					<FieldMessage id="{id}-username-message" error={fieldError('credentials.username')}>
						{#snippet help()}{preset ? l.translate(`registries.user.${preset.id}`) : ''}{/snippet}
					</FieldMessage>
				</div>
				<div class="flex flex-col gap-1.5">
					<Label for="{id}-password">{l.translate('registries.password')}</Label>
					<Input
						id="{id}-password"
						type="password"
						autocomplete="new-password"
						bind:value={password}
						required={!savedPassword}
						placeholder={savedPassword ? l.translate('registries.password.saved') : undefined}
						aria-invalid={!!fieldError('credentials.password') || undefined}
						aria-describedby="{id}-password-message"
					/>
					<FieldMessage id="{id}-password-message" error={fieldError('credentials.password')}>
						{#snippet help()}{preset
								? l.translate(`registries.token.${preset.id}`)
								: l.translate('registries.token.other')}{/snippet}
					</FieldMessage>
				</div>
			</div>

			{#if preset && !initial}
				<div class="flex flex-col gap-1 text-label/[19px]">
					<span class="flex items-center gap-2 font-semibold">
						<RegistryLogo name={preset.id} class="size-4" />
						{l.translate('registries.where')}
					</span>
					<span class="text-muted-foreground">
						{l.translate(`registries.where.${preset.id}`)}
						{l.translate(`registries.note.${preset.id}`)}
						{l.translate('registries.never_shown')}
					</span>
				</div>
			{/if}
		{/if}

		{#if initial}
			<div class="flex flex-col gap-1">
				<strong class="text-label font-semibold">{l.translate('registries.used.title')}</strong>
				<p class="text-label text-muted-foreground">
					{l.translate('registries.used.description', [initial.name])}
				</p>
			</div>
		{/if}
	</fieldset>
</form>
