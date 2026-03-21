import client from './client';
import type { SyncStatus } from '@/types';

export const syncApi = {
  getStatus: () =>
    client.get('/sync/status').then((r) => r.data.data as SyncStatus),
};
