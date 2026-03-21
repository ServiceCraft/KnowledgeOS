import { useQuery, useMutation, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import { themesApi, type ThemeFilter } from '@/api/themes';
import { queryKeys } from '@/lib/queryKeys';

export function useThemesList(filters?: ThemeFilter) {
  return useQuery({
    queryKey: queryKeys.themes.list(filters),
    queryFn: () => themesApi.list(filters),
    placeholderData: keepPreviousData,
  });
}

export function useThemeDetail(id: string) {
  return useQuery({
    queryKey: queryKeys.themes.detail(id),
    queryFn: () => themesApi.getById(id),
    enabled: !!id,
  });
}

export function useCreateTheme() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: themesApi.create,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.themes.all }),
  });
}

export function useUpdateTheme() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Parameters<typeof themesApi.update>[1]> }) =>
      themesApi.update(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.themes.all }),
  });
}

export function useDeleteTheme() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: themesApi.delete,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.themes.all }),
  });
}
