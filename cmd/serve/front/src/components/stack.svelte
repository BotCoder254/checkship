<script lang="ts">
	export let direction: 'row' | 'column' = 'row';
	export let align: 'flex-start' | 'center' | 'stretch' | 'flex-end' =
		direction === 'row' ? 'center' : 'stretch';
	export let justify:
		'flex-start' | 'center' | 'stretch' | 'space-around' | 'space-between' | 'flex-end' =
		'flex-start';
	export let gap: number = 4;
	export let wrap: 'no-wrap' | 'wrap' | 'wrap-reverse' = 'no-wrap';
	export let style: Maybe<string> = undefined;

	let className: string = '';

	/** Additional css classes */
	export { className as class };

	/** DOM element to render (defaults to div) */
	export let as: string = 'div';
</script>

<!--
	Flex layout set inline: the CSS-modules preprocessor doesn't rewrite a dynamic class on
	<svelte:element>, so a <style module> rule here never reached the element under Svelte 5.
	Legacy component (DESIGN.md §11.2): replaced by Tailwind flex utilities as pages migrate.
-->
<svelte:element
	this={as}
	{...$$restProps}
	class={className || undefined}
	style="display: flex; flex-direction: {direction}; align-items: {align}; flex-wrap: {wrap ===
	'no-wrap'
		? 'nowrap'
		: wrap}; justify-content: {justify}; gap: var(--sp-{gap}, 0); {style ?? ''}"
>
	<slot />
</svelte:element>
