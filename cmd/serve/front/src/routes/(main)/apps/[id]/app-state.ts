import type { AppDetail } from '#lib/resources/apps.ts';
import type { DeploymentDetail, Environment } from '#lib/resources/deployments.ts';

/** The branch an environment last deployed from Git, if any. */
export function branchOf(app: AppDetail, environment: Environment): string | undefined {
	const source = app.latest_deployments[environment]?.source;
	return source?.discriminator === 'git' ? source.data.branch : undefined;
}

/** The URL of the first exposed service of a deployment. */
export function liveUrl(deployment?: DeploymentDetail): string | undefined {
	for (const service of deployment?.state.services ?? []) {
		const url = service.entrypoints.find((e) => e.url)?.url;
		if (url) return url;
	}
}
