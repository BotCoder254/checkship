<!--
	What happened to a deployment, in order (DESIGN.md §11.7 Deployment): queued, started, then
	finished (succeeded or failed), each with its time, plus how long it waited and how long it ran.
	The step in progress spins: Queued while it waits for its turn, Started while it runs. Drawn by
	the shared Steps, centered, since the page leads with it.
-->
<script lang="ts">
	import Steps, { type Step } from '#lib/components/steps.svelte';
	import l from '#lib/localization/index.ts';
	import { DeploymentStatus, type State } from '#lib/resources/deployments.ts';

	type Props = { requestedAt: string; state: State };

	let { requestedAt, state }: Props = $props();

	const failed = $derived(state.status === DeploymentStatus.Failed);

	const steps = $derived<Step[]>([
		{
			key: 'queued',
			label: l.translate('deployment.step.queued'),
			time: l.time(requestedAt),
			state: state.started_at || state.finished_at ? 'done' : 'current'
		},
		{
			key: 'started',
			label: l.translate('deployment.step.started'),
			time: state.started_at ? l.time(state.started_at) : undefined,
			note: state.started_at
				? l.translate('deployment.step.waited', [l.duration(requestedAt, state.started_at)])
				: undefined,
			// A deployment can end without ever starting (cancelled, or failed before it ran).
			state: !state.started_at ? 'todo' : state.finished_at ? 'done' : 'current'
		},
		{
			key: 'end',
			label: l.translate(
				!state.finished_at
					? 'deployment.step.finished'
					: failed
						? 'deployment.step.failed'
						: 'deployment.step.succeeded'
			),
			time: state.finished_at ? l.time(state.finished_at) : undefined,
			note:
				state.started_at && state.finished_at
					? l.translate('deployment.step.ran', [l.duration(state.started_at, state.finished_at)])
					: undefined,
			state: !state.finished_at ? 'todo' : failed ? 'failed' : 'done'
		}
	]);
</script>

<div class="py-2">
	<Steps {steps} label={l.translate('deployment.timeline')} align="center" />
</div>
