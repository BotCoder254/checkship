/**
 * Display helpers for backups (DESIGN.md §11.4 states, Spec artboard of the "checkship backups"
 * canvas): the badge, the words of the Trigger and Stored columns, sizes.
 */
import type { BadgeState } from '#lib/components/status-badge.svelte';
import { BadRequestError } from '#lib/error.ts';
import l, { type AppTranslationsString } from '#lib/localization/index.ts';
import type { Backup } from '#lib/resources/backups.ts';

const UNITS = ['B', 'KB', 'MB', 'GB', 'TB'];

/** "412 MB": powers of 1024, one decimal under 10. */
export function formatBytes(bytes: number): string {
	let value = bytes;
	let unit = 0;
	while (value >= 1024 && unit < UNITS.length - 1) {
		value /= 1024;
		unit++;
	}
	const digits = unit > 0 && value < 10 ? 1 : 0;
	return `${value.toFixed(digits)} ${UNITS[unit]}`;
}

/** A succeeded backup whose storage copy failed reads "Saved here only", in warning. */
export function backupBadge(backup: Backup): { state: BadgeState; label: string } {
	if (backup.status === 'succeeded' && backup.store_error) {
		return { state: 'running', label: l.translate('backups.status.local_only') };
	}
	const state: Record<Backup['status'], BadgeState> = {
		pending: 'pending',
		running: 'running',
		succeeded: 'succeeded',
		failed: 'failed'
	};
	return { state: state[backup.status], label: l.translate(`backups.status.${backup.status}`) };
}

export function triggerLabel(trigger: Backup['trigger']): string {
	return l.translate(`backups.trigger.${trigger}`);
}

export function storedLabel(backup: Backup): { label: string; tone: string } {
	if (backup.status === 'failed')
		return { label: l.translate('backups.stored.nothing'), tone: 'text-muted-foreground' };
	if (backup.status !== 'succeeded')
		return { label: l.translate('backups.stored.not_yet'), tone: 'text-muted-foreground' };
	if (backup.store_error)
		return { label: l.translate('backups.stored.upload_failed'), tone: 'text-warning' };
	if (backup.local && backup.stored)
		return { label: l.translate('backups.stored.both'), tone: 'text-muted-foreground' };
	if (backup.stored)
		return { label: l.translate('backups.stored.storage'), tone: 'text-muted-foreground' };
	return { label: l.translate('backups.stored.here'), tone: 'text-muted-foreground' };
}

/** The volumes a backup holds, or asked for while it runs. */
export function volumesOf(backup: Backup): string[] {
	if (backup.result) return backup.result.volumes.map((v) => v.key);
	return backup.volumes;
}

export function sizeOf(backup: Backup): string {
	if (backup.result) return formatBytes(backup.result.size);
	if (backup.status === 'failed') return backup.error_code ?? '';
	return l.translate('backups.copying');
}

/** Translates an error code, falling back to the code itself. */
export function errorText(code: string): string {
	const text = l.translate(code as AppTranslationsString);
	return text || code;
}

export function appName(backup: Pick<Backup, 'app'>): string {
	return backup.app.deleted
		? l.translate('backups.deleted_app', [backup.app.name])
		: backup.app.name;
}

/** The message to show for a failed request: a field's code for validation errors, else the error code. */
export function messageOf(ex: unknown): string {
	if (ex instanceof BadRequestError) {
		const field = Object.values(ex.fields).find(Boolean);
		return errorText(field ?? ex.message);
	}
	return errorText(ex instanceof Error ? ex.message : 'unexpected_error');
}
