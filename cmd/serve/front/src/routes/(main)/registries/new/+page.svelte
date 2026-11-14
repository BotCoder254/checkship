<script lang="ts">
	import { goto } from '$app/navigation';
	import Breadcrumb from '$components/breadcrumb.svelte';
	import Button from '$components/button.svelte';
	import routes from '#lib/path.ts';
	import RegistryForm from '../registry-form.svelte';
	import l from '#lib/localization/index.ts';
	import service, { type CreateRegistry } from '#lib/resources/registries.ts';

	const submit = (data: CreateRegistry) => service.create(data).then(() => goto(routes.registries));
</script>

<RegistryForm handler={submit} let:submitting>
	<Breadcrumb
		segments={[
			{ path: routes.registries, title: l.translate('breadcrumb.registries') },
			l.translate('breadcrumb.registry.new')
		]}
	>
		<Button type="submit" loading={submitting} text="create" />
	</Breadcrumb>
</RegistryForm>
