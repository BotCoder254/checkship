import type { RegistryLogoName } from '#lib/registry-logos.ts';

/**
 * Registry presets (DESIGN.md §11.8): what each one fills in and which token it asks for, from each
 * registry's own docs. checkship only pulls, so read-only scopes are enough.
 */
export type RegistryPreset = {
	id: RegistryLogoName;
	name: string;
	url: string;
	/** Hosts recognised as this registry when reading a saved URL. */
	hosts: string[];
};

export const REGISTRY_PRESETS: RegistryPreset[] = [
	{
		id: 'docker',
		name: 'Docker Hub',
		url: 'https://index.docker.io/v1/',
		hosts: ['index.docker.io', 'docker.io', 'registry-1.docker.io']
	},
	{ id: 'github', name: 'GitHub Container Registry', url: 'https://ghcr.io', hosts: ['ghcr.io'] },
	{
		id: 'gitlab',
		name: 'GitLab Container Registry',
		url: 'https://registry.gitlab.com',
		hosts: ['registry.gitlab.com']
	},
	{ id: 'quay', name: 'Quay', url: 'https://quay.io', hosts: ['quay.io'] }
];

/** The preset a saved registry URL belongs to, if any. */
export function presetForUrl(url: string): RegistryPreset | undefined {
	try {
		const host = new URL(url).hostname;
		return REGISTRY_PRESETS.find((preset) => preset.hosts.includes(host));
	} catch {
		return undefined;
	}
}
