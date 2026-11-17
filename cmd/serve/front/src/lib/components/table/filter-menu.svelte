<!--
	A filter above a table (DESIGN.md §5 Tables): a button reading "Label: value" that opens a
	single-choice menu. Every option has an icon; the chosen one gets a tick.
-->
<script lang="ts" module>
	import type { IconData } from '#lib/components/icon.svelte';

	export type FilterOption<V extends string> = {
		value: V;
		label: string;
		icon: IconData;
		/** Tone for the icon, such as `text-success`. */
		tone?: string;
	};
</script>

<script lang="ts" generics="Value extends string">
	import ArrowDown01Icon from '@hugeicons/core-free-icons/ArrowDown01Icon';
	import Icon from '#lib/components/icon.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import { cn } from '#lib/utils.js';

	type Props = {
		label: string;
		icon: IconData;
		value: Value;
		options: FilterOption<Value>[];
		/** Called with the chosen value, for a filter that is not bound. */
		onValueChange?: (value: Value) => void;
		class?: string;
	};

	let {
		label,
		icon,
		value = $bindable(),
		options,
		onValueChange,
		class: className
	}: Props = $props();

	const current = $derived(options.find((o) => o.value === value));
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<Button {...props} variant="outline" class={cn('max-w-full', className)}>
				<Icon icon={current?.icon ?? icon} class={current?.tone} />
				<span class="truncate"
					>{label}: <span class="font-semibold">{current?.label ?? value}</span></span
				>
				<Icon icon={ArrowDown01Icon} class="size-4 text-muted-foreground" />
			</Button>
		{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="start" class="w-60 p-1.5">
		<DropdownMenu.Group>
			<DropdownMenu.GroupHeading
				class="px-2.5 pt-1.5 pb-1 text-xs font-semibold text-muted-foreground"
			>
				{label}
			</DropdownMenu.GroupHeading>
			<DropdownMenu.RadioGroup
				{value}
				onValueChange={(v) => {
					value = v as Value;
					onValueChange?.(v as Value);
				}}
			>
				{#each options as option (option.value)}
					<DropdownMenu.RadioItem value={option.value} closeOnSelect class="h-9 gap-2.5 pl-2.5">
						<Icon icon={option.icon} class={option.tone ?? 'text-muted-foreground'} />
						{option.label}
					</DropdownMenu.RadioItem>
				{/each}
			</DropdownMenu.RadioGroup>
		</DropdownMenu.Group>
	</DropdownMenu.Content>
</DropdownMenu.Root>
