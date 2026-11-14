<!--
	Hugeicons glyph (DESIGN.md §8), rendered from the icon's data so it paints with the page: no
	mount step, no flash. Usage:

		import Icon from '#lib/components/icon.svelte';
		import Add01Icon from '@hugeicons/core-free-icons/Add01Icon';
		<Icon icon={Add01Icon} class="size-4" />

	Decorative by default (aria-hidden). Pass aria-label to make it meaningful (role="img").
-->
<script lang="ts" module>
	/** One Hugeicons icon: a list of [tag, attributes] (the package doesn't export this type). */
	export type IconData = readonly (readonly [
		string,
		{ readonly [key: string]: string | number }
	])[];
</script>

<script lang="ts">
	import type { SVGAttributes } from 'svelte/elements';
	import { cn } from '#lib/utils.js';

	type Props = SVGAttributes<SVGSVGElement> & {
		icon: IconData;
		/** Width and height in px when no size class is given. */
		size?: number;
		/** Stroke width; the Hugeicons stroke-rounded set is drawn at 1.5. */
		strokeWidth?: number;
	};

	let {
		icon,
		size = 20,
		strokeWidth = 1.5,
		class: className,
		'aria-label': ariaLabel,
		...rest
	}: Props = $props();

	const kebab = (key: string) => key.replace(/[A-Z]/g, (c) => `-${c.toLowerCase()}`);

	// Icon data uses React-style camelCase attributes and a `key`; SVG wants kebab-case.
	const elements = $derived(
		icon.map(([tag, attrs]: IconData[number]) => {
			const out: Record<string, string | number> = {};
			for (const [name, value] of Object.entries(attrs)) {
				if (name === 'key') continue;
				out[kebab(name)] = name === 'strokeWidth' ? strokeWidth : value;
			}
			return { tag, attrs: out };
		})
	);
</script>

<svg
	xmlns="http://www.w3.org/2000/svg"
	width={size}
	height={size}
	viewBox="0 0 24 24"
	fill="none"
	class={cn('shrink-0', className)}
	aria-hidden={ariaLabel ? undefined : 'true'}
	aria-label={ariaLabel}
	role={ariaLabel ? 'img' : undefined}
	focusable="false"
	{...rest}
>
	{#each elements as element, index (index)}
		<svelte:element this={element.tag} xmlns="http://www.w3.org/2000/svg" {...element.attrs} />
	{/each}
</svg>
