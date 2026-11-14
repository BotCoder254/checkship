<!--
	Legacy rich-text wrapper (DESIGN.md §11.2), replaced by Tailwind as pages migrate. Uses Svelte's
	scoped styles: the CSS-modules preprocessor doesn't rewrite a dynamic class on <svelte:element>.
-->
<script lang="ts">
	let className: string = '';

	/** Additional css classes */
	export { className as class };

	export let as = 'div';
</script>

<svelte:element this={as} class="prose {className}">
	<slot />
</svelte:element>

<style>
	.prose :global(a) {
		color: var(--co-primary-4);
	}

	.prose :global(a:hover),
	.prose :global(a:focus) {
		text-decoration: underline;
		text-underline-offset: 2px;
	}

	.prose :global(strong) {
		color: var(--co-text-5);
		font-weight: 600;
	}

	.prose :global(code) {
		font-family: var(--fo-mono);
	}

	.prose > :global(* + *) {
		margin-block-start: var(--sp-4);
	}
</style>
