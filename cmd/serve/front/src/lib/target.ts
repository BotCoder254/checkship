import l from '#lib/localization/index.ts';
import type { App } from '#lib/resources/apps.ts';
import type { Environment } from '#lib/resources/deployments.ts';
import type { Target } from '#lib/resources/targets.ts';

/** Defaults applied by the server (internal/deployment/infra/provider/docker/data.go). */
export const DOCKER_DEFAULT_USER = 'docker';
export const DOCKER_DEFAULT_PORT = 22;

export const isLocal = (target: Pick<Target, 'provider'>) => !target.provider.data.host;

/** "This machine · Docker engine" or "user@host:port", with the server's defaults filled in. */
export function connectionLabel(target: Pick<Target, 'provider'>): string {
	const { host, user, port } = target.provider.data;
	if (!host) return l.translate('targets.kind.local');
	return `${user || DOCKER_DEFAULT_USER}@${host}:${port || DOCKER_DEFAULT_PORT}`;
}

export type TargetUse = { app: App; environment: Environment };

/** The application environments that deploy to a target (what blocks its deletion). */
export function usesOf(apps: App[], targetId: string): TargetUse[] {
	return apps.flatMap((app) =>
		(['production', 'staging'] as const)
			.filter((environment) => app[`${environment}_target`]?.id === targetId)
			.map((environment) => ({ app, environment }))
	);
}
