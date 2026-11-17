<!--
	Drop zone (DESIGN.md §11.6): a button that also takes a dropped file. Its label names both ways
	in (drop, or choose), and it says what it accepts before anything is dropped. While a file is
	dragged over: info tint, inset accent ring, "Release to add the file". Compact is a row (compose
	file, .env), the default is a block (archive).
-->
<script lang="ts">
	import CloudUploadIcon from '@hugeicons/core-free-icons/CloudUploadIcon';
	import type { IconData } from '#lib/components/icon.svelte';
	import Icon from '#lib/components/icon.svelte';
	import l from '#lib/localization/index.ts';
	import { cn } from '#lib/utils.js';

	type Props = {
		/** What to drop, e.g. "a compose.yml" */
		what: string;
		/** What is accepted, shown under it */
		hint: string;
		accept: string;
		onfile: (file: File) => void;
		compact?: boolean;
		icon?: IconData;
		id?: string;
		disabled?: boolean;
		describedby?: string;
	};

	let {
		what,
		hint,
		accept,
		onfile,
		compact = false,
		icon = CloudUploadIcon,
		id,
		disabled = false,
		describedby
	}: Props = $props();

	let input = $state<HTMLInputElement>();
	let over = $state(false);
	// dragenter/dragleave fire for every child: count them to know when the pointer really left.
	let depth = 0;

	function take(files: FileList | null | undefined) {
		const file = files?.[0];
		if (file) onfile(file);
	}

	function dragenter(event: DragEvent) {
		if (disabled || !event.dataTransfer?.types.includes('Files')) return;
		event.preventDefault();
		depth++;
		over = true;
	}

	function dragleave() {
		depth = Math.max(0, depth - 1);
		if (depth === 0) over = false;
	}

	function drop(event: DragEvent) {
		event.preventDefault();
		depth = 0;
		over = false;
		if (!disabled) take(event.dataTransfer?.files);
	}
</script>

<button
	{id}
	type="button"
	{disabled}
	aria-label={l.translate('dropzone.label', [what])}
	aria-describedby={describedby}
	class={cn(
		'flex w-full rounded-md bg-sunken text-left transition-[background-color,box-shadow] outline-none hover:bg-hover focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset disabled:opacity-50',
		compact
			? 'items-center gap-3.5 px-3.5 py-3'
			: 'flex-col items-center gap-2.5 px-6 py-8 text-center',
		over && 'bg-accent-soft ring-2 ring-primary ring-inset hover:bg-accent-soft'
	)}
	onclick={() => input?.click()}
	ondragenter={dragenter}
	ondragover={(event) => {
		if (!disabled) event.preventDefault();
	}}
	ondragleave={dragleave}
	ondrop={drop}
>
	<span
		class={cn(
			'flex shrink-0 items-center justify-center rounded-full',
			compact ? 'size-9' : 'size-11',
			over ? 'bg-primary text-primary-foreground' : 'bg-raised text-primary'
		)}
	>
		<Icon {icon} class={compact ? 'size-4.5' : 'size-5.5'} />
	</span>
	<span class="flex min-w-0 flex-col gap-0.5">
		<span class="text-sm text-ink">
			{#if over}
				<strong class="font-semibold">{l.translate('dropzone.release')}</strong>
			{:else}
				<strong class="font-semibold">{l.translate('dropzone.drop', [what])}</strong>
				{l.translate('dropzone.or')}
				<span class="font-semibold text-primary underline underline-offset-2"
					>{l.translate('dropzone.browse')}</span
				>
			{/if}
		</span>
		<span class="text-label/[18px] text-muted-foreground">{hint}</span>
	</span>
</button>
<input
	bind:this={input}
	type="file"
	{accept}
	class="hidden"
	tabindex="-1"
	aria-hidden="true"
	onchange={(event) => {
		take(event.currentTarget.files);
		event.currentTarget.value = '';
	}}
/>
