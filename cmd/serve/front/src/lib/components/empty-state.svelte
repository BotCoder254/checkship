<!--
	Empty state (DESIGN.md §11.5): an icon, a title, one sentence on what will appear here, and one
	action, the most likely next step.
-->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import Icon, { type IconData } from '#lib/components/icon.svelte';
	import * as Empty from '#lib/components/ui/empty/index.js';

	type Props = { icon: IconData; title: string; description: string; action?: Snippet };

	let { icon, title, description, action }: Props = $props();
</script>

<Empty.Root class="rounded-md bg-sunken py-12">
	<Empty.Header>
		<Empty.Media variant="icon" class="size-12 rounded-md bg-raised text-primary">
			<Icon {icon} class="size-6" />
		</Empty.Media>
		<Empty.Title class="text-base font-semibold">{title}</Empty.Title>
		<Empty.Description class="text-muted-foreground">{description}</Empty.Description>
	</Empty.Header>
	{#if action}
		<Empty.Content>{@render action()}</Empty.Content>
	{/if}
</Empty.Root>
