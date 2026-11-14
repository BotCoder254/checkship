<script lang="ts">
	import Add01Icon from '@hugeicons/core-free-icons/Add01Icon';
	import { goto } from '$app/navigation';
	import Icon from '#lib/components/icon.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service, { type CreateApp } from '#lib/resources/apps.ts';
	import AppForm from '../app-form.svelte';

	let { data } = $props();

	const submit = (app: CreateApp) => service.create(app).then((a) => goto(routes.app(a.id)));
</script>

<AppForm
	title={l.translate('breadcrumb.application.new')}
	crumbs={[{ href: routes.apps, label: l.translate('breadcrumb.applications') }]}
	targets={data.targets}
	handler={submit}
>
	{#snippet actions(submitting)}
		<Button type="submit" loading={submitting} disabled={data.targets.length === 0}>
			<Icon icon={Add01Icon} />
			{l.translate('settings.create')}
		</Button>
	{/snippet}
</AppForm>
