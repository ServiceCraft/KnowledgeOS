import { useRef, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Download, Upload } from 'lucide-react';
import { useExportData, useImportData } from '@/hooks/useExport';
import { ConfirmDialog } from '@/components/shared/ConfirmDialog';
import type { ExportData } from '@/types';
import { toast } from 'sonner';

export function ExportPage() {
  const exportMutation = useExportData();
  const importMutation = useImportData();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [importData, setImportData] = useState<ExportData | null>(null);
  const [showConfirm, setShowConfirm] = useState(false);

  const handleExport = () => {
    exportMutation.mutate(undefined, {
      onSuccess: (data) => {
        const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `knowledgeos-export-${new Date().toISOString().slice(0, 10)}.json`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
        toast.success('Экспорт загружен');
      },
      onError: () => toast.error('Не удалось выполнить экспорт'),
    });
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (ev) => {
      try {
        const data = JSON.parse(ev.target?.result as string) as ExportData;
        setImportData(data);
        setShowConfirm(true);
      } catch {
        toast.error('Некорректный JSON-файл');
      }
    };
    reader.readAsText(file);
    e.target.value = '';
  };

  const handleImport = () => {
    if (!importData) return;
    importMutation.mutate(importData, {
      onSuccess: (result) => {
        setShowConfirm(false);
        setImportData(null);
        toast.success(`Импортировано: ${result.imported}, пропущено: ${result.skipped}`);
        if (result.errors.length > 0) {
          toast.error(`Ошибок при импорте: ${result.errors.length}`);
        }
      },
      onError: () => {
        toast.error('Не удалось выполнить импорт');
        setShowConfirm(false);
      },
    });
  };

  return (
    <div className="space-y-4 max-w-2xl">
      <h1 className="text-2xl font-semibold">Экспорт / Импорт</h1>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Экспорт</CardTitle>
            <CardDescription>
              Скачать все данные базы знаний в формате JSON.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button onClick={handleExport} disabled={exportMutation.isPending}>
              <Download className="h-4 w-4 mr-2" />
              {exportMutation.isPending ? 'Экспорт...' : 'Экспортировать данные'}
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Импорт</CardTitle>
            <CardDescription>
              Загрузить ранее экспортированный JSON-файл для восстановления данных.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <input
              ref={fileInputRef}
              type="file"
              accept=".json"
              onChange={handleFileSelect}
              className="hidden"
            />
            <Button variant="outline" onClick={() => fileInputRef.current?.click()}>
              <Upload className="h-4 w-4 mr-2" />
              Выбрать файл
            </Button>
          </CardContent>
        </Card>
      </div>

      <ConfirmDialog
        open={showConfirm}
        onOpenChange={setShowConfirm}
        title="Подтвердите импорт"
        description="Данные из выбранного файла будут импортированы. Существующие данные могут быть обновлены. Продолжить?"
        onConfirm={handleImport}
        loading={importMutation.isPending}
        destructive={false}
      />
    </div>
  );
}
