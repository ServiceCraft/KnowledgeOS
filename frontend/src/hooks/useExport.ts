import { useMutation } from '@tanstack/react-query';
import { exportApi } from '@/api/export';
import type { ExportData } from '@/types';

export function useExportData() {
  return useMutation({
    mutationFn: () => exportApi.exportData(),
  });
}

export function useImportData() {
  return useMutation({
    mutationFn: (data: ExportData) => exportApi.importData(data),
  });
}
