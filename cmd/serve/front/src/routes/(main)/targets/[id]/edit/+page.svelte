<!--
	Target settings (DESIGN.md §11.8). Delete follows the server: refused while environments deploy
	to the target (the dialog lists them, each linking to its settings) or while it configures;
	otherwise it asks for the name.
-->
<script lang="ts">
	import Settings02Icon from '@hugeicons/core-free-icons/Settings02Icon';
	import ViewIcon from '@hugeicons/core-free-icons/ViewIcon';
	import Delete02Icon from '@hugeicons/core-free-icons/Delete02Icon';
	import Tick02Icon from '@hugeicons/core-free-icons/Tick02Icon';
	import { goto } from '$app/navigation';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import EnvironmentChip from '#lib/components/environment-chip.svelte';
	import RowMenuGroup from '#lib/components/table/row-menu-group.svelte';
	import RowMenuItem from '#lib/components/table/row-menu-item.svelte';
	import RowMenu from '#lib/components/table/row-menu.svelte';
	import * as Table from '#lib/components/ui/table/index.js';
	import { ACTIONS_CELL, ACTIONS_HEAD } from '#lib/table-columns.svelte.ts';
	import Icon from '#lib/components/icon.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import * as AlertDialog from '#lib/components/ui/alert-dialog/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import appsService from '#lib/resources/apps.ts';
	import service, { TargetStatus, type UpdateTarget } from '#lib/resources/targets.ts';
	import { usesOf } from '#lib/target.ts';
	import TargetForm from '../../target-form.svelte';
	import TargetTabs from '../../target-tabs.svelte';

	let { data } = $props();

	const { data: apps } = appsService.queryAll();

	let deleteOpen = $state(false);
	let blockedOpen = $state(false);

	const target = $derived(data.target);
	const configuring = $derived(target.state.status === TargetStatus.Configuring);
	const uses = $derived($apps ? usesOf($apps, target.id) : []);

	function askDelete() {
		if (uses.length > 0) blockedOpen = true;
		else deleteOpen = true;
	}

	const submit = (d: UpdateTarget) =>
		service.update(target.id, d).then(() => goto(routes.target(target.id)));
</script>

<TargetForm
	title={l.translate('breadcrumb.settings')}
	crumbs={[
		{ href: routes.targets, label: l.translate('breadcrumb.targets') },
		{ href: routes.target(target.id), label: target.name }
	]}
	initialData={target}
	handler={submit}
	disabled={!!target.cleanup_requested_at}
>
	{#snippet tabs()}
		<TargetTabs targetId={target.id} active="settings" />
	{/snippet}
	{#snippet actions(submitting)}
		{#if target.cleanup_requested_at}
			<StatusBadge state="deleting" label={l.translate('targets.state.deleting')} />
		{:else}
			<Button
				variant="destructive"
				disabled={configuring}
				title={configuring ? l.translate('targets.delete.configuring') : undefined}
				onclick={askDelete}
			>
				<Icon icon={Delete02Icon} />
				{l.translate('target.delete')}
			</Button>
			<Button type="submit" loading={submitting}>
				<Icon icon={Tick02Icon} />
				{l.translate('save')}
			</Button>
		{/if}
	{/snippet}
</TargetForm>

<ConfirmDialog
	bind:open={deleteOpen}
	destructive
	title={l.translate('targets.delete.title', [target.name])}
	description={target.state.last_ready_version
		? l.translate('target.delete.description', [target.name])
		: l.translate('targets.delete.never_ready')}
	confirmLabel={l.translate('target.delete')}
	icon={Delete02Icon}
	typeToConfirm={target.name}
	typeLabel={l.translate('target.delete.type_name')}
	onconfirm={() => service.delete(target.id).then(() => goto(routes.targets))}
/>

<AlertDialog.Root bind:open={blockedOpen}>
	<AlertDialog.Content class="sm:max-w-[470px]">
		<div class="flex gap-3.5">
			<span
				class="flex size-10 shrink-0 items-center justify-center rounded-full bg-danger-soft text-danger"
			>
				<Icon icon={Delete02Icon} class="size-5" />
			</span>
			<AlertDialog.Header class="gap-1.5 text-left">
				<AlertDialog.Title class="text-lg/[26px] font-semibold">
					{l.translate('targets.blocked.title', [target.name])}
				</AlertDialog.Title>
				<AlertDialog.Description class="text-sm text-ink">
					{l.translate('targets.blocked.description', [target.name])}
				</AlertDialog.Description>
			</AlertDialog.Header>
		</div>
		<Table.Root aria-label={l.translate('targets.apps')}>
			<Table.Header>
				<Table.Row>
					<Table.Head class="w-[55%]">{l.translate('column.name')}</Table.Head>
					<Table.Head class="w-[45%]">{l.translate('column.environment')}</Table.Head>
					<Table.Head class={ACTIONS_HEAD}>{l.translate('column.actions')}</Table.Head>
				</Table.Row>
			</Table.Header>
			<Table.Body>
				{#each uses as use (`${use.app.id}-${use.environment}`)}
					<Table.Row>
						<Table.Cell class="text-sm font-semibold">
							<a href={routes.editApp(use.app.id)} title={use.app.name} class="hover:underline"
								>{use.app.name}</a
							>
						</Table.Cell>
						<Table.Cell><EnvironmentChip environment={use.environment} /></Table.Cell>
						<Table.Cell class={ACTIONS_CELL}>
							<RowMenu label={l.translate('table.menu', [use.app.name])}>
								<RowMenuGroup>
									<RowMenuItem icon={Settings02Icon} href={routes.editApp(use.app.id)}>
										{l.translate('app.settings')}
									</RowMenuItem>
									<RowMenuItem icon={ViewIcon} href={routes.app(use.app.id)}>
										{l.translate('menu.open')}
									</RowMenuItem>
								</RowMenuGroup>
							</RowMenu>
						</Table.Cell>
					</Table.Row>
				{/each}
			</Table.Body>
		</Table.Root>
		<AlertDialog.Footer>
			<AlertDialog.Cancel>{l.translate('close')}</AlertDialog.Cancel>
			<Button variant="destructive" disabled>
				<Icon icon={Delete02Icon} />
				{l.translate('target.delete')}
			</Button>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>
