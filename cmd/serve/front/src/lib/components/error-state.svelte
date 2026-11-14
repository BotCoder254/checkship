<!--
	Not found and unexpected errors (DESIGN.md §11.5): the same empty-state block, with the way back.
	404 names what was missing; anything else offers Try again and shows the code.
-->
<script lang="ts">
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import ArrowLeft01Icon from '@hugeicons/core-free-icons/ArrowLeft01Icon';
	import RefreshIcon from '@hugeicons/core-free-icons/RefreshIcon';
	import Search01Icon from '@hugeicons/core-free-icons/Search01Icon';
	import { page } from '$app/state';
	import Icon from '#lib/components/icon.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import l, { type AppTranslationsString } from '#lib/localization/index.ts';
	import { cn } from '#lib/utils.js';

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
	class="mt-6 flex flex-col items-center gap-3 rounded-md bg-sunken px-6 py-14 text-center"
>
	<span
		class={cn(
			'flex size-12 items-center justify-center rounded-full',
			notFound ? 'bg-raised text-muted-foreground' : 'bg-danger-soft text-danger'
		)}
	>
		<Icon icon={notFound ? Search01Icon : Alert02Icon} class="size-6" />
	</span>
	<div class="flex max-w-100 flex-col gap-1">
		<h1 class="text-base font-semibold">
			{notFound
				? l.translate(`errors.not_found.${what}` as AppTranslationsString)
				: l.translate('errors.title')}
		</h1>
		<p class="text-sm text-muted-foreground">
			{notFound ? l.translate('errors.not_found.description') : l.translate('errors.description')}
		</p>
	</div>
	{#if !notFound}
		<p class="text-xs text-faint">
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
