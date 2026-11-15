<!--
	Registries (DESIGN.md §5 Tables): one table row per private registry, its ⋯ menu has Edit and Delete. New registry and a registry's
	settings open in a drawer, so the list stays in view; deleting asks for its name in a dialog.
-->
<script lang="ts">
	import Add01Icon from '@hugeicons/core-free-icons/Add01Icon';
	import Delete02Icon from '@hugeicons/core-free-icons/Delete02Icon';
	import InformationCircleIcon from '@hugeicons/core-free-icons/InformationCircleIcon';
	import Globe02Icon from '@hugeicons/core-free-icons/Globe02Icon';
	import Key01Icon from '@hugeicons/core-free-icons/Key01Icon';
	import PencilEdit02Icon from '@hugeicons/core-free-icons/PencilEdit02Icon';
	import TextIcon from '@hugeicons/core-free-icons/TextIcon';
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
	import RowMenuGroup from '#lib/components/table/row-menu-group.svelte';
	import RowMenuItem from '#lib/components/table/row-menu-item.svelte';
	import RowMenu from '#lib/components/table/row-menu.svelte';
	import * as Table from '#lib/components/ui/table/index.js';
	import { ACTIONS_CELL, ACTIONS_HEAD, TableColumns } from '#lib/table-columns.svelte.ts';
	import RegistryForm from './registry-form.svelte';
	import RegistryLogo from '#lib/components/registry-logo.svelte';
	import { presetForUrl } from '#lib/registry-presets.ts';

	const { data: registries } = service.queryAll();

	const columns = new TableColumns([
		{ id: 'name', label: l.translate('column.name'), icon: TextIcon, width: 'w-[30%]' },
		{
			id: 'url',
			label: l.translate('column.url'),
			icon: Globe02Icon,
			width: 'w-[40%]',
			priority: 2
		},
		{
			id: 'credentials',
			label: l.translate('column.credentials'),
			icon: Key01Icon,
			width: 'w-[30%]'
		}
	]);

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
		<Table.Root aria-label={l.translate('breadcrumb.registries')}>
			<Table.Header>
				<Table.Row>
					{#each columns.shown as column (column.id)}
						<Table.Head class={columns.head(column.id)}>{column.label}</Table.Head>
					{/each}
					<Table.Head class={ACTIONS_HEAD}>{l.translate('column.actions')}</Table.Head>
				</Table.Row>
			</Table.Header>
			<Table.Body>
				{#each $registries as registry (registry.id)}
					{@const preset = presetForUrl(registry.url)}
					<Table.Row
						class="relative"
						data-state={drawerOpen && editing?.id === registry.id ? 'selected' : undefined}
					>
						<Table.Cell class={columns.cell('name')}>
							<span class="flex min-w-0 items-center gap-2.5">
								{#if preset}<RegistryLogo name={preset.id} class="size-4.5 shrink-0" />{:else}<Icon
										icon={PackageIcon}
										class="size-4.5 shrink-0 text-muted-foreground"
									/>{/if}
								<!-- The whole row opens the drawer; the button stretches over it. -->
								<button
									type="button"
									title={registry.name}
									class="truncate text-left text-sm font-semibold outline-none after:absolute after:inset-0 after:rounded-md focus-visible:after:ring-2 focus-visible:after:ring-ring focus-visible:after:ring-inset"
									onclick={() => openEdit(registry)}>{registry.name}</button
								>
							</span>
						</Table.Cell>
						<Table.Cell
							class={columns.cell('url', 'font-mono text-[12.5px] text-muted-foreground')}
							title={registry.url}
						>
							{registry.url}
						</Table.Cell>
						<Table.Cell
							class={columns.cell(
								'credentials',
								registry.credentials ? 'text-muted-foreground' : 'text-faint'
							)}
						>
							{registry.credentials?.username ?? l.translate('registries.auth.none')}
						</Table.Cell>
						<Table.Cell class={ACTIONS_CELL}>
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
						</Table.Cell>
					</Table.Row>
				{/each}
			</Table.Body>
		</Table.Root>
		<div class="grid grid-cols-[20px_minmax(0,1fr)] gap-3 rounded-lg bg-sunken px-4 py-3.5">
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
