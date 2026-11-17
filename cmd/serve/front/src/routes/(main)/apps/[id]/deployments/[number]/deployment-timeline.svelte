<!--
	What happened to a deployment, in order (DESIGN.md §11.7 Deployment): queued, started, then
	finished (or failed, or still running), each with its time, plus how long it waited and how long
	it ran. Drawn by the shared Steps.
-->
<script lang="ts">
	import Steps, { type Step } from '#lib/components/steps.svelte';
	import l from '#lib/localization/index.ts';
	import { DeploymentStatus, type State } from '#lib/resources/deployments.ts';

	type Props = { requestedAt: string; state: State };

	let { requestedAt, state }: Props = $props();

	const last = $derived(
		state.finished_at
			? state.status === DeploymentStatus.Failed
				? 'failed'
				: 'succeeded'
			: state.started_at
				? 'running'
				: 'waiting'
	);

	const steps = $derived<Step[]>([
		{
			key: 'queued',
			label: l.translate('deployment.step.queued'),
			time: l.time(requestedAt),
			state: 'done'
		},
		{
			key: 'started',
			label: l.translate('deployment.step.started'),
			time: state.started_at ? l.time(state.started_at) : undefined,
			note: state.started_at
				? l.translate('deployment.step.waited', [l.duration(requestedAt, state.started_at)])
				: undefined,
			state: state.started_at ? 'done' : 'current'
		},
		{
			key: 'end',
			label: l.translate(`deployment.step.${last}`),
			time: state.finished_at ? l.time(state.finished_at) : undefined,
			note:
				state.started_at && state.finished_at
					? l.translate('deployment.step.ran', [l.duration(state.started_at, state.finished_at)])
					: undefined,
			state:
				last === 'failed'
					? 'failed'
					: state.finished_at
						? 'done'
						: state.started_at
							? 'current'
							: 'todo'
		}
	]);
</script>

<Steps {steps} label={l.translate('deployment.timeline')} />
