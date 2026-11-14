<!--
	A registry's mark (DESIGN.md §11.8), in its own colours; GitHub's in the text colour, which its
	brand rules allow. Decorative: the registry's name is always written next to it.
-->
<script lang="ts">
	import { REGISTRY_LOGOS, type RegistryLogoName } from '#lib/registry-logos.ts';
	import { cn } from '#lib/utils.js';

	type Props = { name: RegistryLogoName; class?: string };

	let { name, class: className }: Props = $props();

	const logo = $derived(REGISTRY_LOGOS[name]);
</script>

<svg
	viewBox={logo.viewBox}
	fill={'fill' in logo && logo.fill ? logo.fill : 'currentColor'}
	aria-hidden="true"
	class={cn('size-5 shrink-0', name === 'github' && 'text-ink', className)}
>
	{#each logo.shapes as shape, i (i)}
		{#if shape.kind === 'path'}
			<path d={shape.data} />
		{:else}
			<polygon points={shape.data} fill={shape.fill} />
		{/if}
	{/each}
</svg>
