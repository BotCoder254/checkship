<script lang="ts">
	/**
	 * checkship lockup: the ring-and-check mark followed by the wordmark.
	 * Source of truth: assets/brand/logo-mark.svg and assets/brand/logo.svg at the repository root.
	 *
	 * Rules: stroke only, brand blue (--co-brand), wordmark in ink, Inter 600, -0.02em tracking,
	 * gap of 0.4x the mark size.
	 */

	/** Height of the mark in pixels (22 title bar, 28 sidebar header, 56 welcome screen, 72 about). */
	export let size = 22;

	/** Render only the mark, without the wordmark. */
	export let markOnly = false;

	let className: string = '';

	/** Additional css classes */
	export { className as class };
</script>

<span
	class="lockup {className}"
	style="--logo-size: {size}px"
	role={markOnly ? 'img' : undefined}
	aria-label={markOnly ? 'checkship' : undefined}
>
	<svg
		class="mark"
		viewBox="0 0 24 24"
		width={size}
		height={size}
		fill="none"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
		aria-hidden="true"
	>
		<path d="M15.37 3.66A9 9 0 1 0 20.56 9.22" />
		<path d="M7.25 11.75 10.75 15.25 20 4" />
	</svg>
	{#if !markOnly}
		<span class="wordmark">checkship</span>
	{/if}
</span>

<style module>
	.lockup {
		display: inline-flex;
		align-items: center;
		gap: calc(var(--logo-size) * 0.4);
	}

	.mark {
		display: block;
		flex: none;
		stroke: var(--co-brand);
	}

	.wordmark {
		color: var(--co-text-5);
		font-family: 'Inter', var(--fo-body);
		font-size: calc(var(--logo-size) * 0.75);
		font-weight: 600;
		letter-spacing: -0.02em;
		line-height: 1;
		text-transform: lowercase;
	}
</style>
