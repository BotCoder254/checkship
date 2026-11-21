<!--
	Connect Cloudflare (DESIGN.md §11.7), inside the Domains drawer: shown when a domain's DNS is at
	Cloudflare or the target is already connected. Three numbered steps: open Cloudflare's token page
	pre-filled with the permissions, paste the token, choose how visitors reach the target (a tunnel
	or a DNS record). The drawer's footer runs `setup()` through "Set up and redeploy"; the tasks done
	tick off below, then the live address. Problems show above the steps: a refused token, a missing
	permission, domains outside the account, records checkship didn't create (Keep it / Replace).
	The token is sent once to checkship, which only ever sends it to Cloudflare.
-->
<script lang="ts">
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import CheckmarkCircle02Icon from '@hugeicons/core-free-icons/CheckmarkCircle02Icon';
	import Copy01Icon from '@hugeicons/core-free-icons/Copy01Icon';
	import LinkSquare02Icon from '@hugeicons/core-free-icons/LinkSquare02Icon';
	import RefreshIcon from '@hugeicons/core-free-icons/RefreshIcon';
	import Tick02Icon from '@hugeicons/core-free-icons/Tick02Icon';
	import Icon from '#lib/components/icon.svelte';
	import Notice from '#lib/components/notice.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { copyText } from '#lib/clipboard.ts';
	import { BadRequestError } from '#lib/error.ts';
	import {
		CLOUDFLARE_PERMISSIONS,
		cloudflareTokenLink,
		looksLikeCloudflareToken
	} from '#lib/cloudflare.ts';
	import l, { type AppTranslationsString } from '#lib/localization/index.ts';
	import service, {
		type CloudflareMode,
		type CloudflareSetupResult,
		type CloudflareStatus
	} from '#lib/resources/targets.ts';
	import { cn } from '#lib/utils.js';

	type Props = {
		targetId: string;
		targetName: string;
		/** The target answers in https (a public server) rather than http (behind a tunnel). */
		https: boolean;
		/** Whether the drawer should show this section at all: a domain's DNS is at Cloudflare. */
		atCloudflare: boolean;
		/** Ready to set up: connected and a way chosen. The drawer's primary button follows it. */
		ready?: boolean;
		/** Shown or not, so the drawer can place it. */
		visible?: boolean;
	};

	let {
		targetId,
		targetName,
		https,
		atCloudflare,
		ready = $bindable(false),
		visible = $bindable(false)
	}: Props = $props();

	type Problem = { code: string; permission?: string; cloudflare?: string };

	let status = $state<CloudflareStatus>();
	let token = $state('');
	let replacing = $state(false); // Typing another token while connected
	let connecting = $state(false);
	let tokenProblem = $state<Problem>();
	let problem = $state<Problem>();
	let mode = $state<CloudflareMode>();
	let result = $state<CloudflareSetupResult>();
	let kept = $state<string[]>([]); // Conflicts the person chose to keep
	let resolved = $state<string[]>([]); // Conflicts fixed since: by hand or replaced
	let copied = $state<string>();

	async function copy(text: string) {
		if (await copyText(text)) {
			copied = text;
			setTimeout(() => (copied = undefined), 2000);
		}
	}
	let running = $state(false);
	let opened = $state(false); // Open Cloudflare was clicked: step 2 is next

	const connected = $derived(!!status?.connected);
	const tunnelAllowed = $derived(!!status?.tunnel_allowed);
	const chosen = $derived(mode ?? status?.mode ?? (https || !tunnelAllowed ? 'dns' : 'tunnel'));
	const conflicts = $derived((result?.conflicts ?? []).filter((c) => !kept.includes(c.hostname)));
	// The first domain whose record is in place (one added by hand may not be checkship's own).
	const live = $derived(
		result && conflicts.length === 0
			? (result.tasks.find((t) => t.key === 'records')?.detail.split(', ')[0] ??
					result.status.records[0]?.hostname)
			: undefined
	);

	$effect(() => {
		visible = connected || atCloudflare;
	});
	$effect(() => {
		ready = connected && !replacing;
	});

	// Fresh each time the section mounts (the drawer opens).
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

	function problemOf(ex: unknown, field?: string): Problem {
		if (ex instanceof BadRequestError) {
			const raw = (field && ex.isValidationError ? ex.details[field] : ex.details) as
				| {
						code?: string;
						detail?: { permission?: string; cloudflare?: string };
						permission?: string;
						cloudflare?: string;
				  }
				| undefined;
			if (field && ex.isValidationError && raw?.code)
				return {
					code: raw.code,
					permission: raw.detail?.permission,
					cloudflare: raw.detail?.cloudflare
				};
			return { code: ex.message, permission: raw?.permission, cloudflare: raw?.cloudflare };
		}
		return { code: ex instanceof Error ? ex.message : 'unexpected_error' };
	}

	function problemText(p: Problem): string {
		if (p.code === 'cloudflare_permission_missing')
			return (
				l.translate('cloudflare.problem.permission', [p.permission ?? '']) +
				(p.cloudflare ? ` ${l.translate('cloudflare.problem.said', [p.cloudflare])}` : '')
			);
		return l.translate(p.code as AppTranslationsString);
	}

	async function connect(event: SubmitEvent) {
		event.preventDefault();
		tokenProblem = undefined;
		if (!looksLikeCloudflareToken(token)) {
			tokenProblem = { code: 'cloudflare_token_invalid' };
			return;
		}
		connecting = true;
		try {
			status = await service.connectCloudflare(targetId, token.trim());
			token = '';
			replacing = false;
			mode = undefined;
			result = undefined;
		} catch (ex) {
			tokenProblem = problemOf(ex, 'token');
		} finally {
			connecting = false;
		}
	}

	/** Sets everything up on Cloudflare; called by the drawer after the domains are saved. */
	export async function setup(replace: string[] = []) {
		running = true;
		problem = undefined;
		try {
			const before = (result?.conflicts ?? []).map((c) => c.hostname);
			result = await service.setupCloudflare(targetId, { mode: chosen, replace });
			status = result.status;
			const now = result.conflicts.map((c) => c.hostname);
			resolved = [...resolved, ...before.filter((h) => !now.includes(h) && !kept.includes(h))];
			kept = [];
		} catch (ex) {
			problem = problemOf(ex);
			void load(targetId); // The token may have turned out unable to create a tunnel
		} finally {
			running = false;
		}
	}

	function stepTone(done: boolean, current: boolean) {
		if (done) return 'bg-success-soft text-success';
		if (current) return 'bg-primary text-primary-foreground';
		return 'bg-raised text-muted-foreground';
	}
