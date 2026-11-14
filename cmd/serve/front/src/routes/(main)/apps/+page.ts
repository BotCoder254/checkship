import service from '#lib/resources/apps.ts';

export const load = async ({ fetch, depends }) => {
	const apps = await service.fetchAll({ fetch, depends });

	return {
		apps
	};
};
