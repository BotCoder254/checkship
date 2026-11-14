<!--
	App shell (DESIGN.md §5). The title bar and sidebar share `canvas` and read as one frame; the
	workspace is a `surface` panel flush right and bottom with one 6px corner at the top left. No
	lines anywhere: the change of tone is the boundary.

	- 1024px and up: sidebar 240px, collapsible to 56px from the title bar (remembered).
	- 768 to 1023px: sidebar 56px (icons and tooltips); the title bar button opens the full menu.
	- Under 768px: no sidebar; the menu button opens it as a 300px sheet.
-->
<script lang="ts">
	import Cancel01Icon from '@hugeicons/core-free-icons/Cancel01Icon';
	import Menu01Icon from '@hugeicons/core-free-icons/Menu01Icon';
	import SidebarLeftIcon from '@hugeicons/core-free-icons/SidebarLeftIcon';
	import type { Snippet } from 'svelte';
	import { MediaQuery } from 'svelte/reactivity';
	import { navigating } from '$app/state';
	import Logo from '$assets/logo.svelte';
	import Icon from '#lib/components/icon.svelte';
	import AccountMenu from '#lib/components/shell/account-menu.svelte';
	import AppNav from '#lib/components/shell/app-nav.svelte';
	import * as Sheet from '#lib/components/ui/sheet/index.js';
	import * as Tooltip from '#lib/components/ui/tooltip/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import { CONTENT_WIDTHS, preferences } from '#lib/preferences.svelte.ts';
	import type { Profile } from '#lib/resources/users.ts';
	import { cn } from '#lib/utils.js';

	type Props = { user: Profile; version: string; children: Snippet };

	let { user, version, children }: Props = $props();

	const wide = new MediaQuery('min-width: 1024px');
	let menuOpen = $state(false);

	const collapsed = $derived(!wide.current || preferences.sidebarCollapsed);

	$effect(() => {
		document.documentElement.style.setProperty(
			'--content-max',
			CONTENT_WIDTHS[preferences.contentWidth]
		);
	});

	$effect(() => {
		// The sheet only exists below 1024px: close it if the window grows past that.
		if (wide.current) menuOpen = false;
	});
</script>

<Tooltip.Provider delayDuration={300}>
	{#if navigating.type}
		<div
			class="fixed inset-x-0 top-0 z-50 h-0.5 animate-pulse bg-primary"
			role="progressbar"
			aria-label="Loading"
		></div>
	{/if}

	<div
		class={cn(
			'grid h-svh grid-rows-[44px_minmax(0,1fr)] bg-canvas font-sans text-ink md:grid-cols-[56px_minmax(0,1fr)]',
			preferences.sidebarCollapsed
				? 'lg:grid-cols-[56px_minmax(0,1fr)]'
				: 'lg:grid-cols-[240px_minmax(0,1fr)]'
		)}
	>
		<header
			class="col-span-full flex items-center justify-between gap-3 pr-2 pl-1 md:pr-3 md:pl-2.5"
		>
			<div class="flex items-center gap-2 md:gap-3">
				<button
					type="button"
					class="hidden size-9 items-center justify-center rounded-md text-muted-foreground outline-none hover:bg-hover hover:text-ink focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset lg:flex"
					aria-label={l.translate(preferences.sidebarCollapsed ? 'nav.expand' : 'nav.collapse')}
					aria-expanded={!preferences.sidebarCollapsed}
					onclick={() => preferences.toggleSidebar()}
				>
					<Icon icon={SidebarLeftIcon} class="size-5" />
				</button>
				<button
					type="button"
					class="flex size-11 items-center justify-center rounded-md text-ink outline-none hover:bg-hover focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset md:size-9 md:text-muted-foreground lg:hidden"
					aria-label={l.translate('nav.open_menu')}
					aria-expanded={menuOpen}
					onclick={() => (menuOpen = true)}
				>
					<Icon icon={Menu01Icon} class="size-5.5 md:size-5" />
				</button>
				<a
					href={routes.apps}
					class="rounded-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
					aria-label="checkship"
				>
					<Logo size={22} />
				</a>
			</div>
			<AccountMenu {user} />
		</header>

		<aside class="hidden min-h-0 md:block">
			<AppNav {collapsed} {version} />
		</aside>

		<main
			class="min-h-0 min-w-0 overflow-y-auto rounded-t-md bg-surface md:rounded-tl-md md:rounded-tr-none"
		>
			<div class="mx-auto w-full max-w-(--content-max) px-5 pb-12 min-[1100px]:px-10 md:px-6">
				{@render children()}
			</div>
		</main>
	</div>

	<Sheet.Root bind:open={menuOpen}>
		<Sheet.Content
			side="left"
			showCloseButton={false}
			class="w-75 max-w-[85vw] gap-0 bg-canvas p-0 shadow-[12px_0_32px_rgba(0,0,0,0.45)] sm:max-w-75"
		>
			<div class="flex h-16 shrink-0 items-center justify-between pr-1.5 pl-5">
				<Sheet.Title class="sr-only">{l.translate('nav.main')}</Sheet.Title>
				<Logo size={28} />
				<Sheet.Close
					class="flex size-11 items-center justify-center rounded-md text-muted-foreground outline-none hover:bg-hover hover:text-ink focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset"
					aria-label={l.translate('nav.close_menu')}
				>
					<Icon icon={Cancel01Icon} class="size-5" />
				</Sheet.Close>
			</div>
			<div class="min-h-0 flex-1">
				<AppNav {version} onnavigate={() => (menuOpen = false)} />
			</div>
		</Sheet.Content>
	</Sheet.Root>
</Tooltip.Provider>
