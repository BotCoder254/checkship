<!--
	The last 20 deploys of one environment (DESIGN.md §11.7 Overview), oldest to newest: a row of
	16×40px bars (owner's call 2026-11-20: taller, each one's status in a tooltip on hover and focus,
	refreshed every 2.5s so new deploys slide in while the page is open), `success`, `danger` or `faint` (running or waiting), the live one outlined. A run of
	failures shows before a list is read. Cancelled deploys are `faint`: they are not failures. Each square opens that deploy's preview and is named by
	what it shows ("Deployment 21, Production, Failed, …").
-->
<script lang="ts">
	import { deploymentLabel } from '#lib/deployment-row.ts';
	import l from '#lib/localization/index.ts';
	import service, {
		DeploymentStatus,
		type Deployment,
		type Environment,
		type State
	} from '#lib/resources/deployments.ts';
	import * as Tooltip from '#lib/components/ui/tooltip/index.js';
	import { deploymentBadge, isFailure } from '#lib/status.ts';
	import { clock, deploymentDuration } from '#lib/deployment-row.ts';
	import { cn } from '#lib/utils.js';

	type Props = {
		appId: string;
		/** One environment; none for a list that mixes them (the Deployments tab). */
		environment?: Environment;
		/** Rows already loaded, newest first; otherwise the last 20 of the environment are fetched. */
		deployments?: Deployment[];
		/** What the strip is for when it is not one environment ("blog" on a target's page). */
		name?: string;
		/** The deployment this environment serves, outlined. */
		live?: number;
		onopen: (number: number) => void;
	};

	let { appId, environment, deployments, name, live, onopen }: Props = $props();

	const query = $derived(
		deployments
			? undefined
			: service.queryAllByApp(
					appId,
					{
						page: 1,
						per_page: 20,
						...(environment ? { environment } : {})
					},
					true
				)
	);
	const store = $derived(query?.data);
	const deploys = $derived([...(deployments ?? $store?.data ?? [])].slice(0, 20).reverse());

	// Cancelled deploys are grey like the ones still waiting: they are not failures.
	const tone = (state: State) =>
		state.status === DeploymentStatus.Succeeded
			? 'bg-success'
			: isFailure(state)
				? 'bg-danger'
				: 'bg-faint';
</script>

{#if deploys.length > 1}
	<div class="flex flex-col gap-1">
		<ul
			class="flex flex-wrap items-end gap-[5px]"
			aria-label={l.translate('detail.strip.label', [
				name ??
					(environment
						? l.translate(`deploy.environment.${environment}`)
						: l.translate('detail.strip.shown')),
				deploys.length
			])}
		>
			{#each deploys as d (d.deployment_number)}
				{@const badge = deploymentBadge(d.state)}
				<li>
					<Tooltip.Root>
						<Tooltip.Trigger>
							{#snippet child({ props })}
								<button
									{...props}
									type="button"
									class={cn(
										'block h-10 w-4 rounded-[3px] transition-transform outline-none hover:scale-y-110 focus-visible:ring-2 focus-visible:ring-ring motion-reduce:transition-none',
										tone(d.state),
										d.deployment_number === live &&
											'outline-[1.5px] outline-offset-[1.5px] outline-ink outline-solid'
									)}
									aria-label={deploymentLabel(d)}
									onclick={() => onopen(d.deployment_number)}
								></button>
							{/snippet}
						</Tooltip.Trigger>
						<Tooltip.Content side="bottom">
							<span class="flex flex-col gap-0.5">
								<b class="font-semibold">#{d.deployment_number} · {badge.label}</b>
								<span
									>{clock(d.requested_at)}{#if deploymentDuration(d)}&nbsp;· {deploymentDuration(
											d
										)}{/if}{#if d.deployment_number === live}&nbsp;· {l.translate(
											'detail.strip.live'
										)}{/if}</span
								>
							</span>
						</Tooltip.Content>
					</Tooltip.Root>
				</li>
			{/each}
		</ul>
		<span class="flex max-w-[330px] justify-between text-xs text-muted-foreground">
			<span>{l.translate('detail.strip.oldest', [deploys.length])}</span>
			<span>{l.translate('detail.strip.latest')}</span>
		</span>
	</div>
{/if}
