<script lang="ts">
	import Delete02Icon from '@hugeicons/core-free-icons/Delete02Icon';
	import RefreshIcon from '@hugeicons/core-free-icons/RefreshIcon';
	import Tick02Icon from '@hugeicons/core-free-icons/Tick02Icon';
	import { goto } from '$app/navigation';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import Icon from '#lib/components/icon.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service, { TargetStatus, type UpdateTarget } from '#lib/resources/targets.ts';
	import TargetForm from '../../target-form.svelte';

	let { data } = $props();

	let deleteOpen = $state(false);
	let reconfigureOpen = $state(false);

	const target = $derived(data.target);
	const configuring = $derived(target.state.status === TargetStatus.Configuring);

	const submit = (d: UpdateTarget) => service.update(target.id, d).then(() => goto(routes.targets));
</script>

<TargetForm
	title={l.translate('breadcrumb.target.settings', [target.name])}
	crumbs={[{ href: routes.targets, label: l.translate('breadcrumb.targets') }]}
	initialData={target}
	handler={submit}
>
	{#snippet actions(submitting)}
		{#if target.cleanup_requested_at}
			<StatusBadge state="deleting" label={l.translate('apps.deletion_requested')} />
		{:else}
			<Button variant="destructive" disabled={configuring} onclick={() => (deleteOpen = true)}>
				<Icon icon={Delete02Icon} />
				{l.translate('target.delete')}
			</Button>
			<Button variant="outline" disabled={configuring} onclick={() => (reconfigureOpen = true)}>
				<Icon icon={RefreshIcon} />
				{l.translate('target.reconfigure')}
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
	title={l.translate('target.delete')}
	description={l.translate('target.delete.description', [target.name])}
	confirmLabel={l.translate('target.delete')}
	icon={Delete02Icon}
	typeToConfirm={target.name}
	typeLabel={l.translate('target.delete.type_name')}
	onconfirm={() => service.delete(target.id).then(() => goto(routes.targets))}
/>

<ConfirmDialog
	bind:open={reconfigureOpen}
	title={l.translate('target.reconfigure.title')}
	description={l.translate('target.reconfigure.confirm')}
	confirmLabel={l.translate('target.reconfigure')}
	icon={RefreshIcon}
	onconfirm={() => service.reconfigure(target.id).then(() => goto(routes.targets))}
/>
