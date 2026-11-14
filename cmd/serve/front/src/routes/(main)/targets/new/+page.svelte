<script lang="ts">
	import Add01Icon from '@hugeicons/core-free-icons/Add01Icon';
	import { goto } from '$app/navigation';
	import Icon from '#lib/components/icon.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service, { type CreateTarget } from '#lib/resources/targets.ts';
	import TargetForm from '../target-form.svelte';

	const submit = (data: CreateTarget) => service.create(data).then(() => goto(routes.targets));
</script>

<TargetForm
	title={l.translate('breadcrumb.target.new')}
	crumbs={[{ href: routes.targets, label: l.translate('breadcrumb.targets') }]}
	handler={submit}
>
	{#snippet actions(submitting)}
		<Button type="submit" loading={submitting}>
			<Icon icon={Add01Icon} />
			{l.translate('target.create')}
		</Button>
	{/snippet}
</TargetForm>
