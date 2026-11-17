<!--
	Read-only highlighted code (DESIGN.md §11.6): on canvas, mono 12.5/20, wraps long lines, with an
	optional Copy button in the corner. Shows plain text until the highlighter has loaded.
-->
<script lang="ts">
	import { copyText } from '#lib/clipboard.ts';
	import Copy01Icon from '@hugeicons/core-free-icons/Copy01Icon';
	import { toast } from 'svelte-sonner';
	import Icon from '#lib/components/icon.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { escapeHtml, highlight, type CodeLang } from '#lib/highlight.ts';
	import l from '#lib/localization/index.ts';
	import { cn } from '#lib/utils.js';

	type Props = {
		code: string;
		lang: CodeLang;
		copy?: boolean;
		label?: string;
		class?: string;
		element?: HTMLPreElement;
	};

	let {
		code,
		lang,
		copy = false,
		label,
		class: className,
		element = $bindable()
	}: Props = $props();

	let html = $state<string>();
	// Shiki escapes the code it colours; the plain fallback is escaped here.
	const rendered = $derived(html ?? escapeHtml(code));

	$effect(() => {
		const source = code;
		let current = true;
		highlight(source, lang)
			.then((result) => {
				if (current) html = result;
			})
			.catch(() => (html = undefined));
		return () => (current = false);
	});

	async function copyCode() {
		if (await copyText(code)) toast.success(l.translate('copied'));
		else toast.error(l.translate('copy_blocked'));
	}
</script>

<div class="relative min-w-0">
	<pre
		bind:this={element}
		aria-label={label}
		class={cn(
			'overflow-auto rounded-md bg-canvas px-4 py-3 font-mono text-xs/5 break-all whitespace-pre-wrap text-ink',
			copy && 'pr-24',
			className
		)}><code>{@html rendered}</code></pre>
	{#if copy}
		<Button variant="outline" size="sm" class="absolute top-2 right-2" onclick={copyCode}>
			<Icon icon={Copy01Icon} />
			{l.translate('copy')}
		</Button>
	{/if}
</div>
