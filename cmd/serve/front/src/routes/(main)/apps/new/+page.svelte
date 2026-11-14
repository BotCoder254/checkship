<script lang="ts">
	import { goto } from '$app/navigation';
	import Breadcrumb from '$components/breadcrumb.svelte';
	import Button from '$components/button.svelte';
	import routes from '#lib/path.ts';
	import service, { type CreateApp } from '#lib/resources/apps.ts';
	import AppForm from '../app-form.svelte';
	import l from '#lib/localization/index.ts';

	export let data;

	const submit = (data: CreateApp) => service.create(data).then((a) => goto(routes.app(a.id)));
</script>

<AppForm handler={submit} targets={data.targets} let:submitting>
	<Breadcrumb
		segments={[
			{ path: routes.apps, title: l.translate('breadcrumb.applications') },
			l.translate('breadcrumb.application.new')
		]}
	>
		<Button type="submit" loading={submitting} text="create" />
	</Breadcrumb>
</AppForm>
