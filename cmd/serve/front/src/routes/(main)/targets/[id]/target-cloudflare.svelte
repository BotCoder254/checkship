<!--
	Target page › Cloudflare (DESIGN.md §11.7 Connect Cloudflare): shown only once connected. The
	account and the token's permissions, then the tunnel (its name and connector state), its routes
	and the DNS records checkship manages, with Replace token (in an application's Domains drawer)
	and Disconnect, which removes the tunnel, its connector and the records checkship created.
-->
<script lang="ts">
	import CloudIcon from '@hugeicons/core-free-icons/CloudIcon';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import Icon from '#lib/components/icon.svelte';
	import SectionHeading from '#lib/components/section-heading.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { CLOUDFLARE_PERMISSIONS } from '#lib/cloudflare.ts';
	import l from '#lib/localization/index.ts';
	import service, { type CloudflareStatus } from '#lib/resources/targets.ts';
	import { cn } from '#lib/utils.js';

	type Props = { targetId: string; targetName: string };

	let { targetId, targetName }: Props = $props();

	let status = $state<CloudflareStatus>();
	let disconnecting = $state(false);

	$effect(() => {
		void load(targetId);
	});

	async function load(id: string) {
		try {
			status = await service.cloudflare(id);
		} catch {
			status = undefined;
		}
	}

	async function disconnect() {
		await service.disconnectCloudflare(targetId);
		status = await service.cloudflare(targetId);
	}

	const tunnelTone = $derived(
		status?.tunnel?.status === 'healthy'
			? 'success'
			: status?.tunnel?.status === 'degraded'
				? 'warning'
				: 'danger'
	);
</script>

{#if status?.connected}
	<section class="flex flex-col gap-3" aria-labelledby="cloudflare-title">
		<SectionHeading id="cloudflare-title">
			<Icon icon={CloudIcon} class="size-4" />
			Cloudflare
			{#snippet aside()}
				<Button variant="destructive" size="sm" onclick={() => (disconnecting = true)}>
					{l.translate('cloudflare.disconnect')}
				</Button>
			{/snippet}
		</SectionHeading>
		<div class="flex min-w-0 flex-col gap-3 track p-4 sm:p-5">
			<dl class="grid grid-cols-[120px_minmax(0,1fr)] gap-x-3 gap-y-2 text-label/5">
				<dt class="text-muted-foreground">{l.translate('cloudflare.account')}</dt>
				<dd class="flex flex-wrap items-center gap-2">
					<span
						class="inline-flex min-h-6 items-center gap-1.5 rounded-sm bg-success-soft px-2 py-0.5 text-xs font-semibold break-all"
					>
						<span aria-hidden="true" class="size-2 rounded-full bg-success"></span>
						{l.translate('cloudflare.connected', [status.account_name ?? ''])}
					</span>
				</dd>
				<dt class="text-muted-foreground">{l.translate('cloudflare.permissions')}</dt>
				<dd class="flex flex-wrap gap-1.5">
					{#each CLOUDFLARE_PERMISSIONS as permission (permission.key)}
						{#if permission.key !== 'argotunnel' || status.tunnel_allowed}
							<span
								class="inline-flex h-6 items-center rounded-sm bg-raised px-2 text-xs font-semibold"
								>{permission.label}</span
							>
						{/if}
					{/each}
				</dd>
				<dt class="text-muted-foreground">{l.translate('cloudflare.way')}</dt>
				<dd>{l.translate(`cloudflare.way.${status.mode ?? 'dns'}`)}</dd>
				{#if status.tunnel}
					<dt class="text-muted-foreground">{l.translate('cloudflare.tunnel')}</dt>
					<dd class="flex flex-wrap items-center gap-2">
						<span class="font-mono">{status.tunnel.name}</span>
						<span
							class={cn(
								'inline-flex h-6 items-center gap-1.5 rounded-sm px-2 text-xs font-semibold',
								tunnelTone === 'success' && 'bg-success-soft',
								tunnelTone === 'warning' && 'bg-warning-soft',
								tunnelTone === 'danger' && 'bg-danger-soft'
							)}
						>
							<span
								aria-hidden="true"
								class={cn(
									'size-2 rounded-full',
									tunnelTone === 'success' && 'bg-success',
									tunnelTone === 'warning' && 'bg-warning',
									tunnelTone === 'danger' && 'bg-danger'
								)}
							></span>
							{l.translate('cloudflare.tunnel.state', [
								status.tunnel.status,
								status.tunnel.connections
							])}
						</span>
					</dd>
				{/if}
				<dt class="text-muted-foreground">{l.translate('cloudflare.zones.label')}</dt>
				<dd class="font-mono">{status.zones.join(', ') || '—'}</dd>
			</dl>

			<h3 class="mt-2 text-xs font-semibold text-muted-foreground">
				{l.translate('cloudflare.records')}
			</h3>
			{#if status.records.length}
				<div class="overflow-x-auto">
					<table class="w-full text-left text-xs">
						<thead class="text-muted-foreground">
							<tr>
								<th scope="col" class="py-1.5 pr-3 font-semibold"
									>{l.translate('domains.records.domain')}</th
								>
								<th scope="col" class="py-1.5 pr-3 font-semibold"
									>{l.translate('domains.records.type')}</th
								>
								<th scope="col" class="py-1.5 font-semibold"
									>{l.translate('domains.records.value')}</th
								>
							</tr>
						</thead>
						<tbody class="font-mono">
							{#each status.records as record (record.hostname + record.type)}
								<tr class="align-top">
									<td class="py-1.5 pr-3 break-all">{record.hostname}</td>
									<td class="py-1.5 pr-3">{record.type}</td>
									<td class="py-1.5 break-all">{record.content}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{:else}
				<p class="text-xs text-muted-foreground">{l.translate('cloudflare.records.none')}</p>
			{/if}
			<p class="text-xs text-muted-foreground">{l.translate('cloudflare.target.help')}</p>
		</div>
	</section>

	<ConfirmDialog
		bind:open={disconnecting}
		destructive
		title={l.translate('cloudflare.disconnect.title', [targetName])}
		description={l.translate('cloudflare.disconnect.description')}
		confirmLabel={l.translate('cloudflare.disconnect')}
		onconfirm={disconnect}
	/>
{/if}
