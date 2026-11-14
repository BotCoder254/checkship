<script lang="ts">
	import Add01Icon from '@hugeicons/core-free-icons/Add01Icon';
	import { goto } from '$app/navigation';
	import Icon from '#lib/components/icon.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service, { type CreateTarget } from '#lib/resources/targets.ts';
	import TargetForm from '../target-form.svelte';

	let { data } = $props();

	const submit = (target: CreateTarget) =>
		service.create(target).then((t) => goto(routes.target(t.id)));
</script>

<TargetForm
	title={l.translate('breadcrumb.target.new')}
	crumbs={[{ href: routes.targets, label: l.translate('breadcrumb.targets') }]}
	localTarget={data.localTarget}
	handler={submit}
>
	{#snippet actions(submitting)}
		<Button type="submit" loading={submitting}>
			<Icon icon={Add01Icon} />
			{l.translate('target.create')}
		</Button>
	{/snippet}
</TargetForm>
