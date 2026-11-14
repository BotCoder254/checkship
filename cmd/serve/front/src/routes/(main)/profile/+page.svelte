<!--
	Profile, opened from the avatar (DESIGN.md §5): a 760px column. Account fields, interface rows on
	a sunken fill (control on the right), and the API key with Copy and Regenerate.
-->
<script lang="ts">
	import Copy01Icon from '@hugeicons/core-free-icons/Copy01Icon';
	import RefreshIcon from '@hugeicons/core-free-icons/RefreshIcon';
	import Tick02Icon from '@hugeicons/core-free-icons/Tick02Icon';
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import { toast } from 'svelte-sonner';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import FieldMessage from '#lib/components/field-message.svelte';
	import Icon from '#lib/components/icon.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { GLOBAL_ERROR_NAME, submitter } from '#lib/form.ts';
	import l, {
		type AppLocales,
		type AppTranslationsString,
		type LocaleCode
	} from '#lib/localization/index.ts';
	import service from '#lib/resources/users.ts';

	let { data } = $props();

	// The form starts from the profile loaded with the page.
	// svelte-ignore state_referenced_locally
	let email = $state(data.user.email);
	let password = $state('');
	// svelte-ignore state_referenced_locally
	let apiKey = $state(data.user.api_key);
	let locale = $state<string>(l.locale());
	let regenerateOpen = $state(false);

	const locales = l.locales().map((item) => ({ value: item.code, label: item.displayName }));
	const localeLabel = $derived(locales.find((item) => item.value === locale)?.label ?? locale);
	const initial = $derived((data.user.email.trim()[0] ?? '?').toUpperCase());

	const { submit, loading, errors } = submitter(async () => {
		await service.update({ email, password: password || undefined });
		password = '';
		toast.success(l.translate('profile.saved'));
		l.locale(locale as LocaleCode<AppLocales>); // Reloads the page when it changed.
	});

	const globalError = $derived($errors?.[GLOBAL_ERROR_NAME]);

	function onsubmit(event: SubmitEvent) {
		event.preventDefault();
		if ($loading) return;
		submit().catch(() => {});
	}

	async function copyKey() {
		try {
			await navigator.clipboard.writeText(apiKey);
			toast.success(l.translate('copied'));
		} catch {
			toast.error(l.translate('unexpected_error'));
		}
	}
</script>

<form class="flex flex-col" novalidate {onsubmit}>
	<PageHeader title={l.translate('breadcrumb.profile')}>
		{#snippet actions()}
			<Button type="submit" loading={$loading}>
				<Icon icon={Tick02Icon} />
				{l.translate('save')}
			</Button>
		{/snippet}
	</PageHeader>

	<div class="flex items-center gap-4.5 pb-9">
		<span
			aria-hidden="true"
			class="flex size-16 shrink-0 items-center justify-center rounded-full bg-primary text-2xl font-semibold text-primary-foreground"
		>
			{initial}
		</span>
		<span class="flex min-w-0 flex-col gap-0.5">
			<span class="truncate text-lg/7 font-semibold">{data.user.email}</span>
			<span class="text-sm text-muted-foreground">
				{l.translate('profile.signed_in_since', [l.date(data.user.registered_at)])}
			</span>
		</span>
	</div>

	<div class="flex flex-col gap-9">
		{#if globalError}
			<Alert.Root variant="destructive">
				<Icon icon={Alert02Icon} />
				<Alert.Title class="font-semibold">
					{l.translate(globalError as AppTranslationsString)}
				</Alert.Title>
			</Alert.Root>
		{/if}

		<SettingsSection
			title={l.translate('profile.account')}
			description={l.translate('profile.account.description')}
		>
			<div class="grid gap-4.5 md:grid-cols-2">
				<div class="flex flex-col gap-1.5">
					<Label for="profile-email">{l.translate('email')}</Label>
					<Input
						id="profile-email"
						type="email"
						autocomplete="username"
						bind:value={email}
						readonly={$loading}
						aria-invalid={!!$errors?.email || undefined}
						aria-describedby="profile-email-message"
					/>
					<FieldMessage id="profile-email-message" error={$errors?.email ?? undefined} />
				</div>
				<div class="flex flex-col gap-1.5">
					<Label for="profile-password">{l.translate('profile.password')}</Label>
					<Input
						id="profile-password"
						type="password"
						autocomplete="new-password"
						bind:value={password}
						readonly={$loading}
						aria-invalid={!!$errors?.password || undefined}
						aria-describedby="profile-password-message"
					/>
					<FieldMessage id="profile-password-message" error={$errors?.password ?? undefined}>
						{#snippet help()}{l.translate('profile.password.help')}{/snippet}
					</FieldMessage>
				</div>
			</div>
		</SettingsSection>

		<SettingsSection
			title={l.translate('profile.interface')}
			description={l.translate('profile.interface.description')}
		>
			<div class="flex flex-wrap items-center gap-x-6 gap-y-3 rounded-md bg-sunken px-4 py-3.5">
				<div class="flex min-w-48 flex-1 flex-col gap-0.5">
					<Label for="profile-locale">{l.translate('profile.locale')}</Label>
					<p class="text-sm text-muted-foreground">{l.translate('profile.locale.help')}</p>
				</div>
				<Select.Root type="single" bind:value={locale}>
					<Select.Trigger id="profile-locale" class="w-45 shrink-0 bg-surface">
						{localeLabel}
					</Select.Trigger>
					<Select.Content>
						{#each locales as item (item.value)}
							<Select.Item value={item.value} label={item.label}>{item.label}</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</div>
		</SettingsSection>

		<SettingsSection
			id="integration"
			title={l.translate('profile.integration')}
			description={l.translate('profile.integration.short')}
		>
			<p class="text-sm text-muted-foreground">
				<!-- eslint-disable-next-line svelte/no-at-html-tags -- our own translation string -->
				{@html l.translate('profile.integration.description')}
			</p>
			<div class="flex flex-wrap gap-1.5">
				<Input
					aria-label={l.translate('profile.key')}
					readonly
					value={apiKey}
					class="min-w-60 flex-1 font-mono text-[13px]"
				/>
				<Button variant="outline" onclick={copyKey}>
					<Icon icon={Copy01Icon} />
					{l.translate('copy')}
				</Button>
				<Button variant="outline" onclick={() => (regenerateOpen = true)}>
					<Icon icon={RefreshIcon} />
					{l.translate('profile.key.refresh')}
				</Button>
			</div>
		</SettingsSection>
	</div>
</form>

<ConfirmDialog
	bind:open={regenerateOpen}
	title={l.translate('profile.key.refresh.title')}
	description={l.translate('profile.key.refresh.confirm')}
	confirmLabel={l.translate('profile.key.refresh')}
	icon={RefreshIcon}
	onconfirm={() =>
		service.refreshAPIKey().then((result) => {
			apiKey = result.api_key;
			toast.success(l.translate('profile.key.regenerated'));
		})}
/>
