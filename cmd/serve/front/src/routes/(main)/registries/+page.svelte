<!--
	Registries: one row per private registry on a sunken fill. New registry and a registry's
	settings open in a drawer, so the list stays in view; deleting asks for its name in a dialog.
-->
<script lang="ts">
	import Add01Icon from '@hugeicons/core-free-icons/Add01Icon';
	import ArrowRight01Icon from '@hugeicons/core-free-icons/ArrowRight01Icon';
	import Delete02Icon from '@hugeicons/core-free-icons/Delete02Icon';
	import InformationCircleIcon from '@hugeicons/core-free-icons/InformationCircleIcon';
	import Key01Icon from '@hugeicons/core-free-icons/Key01Icon';
	import PackageIcon from '@hugeicons/core-free-icons/PackageIcon';
	import Tick02Icon from '@hugeicons/core-free-icons/Tick02Icon';
	import { toast } from 'svelte-sonner';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import Drawer from '#lib/components/drawer.svelte';
	import EmptyState from '#lib/components/empty-state.svelte';
	import Icon from '#lib/components/icon.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import service, { type Registry } from '#lib/resources/registries.ts';
	import { cn } from '#lib/utils.js';
	import RegistryForm from './registry-form.svelte';
	import RegistryLogo from '#lib/components/registry-logo.svelte';
	import { presetForUrl } from '#lib/registry-presets.ts';

	const { data: registries } = service.queryAll();

	let drawerOpen = $state(false);
	let editing = $state<Registry>();
	let saving = $state(false);
	let deleteOpen = $state(false);
	// Remount the form each time the drawer opens, so it starts from the registry shown.
	let formKey = $state(0);

	function openNew() {
		editing = undefined;
		formKey++;
		drawerOpen = true;
	}

	function openEdit(registry: Registry) {
		editing = registry;
		formKey++;
		drawerOpen = true;
	}

	async function saved(registry: Registry) {
		await service.fetchAll({ cache: 'no-store' });
		toast.success(
			l.translate(editing ? 'registries.saved' : 'registries.created', [registry.name])
		);
		drawerOpen = false;
	}

	async function remove() {
		await service.delete(editing!.id);
		await service.fetchAll({ cache: 'no-store' });
		drawerOpen = false;
	}
</script>

<PageHeader title={l.translate('breadcrumb.registries')}>
	{#snippet actions()}
		<Button onclick={openNew}>
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
			class="hidden grid-cols-[36px_minmax(0,1.6fr)_minmax(0,1fr)_16px] gap-4 px-4 pb-1.5 text-xs font-semibold text-muted-foreground md:grid"
		>
			<span role="columnheader"></span>
			<span role="columnheader">{l.translate('registries.column.registry')}</span>
			<span role="columnheader">{l.translate('registries.column.auth')}</span>
			<span role="columnheader"></span>
		</div>

		{#each $registries as registry (registry.id)}
			{@const preset = presetForUrl(registry.url)}
			<button
				type="button"
				role="row"
				class={cn(
					'grid grid-cols-[36px_minmax(0,1fr)] items-center gap-x-4 gap-y-1 rounded-md bg-sunken px-4 py-3 text-left text-ink transition-colors outline-none hover:bg-hover focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset md:grid-cols-[36px_minmax(0,1.6fr)_minmax(0,1fr)_16px]',
					drawerOpen && editing?.id === registry.id && 'bg-raised'
				)}
				onclick={() => openEdit(registry)}
			>
				<span
					role="cell"
					class="flex size-9 items-center justify-center rounded-md bg-raised text-muted-foreground max-md:row-span-2"
				>
					{#if preset}<RegistryLogo name={preset.id} class="size-5" />{:else}<Icon
							icon={PackageIcon}
							class="size-4.5"
						/>{/if}
				</span>
				<span role="cell" class="flex min-w-0 flex-col gap-0.5">
					<span class="truncate text-[15px]/[22px] font-semibold">{registry.name}</span>
					<span class="truncate font-mono text-xs text-muted-foreground">{registry.url}</span>
				</span>
				<span role="cell" class="flex items-center gap-1.5 text-sm text-muted-foreground">
					{#if registry.credentials}
						<Icon icon={Key01Icon} class="size-3.5" />
						{l.translate('registries.auth.user', [registry.credentials.username])}
					{:else}
						{l.translate('registries.auth.none')}
					{/if}
				</span>
				<Icon icon={ArrowRight01Icon} class="hidden size-4 text-faint md:block" />
			</button>
		{/each}
		<div class="mt-2.5 grid grid-cols-[20px_minmax(0,1fr)] gap-3 rounded-lg bg-sunken px-4 py-3.5">
			<Icon icon={InformationCircleIcon} class="size-4.5 text-muted-foreground" />
			<p class="text-[13px]/5 text-muted-foreground">{l.translate('registries.note')}</p>
		</div>
	</div>
{:else if $registries}
	<EmptyState
		icon={PackageIcon}
		title={l.translate('registries.empty.title')}
		description={l.translate('registries.empty.description')}
	>
		{#snippet action()}
			<Button onclick={openNew}>
				<Icon icon={Add01Icon} />
				{l.translate('registry.new')}
			</Button>
		{/snippet}
	</EmptyState>
{/if}

<Drawer
	bind:open={drawerOpen}
	title={editing ? editing.name : l.translate('registry.new')}
	description={l.translate('registries.drawer.description')}
>
	{#key formKey}
		<RegistryForm
			id="registry-drawer"
			initialData={editing}
			bind:loading={saving}
			onsaved={saved}
		/>
	{/key}
	{#snippet footer()}
		{#if editing}
			<Button variant="destructive" class="md:mr-auto" onclick={() => (deleteOpen = true)}>
				<Icon icon={Delete02Icon} />
				{l.translate('registry.delete')}
			</Button>
		{/if}
		<Button variant="outline" onclick={() => (drawerOpen = false)}>{l.translate('cancel')}</Button>
		<Button type="submit" form="registry-drawer" loading={saving}>
			<Icon icon={editing ? Tick02Icon : Add01Icon} />
			{editing ? l.translate('save') : l.translate('registries.create')}
		</Button>
	{/snippet}
</Drawer>

{#if editing}
	<ConfirmDialog
		bind:open={deleteOpen}
		destructive
		icon={Delete02Icon}
		title={l.translate('registries.delete.title', [editing.name])}
		description={l.translate('registries.delete.description')}
		confirmLabel={l.translate('registry.delete')}
		typeToConfirm={editing.name}
		typeLabel={l.translate('registries.delete.type_name', [editing.name])}
		onconfirm={remove}
	/>
{/if}
