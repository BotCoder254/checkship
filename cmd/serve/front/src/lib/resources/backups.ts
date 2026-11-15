import { POLLING_INTERVAL_MS } from '#lib/config.ts';
import fetcher, {
	type FetchOptions,
	type FetchService,
	type QueryResult
} from '#lib/fetcher/index.ts';
import { buildFormData } from '#lib/form.ts';
import type { Environment } from '#lib/resources/deployments.ts';

export type BackupStatus = 'pending' | 'running' | 'succeeded' | 'failed';

export type BackupTrigger =
	'manual' | 'upload' | 'before_delete' | 'before_target_change' | 'before_restore';

export type VolumeStats = { key: string; size: number; files: number };

/** A mount of the compose project that a backup leaves out. */
export type SkippedMount = {
	service: string;
	kind: 'bind' | 'anonymous' | 'external';
	source?: string;
	target: string;
};

export type BackupResult = {
	size: number;
	sha256: string;
	volumes: VolumeStats[];
	skipped?: SkippedMount[];
};

export type Backup = {
	id: string;
	app: { id: string; name: string; deleted: boolean };
	environment: Environment;
	target: { id: string; name?: string | null };
	trigger: BackupTrigger;
	status: BackupStatus;
	stop_containers: boolean;
	volumes: string[];
	result?: BackupResult | null;
	local: boolean;
	stored: boolean;
	store_error?: string | null;
	error_code?: string | null;
	requested_at: string;
	requested_by: { id: string; email: string };
	finished_at?: string | null;
};

export type Restore = {
	id: string;
	app: { id: string; name: string; deleted: boolean };
	environment: Environment;
	status: BackupStatus;
	volumes: string[];
	restored: string[];
	snapshot_id?: string | null;
	error_code?: string | null;
	requested_at: string;
	requested_by: { id: string; email: string };
	finished_at?: string | null;
};

export type BackupDetail = Backup & { restores: Restore[] };

export type Volume = { key: string; docker_name: string; driver: string; services: string[] };

export type AppVolumes = { target: string; volumes: Volume[]; skipped: SkippedMount[] };

export type StorageMode = 'local' | 'builtin' | 'external';

export type BackupStorage = {
	mode: StorageMode;
	target_id?: string | null;
	endpoint: string;
	use_ssl: boolean;
	path_style: boolean;
	region: string;
	bucket: string;
	prefix: string;
	access_key: string;
	secret_key: string;
	keep_local: boolean;
	status: 'ready' | 'configuring' | 'failed';
	error_code?: string | null;
	updated_at?: string | null;
};

export type ExternalStorage = {
	endpoint: string;
	region: string;
	bucket: string;
	prefix: string;
	access_key: string;
	secret_key: string;
	use_ssl: boolean;
	path_style: boolean;
};

export type UpdateStorage = {
	mode: StorageMode;
	target_id?: string;
	external?: ExternalStorage;
	keep_local?: boolean;
};

export type RequestBackup = {
	environment: Environment;
	volumes: string[];
	stop_containers: boolean;
};

export type RequestRestore = {
	app_id?: string;
	environment?: Environment;
	volumes?: string[];
	snapshot_first: boolean;
};

const BACKUPS = '/api/v1/backups';
const STORAGE = '/api/v1/backup-storage';

/** A backup still being made or a restore still running: worth refreshing often. */
export function isBusy(status: BackupStatus) {
	return status === 'pending' || status === 'running';
}

export interface BackupsService {
	queryAll(): QueryResult<Backup[]>;
	queryByApp(appId: string): QueryResult<Backup[]>;
	queryOne(id: string): QueryResult<BackupDetail>;
	queryVolumes(appId: string, environment: Environment): QueryResult<AppVolumes>;
	fetchVolumes(
		appId: string,
		environment: Environment,
		options?: FetchOptions
	): Promise<AppVolumes>;
	request(appId: string, payload: RequestBackup): Promise<BackupDetail>;
	upload(appId: string, archive: File, environment?: Environment): Promise<BackupDetail>;
	restore(id: string, payload: RequestRestore): Promise<BackupDetail>;
	retry(id: string): Promise<BackupDetail>;
	store(id: string): Promise<void>;
	delete(id: string): Promise<void>;
	downloadUrl(id: string): string;
	refresh(appId?: string): Promise<unknown>;
	queryStorage(): QueryResult<BackupStorage>;
	updateStorage(payload: UpdateStorage): Promise<BackupStorage>;
	testStorage(external: ExternalStorage): Promise<void>;
}

