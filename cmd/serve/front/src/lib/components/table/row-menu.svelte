<!--
	A table row's ⋯ menu (DESIGN.md §5 Tables): a ghost icon button in the Actions column, the menu
	opening under it aligned to the right edge. Put each group of items in a RowMenuGroup: groups are
	split by a 6px gap, never a line, and the destructive item comes last.

		<RowMenu label={l.translate('table.menu', [app.name])}>
			<RowMenuGroup>
				<RowMenuItem icon={ViewIcon} href={routes.app(app.id)}>Open</RowMenuItem>
			</RowMenuGroup>
			<RowMenuGroup>
				<RowMenuItem icon={Delete02Icon} variant="destructive" onSelect={…}>Delete</RowMenuItem>
			</RowMenuGroup>
		</RowMenu>
-->
<script lang="ts">
	import MoreHorizontalIcon from '@hugeicons/core-free-icons/MoreHorizontalIcon';
	import type { Snippet } from 'svelte';
	import Icon from '#lib/components/icon.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';

	type Props = {
		/** The menu's accessible name, such as "Actions for blog". */
		label: string;
		children: Snippet;
	};

	let { label, children }: Props = $props();
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<Button
				{...props}
				variant="ghost"
				size="icon-sm"
				class="text-muted-foreground hover:bg-overlay hover:text-ink aria-expanded:bg-overlay aria-expanded:text-ink"
				aria-label={label}
				title={label}
			>
				<Icon icon={MoreHorizontalIcon} />
			</Button>
		{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="end" class="flex w-56 flex-col gap-1.5 p-1.5">
		{@render children()}
	</DropdownMenu.Content>
</DropdownMenu.Root>
