/**
 * Expose application routes.
 */
const routes = {
	signin: (redirectTo?: string) =>
		redirectTo ? `/signin?redirectTo=${encodeURIComponent(redirectTo)}` : '/signin',
	profile: '/profile',
	dashboard: '/',
	apps: '/apps',
	createApp: '/apps/new',
	editApp: (id: string) => `/apps/${id}/edit`,
	app: (id: string) => `/apps/${id}`,
	appDeployments: (id: string) => `/apps/${id}/deployments`,
	appLogs: (id: string, environment?: 'production' | 'staging') =>
		environment && environment !== 'production'
			? `/apps/${id}/logs?environment=${environment}`
			: `/apps/${id}/logs`,
	createDeployment: (id: string) => `/apps/${id}/deployments/new`,
	deployment: (id: string, number: number) => `/apps/${id}/deployments/${number}`,
	targets: '/targets',
	createTarget: '/targets/new',
	target: (id: string) => `/targets/${id}`,
	editTarget: (id: string) => `/targets/${id}/edit`,
	jobs: '/jobs',
	registries: '/registries',
	createRegistry: '/registries/new',
	editRegistry: (id: string) => `/registries/${id}/edit`
} as const;

export default routes;
