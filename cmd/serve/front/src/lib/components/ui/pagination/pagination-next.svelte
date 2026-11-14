<script lang="ts">
	import { Pagination as PaginationPrimitive } from 'bits-ui';
	import Icon from '#lib/components/icon.svelte';
	import ArrowRight01Icon from '@hugeicons/core-free-icons/ArrowRight01Icon';
	import { buttonVariants } from '#lib/components/ui/button/index.js';
	import { cn } from '#lib/utils.js';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: PaginationPrimitive.NextButtonProps = $props();
</script>

<PaginationPrimitive.NextButton
	bind:ref
	aria-label="Go to next page"
	class={cn(buttonVariants({ variant: 'ghost', size: 'default' }), 'pr-2!', className)}
	{...restProps}
>
	<span class="hidden sm:block">Next</span>
	<Icon icon={ArrowRight01Icon} data-icon="inline-end" class="cn-rtl-flip" />
</PaginationPrimitive.NextButton>
