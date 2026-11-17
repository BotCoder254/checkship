<!--
	Icon-only button (DESIGN.md §8): its label is both the tooltip and the accessible name, so an
	icon button can never ship without either. Ghost: no fill until hovered, focus drawn inside.

	- `tone`: "frame" for the title bar, sidebar and mobile menu, which sit on canvas (their hover
	  fill must differ from canvas); "surface" for the workspace, drawers and dialogs.
	- `touch`: 44px on phones, 36px from md up (title bar, drawer header).

	For a trigger owned by another Bits UI part (a menu, a sheet close), use `iconButtonClass` on that
	part's own element and wrap it in a Tooltip, as `language-menu.svelte` does.
-->
<script lang="ts" module>
	import { cn } from '#lib/utils.js';

	export type IconButtonTone = 'frame' | 'surface';

	export function iconButtonClass(
		{ tone = 'surface', touch = false }: { tone?: IconButtonTone; touch?: boolean } = {},
		className?: string
	) {
		return cn(
			'flex shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors outline-none hover:text-ink focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset data-[state=open]:text-ink',
			tone === 'frame'
				? 'hover:bg-hover-frame data-[state=open]:bg-hover-frame'
				: 'hover:bg-hover data-[state=open]:bg-hover',
			touch ? 'size-11 md:size-9' : 'size-9',
			className
		);
	}
</script>

<script lang="ts">
	import type { HTMLButtonAttributes } from 'svelte/elements';
	import Icon, { type IconData } from '#lib/components/icon.svelte';
	import * as Tooltip from '#lib/components/ui/tooltip/index.js';

	type Props = Omit<HTMLButtonAttributes, 'children'> & {
		label: string;
		icon: IconData;
		tone?: IconButtonTone;
		touch?: boolean;
		side?: 'top' | 'right' | 'bottom' | 'left';
		iconClass?: string;
	};

	let {
		label,
		icon,
		tone = 'surface',
		touch = false,
		side = 'bottom',
		iconClass = 'size-5',
		class: className,
		...rest
	}: Props = $props();

	/** The tooltip trigger's props, then the caller's; an event both handle runs both handlers. */
	function merge(trigger: Record<string, unknown>, own: Record<string, unknown>) {
		const merged = { ...trigger, ...own };
		for (const key of Object.keys(own)) {
			const a = trigger[key];
			const b = own[key];
			if (key.startsWith('on') && typeof a === 'function' && typeof b === 'function') {
				merged[key] = (event: Event) => {
					a(event);
					b(event);
				};
			}
		}
		return merged;
	}
</script>

<Tooltip.Root>
	<Tooltip.Trigger>
		{#snippet child({ props })}
			<button
				{...merge(props, rest)}
				type="button"
				aria-label={label}
				class={iconButtonClass({ tone, touch }, className as string | undefined)}
			>
				<Icon {icon} class={iconClass} />
			</button>
		{/snippet}
	</Tooltip.Trigger>
	<Tooltip.Content {side}>{label}</Tooltip.Content>
</Tooltip.Root>
