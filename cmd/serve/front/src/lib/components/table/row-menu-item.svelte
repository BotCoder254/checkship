<!--
	One item of a table menu: always an icon, then one or two words. `href` navigates (or opens a new
	tab when `external`); `onSelect` runs an action. A destructive item is red and comes last.
-->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import { goto } from '$app/navigation';
	import type { IconData } from '#lib/components/icon.svelte';
	import Icon from '#lib/components/icon.svelte';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';

	type Props = {
		icon: IconData;
		href?: string;
		external?: boolean;
		onSelect?: () => void;
		variant?: 'default' | 'destructive';
		disabled?: boolean;
		children: Snippet;
	};

	let { icon, href, external, onSelect, variant = 'default', disabled, children }: Props = $props();

	function select() {
		if (onSelect) onSelect();
		else if (href && external) window.open(href, '_blank', 'noopener,noreferrer');
		else if (href) goto(href);
	}
</script>

<DropdownMenu.Item
	{variant}
	{disabled}
	class="h-9 gap-2.5 px-2.5 whitespace-nowrap"
	onSelect={select}
>
	<Icon {icon} class={variant === 'destructive' ? undefined : 'text-muted-foreground'} />
	{@render children()}
</DropdownMenu.Item>
