<!--
	Account avatar and menu, top right of the title bar (DESIGN.md §5). A round accent avatar,
	36px (40px on phones). The menu floats on its own tone with a soft shadow: no lines inside,
	groups are spacing.
-->
<script lang="ts">
	import Key01Icon from '@hugeicons/core-free-icons/Key01Icon';
	import Logout03Icon from '@hugeicons/core-free-icons/Logout03Icon';
	import UserIcon from '@hugeicons/core-free-icons/UserIcon';
	import { goto } from '$app/navigation';
	import Icon from '#lib/components/icon.svelte';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import auth from '#lib/auth.ts';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import type { Profile } from '#lib/resources/users.ts';

	let { user }: { user: Profile } = $props();

	const initial = $derived((user.email.trim()[0] ?? '?').toUpperCase());
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<button
				{...props}
				type="button"
				aria-label={l.translate('account.menu', [user.email])}
				class="flex size-10 items-center justify-center rounded-full bg-primary text-base font-semibold text-primary-foreground outline-none hover:bg-accent-hover focus-visible:ring-2 focus-visible:ring-primary-foreground focus-visible:ring-inset md:size-9"
			>
				{initial}
			</button>
		{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="end" sideOffset={6} class="w-66 p-1.5">
		<div class="mb-1.5 flex items-center gap-3 rounded-sm bg-sunken px-2.5 pt-2.5 pb-3">
			<span
				aria-hidden="true"
				class="flex size-10 shrink-0 items-center justify-center rounded-full bg-primary text-base font-semibold text-primary-foreground"
			>
				{initial}
			</span>
			<span class="flex min-w-0 flex-col">
				<span class="truncate text-sm font-semibold">{user.email}</span>
				<span class="text-xs text-muted-foreground">{l.translate('account.signed_in')}</span>
			</span>
		</div>
		<DropdownMenu.Item class="h-9 gap-3 px-2.5" onSelect={() => goto(routes.profile)}>
			<Icon icon={UserIcon} class="size-4.5" />
			{l.translate('account.profile')}
		</DropdownMenu.Item>
		<DropdownMenu.Item
			class="h-9 gap-3 px-2.5"
			onSelect={() => goto(`${routes.profile}#integration`)}
		>
			<Icon icon={Key01Icon} class="size-4.5" />
			{l.translate('account.api_key')}
		</DropdownMenu.Item>
		<DropdownMenu.Item class="mt-1.5 h-9 gap-3 px-2.5" onSelect={() => auth.logout()}>
			<Icon icon={Logout03Icon} class="size-4.5" />
			{l.translate('account.sign_out')}
		</DropdownMenu.Item>
	</DropdownMenu.Content>
</DropdownMenu.Root>
