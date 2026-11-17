<!--
	Not found and unexpected errors (DESIGN.md §11.5): written on the page like an empty state (no box): the notFound or error drawing at page size, then the way back.
	404 names what was missing; anything else offers Try again and shows the code.
-->
<script lang="ts">
	import ArrowLeft01Icon from '@hugeicons/core-free-icons/ArrowLeft01Icon';
	import RefreshIcon from '@hugeicons/core-free-icons/RefreshIcon';
	import { page } from '$app/state';
	import Icon from '#lib/components/icon.svelte';
	import Illustration from '#lib/components/illustration.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import l, { type AppTranslationsString } from '#lib/localization/index.ts';

	type Props = {
		/** What was being looked for, e.g. errors.app */
		what: 'app' | 'target' | 'registry' | 'deployment' | 'page';
		back: { href: string; label: string };
	};

	let { what, back }: Props = $props();

	const notFound = $derived(page.status === 404);
</script>

<svelte:head>
	<title
		>checkship - {notFound ? l.translate('errors.not_found') : l.translate('errors.title')}</title
	>
</svelte:head>

<div
	role={notFound ? undefined : 'alert'}
	class="flex flex-col items-center gap-3 px-6 py-12 text-center"
>
	<Illustration name={notFound ? 'notFound' : 'error'} class="w-[280px] max-w-full" />
	<div class="mt-2 flex max-w-100 flex-col gap-1.5">
		<h1 class="text-lg/[26px] font-semibold">
			{notFound
				? l.translate(`errors.not_found.${what}` as AppTranslationsString)
				: l.translate('errors.title')}
		</h1>
		<p class="text-sm text-muted-foreground">
			{notFound ? l.translate('errors.not_found.description') : l.translate('errors.description')}
		</p>
	</div>
	{#if !notFound}
		<p class="text-xs text-muted-foreground">
			{l.translate('errors.code', [page.status])}{#if page.error?.message}
				· <span class="font-mono">{page.error.message}</span>{/if}
		</p>
	{/if}
	<div class="flex flex-wrap justify-center gap-2">
		{#if !notFound}
			<Button onclick={() => location.reload()}>
				<Icon icon={RefreshIcon} />
				{l.translate('errors.retry')}
			</Button>
		{/if}
		<Button variant="outline" href={back.href}>
			<Icon icon={ArrowLeft01Icon} />
			{back.label}
		</Button>
	</div>
</div>
