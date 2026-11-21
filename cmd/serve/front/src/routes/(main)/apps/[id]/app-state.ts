import type { AppDetail } from '#lib/resources/apps.ts';
import type { DeploymentDetail, Environment } from '#lib/resources/deployments.ts';

/** The branch an environment last deployed from Git, if any. */
export function branchOf(app: AppDetail, environment: Environment): string | undefined {
	const source = app.latest_deployments[environment]?.source;
	return source?.discriminator === 'git' ? source.data.branch : undefined;
}

/**
 * The URL of the first exposed service of a deployment: its primary custom domain when it has
 * one (seelf#5), else its default address.
 */
export function liveUrl(deployment?: DeploymentDetail): string | undefined {
	return liveUrls(deployment)[0];
}

/** Every http address of the main service: custom domains first, then the default one. */
export function liveUrls(deployment?: DeploymentDetail): string[] {
	for (const service of deployment?.state.services ?? []) {
		if (service.urls?.length) return service.urls;
		const url = service.entrypoints.find((e) => e.url)?.url;
		if (url) return [url];
	}
	return [];
}