</script>

{#if visible}
	<section class="flex flex-col gap-4" aria-labelledby="cloudflare-title" aria-busy={running}>
		<div class="flex flex-col gap-1">
			<h3 id="cloudflare-title" class="text-sm font-semibold">
				{l.translate('cloudflare.title')}
			</h3>
			<p class="text-xs text-muted-foreground">
				{l.translate('cloudflare.intro', [targetName])}
			</p>
		</div>

		{#if problem}
			<Notice tone="danger" role="alert">
				<strong class="font-semibold">{problemText(problem)}</strong>
				{#if problem.code === 'cloudflare_permission_missing' && problem.permission === 'Cloudflare Tunnel · Edit'}
					<span>{l.translate('cloudflare.problem.fix.tunnel')}</span>
				{:else if problem.code === 'cloudflare_permission_missing' || problem.code === 'cloudflare_token_refused'}
					<span>{l.translate('cloudflare.problem.fix')}</span>
				{/if}
			</Notice>
		{/if}

		{#if result?.outside.length}
			<Notice tone="warning" role="status">
				<strong class="font-semibold">
					{l.translate('cloudflare.outside', [result.outside.join(', ')])}
				</strong>
				<span>{l.translate('cloudflare.outside.help')}</span>
			</Notice>
		{/if}

		{#each conflicts as conflict (conflict.hostname)}
			{@const dnsPage = status?.account_id
				? `https://dash.cloudflare.com/${status.account_id}/${conflict.zone}/dns/records`
				: 'https://dash.cloudflare.com/?to=/:account/:zone/dns/records'}
			<section
				class="flex flex-col gap-3 rounded-md bg-warning-soft px-3.5 py-3"
				aria-labelledby="conflict-{conflict.hostname}"
			>
				<div class="flex flex-col gap-1">
					<strong id="conflict-{conflict.hostname}" class="text-label font-semibold">
						{l.translate('cloudflare.fix.title', [conflict.hostname])}
					</strong>
					<span class="text-xs">
						{conflict.reason === 'dns_edit'
							? l.translate('cloudflare.fix.why.dns_edit')
							: l.translate('cloudflare.fix.why', [conflict.type, conflict.content])}
					</span>
				</div>

				<ol class="flex flex-col gap-3 text-xs">
					<li class="grid grid-cols-[20px_minmax(0,1fr)] gap-x-2.5 gap-y-1">
						<span
							aria-hidden="true"
							class="flex size-5 items-center justify-center rounded-full bg-raised text-[11px] font-semibold"
							>1</span
						>
						<div class="flex flex-col items-start gap-1.5">
							<b class="font-semibold">{l.translate('cloudflare.fix.open')}</b>
							<Button
								size="sm"
								variant="outline"
								href={dnsPage}
								target="_blank"
								rel="noopener noreferrer"
							>
								<Icon icon={LinkSquare02Icon} />
								{l.translate('cloudflare.fix.open.button', [conflict.zone])}
							</Button>
						</div>
					</li>
					{#if conflict.type}
						<li class="grid grid-cols-[20px_minmax(0,1fr)] gap-x-2.5 gap-y-1">
							<span
								aria-hidden="true"
								class="flex size-5 items-center justify-center rounded-full bg-raised text-[11px] font-semibold"
								>2</span
							>
							<div class="flex flex-col gap-1">
								<b class="font-semibold">{l.translate('cloudflare.fix.delete')}</b>
								<span class="font-mono break-all"
									>{conflict.type} · {conflict.want.name} → {conflict.content}</span
								>
								<span class="text-muted-foreground"
									>{l.translate('cloudflare.fix.delete.help')}</span
								>
							</div>
						</li>
					{/if}
					<li class="grid grid-cols-[20px_minmax(0,1fr)] gap-x-2.5 gap-y-1">
						<span
							aria-hidden="true"
							class="flex size-5 items-center justify-center rounded-full bg-raised text-[11px] font-semibold"
							>{conflict.type ? 3 : 2}</span
						>
						<div class="flex min-w-0 flex-col gap-1.5">
							<b class="font-semibold">{l.translate('cloudflare.fix.add')}</b>
							<dl
								class="grid grid-cols-[96px_minmax(0,1fr)] items-center gap-x-3 gap-y-1.5 rounded-md bg-canvas px-3 py-2.5"
							>
								<dt class="text-muted-foreground">{l.translate('cloudflare.fix.type')}</dt>
								<dd class="font-mono font-semibold">{conflict.want.type}</dd>
								<dt class="text-muted-foreground">{l.translate('cloudflare.fix.name')}</dt>
								<dd class="font-mono font-semibold">
									{conflict.want.name}
									{#if conflict.want.name === '@'}
										<span class="font-sans font-normal text-muted-foreground"
											>· {l.translate('cloudflare.fix.name.root', [conflict.zone])}</span
										>
									{/if}
								</dd>
								<dt class="text-muted-foreground">{l.translate('cloudflare.fix.target')}</dt>
								<dd class="flex min-w-0 items-center gap-1.5">
									<span class="min-w-0 font-mono font-semibold break-all"
										>{conflict.want.content}</span
									>
									<Button
										variant="ghost"
										size="icon-xs"
										aria-label={l.translate('cloudflare.fix.copy')}
										onclick={() => copy(conflict.want.content)}
									>
										<Icon icon={copied === conflict.want.content ? Tick02Icon : Copy01Icon} />
									</Button>
								</dd>
								<dt class="text-muted-foreground">{l.translate('cloudflare.fix.proxy')}</dt>
								<dd class="flex items-center gap-1.5 font-semibold">
									<span aria-hidden="true" class="size-2.5 rounded-full bg-[#f38020]"></span>
									{l.translate('cloudflare.fix.proxied')}
								</dd>
								<dt class="text-muted-foreground">TTL</dt>
								<dd class="font-semibold">Auto</dd>
							</dl>
							<span class="text-muted-foreground">{l.translate('cloudflare.fix.add.help')}</span>
						</div>
					</li>
					<li class="grid grid-cols-[20px_minmax(0,1fr)] gap-x-2.5 gap-y-1">
						<span
							aria-hidden="true"
							class="flex size-5 items-center justify-center rounded-full bg-raised text-[11px] font-semibold"
							>{conflict.type ? 4 : 3}</span
						>
						<div class="flex flex-col items-start gap-1.5">
							<b class="font-semibold">{l.translate('cloudflare.fix.check')}</b>
							<span class="text-muted-foreground">{l.translate('cloudflare.fix.check.help')}</span>
							<Button size="sm" loading={running} onclick={() => setup()}>
								<Icon icon={RefreshIcon} />
								{l.translate('cloudflare.fix.check.button')}
							</Button>
						</div>
					</li>
				</ol>

				{#if conflict.reason !== 'dns_edit'}
					<div class="flex flex-col gap-1.5 border-t border-line pt-3 text-xs">
						<span>{l.translate('cloudflare.fix.or')}</span>
						<span class="flex flex-wrap gap-2">
							<Button
								variant="destructive"
								size="sm"
								loading={running}
								onclick={() => setup([conflict.hostname])}
							>
								{l.translate('cloudflare.conflict.replace')}
							</Button>
							<Button
								variant="ghost"
								size="sm"
								onclick={() => (kept = [...kept, conflict.hostname])}
							>
								{l.translate('cloudflare.conflict.keep')}
							</Button>
						</span>
					</div>
				{/if}
			</section>
		{/each}

		{#each resolved as hostname (hostname)}
			<p
				role="status"
				class="flex items-start gap-2 rounded-md bg-success-soft px-3.5 py-3 text-label font-semibold"
			>
				<Icon icon={CheckmarkCircle02Icon} class="mt-0.5 size-5 shrink-0 text-success" />
				<span class="flex flex-col gap-0.5">
					{l.translate('cloudflare.fix.done', [hostname])}
					<span class="text-xs font-normal text-muted-foreground"
						>{l.translate('cloudflare.fix.done.help')}</span
					>
				</span>
			</p>
		{/each}

		<ol class="flex flex-col gap-5">
			<!-- 1 · Create a token -->
			<li class="grid grid-cols-[24px_minmax(0,1fr)] gap-x-3 gap-y-1.5">
				<span
					aria-hidden="true"
					class={cn(
						'flex size-6 items-center justify-center rounded-full text-xs font-semibold',
						stepTone(connected || opened, !connected && !opened)
					)}
				>
					{#if connected || opened}<Icon icon={Tick02Icon} class="size-3.5" />{:else}1{/if}
				</span>
				<div class="flex min-w-0 flex-col gap-2">
					<strong class="text-label font-semibold">{l.translate('cloudflare.step1')}</strong>
					<p class="text-xs text-muted-foreground">{l.translate('cloudflare.step1.help')}</p>
					<ul class="flex flex-wrap gap-1.5" aria-label={l.translate('cloudflare.permissions')}>
						{#each CLOUDFLARE_PERMISSIONS as permission (permission.key)}
							<li
								class="inline-flex h-6 items-center rounded-sm bg-raised px-2 text-xs font-semibold"
							>
								{permission.label}
							</li>
						{/each}
					</ul>
					<div>
						<Button
							variant={connected ? 'outline' : 'default'}
							size="sm"
							href={cloudflareTokenLink(l.translate('cloudflare.token_name', [targetName]))}
							target="_blank"
							rel="noopener noreferrer"
							onclick={() => (opened = true)}
						>
							<Icon icon={LinkSquare02Icon} />
							{l.translate('cloudflare.open')}
						</Button>
					</div>
				</div>
			</li>

			<!-- 2 · Paste the token -->
			<li class="grid grid-cols-[24px_minmax(0,1fr)] gap-x-3 gap-y-1.5">
				<span
					aria-hidden="true"
					class={cn(
						'flex size-6 items-center justify-center rounded-full text-xs font-semibold',
						stepTone(connected && !replacing, (opened && !connected) || replacing)
					)}
				>
					{#if connected && !replacing}<Icon icon={Tick02Icon} class="size-3.5" />{:else}2{/if}
				</span>
				<div class="flex min-w-0 flex-col gap-2">
					<strong class="text-label font-semibold">{l.translate('cloudflare.step2')}</strong>
					{#if connected && !replacing}
						<span class="flex flex-wrap items-center gap-2">
							<span
								class="inline-flex min-h-6 items-center gap-1.5 rounded-sm bg-success-soft px-2 py-0.5 text-xs font-semibold break-all text-ink"
							>
								<Icon icon={CheckmarkCircle02Icon} class="size-3.5 text-success" />
								{l.translate('cloudflare.connected', [status?.account_name ?? ''])}
							</span>
							{#if status?.zones.length}
								<span class="text-xs text-muted-foreground">
									{l.translate('cloudflare.zones', [status.zones.join(', ')])}
								</span>
							{/if}
						</span>
						<div>
							<Button variant="ghost" size="sm" onclick={() => (replacing = true)}>
								{l.translate('cloudflare.replace_token')}
							</Button>
						</div>
					{:else}
						<form class="flex flex-col gap-1.5" onsubmit={connect}>
							<Label for="cloudflare-token" class="sr-only">{l.translate('cloudflare.token')}</Label
							>
							<div class="flex gap-2">
								<Input
									id="cloudflare-token"
									type="password"
									bind:value={token}
									oninput={() => (tokenProblem = undefined)}
									placeholder={l.translate('cloudflare.token.placeholder')}
									class="font-mono text-label"
									autocomplete="off"
									spellcheck="false"
									aria-invalid={!!tokenProblem || undefined}
									aria-describedby="cloudflare-token-help"
								/>
								<Button type="submit" loading={connecting}>
									{l.translate('cloudflare.connect')}
								</Button>
							</div>
							{#if tokenProblem}
								<p class="text-xs text-danger" role="alert">
									{problemText(tokenProblem)}
									{#if tokenProblem.code === 'cloudflare_permission_missing'}
										{l.translate('cloudflare.problem.fix')}
									{/if}
								</p>
							{/if}
							<p id="cloudflare-token-help" class="text-xs text-muted-foreground">
								{l.translate('cloudflare.token.help')}
							</p>
							{#if replacing}
								<div>
									<Button variant="ghost" size="sm" onclick={() => (replacing = false)}>
										{l.translate('cancel')}
									</Button>
								</div>
							{/if}
						</form>
					{/if}
				</div>
			</li>

			<!-- 3 · Tunnel or DNS record -->
			<li class="grid grid-cols-[24px_minmax(0,1fr)] gap-x-3 gap-y-1.5">
				<span
					aria-hidden="true"
					class={cn(
						'flex size-6 items-center justify-center rounded-full text-xs font-semibold',
						stepTone(!!live, connected && !replacing && !live)
					)}
				>
					{#if live}<Icon icon={Tick02Icon} class="size-3.5" />{:else}3{/if}
				</span>
				<div class="flex min-w-0 flex-col gap-2">
					<strong id="cloudflare-way" class="text-label font-semibold"
						>{l.translate('cloudflare.step3')}</strong
					>
					<div role="radiogroup" aria-labelledby="cloudflare-way" class="grid gap-2 sm:grid-cols-2">
						{#each ['tunnel', 'dns'] as const as way (way)}
							{@const disabled = !connected}
							<button
								type="button"
								role="radio"
								aria-checked={chosen === way}
								{disabled}
								onclick={() => (mode = way)}
								class={cn(
									'flex flex-col gap-1 rounded-md bg-sunken p-3 text-left transition-colors',
									'focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none',
									chosen === way && connected && 'bg-accent-soft ring-2 ring-primary ring-inset',
									disabled && 'cursor-not-allowed opacity-60'
								)}
							>
								<span class="flex flex-wrap items-center gap-x-2 text-label font-semibold">
									{l.translate(`cloudflare.way.${way}`)}
									{#if way === 'tunnel' && !https}
										<span class="text-xs font-normal text-muted-foreground"
											>{l.translate('cloudflare.way.recommended')}</span
										>
									{/if}
								</span>
								<span class="text-xs text-muted-foreground">
									{way === 'tunnel' && connected && !tunnelAllowed
										? l.translate('cloudflare.way.tunnel.not_allowed')
										: l.translate(`cloudflare.way.${way}.help`)}
								</span>
							</button>
						{/each}
					</div>
					{#if connected && !replacing && !result}
						<p class="text-xs text-muted-foreground">{l.translate('cloudflare.step3.help')}</p>
					{/if}

					{#if running}
						<p role="status" class="text-xs text-muted-foreground">
							{l.translate('cloudflare.running')}
						</p>
					{/if}
					{#if result}
						<ul class="flex flex-col gap-1.5" aria-label={l.translate('cloudflare.tasks')}>
							{#each result.tasks as task (task.key)}
								<li class="grid grid-cols-[16px_minmax(0,1fr)] items-start gap-2 text-xs">
									<Icon icon={Tick02Icon} class="mt-px size-4 text-success" />
									<span>
										<b class="font-semibold">{l.translate(`cloudflare.task.${task.key}`)}</b>
										{#if task.detail}<span class="text-muted-foreground">
												· {task.detail}</span
											>{/if}
									</span>
								</li>
							{/each}
						</ul>
						{#if live}
							<p
								role="status"
								class="flex items-center gap-2 rounded-md bg-success-soft px-3 py-2 text-label font-semibold"
							>
								<Icon icon={CheckmarkCircle02Icon} class="size-4 text-success" />
								<a
									href="https://{live}"
									target="_blank"
									rel="noopener noreferrer"
									class="hover:underline">{l.translate('cloudflare.live', [`https://${live}`])}</a
								>
							</p>
							<p class="text-xs text-muted-foreground">{l.translate('cloudflare.live.help')}</p>
						{/if}
					{/if}
				</div>
			</li>
		</ol>

		{#if !connected}
			<p class="flex items-start gap-2 text-xs text-muted-foreground">
				<Icon icon={Alert02Icon} class="mt-px size-3.5 shrink-0" />
				{l.translate('cloudflare.what')}
			</p>
		{/if}
	</section>
{/if}
