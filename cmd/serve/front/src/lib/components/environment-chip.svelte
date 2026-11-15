<!-- The environment as a small chip: production in the accent tone, staging neutral. -->
<script lang="ts">
	import { cn } from '#lib/utils.js';

	type Props = { environment: string; class?: string };

	let { environment, class: className }: Props = $props();
</script>

<span
	class={cn(
		'inline-block rounded-sm px-1.5 text-[11.5px]/5 font-semibold',
		environment === 'production'
			? 'bg-accent-soft text-primary'
			: 'bg-raised text-muted-foreground',
		className
	)}>{environment}</span
>
