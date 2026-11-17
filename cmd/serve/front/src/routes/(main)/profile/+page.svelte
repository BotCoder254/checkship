<!--
	Profile v4 (DESIGN.md §11.9): who you are and Sign out, then sections that each save on their own
	and say so: Account (email, unique), Password (typed twice, checked here; the API only asks for
	a non-empty value), Interface (language, applied at once) and the API key. The server returns
	the key in full, so it is masked (first and last four) with Reveal and Copy; Regenerate sits in
	a danger-tinted block behind a confirmation, because the old key stops working at once.
-->
<script lang="ts">
	import TranslateIcon from '@hugeicons/core-free-icons/TranslateIcon';
	import { copyText } from '#lib/clipboard.ts';
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import CheckmarkCircle02Icon from '@hugeicons/core-free-icons/CheckmarkCircle02Icon';
	import Copy01Icon from '@hugeicons/core-free-icons/Copy01Icon';
	import Key01Icon from '@hugeicons/core-free-icons/Key01Icon';
	import Logout03Icon from '@hugeicons/core-free-icons/Logout03Icon';
	import RefreshIcon from '@hugeicons/core-free-icons/RefreshIcon';
	import Tick02Icon from '@hugeicons/core-free-icons/Tick02Icon';
	import ViewIcon from '@hugeicons/core-free-icons/ViewIcon';
	import ViewOffIcon from '@hugeicons/core-free-icons/ViewOffIcon';
	import { invalidateAll } from '$app/navigation';
	import { toast } from 'svelte-sonner';
	import auth from '#lib/auth.ts';
	import CodeBlock from '#lib/components/code-block.svelte';
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

	// The forms start from the profile loaded with the page.
	// svelte-ignore state_referenced_locally
	let email = $state(data.user.email);
	// svelte-ignore state_referenced_locally
	let savedEmail = $state(data.user.email);
	let emailSaved = $state(false);

	let password = $state('');
	let repeat = $state('');
	let passwordProblem = $state<'empty' | 'mismatch'>();
	let passwordChanged = $state(false);

	// svelte-ignore state_referenced_locally
	let apiKey = $state(data.user.api_key);
	let revealed = $state(false);
	let regenerateOpen = $state(false);

	const locale = l.locale();
	const locales = l.locales().map((item) => ({ value: item.code, label: item.displayName }));
	const localeLabel = locales.find((item) => item.value === locale)?.label ?? locale;
	const initial = $derived((savedEmail.trim()[0] ?? '?').toUpperCase());

	const masked = $derived(
		apiKey.length > 12
			? `${apiKey.slice(0, 4)}${'•'.repeat(apiKey.length - 8)}${apiKey.slice(-4)}`
			: '•'.repeat(apiKey.length)
	);
	const curl = $derived(
		[
			'curl -i -X POST \\',
			'  -H "Authorization: Bearer $CHECKSHIP_API_KEY" \\',
			'  -H "Content-Type: application/json" \\',
			`  -d '{"environment":"production","git":{"branch":"main"}}' \\`,
			`  ${typeof location === 'undefined' ? '' : location.origin}/api/v1/apps/$APP_ID/deployments`
		].join('\n')
	);

	const account = submitter(async () => {
		await service.update({ email });
		savedEmail = email;
		emailSaved = true;
		await invalidateAll();
	});
	const accountLoading = account.loading;
	const accountErrors = account.errors;

	const passwordForm = submitter(async () => {
		await service.update({ password });
		password = '';
		repeat = '';
		passwordChanged = true;
	});
	const passwordLoading = passwordForm.loading;
	const passwordErrors = passwordForm.errors;

	const globalError = $derived(
		$accountErrors?.[GLOBAL_ERROR_NAME] ?? $passwordErrors?.[GLOBAL_ERROR_NAME]
	);

	function saveEmail(event: SubmitEvent) {
		event.preventDefault();
		if ($accountLoading) return;
		account.submit().catch(() => {});
	}

	function changePassword(event: SubmitEvent) {
		event.preventDefault();
		if ($passwordLoading) return;
		passwordProblem = !password ? 'empty' : password !== repeat ? 'mismatch' : undefined;
		if (passwordProblem) return;
		passwordForm.submit().catch(() => {});
	}

	async function copyKey() {
		if (await copyText(apiKey)) toast.success(l.translate('profile.key.copied'));
		else toast.error(l.translate('copy_blocked'));
	}

	async function regenerate() {
		const result = await service.refreshAPIKey();
		apiKey = result.api_key;
		toast.success(l.translate('profile.key.regenerated'));
	}
