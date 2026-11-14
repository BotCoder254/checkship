<!--
	New deployment as a page (DESIGN.md §11.6): the application page opens the same form in a drawer;
	this route keeps links and bookmarks working. 760px column, Deploy in the header.
-->
<script lang="ts">
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import Rocket01Icon from '@hugeicons/core-free-icons/Rocket01Icon';
	import { goto } from '$app/navigation';
	import Icon from '#lib/components/icon.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import SettingsSection from '#lib/components/settings-section.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import DeployForm from '../deploy-form.svelte';

	let { data } = $props();
	let loading = $state(false);
</script>

<div class="flex flex-col">
	<PageHeader
		title={l.translate('breadcrumb.deployment.new')}
		crumbs={[
			{ href: routes.apps, label: l.translate('breadcrumb.applications') },
			{ href: routes.app(data.app.id), label: data.app.name }
		]}
	>
		{#snippet actions()}
			{#if !data.app.cleanup_requested_at}
				<Button type="submit" form="deploy-page" {loading}>
					<Icon icon={Rocket01Icon} />
					{l.translate('deployment.deploy')}
				</Button>
			{/if}
		{/snippet}
	</PageHeader>

	{#if data.app.cleanup_requested_at}
		<Alert.Root variant="destructive">
			<Icon icon={Alert02Icon} />
			<Alert.Title class="font-semibold">{l.translate('apps.deletion_requested')}</Alert.Title>
		</Alert.Root>
	{:else}
		<SettingsSection
			title={l.translate('deploy.section')}
			description={l.translate('deploy.section.description')}
		>
			<DeployForm
				id="deploy-page"
				app={data.app}
				apiKey={data.user.api_key}
				bind:loading
				onqueued={(d) => goto(routes.deployment(data.app.id, d.deployment_number))}
			/>
		</SettingsSection>
	{/if}
</div>
