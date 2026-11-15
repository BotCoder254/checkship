<!--
	The Columns menu above a table (DESIGN.md §5 Tables): one checkbox item per column, each with its
	icon. It stays open while columns are ticked; locked columns (name, main state) are greyed out.
-->
<script lang="ts" generics="Id extends string">
	import LayoutTable01Icon from '@hugeicons/core-free-icons/LayoutTable01Icon';
	import Icon from '#lib/components/icon.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import l from '#lib/localization/index.ts';
	import type { TableColumns } from '#lib/table-columns.svelte.ts';

	let { columns }: { columns: TableColumns<Id> } = $props();
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<Button {...props} variant="outline">
				<Icon icon={LayoutTable01Icon} />
				{l.translate('table.columns')}
			</Button>
		{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="end" class="w-56 p-1.5">
		<DropdownMenu.Group>
			<DropdownMenu.GroupHeading class="px-2.5 pt-1.5 pb-1 text-xs font-semibold text-faint">
				{l.translate('table.columns.show')}
			</DropdownMenu.GroupHeading>
			{#each columns.all as column (column.id)}
				<DropdownMenu.CheckboxItem
					class="h-9 gap-2.5 pl-2.5"
					disabled={column.locked}
					checked={columns.visible(column.id)}
					onCheckedChange={(checked) => columns.setVisible(column.id, checked)}
				>
					<Icon icon={column.icon} class="text-muted-foreground" />
					{column.label}
				</DropdownMenu.CheckboxItem>
			{/each}
		</DropdownMenu.Group>
	</DropdownMenu.Content>
</DropdownMenu.Root>
