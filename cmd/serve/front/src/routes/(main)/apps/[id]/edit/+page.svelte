<script lang="ts">
	import Delete02Icon from '@hugeicons/core-free-icons/Delete02Icon';
	import Tick02Icon from '@hugeicons/core-free-icons/Tick02Icon';
	import { goto } from '$app/navigation';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import Icon from '#lib/components/icon.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service, { type UpdateApp } from '#lib/resources/apps.ts';
	import AppForm from '../../app-form.svelte';
	import AppTabs from '../app-tabs.svelte';

	let { data } = $props();

	let deleteOpen = $state(false);
	let deleting = $state(false);

	const submit = (form: UpdateApp) =>
		service.update(data.app.id, form).then(() => goto(routes.app(data.app.id)));

	async function remove() {
		deleting = true;
		try {
			await service.delete(data.app.id);
			await goto(routes.apps);
		} finally {
			deleting = false;
		}
	}
</script>

<AppForm
	title={l.translate('breadcrumb.settings')}
	crumbs={[
		{ href: routes.apps, label: l.translate('breadcrumb.applications') },
		{ href: routes.app(data.app.id), label: data.app.name }
	]}
	targets={data.targets}
	initialData={data.app}
	handler={submit}
	disabled={deleting || !!data.app.cleanup_requested_at}
>
	{#snippet tabs()}
		<AppTabs appId={data.app.id} active="settings" />
	{/snippet}
	{#snippet actions(submitting)}
		{#if !data.app.cleanup_requested_at}
			<Button variant="destructive" onclick={() => (deleteOpen = true)} disabled={submitting}>
				<Icon icon={Delete02Icon} />
				{l.translate('app.delete')}
			</Button>
			<Button type="submit" loading={submitting}>
				<Icon icon={Tick02Icon} />
				{l.translate('save')}
			</Button>
		{/if}
	{/snippet}
</AppForm>

<ConfirmDialog
	bind:open={deleteOpen}
	destructive
	icon={Delete02Icon}
	title={l.translate('app.delete.title', [data.app.name])}
	description={l.translate('app.delete.description')}
	confirmLabel={l.translate('app.delete')}
	typeToConfirm={data.app.name}
	typeLabel={l.translate('app.delete.type_name', [data.app.name])}
	onconfirm={remove}
/>
