<!--
	A setting row (DESIGN.md §5 Setting rows): no fill and no line (§1 Cardless). The label and one
	line of help on the left (it flexes), the control on the right (it never shrinks), at least 54px
	tall, so down a form the controls form one column on the right edge. The label points at the
	control by `for`, so clicking the words flips a switch.
-->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Label } from '#lib/components/ui/label/index.js';
	import { cn } from '#lib/utils.js';

	type Props = {
		/** The control's id. */
		for: string;
		label: string;
		description?: string;
		class?: string;
		/** The switch, select or segmented control. */
		children: Snippet;
	};

	let { for: controlId, label, description, class: className, children }: Props = $props();
</script>

<div class={cn('flex min-h-[54px] items-center gap-4 py-2', className)}>
	<div class="flex min-w-0 flex-1 flex-col gap-0.5">
		<Label for={controlId}>{label}</Label>
		{#if description}
			<p class="text-label text-muted-foreground">{description}</p>
		{/if}
	</div>
	<div class="flex shrink-0 items-center">{@render children()}</div>
</div>
