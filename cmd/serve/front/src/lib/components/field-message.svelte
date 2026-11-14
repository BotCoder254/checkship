<!--
	Under a field (DESIGN.md §11.5): the error when there is one (danger, with an icon, linked to the
	input by its id), otherwise the help text in muted.
-->
<script lang="ts">
	import AlertCircleIcon from '@hugeicons/core-free-icons/AlertCircleIcon';
	import type { Snippet } from 'svelte';
	import Icon from '#lib/components/icon.svelte';
	import l, { type AppTranslationsString } from '#lib/localization/index.ts';

	type Props = { id: string; error?: string; help?: Snippet };

	let { id, error, help }: Props = $props();
</script>

{#if error}
	<p {id} role="alert" class="flex items-center gap-1.5 text-sm text-danger">
		<Icon icon={AlertCircleIcon} class="size-3.5" />
		{l.translate(error as AppTranslationsString)}
	</p>
{:else if help}
	<div
		{id}
		class="text-sm text-muted-foreground [&_a]:text-primary [&_a]:underline [&_code]:font-mono"
	>
		{@render help()}
	</div>
{/if}
