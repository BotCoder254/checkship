<!--
	Registries (DESIGN.md §5 Lists): one list row per private registry, its logo in the tile, the URL
	and who it signs in as on the details line; its ⋯ menu has Edit and Delete. New registry and a registry's
	settings open in a drawer, so the list stays in view; deleting asks for its name in a dialog.
-->
<script lang="ts">
	import Add01Icon from '@hugeicons/core-free-icons/Add01Icon';
	import Delete02Icon from '@hugeicons/core-free-icons/Delete02Icon';
	import PencilEdit02Icon from '@hugeicons/core-free-icons/PencilEdit02Icon';
	import PackageIcon from '@hugeicons/core-free-icons/PackageIcon';
	import Tick02Icon from '@hugeicons/core-free-icons/Tick02Icon';
	import { toast } from 'svelte-sonner';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import Drawer from '#lib/components/drawer.svelte';
	import EmptyState from '#lib/components/empty-state.svelte';
	import Notice from '#lib/components/notice.svelte';
	import Icon from '#lib/components/icon.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import service, { type Registry } from '#lib/resources/registries.ts';
	import RowMenuGroup from '#lib/components/table/row-menu-group.svelte';
	import RowMenuItem from '#lib/components/table/row-menu-item.svelte';
	import RowMenu from '#lib/components/table/row-menu.svelte';
	import TimelineList from '#lib/components/timeline/timeline-list.svelte';
	import TimelineRow from '#lib/components/timeline/timeline-row.svelte';
	import RegistryForm from './registry-form.svelte';
	import RegistryLogo from '#lib/components/registry-logo.svelte';
	import { presetForUrl } from '#lib/registry-presets.ts';

	const { data: registries } = service.queryAll();

	function detailsOf(registry: Registry): string[] {
		return [
			registry.url,
			registry.credentials
				? l.translate('registries.auth.user', [registry.credentials.username])
				: l.translate('registries.auth.none')
		];
	}

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
	<div class="flex flex-col gap-3">
		<TimelineList label={l.translate('breadcrumb.registries')}>
			{#each $registries as registry (registry.id)}
				{@const preset = presetForUrl(registry.url)}
				<TimelineRow
					icon={PackageIcon}
					label={[registry.name, ...detailsOf(registry)].join(', ')}
					onclick={() => openEdit(registry)}
					selected={drawerOpen && editing?.id === registry.id}
				>
					{#snippet tile()}
						{#if preset}<RegistryLogo name={preset.id} class="size-4.5" />{:else}<Icon
								icon={PackageIcon}
								class="size-4.5"
							/>{/if}
					{/snippet}
					{#snippet title()}{registry.name}{/snippet}
					{#snippet details()}{detailsOf(registry).join(' · ')}{/snippet}
					{#snippet actions()}
						<RowMenu label={l.translate('table.menu', [registry.name])}>
							<RowMenuGroup>
								<RowMenuItem icon={PencilEdit02Icon} onSelect={() => openEdit(registry)}>
									{l.translate('menu.edit')}
								</RowMenuItem>
							</RowMenuGroup>
							<RowMenuGroup>
								<RowMenuItem
									icon={Delete02Icon}
									variant="destructive"
									onSelect={() => {
										editing = registry;
										deleteOpen = true;
									}}
								>
									{l.translate('menu.delete')}
								</RowMenuItem>
							</RowMenuGroup>
						</RowMenu>
					{/snippet}
				</TimelineRow>
			{/each}
		</TimelineList>
		<Notice><p>{l.translate('registries.note')}</p></Notice>
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
