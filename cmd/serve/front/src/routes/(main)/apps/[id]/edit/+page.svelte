<script lang="ts">
	import Delete02Icon from '@hugeicons/core-free-icons/Delete02Icon';
	import Tick02Icon from '@hugeicons/core-free-icons/Tick02Icon';
	import { goto } from '$app/navigation';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import Icon from '#lib/components/icon.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Switch } from '#lib/components/ui/switch/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service, { type UpdateApp } from '#lib/resources/apps.ts';
	import appsService from '#lib/resources/apps.ts';
	import AppForm from '../../app-form.svelte';
	import AppWorkspace from '../app-workspace.svelte';

	let { data } = $props();

	// The header polls like on the other tabs; the form keeps the values it was opened with.
	const appQuery = $derived(appsService.queryById(data.app.id));
	const appStore = $derived(appQuery.data);
	const app = $derived($appStore ?? data.app);

	let deleteOpen = $state(false);
	let deleting = $state(false);
	let backupFirst = $state(true);

	const submit = (form: UpdateApp) =>
		service.update(data.app.id, form).then(() => goto(routes.app(data.app.id)));

	async function remove() {
		deleting = true;
		try {
			await service.delete(data.app.id, backupFirst);
			await goto(routes.apps);
		} finally {
			deleting = false;
		}
	}
</script>

<AppWorkspace {app} apiKey={data.user.api_key} tab="settings">
	<AppForm
		embedded
		targets={data.targets}
		initialData={data.app}
		handler={submit}
		disabled={deleting || !!data.app.cleanup_requested_at}
	>
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
</AppWorkspace>

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
>
	{#snippet extra()}
		<div class="flex items-start gap-4 py-1">
			<span class="flex min-w-0 flex-1 flex-col gap-1">
				<label for="delete-backup" class="text-sm font-medium"
					>{l.translate('app.delete.backup')}</label
				>
				<span class="text-xs/[18px] {backupFirst ? 'text-muted-foreground' : 'text-danger'}">
					{backupFirst ? l.translate('app.delete.backup.on') : l.translate('app.delete.backup.off')}
				</span>
			</span>
			<Switch id="delete-backup" bind:checked={backupFirst} />
		</div>
	{/snippet}
</ConfirmDialog>
