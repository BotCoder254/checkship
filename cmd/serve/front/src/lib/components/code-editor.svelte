<!--
	Code field highlighted as you type (DESIGN.md §11.6): a transparent textarea over the
	highlighted text, both in the same grid cell with the same font and wrapping, so the field grows
	with its content and the caret lines up without scroll syncing. Looks like any input: sunken
	fill, faint border, focus drawn inside.
-->
<script lang="ts">
	import type { HTMLTextareaAttributes } from 'svelte/elements';
	import { escapeHtml, highlight, type CodeLang } from '#lib/highlight.ts';
	import { cn } from '#lib/utils.js';

	type Props = Omit<HTMLTextareaAttributes, 'value' | 'class'> & {
		value: string;
		lang: CodeLang;
		rows?: number;
		class?: string;
		tone?: 'sunken' | 'canvas';
	};

	let {
		value = $bindable(''),
		lang,
		rows = 6,
		class: className,
		tone = 'sunken',
		...rest
	}: Props = $props();

	let html = $state<string>();
	// Shiki escapes the code it colours; the plain fallback is escaped here. The trailing space gives
	// a last empty line its height, so the field grows as soon as Enter is pressed.
	const rendered = $derived((html ?? escapeHtml(value)) + ' ');

	$effect(() => {
		const source = value;
		let current = true;
		highlight(source, lang)
			.then((result) => {
				if (current) html = result;
			})
			.catch(() => (html = undefined));
		return () => (current = false);
	});

	const shared =
		'col-start-1 row-start-1 m-0 px-3 py-2.5 font-mono text-xs/5 break-all whitespace-pre-wrap';
</script>

<div
	class={cn(
		'grid rounded-md border border-input transition-[color,box-shadow] focus-within:border-ring focus-within:ring-1 focus-within:ring-ring focus-within:ring-inset hover:border-muted-foreground has-aria-invalid:border-destructive',
		tone === 'sunken' ? 'bg-sunken' : 'bg-canvas',
		className
	)}
	style:min-height="{rows * 20 + 22}px"
>
	<pre aria-hidden="true" class={cn(shared, 'pointer-events-none overflow-hidden text-ink')}><code
			>{@html rendered}</code
		></pre>
	<textarea
		bind:value
		spellcheck="false"
		autocapitalize="off"
		autocomplete="off"
		class={cn(
			shared,
			'resize-none overflow-hidden bg-transparent text-transparent caret-ink outline-none selection:bg-accent-soft selection:text-transparent placeholder:text-muted-foreground'
		)}
		{...rest}></textarea>
</div>
