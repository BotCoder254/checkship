<!-- New registry as a page; the registries list opens the same form in a drawer. -->
<script lang="ts">
	import Add01Icon from '@hugeicons/core-free-icons/Add01Icon';
	import { goto } from '$app/navigation';
	import Icon from '#lib/components/icon.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import RegistryForm from '../registry-form.svelte';

	let loading = $state(false);
</script>

<div class="flex flex-col">
	<PageHeader
		title={l.translate('breadcrumb.registry.new')}
		crumbs={[{ href: routes.registries, label: l.translate('breadcrumb.registries') }]}
	>
		{#snippet actions()}
			<Button type="submit" form="registry-page" {loading}>
				<Icon icon={Add01Icon} />
				{l.translate('registries.create')}
			</Button>
		{/snippet}
	</PageHeader>
	<SettingsSection
		title={l.translate('settings.general')}
		description={l.translate('registries.drawer.description')}
	>
		<RegistryForm id="registry-page" bind:loading onsaved={() => goto(routes.registries)} />
	</SettingsSection>
</div>
