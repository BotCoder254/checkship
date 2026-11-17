<!--
	Target settings (DESIGN.md §11.8). Delete follows the server: refused while environments deploy
	to the target (the dialog lists them, each linking to its settings) or while it configures;
	otherwise it asks for the name. The header is the one Overview has (target-header.svelte).
-->
<script lang="ts">
	import DashboardSquare01Icon from '@hugeicons/core-free-icons/DashboardSquare01Icon';
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
	import TimelineList from '#lib/components/timeline/timeline-list.svelte';
	import TimelineRow from '#lib/components/timeline/timeline-row.svelte';
	import Icon from '#lib/components/icon.svelte';
	import * as AlertDialog from '#lib/components/ui/alert-dialog/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import appsService from '#lib/resources/apps.ts';
	import service, { TargetStatus, type UpdateTarget } from '#lib/resources/targets.ts';
	import { usesOf } from '#lib/target.ts';
	import TargetForm from '../../target-form.svelte';
	import TargetHeader from '../target-header.svelte';

	let { data } = $props();

	const { data: apps } = appsService.queryAll();
	const { data: targets } = service.queryAll();

	let deleteOpen = $state(false);
	let blockedOpen = $state(false);

	// The header polls like on Overview; the form keeps the values it was opened with.
	const target = $derived($targets?.find((t) => t.id === data.target.id) ?? data.target);
	const configuring = $derived(target.state.status === TargetStatus.Configuring);
	const uses = $derived($apps ? usesOf($apps, target.id) : []);

	function askDelete() {
		if (uses.length > 0) blockedOpen = true;
		else deleteOpen = true;
	}

	const submit = (d: UpdateTarget) =>
		service.update(target.id, d).then(() => goto(routes.target(target.id)));
</script>

<TargetHeader {target} tab="settings" />

<TargetForm
	embedded
	initialData={data.target}
	handler={submit}
	disabled={!!target.cleanup_requested_at}
>
	{#snippet actions(submitting)}
		<!-- While it is being deleted, the header says so and there is nothing to do here. -->
		{#if !target.cleanup_requested_at}
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
		<TimelineList label={l.translate('targets.apps')}>
			{#each uses as use (`${use.app.id}-${use.environment}`)}
				<TimelineRow
					icon={DashboardSquare01Icon}
					label="{use.app.name}, {l.translate(`deploy.environment.${use.environment}`)}"
					href={routes.editApp(use.app.id)}
				>
					{#snippet title()}{use.app.name}{/snippet}
					{#snippet badge()}<EnvironmentChip environment={use.environment} />{/snippet}
					{#snippet actions()}
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
					{/snippet}
				</TimelineRow>
			{/each}
		</TimelineList>
		<AlertDialog.Footer>
			<AlertDialog.Cancel>{l.translate('close')}</AlertDialog.Cancel>
			<Button variant="destructive" disabled>
				<Icon icon={Delete02Icon} />
				{l.translate('target.delete')}
			</Button>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>
