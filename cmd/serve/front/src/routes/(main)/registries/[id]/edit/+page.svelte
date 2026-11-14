<!-- Registry settings as a page; the registries list opens the same form in a drawer. -->
<script lang="ts">
	import Delete02Icon from '@hugeicons/core-free-icons/Delete02Icon';
	import Tick02Icon from '@hugeicons/core-free-icons/Tick02Icon';
	import { goto } from '$app/navigation';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import Icon from '#lib/components/icon.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service from '#lib/resources/registries.ts';
	import RegistryForm from '../../registry-form.svelte';

	let { data } = $props();
	let loading = $state(false);
	let deleteOpen = $state(false);
</script>

<div class="flex flex-col">
	<PageHeader
		title={data.registry.name}
		crumbs={[{ href: routes.registries, label: l.translate('breadcrumb.registries') }]}
	>
		{#snippet actions()}
			<Button variant="destructive" onclick={() => (deleteOpen = true)}>
				<Icon icon={Delete02Icon} />
				{l.translate('registry.delete')}
			</Button>
			<Button type="submit" form="registry-page" {loading}>
				<Icon icon={Tick02Icon} />
				{l.translate('save')}
			</Button>
		{/snippet}
	</PageHeader>
	<SettingsSection
		title={l.translate('settings.general')}
		description={l.translate('registries.drawer.description')}
	>
		<RegistryForm
			id="registry-page"
			initialData={data.registry}
			bind:loading
			onsaved={() => goto(routes.registries)}
		/>
	</SettingsSection>
</div>

<ConfirmDialog
	bind:open={deleteOpen}
	destructive
	icon={Delete02Icon}
	title={l.translate('registries.delete.title', [data.registry.name])}
	description={l.translate('registries.delete.description')}
	confirmLabel={l.translate('registry.delete')}
	typeToConfirm={data.registry.name}
	typeLabel={l.translate('registries.delete.type_name', [data.registry.name])}
	onconfirm={() => service.delete(data.registry.id).then(() => goto(routes.registries))}
/>
