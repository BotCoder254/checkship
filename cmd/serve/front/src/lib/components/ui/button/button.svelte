<script lang="ts" module>
	import { type VariantProps, tv } from 'tailwind-variants';
	import { cn, type WithElementRef } from '#lib/utils.js';
	import type { HTMLAnchorAttributes, HTMLButtonAttributes } from 'svelte/elements';

	// checkship edits (DESIGN.md §6, §11.5): flat fills, accent-hover on the primary button, and an
	// inset 2px focus ring whose colour is set per variant (no outer ring, no offset).
	export const buttonVariants = tv({
		base: "rounded-md border border-transparent bg-clip-padding text-sm font-medium focus-visible:ring-2 focus-visible:ring-inset aria-invalid:border-destructive active:not-aria-[haspopup]:translate-y-px [&_svg:not([class*='size-'])]:size-4.5 has-[>svg:first-child]:pl-3 group/button inline-flex shrink-0 items-center justify-center whitespace-nowrap transition-all outline-none select-none disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0",
		variants: {
			variant: {
				default:
					'bg-primary font-semibold text-primary-foreground hover:bg-accent-hover focus-visible:ring-primary-foreground',
				// No outlined buttons (DESIGN.md §1): the secondary action is a flat raised fill.
				outline:
					'bg-secondary text-secondary-foreground hover:bg-accent aria-expanded:bg-accent focus-visible:ring-ring',
				secondary:
					'bg-secondary text-secondary-foreground hover:bg-[color-mix(in_oklch,var(--secondary),var(--foreground)_5%)] aria-expanded:bg-secondary aria-expanded:text-secondary-foreground focus-visible:ring-ring',
				ghost:
					'hover:bg-muted hover:text-foreground aria-expanded:bg-muted aria-expanded:text-foreground focus-visible:ring-ring',
				destructive:
					'bg-danger-soft font-semibold text-danger hover:bg-danger-soft/80 focus-visible:ring-danger',
				link: 'text-primary underline-offset-4 hover:underline focus-visible:ring-ring'
			},
			size: {
				default:
					'h-9 gap-2 px-3.5 in-data-[slot=button-group]:rounded-md has-data-[icon=inline-end]:pr-2 has-data-[icon=inline-start]:pl-2',
				xs: "h-6 gap-1 rounded-[min(var(--radius-md),8px)] px-2 text-xs in-data-[slot=button-group]:rounded-md has-data-[icon=inline-end]:pr-1.5 has-data-[icon=inline-start]:pl-1.5 [&_svg:not([class*='size-'])]:size-3",
				sm: 'h-8 gap-1 rounded-[min(var(--radius-md),10px)] px-2.5 in-data-[slot=button-group]:rounded-md has-data-[icon=inline-end]:pr-1.5 has-data-[icon=inline-start]:pl-1.5',
				lg: 'h-10 gap-1.5 px-2.5 has-data-[icon=inline-end]:pr-2 has-data-[icon=inline-start]:pl-2',
				icon: 'size-9',
				'icon-xs':
					"size-6 rounded-[min(var(--radius-md),8px)] in-data-[slot=button-group]:rounded-md [&_svg:not([class*='size-'])]:size-3",
				'icon-sm':
					'size-8 rounded-[min(var(--radius-md),10px)] in-data-[slot=button-group]:rounded-md',
				'icon-lg': 'size-10'
			}
		},
		defaultVariants: {
			variant: 'default',
			size: 'default'
		}
	});

	export type ButtonVariant = VariantProps<typeof buttonVariants>['variant'];
	export type ButtonSize = VariantProps<typeof buttonVariants>['size'];

	export type ButtonProps = WithElementRef<HTMLButtonAttributes> &
		WithElementRef<HTMLAnchorAttributes> & {
			variant?: ButtonVariant;
			size?: ButtonSize;
			/**
			 * Async work in progress (DESIGN.md §11.5): a spinner is mounted next to the label, which
			 * stays, so the width never changes. The button keeps focus (aria-busy, never `disabled`)
			 * and ignores clicks, which also blocks a second form submit.
			 */
			loading?: boolean;
		};
</script>

<script lang="ts">
	import { Spinner } from '#lib/components/ui/spinner/index.js';

	let {
		class: className,
		variant = 'default',
		size = 'default',
		ref = $bindable(null),
		href = undefined,
		type = 'button',
		disabled,
		loading = false,
		onclick,
		children,
		...restProps
	}: ButtonProps = $props();

	function guard(
		event: MouseEvent & { currentTarget: EventTarget & (HTMLButtonElement | HTMLAnchorElement) }
	) {
		if (loading) {
			event.preventDefault();
			return;
		}

		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		(onclick as any)?.(event);
	}
</script>

{#snippet content()}
	{#if loading}
		<Spinner role="presentation" aria-hidden="true" />
	{/if}
	{@render children?.()}
{/snippet}

{#if href}
	<a
		bind:this={ref}
		data-slot="button"
		class={cn(buttonVariants({ variant, size }), className)}
		href={disabled ? undefined : href}
		aria-disabled={disabled || undefined}
		aria-busy={loading || undefined}
		role={disabled ? 'link' : undefined}
		tabindex={disabled ? -1 : undefined}
		onclick={guard}
		{...restProps}
	>
		{@render content()}
	</a>
{:else}
	<button
		bind:this={ref}
		data-slot="button"
		class={cn(buttonVariants({ variant, size }), className)}
		{type}
		{disabled}
		aria-busy={loading || undefined}
		onclick={guard}
		{...restProps}
	>
		{@render content()}
	</button>
{/if}
