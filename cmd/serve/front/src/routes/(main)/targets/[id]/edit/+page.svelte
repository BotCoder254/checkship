<!--
	Target settings (DESIGN.md §11.8). Delete follows the server: refused while environments deploy
	to the target (the dialog lists them, each linking to its settings) or while it configures;
	otherwise it asks for the name.
-->
<script lang="ts">
	import ArrowRight01Icon from '@hugeicons/core-free-icons/ArrowRight01Icon';
	import Delete02Icon from '@hugeicons/core-free-icons/Delete02Icon';
	import Tick02Icon from '@hugeicons/core-free-icons/Tick02Icon';
	import { goto } from '$app/navigation';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import Icon from '#lib/components/icon.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import * as AlertDialog from '#lib/components/ui/alert-dialog/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import appsService from '#lib/resources/apps.ts';
	import service, { TargetStatus, type UpdateTarget } from '#lib/resources/targets.ts';
	import { usesOf } from '#lib/target.ts';
	import { cn } from '#lib/utils.js';
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
		<ul class="flex flex-col gap-1.5">
			{#each uses as use (`${use.app.id}-${use.environment}`)}
				<li>
					<a
						href={routes.editApp(use.app.id)}
						class="flex h-10 items-center justify-between rounded-md bg-sunken px-3 text-[13.5px] outline-none hover:bg-hover focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset"
					>
						<span class="flex items-center gap-2.5">
							<strong class="font-semibold">{use.app.name}</strong>
							<span
								class={cn(
									'rounded-sm px-1.5 text-[11.5px]/5 font-semibold',
									use.environment === 'production'
										? 'bg-accent-soft text-primary'
										: 'bg-raised text-muted-foreground'
								)}>{use.environment}</span
							>
						</span>
						<span class="flex items-center gap-1 text-[13px] font-semibold text-primary">
							{l.translate('app.settings')}
							<Icon icon={ArrowRight01Icon} class="size-3.5" />
						</span>
					</a>
				</li>
			{/each}
		</ul>
		<AlertDialog.Footer>
			<AlertDialog.Cancel>{l.translate('close')}</AlertDialog.Cancel>
			<Button variant="destructive" disabled>
				<Icon icon={Delete02Icon} />
				{l.translate('target.delete')}
			</Button>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>
