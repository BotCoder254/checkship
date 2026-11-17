<!--
	Sign-in (DESIGN.md §5, §11.5). Errors come from the API: `invalid_email_or_password` is set on
	both fields, so it is shown once as an alert and both inputs are marked invalid. Any other field
	code shows under its field until that field is edited.
-->
<script lang="ts">
	import Icon from '#lib/components/icon.svelte';
	import AlertCircleIcon from '@hugeicons/core-free-icons/AlertCircleIcon';
	import CancelCircleIcon from '@hugeicons/core-free-icons/CancelCircleIcon';
	import Logo from '#lib/components/logo.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import auth from '#lib/auth.ts';
	import { GLOBAL_ERROR_NAME, submitter } from '#lib/form.ts';
	import l, { type AppTranslationsString } from '#lib/localization/index.ts';

	const CREDENTIALS_ERROR = 'invalid_email_or_password';

	let email = $state('');
	let password = $state('');
	let emailInput = $state<HTMLInputElement | null>(null);

	const { submit, loading, errors } = submitter(() => auth.login(email, password));

	const credentialsRejected = $derived(
		$errors?.email === CREDENTIALS_ERROR || $errors?.password === CREDENTIALS_ERROR
	);
	const alertCode = $derived(
		credentialsRejected ? CREDENTIALS_ERROR : $errors?.[GLOBAL_ERROR_NAME]
	);

	/** Field codes still shown: follows each submit, and a field's code is dropped once it is edited. */
	let fieldErrors = $derived<{ email?: string; password?: string }>(
		credentialsRejected
			? {}
			: { email: $errors?.email ?? undefined, password: $errors?.password ?? undefined }
	);

	$effect(() => {
		emailInput?.focus();
	});

	function translate(code: string) {
		return l.translate(code as AppTranslationsString);
	}

	function onsubmit(event: SubmitEvent) {
		event.preventDefault();
		if ($loading) return;

		// The submitter keeps the error for display; nothing else to do with it here.
		submit().catch(() => {});
	}
</script>

<svelte:head>
	<title>checkship - {l.translate('auth.signin.title')}</title>
</svelte:head>

<form class="flex w-full max-w-90 flex-col gap-7.5 md:gap-9" novalidate {onsubmit}>
	<Logo size={56} />

	<div class="flex flex-col gap-6">
		<div class="flex flex-col gap-1.5">
			<h1 class="text-xl/7 font-semibold tracking-tight">{l.translate('auth.signin.title')}</h1>
			<p class="text-base text-muted-foreground">{l.translate('auth.signin.description')}</p>
		</div>

		{#if alertCode}
			<Alert.Root variant="destructive" class="p-3">
				<Icon icon={CancelCircleIcon} />
				<Alert.Title class="font-semibold">{translate(alertCode)}</Alert.Title>
			</Alert.Root>
		{/if}

		<div class="flex flex-col gap-4.5">
			<Field.Field class="gap-1.5">
				<Field.Label for="email">{l.translate('email')}</Field.Label>
				<Input
					id="email"
					type="email"
					autocomplete="username"
					placeholder="admin@example.com"
					class="h-11 md:h-9"
					bind:ref={emailInput}
					bind:value={email}
					readonly={$loading}
					aria-invalid={credentialsRejected || !!fieldErrors.email || undefined}
					aria-describedby={fieldErrors.email ? 'email-error' : undefined}
					oninput={() => (fieldErrors = { ...fieldErrors, email: undefined })}
				/>
				{#if fieldErrors.email}
					<Field.Error id="email-error" class="flex items-center gap-1.5">
						<Icon icon={AlertCircleIcon} class="size-3.5 shrink-0" aria-hidden="true" />
						{translate(fieldErrors.email)}
					</Field.Error>
				{/if}
			</Field.Field>

			<Field.Field class="gap-1.5">
				<Field.Label for="password">{l.translate('password')}</Field.Label>
				<Input
					id="password"
					type="password"
					autocomplete="current-password"
					class="h-11 md:h-9"
					bind:value={password}
					readonly={$loading}
					aria-invalid={credentialsRejected || !!fieldErrors.password || undefined}
					aria-describedby={fieldErrors.password ? 'password-error' : undefined}
					oninput={() => (fieldErrors = { ...fieldErrors, password: undefined })}
				/>
				{#if fieldErrors.password}
					<Field.Error id="password-error" class="flex items-center gap-1.5">
						<Icon icon={AlertCircleIcon} class="size-3.5 shrink-0" aria-hidden="true" />
						{translate(fieldErrors.password)}
					</Field.Error>
				{/if}
			</Field.Field>
		</div>

		<Button
			type="submit"
			loading={$loading}
			class="h-11 w-full text-base font-semibold md:h-9 md:text-sm"
		>
			{l.translate('auth.signin.title')}
		</Button>
	</div>
</form>
