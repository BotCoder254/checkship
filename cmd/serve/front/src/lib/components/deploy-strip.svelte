<!--
	The last 20 deploys of one environment (DESIGN.md §11.7 Overview), oldest to newest: a row of
	12px squares, `success`, `danger` or `faint` (running or waiting), the live one outlined. A run of
	failures shows before a list is read. Each square opens that deploy's preview and is named by
	what it shows ("Deployment 21, Production, Failed, …").
-->
<script lang="ts">
	import { deploymentLabel } from '#lib/deployment-row.ts';
	import l from '#lib/localization/index.ts';
	import service, {
		DeploymentStatus,
		type Deployment,
		type Environment
	} from '#lib/resources/deployments.ts';
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
			: service.queryAllByApp(appId, {
					page: 1,
					per_page: 20,
					...(environment ? { environment } : {})
				})
	);
	const store = $derived(query?.data);
	const deploys = $derived([...(deployments ?? $store?.data ?? [])].slice(0, 20).reverse());

	const tone = (status: DeploymentStatus) =>
		status === DeploymentStatus.Succeeded
			? 'bg-success'
			: status === DeploymentStatus.Failed
				? 'bg-danger'
				: 'bg-faint';
</script>

{#if deploys.length > 1}
	<div class="flex flex-col gap-1">
		<ul
			class="flex flex-wrap gap-[3px]"
			aria-label={l.translate('detail.strip.label', [
				name ??
					(environment
						? l.translate(`deploy.environment.${environment}`)
						: l.translate('detail.strip.shown')),
				deploys.length
			])}
		>
			{#each deploys as d (d.deployment_number)}
				<li>
					<button
						type="button"
						class={cn(
							'block size-3 rounded-xs outline-none focus-visible:ring-2 focus-visible:ring-ring',
							tone(d.state.status),
							d.deployment_number === live &&
								'outline-[1.5px] outline-offset-[1.5px] outline-ink outline-solid'
						)}
						aria-label={deploymentLabel(d)}
						title={deploymentLabel(d)}
						onclick={() => onopen(d.deployment_number)}
					></button>
				</li>
			{/each}
		</ul>
		<span class="flex max-w-[330px] justify-between text-xs text-muted-foreground">
			<span>{l.translate('detail.strip.oldest', [deploys.length])}</span>
			<span>{l.translate('detail.strip.latest')}</span>
		</span>
	</div>
{/if}
