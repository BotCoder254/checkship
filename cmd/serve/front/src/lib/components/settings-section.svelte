<!--
	A form section in two columns (DESIGN.md §11.6): the title and one sentence on the left (260px),
	the fields on the right, so forms use the page width instead of a narrow centred column. One
	column below md.
-->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '#lib/utils.js';

	type Props = {
		title: string;
		description?: string;
		id?: string;
		class?: string;
		children: Snippet;
	};

	let { title, description, id, class: className, children }: Props = $props();
</script>

<section
	{id}
	class={cn('grid scroll-mt-6 gap-4 md:grid-cols-[260px_minmax(0,1fr)] md:gap-10', className)}
>
	<div class="flex flex-col gap-1.5">
		<h2 class="text-base font-semibold">{title}</h2>
		{#if description}
			<p class="text-[13px] text-muted-foreground">{description}</p>
		{/if}
	</div>
	<div class="flex min-w-0 flex-col gap-4">
		{@render children()}
	</div>
</section>
