<!--
	Registries: one row per private registry on a sunken fill. Rows open the registry's settings.
-->
<script lang="ts">
	import Add01Icon from '@hugeicons/core-free-icons/Add01Icon';
	import ArrowRight01Icon from '@hugeicons/core-free-icons/ArrowRight01Icon';
	import PackageIcon from '@hugeicons/core-free-icons/PackageIcon';
	import EmptyState from '#lib/components/empty-state.svelte';
	import Icon from '#lib/components/icon.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service from '#lib/resources/registries.ts';

	const { data: registries } = service.queryAll();
</script>

<PageHeader title={l.translate('breadcrumb.registries')}>
	{#snippet actions()}
		<Button href={routes.createRegistry}>
			<Icon icon={Add01Icon} />
			{l.translate('registry.new')}
		</Button>
	{/snippet}
	{#snippet status()}
		{#if $registries}
			<span>{l.translate('registries.summary', [$registries.length])}</span>
		{/if}
	{/snippet}
</PageHeader>

{#if $registries && $registries.length > 0}
	<div role="table" aria-label={l.translate('breadcrumb.registries')} class="flex flex-col gap-1.5">
		<div
			role="row"
			class="hidden grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)_20px] gap-4.5 px-4.5 pb-1.5 text-xs font-semibold text-muted-foreground md:grid"
		>
			<span role="columnheader">{l.translate('registries.column.registry')}</span>
			<span role="columnheader">{l.translate('registries.column.auth')}</span>
			<span role="columnheader"
				><span class="sr-only">{l.translate('breadcrumb.registries')}</span></span
			>
		</div>

		{#each $registries as registry (registry.id)}
			<a
				role="row"
				href={routes.editRegistry(registry.id)}
				class="grid grid-cols-1 gap-3 rounded-md bg-sunken px-4 py-3.5 text-ink transition-colors outline-none hover:bg-hover focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset md:grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)_20px] md:items-center md:gap-4.5 md:px-4.5"
			>
				<span role="cell" class="flex min-w-0 flex-col gap-0.5">
					<span class="truncate text-[15px]/[22px] font-semibold">{registry.name}</span>
					<span class="truncate font-mono text-xs text-muted-foreground">{registry.url}</span>
				</span>
				<span role="cell" class="text-sm text-muted-foreground">
					{registry.credentials
						? l.translate('registries.auth.user', [registry.credentials.username])
						: l.translate('registries.auth.none')}
				</span>
				<Icon icon={ArrowRight01Icon} class="hidden size-4.5 text-muted-foreground md:block" />
			</a>
		{/each}
	</div>
{:else if $registries}
	<EmptyState
		icon={PackageIcon}
		title={l.translate('registries.empty.title')}
		description={l.translate('registries.empty.description')}
	>
		{#snippet action()}
			<Button href={routes.createRegistry}>
				<Icon icon={Add01Icon} />
				{l.translate('registry.new')}
			</Button>
		{/snippet}
	</EmptyState>
{/if}
