<!--
	checkship lockup (DESIGN.md §10): the check-of-containers mark followed by the wordmark.
	Source of truth: assets/brand/logo-mark.svg and assets/brand/logo.svg at the repository root.

	Rules: stroke only, brand blue (--brand, light and dark), wordmark in ink, Inter 600, -0.02em
	tracking, gap of 0.4x the mark size.
-->
<script lang="ts">
	import { cn } from '#lib/utils.js';

	type Props = {
		/** Height of the mark in pixels (22 title bar, 28 sidebar header, 56 sign-in, 72 about). */
		size?: number;
		/** Render only the mark, without the wordmark. */
		markOnly?: boolean;
		class?: string;
	};

	let { size = 22, markOnly = false, class: className }: Props = $props();
</script>

<span
	class={cn('inline-flex items-center', className)}
	style:gap="{size * 0.4}px"
	role={markOnly ? 'img' : undefined}
	aria-label={markOnly ? 'checkship' : undefined}
>
	<svg
		class="block flex-none stroke-brand"
		viewBox="0 0 24 24"
		width={size}
		height={size}
		fill="none"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		aria-hidden="true"
	>
		<!-- Three containers (a pod) on the three points of a check -->
		<path d="M2.5 7.5h4.5v4.5h-4.5z" />
		<path d="M9 15.5h4.5v4.5H9z" />
		<path d="M15 4.5h6v6h-6z" />
	</svg>
	{#if !markOnly}
		<span
			class="font-sans leading-none font-semibold tracking-[-0.02em] text-ink lowercase"
			style:font-size="{size * 0.75}px">checkship</span
		>
	{/if}
</span>
