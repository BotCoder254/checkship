/** Words for where backups are stored, shared by the storage page and the Back up drawer. */
import l from '#lib/localization/index.ts';
import type { BackupStorage } from '#lib/resources/backups.ts';
import type { Target } from '#lib/resources/targets.ts';

export function storageSummary(storage: BackupStorage, targets: Target[] = []): string {
	switch (storage.mode) {
		case 'builtin': {
			const name = targets.find((t) => t.id === storage.target_id)?.name ?? storage.target_id ?? '';
			return l.translate('storage.summary.builtin', [name]);
		}
		case 'external':
			return l.translate('storage.summary.external', [storage.endpoint]);
		default:
			return l.translate('storage.summary.local');
	}
}
