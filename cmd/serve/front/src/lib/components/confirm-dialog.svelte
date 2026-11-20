<!--
	Confirmation for actions that change or remove something (DESIGN.md §11.5): replaces
	window.confirm. The confirm button has the loading state and the dialog stays open on error, so
	the reason shows where the decision was made. With `typeToConfirm`, the action stays disabled
	until that exact text is typed.
-->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { IconData } from '#lib/components/icon.svelte';
	import Icon from '#lib/components/icon.svelte';
	import * as AlertDialog from '#lib/components/ui/alert-dialog/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { BadRequestError } from '#lib/error.ts';
	import l, { type AppTranslationsString } from '#lib/localization/index.ts';

	type Props = {
		open: boolean;
		title: string;
		description: string;
		confirmLabel: string;
		/** The button that closes without doing anything; "Cancel" by default. */
		cancelLabel?: string;
		icon?: IconData;
		destructive?: boolean;
		typeToConfirm?: string;
		typeLabel?: string;
		/** More controls between the description and the confirmation, such as an option. */
		extra?: Snippet;
		onconfirm: () => Promise<unknown>;
	};

	let {
		open = $bindable(false),
		title,
		description,
		confirmLabel,
		cancelLabel,
		icon,
		destructive = false,
		typeToConfirm,
		typeLabel,
		extra,
		onconfirm
	}: Props = $props();

	let pending = $state(false);
	let error = $state<string>();
	let typed = $state('');

	const blocked = $derived(typeToConfirm !== undefined && typed !== typeToConfirm);

	$effect(() => {
		if (open) {
			error = undefined;
			typed = '';
		}
	});

	async function confirm() {
		if (pending || blocked) return;
		pending = true;
		error = undefined;
		try {
			await onconfirm();
			open = false;
		} catch (ex) {
			const code =
				ex instanceof BadRequestError
					? (Object.values(ex.fields)[0] ?? ex.message)
					: ex instanceof Error
						? ex.message
						: 'unexpected_error';
			error = l.translate(code as AppTranslationsString);
		} finally {
			pending = false;
		}
	}
</script>

<AlertDialog.Root bind:open>
	<AlertDialog.Content class="gap-5">
		<AlertDialog.Header>
			<AlertDialog.Title>{title}</AlertDialog.Title>
			<AlertDialog.Description>{description}</AlertDialog.Description>
		</AlertDialog.Header>

		{@render extra?.()}

		{#if typeToConfirm !== undefined}
			<div class="flex flex-col gap-1.5">
				<Label for="confirm-type">{typeLabel}</Label>
				<Input id="confirm-type" bind:value={typed} autocomplete="off" spellcheck="false" />
			</div>
		{/if}

		{#if error}
			<p role="alert" class="rounded-md bg-danger-soft px-3 py-2.5 text-sm font-semibold text-ink">
				{error}
			</p>
		{/if}

		<AlertDialog.Footer>
			<AlertDialog.Cancel disabled={pending}
				>{cancelLabel ?? l.translate('cancel')}</AlertDialog.Cancel
			>
			<Button
				variant={destructive ? 'destructive' : 'default'}
				loading={pending}
				disabled={blocked}
				onclick={confirm}
			>
				{#if icon && !pending}<Icon {icon} />{/if}
				{confirmLabel}
			</Button>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>
