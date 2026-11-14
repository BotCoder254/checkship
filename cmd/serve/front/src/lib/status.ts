import type { BadgeState } from '#lib/components/status-badge.svelte';
import l from '#lib/localization/index.ts';
import { DeploymentStatus, type State } from '#lib/resources/deployments.ts';
import { TargetStatus, type TargetState } from '#lib/resources/targets.ts';

/** Maps API states to badges (DESIGN.md §11.4). */

export function deploymentBadge(state: State): { state: BadgeState; label: string } {
	switch (state.status) {
		case DeploymentStatus.Succeeded:
			return { state: 'succeeded', label: l.translate('status.succeeded') };
		case DeploymentStatus.Failed:
			return { state: 'failed', label: l.translate('status.failed') };
		case DeploymentStatus.Running:
			return { state: 'running', label: l.translate('status.running') };
		default:
			return { state: 'pending', label: l.translate('status.pending') };
	}
}

export function targetBadge(state: TargetState): { state: BadgeState; label: string } {
	switch (state.status) {
		case TargetStatus.Ready:
			return { state: 'ready', label: l.translate('status.ready') };
		case TargetStatus.Failed:
			return { state: 'failed', label: l.translate('status.failed') };
		default:
			return { state: 'configuring', label: l.translate('status.configuring') };
	}
}

/** The most recent moment a deployment did something, for "2 min ago". */
export function deploymentMoment(deployment: { state: State; requested_at: string }): string {
	return deployment.state.finished_at ?? deployment.state.started_at ?? deployment.requested_at;
}
