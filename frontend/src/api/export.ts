import client from './client';
import type { ExportData, ImportResult } from '@/types';

export const exportApi = {
  exportData: () =>
    client.get('/export').then((r) => r.data.data as ExportData),
  importData: (data: ExportData) =>
    client.post('/import', data).then((r) => r.data.data as ImportResult),
};