</script>

<PageHeader title={l.translate('breadcrumb.profile')} />

<div class="flex flex-col gap-10">
	<div class="flex flex-wrap items-center gap-4">
		<span
			aria-hidden="true"
			class="flex size-14 shrink-0 items-center justify-center rounded-full bg-primary text-xl font-semibold text-primary-foreground"
		>
			{initial}
		</span>
		<span class="flex min-w-0 flex-1 flex-col gap-0.5">
			<span class="truncate text-lg/7 font-semibold">{savedEmail}</span>
			<span class="text-label text-muted-foreground">
				{l.translate('profile.created', [l.date(data.user.registered_at)])}
			</span>
		</span>
		<Button variant="outline" onclick={() => auth.logout()}>
			<Icon icon={Logout03Icon} />
			{l.translate('profile.sign_out')}
		</Button>
	</div>

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
		description={l.translate('profile.email.description')}
	>
		<form class="flex flex-col gap-4" novalidate onsubmit={saveEmail}>
			<div class="flex flex-col gap-1.5">
				<Label for="profile-email">{l.translate('email')}</Label>
				<Input
					id="profile-email"
					type="email"
					autocomplete="username"
					bind:value={email}
					oninput={() => (emailSaved = false)}
					readonly={$accountLoading}
					aria-invalid={!!$accountErrors?.email || undefined}
					aria-describedby="profile-email-message"
				/>
				<FieldMessage id="profile-email-message" error={$accountErrors?.email ?? undefined} />
			</div>
			<div class="flex flex-wrap items-center gap-3">
				<!-- Section saves are secondary and wait for a change: a page of settings has no single
					primary action (DESIGN.md §3, one primary per view). -->
				<Button
					type="submit"
					variant="outline"
					loading={$accountLoading}
					disabled={email === savedEmail && !$accountErrors}
				>
					<Icon icon={Tick02Icon} />
					{l.translate('profile.email.save')}
				</Button>
				{#if emailSaved}
					<span role="status" class="flex items-center gap-1.5 text-label text-success">
						<Icon icon={CheckmarkCircle02Icon} class="size-3.5" />
						{l.translate('profile.email.saved')}
					</span>
				{/if}
			</div>
		</form>
	</SettingsSection>

	<SettingsSection
		title={l.translate('profile.password.section')}
		description={l.translate('profile.password.description')}
	>
		<form class="flex flex-col gap-4" novalidate onsubmit={changePassword}>
			<!-- Lets password managers pair the new password with the account. -->
			<input type="email" autocomplete="username" value={savedEmail} hidden readonly />
			<div class="grid gap-4 md:grid-cols-2">
				<div class="flex flex-col gap-1.5">
					<Label for="profile-password">{l.translate('profile.password')}</Label>
					<Input
						id="profile-password"
						type="password"
						autocomplete="new-password"
						bind:value={password}
						oninput={() => ((passwordChanged = false), (passwordProblem = undefined))}
						readonly={$passwordLoading}
						aria-invalid={passwordProblem === 'empty' || !!$passwordErrors?.password || undefined}
						aria-describedby="profile-password-message"
					/>
					<FieldMessage
						id="profile-password-message"
						error={passwordProblem === 'empty'
							? l.translate('profile.password.empty')
							: ($passwordErrors?.password ?? undefined)}
					/>
				</div>
				<div class="flex flex-col gap-1.5">
					<Label for="profile-password-repeat">{l.translate('profile.password.repeat')}</Label>
					<Input
						id="profile-password-repeat"
						type="password"
						autocomplete="new-password"
						bind:value={repeat}
						oninput={() => (passwordProblem = undefined)}
						readonly={$passwordLoading}
						aria-invalid={passwordProblem === 'mismatch' || undefined}
						aria-describedby="profile-password-repeat-message"
					/>
					<FieldMessage
						id="profile-password-repeat-message"
						error={passwordProblem === 'mismatch'
							? l.translate('profile.password.mismatch')
							: undefined}
					>
						{#snippet help()}{l.translate('profile.password.repeat.help')}{/snippet}
					</FieldMessage>
				</div>
			</div>
			<div class="flex flex-wrap items-center gap-3">
				<Button
					type="submit"
					variant="outline"
					loading={$passwordLoading}
					disabled={!password && !repeat}
				>
					<Icon icon={Key01Icon} />
					{l.translate('profile.password.change')}
				</Button>
				{#if passwordChanged}
					<span role="status" class="flex items-center gap-1.5 text-label text-success">
						<Icon icon={CheckmarkCircle02Icon} class="size-3.5" />
						{l.translate('profile.password.changed')}
					</span>
				{/if}
			</div>
		</form>
	</SettingsSection>

	<SettingsSection
		title={l.translate('profile.interface')}
		description={l.translate('profile.interface.browser')}
	>
		<div class="flex flex-wrap items-end gap-x-4 gap-y-2">
			<div class="flex w-full flex-col gap-1.5 sm:w-65">
				<Label for="profile-locale">{l.translate('profile.locale')}</Label>
				<Select.Root
					type="single"
					value={locale}
					onValueChange={(value) => l.locale(value as LocaleCode<AppLocales>)}
				>
					<Select.Trigger id="profile-locale" class="w-full">{localeLabel}</Select.Trigger>
					<Select.Content>
						{#each locales as item (item.value)}
							<Select.Item value={item.value} label={item.label}>
								<Icon icon={TranslateIcon} class="text-muted-foreground" />
								{item.label}
							</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
			</div>
			<p class="pb-2 text-label text-muted-foreground">{l.translate('profile.locale.help')}</p>
		</div>
	</SettingsSection>

	<SettingsSection
		id="integration"
		title={l.translate('profile.key.section')}
		description={l.translate('profile.key.description')}
	>
		<div class="flex flex-col gap-4">
			<div
				class="flex h-10 items-center gap-2 rounded-md border border-faint bg-sunken pr-1.5 pl-3"
			>
				<span class="sr-only">{l.translate('profile.key')}</span>
				<span class="min-w-0 flex-1 truncate font-mono text-label">
					{revealed ? apiKey : masked}
				</span>
				<Button
					variant="ghost"
					size="sm"
					aria-pressed={revealed}
					onclick={() => (revealed = !revealed)}
				>
					<Icon icon={revealed ? ViewOffIcon : ViewIcon} />
					{l.translate(revealed ? 'profile.key.hide' : 'profile.key.reveal')}
				</Button>
				<Button variant="outline" size="sm" onclick={copyKey}>
					<Icon icon={Copy01Icon} />
					{l.translate('copy')}
				</Button>
			</div>
			<p class="text-xs text-muted-foreground">{l.translate('profile.key.usage')}</p>
			<CodeBlock code={curl} lang="shellscript" copy label={l.translate('profile.key.example')} />
			<div class="flex min-h-[54px] flex-wrap items-center justify-between gap-4 py-2">
				<span class="flex min-w-0 flex-col gap-0.5">
					<strong class="text-sm font-semibold"
						>{l.translate('profile.key.regenerate.title')}</strong
					>
					<span class="text-label">{l.translate('profile.key.regenerate.description')}</span>
				</span>
				<Button variant="destructive" onclick={() => (regenerateOpen = true)}>
					<Icon icon={RefreshIcon} />
					{l.translate('profile.key.regenerate')}
				</Button>
			</div>
		</div>
	</SettingsSection>
</div>

<ConfirmDialog
	bind:open={regenerateOpen}
	destructive
	icon={RefreshIcon}
	title={l.translate('profile.key.refresh.title')}
	description={l.translate('profile.key.regenerate.confirm')}
	confirmLabel={l.translate('profile.key.regenerate.confirm_label')}
	onconfirm={regenerate}
/>
