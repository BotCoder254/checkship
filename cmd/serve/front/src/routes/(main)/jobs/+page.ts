import service from '#lib/resources/jobs.ts';

export const load = async ({ fetch, depends }) => {
	const jobs = await service.fetchAll({ page: 1 }, { fetch, depends });

	return {
		jobs
	};
};