export class RemoteBackupsService implements BackupsService {
	constructor(
		private readonly _fetcher: FetchService,
		private readonly _pollingInterval: number
	) {}

	// Every change shows on both lists and on the app's own list.
	private invalidations(appId?: string) {
		return appId ? [BACKUPS, `/api/v1/apps/${appId}/backups`] : [BACKUPS];
	}

	queryAll() {
		return this._fetcher.query<Backup[]>(BACKUPS, { refreshInterval: this._pollingInterval });
	}

	queryByApp(appId: string) {
		return this._fetcher.query<Backup[]>(`/api/v1/apps/${appId}/backups`, {
			refreshInterval: this._pollingInterval
		});
	}

	queryOne(id: string) {
		return this._fetcher.query<BackupDetail>(`${BACKUPS}/${id}`, {
			refreshInterval: this._pollingInterval
		});
	}

	queryVolumes(appId: string, environment: Environment) {
		return this._fetcher.query<AppVolumes>(`/api/v1/apps/${appId}/volumes`, {
			params: { environment }
		});
	}

	fetchVolumes(appId: string, environment: Environment, options?: FetchOptions) {
		return this._fetcher.get<AppVolumes>(`/api/v1/apps/${appId}/volumes`, {
			...options,
			params: { environment }
		});
	}

	request(appId: string, payload: RequestBackup) {
		return this._fetcher.post<BackupDetail, RequestBackup>(
			`/api/v1/apps/${appId}/backups`,
			payload,
			{
				invalidate: this.invalidations(appId)
			}
		);
	}

	upload(appId: string, archive: File, environment?: Environment) {
		return this._fetcher.post<BackupDetail, FormData>(
			`/api/v1/apps/${appId}/backups/upload`,
			buildFormData(environment ? { archive, environment } : { archive }),
			{ invalidate: this.invalidations(appId) }
		);
	}

	restore(id: string, payload: RequestRestore) {
		return this._fetcher.post<BackupDetail, RequestRestore>(`${BACKUPS}/${id}/restore`, payload, {
			invalidate: [`${BACKUPS}/${id}`, ...this.invalidations()]
		});
	}

	retry(id: string) {
		return this._fetcher.post<BackupDetail, undefined>(`${BACKUPS}/${id}/retry`, undefined, {
			invalidate: [`${BACKUPS}/${id}`, ...this.invalidations()]
		});
	}

	store(id: string) {
		return this._fetcher.post<void, undefined>(`${BACKUPS}/${id}/store`, undefined, {
			invalidate: [`${BACKUPS}/${id}`, ...this.invalidations()]
		});
	}

	delete(id: string) {
		return this._fetcher.delete(`${BACKUPS}/${id}`, { invalidate: this.invalidations() });
	}

	downloadUrl(id: string) {
		return `${BACKUPS}/${id}/download`;
	}

	refresh(appId?: string) {
		return Promise.all([
			this._fetcher.get(BACKUPS, { cache: 'no-store' }),
			appId ? this._fetcher.get(`/api/v1/apps/${appId}/backups`, { cache: 'no-store' }) : null
		]);
	}

	queryStorage() {
		return this._fetcher.query<BackupStorage>(STORAGE, { refreshInterval: this._pollingInterval });
	}

	updateStorage(payload: UpdateStorage) {
		return this._fetcher.put<BackupStorage, UpdateStorage>(STORAGE, payload);
	}

	testStorage(external: ExternalStorage) {
		return this._fetcher.post<void, { external: ExternalStorage }>(
			`${STORAGE}/test`,
			{ external },
			{ skipUrlInvalidate: true }
		);
	}
}

const service: BackupsService = new RemoteBackupsService(fetcher, POLLING_INTERVAL_MS);

export default service;
