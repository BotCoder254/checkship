/**
 * How a deployment reads as a list row (DESIGN.md §5 Lists and tables), shared by the Deployments
 * tab, Recent deployments, Activity and the Failures drawer, so the same deployment says the same
 * thing everywhere: a tile (icon and tone), then one details line, time first.
 */
import CancelCircleIcon from '@hugeicons/core-free-icons/CancelCircleIcon';
import CheckmarkCircle02Icon from '@hugeicons/core-free-icons/CheckmarkCircle02Icon';
import Clock01Icon from '@hugeicons/core-free-icons/Clock01Icon';
import Loading03Icon from '@hugeicons/core-free-icons/Loading03Icon';
import type { IconData } from '#lib/components/icon.svelte';
import type { TimelineTone } from '#lib/components/timeline/timeline-row.svelte';
import l from '#lib/localization/index.ts';
import { DeploymentStatus, type SourceData, type State } from '#lib/resources/deployments.ts';
import { deploymentBadge, deploymentMoment } from '#lib/status.ts';

export function deploymentTile(state: State): {
	icon: IconData;
	tone: TimelineTone;
	spin: boolean;
} {
	switch (state.status) {
		case DeploymentStatus.Succeeded:
			return { icon: CheckmarkCircle02Icon, tone: 'success', spin: false };
		case DeploymentStatus.Failed:
			return { icon: CancelCircleIcon, tone: 'danger', spin: false };
		case DeploymentStatus.Running:
			return { icon: Loading03Icon, tone: 'warning', spin: true };
		default:
			return { icon: Clock01Icon, tone: 'muted', spin: false };
	}
}

/** "14:32", in the viewer's time zone. The day is the group's heading. */
export function clock(value: DateValue): string {
	return new Date(value).toLocaleTimeString(l.locale(), { hour: '2-digit', minute: '2-digit' });
}

type RowDeployment = {
	deployment_number: number;
	environment: string;
	source?: SourceData;
	state: State;
	requested_at: string;
};

/** "took 48s", "1m 2s so far", or nothing while it waits. */
export function deploymentDuration(d: { state: State }, now = new Date()): string | undefined {
	if (d.state.started_at && d.state.finished_at)
		return l.translate('list.took', [l.duration(d.state.started_at, d.state.finished_at)]);
	if (d.state.started_at) return l.translate('list.so_far', [l.duration(d.state.started_at, now)]);
	return undefined;
}

/** What the source was: the branch for Git, else the kind (Compose file, Archive). */
export function deploymentSource(source: SourceData | undefined): string | undefined {
	if (!source) return undefined;
	if (source.discriminator === 'git') return source.data.branch;
	return l.translate(`deployment.source.${source.discriminator}`);
}

/** The short commit, when there is one. */
export function deploymentCommit(source: SourceData | undefined): string | undefined {
	return source?.discriminator === 'git' ? source.data.hash?.slice(0, 7) : undefined;
}

/** Everything a row shows, for its accessible name. */
export function deploymentLabel(d: RowDeployment, prefix?: string): string {
	const parts = [
		l.translate('deployment.row.label', [
			d.deployment_number,
			l.translate(`deploy.environment.${d.environment as 'production' | 'staging'}`),
			deploymentBadge(d.state).label,
			l.datetime(deploymentMoment(d))
		]),
		deploymentSource(d.source),
		deploymentCommit(d.source),
		d.state.error_code ?? undefined
	];
	return [prefix, ...parts].filter(Boolean).join(', ');
}
