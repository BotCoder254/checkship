<!--
	Empty state (DESIGN.md §11.5 Empty states): a drawing, a title, one sentence on what will appear
	here or what to do, then at most one primary action and an optional documentation link. Written
	straight on the page, centred, with no box, fill or frame (§1 Cardless).

	Sizes follow the space: `page` (280px drawing) when the empty state is the page's main content,
	`section` (200px) for a tab or a filtered list, `compact` (140px) in a dashboard section, a
	drawer or a page part. The drawing is decorative; the words carry the meaning.
-->
<script lang="ts">
	import ArrowUpRight01Icon from '@hugeicons/core-free-icons/ArrowUpRight01Icon';
	import type { Snippet } from 'svelte';
	import Icon from '#lib/components/icon.svelte';
	import Illustration, { type IllustrationName } from '#lib/components/illustration.svelte';
	import { cn } from '#lib/utils.js';

	type Props = {
		illustration: IllustrationName;
		title: string;
		description?: string;
		size?: 'page' | 'section' | 'compact';
		/** Heading level of the title (2 on a page, 3 inside a section). */
		level?: 2 | 3;
		/** A documentation page, shown as a quiet link after the action. */
		link?: { href: string; label: string };
		role?: 'alert' | 'status';
		class?: string;
		action?: Snippet;
	};

	let {
		illustration,
		title,
		description,
		size = 'page',
		level = 2,
		link,
		role,
		class: className,
		action
	}: Props = $props();

	const art = { page: 'w-[280px]', section: 'w-[200px]', compact: 'w-[140px]' } as const;
</script>

<div
	{role}
	class={cn(
		'flex flex-col items-center text-center',
		size === 'page' && 'py-12',
		size === 'section' && 'py-8',
		size === 'compact' && 'py-4',
		className
	)}
>
	<Illustration name={illustration} class={cn('max-w-full', art[size])} />
	<svelte:element
		this={`h${level}`}
		class={cn(
			'font-semibold text-ink',
			size === 'compact' ? 'mt-3 text-sm' : 'mt-5 text-lg/[26px]'
		)}
	>
		{title}
	</svelte:element>
	{#if description}
		<p
			class={cn(
				'mt-1.5 max-w-[46ch] text-muted-foreground',
				size === 'compact' ? 'text-label' : 'text-sm'
			)}
		>
			{description}
		</p>
	{/if}
	{#if action || link}
		<div
			class={cn(
				'flex flex-wrap items-center justify-center gap-x-4 gap-y-2',
				size === 'compact' ? 'mt-3' : 'mt-5'
			)}
		>
			{@render action?.()}
			{#if link}
				<a
					href={link.href}
					target="_blank"
					rel="noopener noreferrer"
					class="inline-flex items-center gap-1 rounded-sm text-label font-semibold text-primary outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring"
				>
					{link.label}
					<Icon icon={ArrowUpRight01Icon} class="size-3.5" />
				</a>
			{/if}
		</div>
	{/if}
</div>
