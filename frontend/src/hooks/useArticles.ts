import { useQuery, useMutation, useQueryClient, keepPreviousData } from '@tanstack/react-query';
import { articlesApi, type ArticleFilter } from '@/api/articles';
import { queryKeys } from '@/lib/queryKeys';

export function useArticlesList(filters?: ArticleFilter) {
  return useQuery({
    queryKey: queryKeys.articles.list(filters),
    queryFn: () => articlesApi.list(filters),
    placeholderData: keepPreviousData,
  });
}

export function useArticleDetail(id: string) {
  return useQuery({
    queryKey: queryKeys.articles.detail(id),
    queryFn: () => articlesApi.getById(id),
    enabled: !!id,
  });
}

export function useCreateArticle() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: articlesApi.create,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.articles.all }),
  });
}

export function useUpdateArticle() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Parameters<typeof articlesApi.update>[1]> }) =>
      articlesApi.update(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.articles.all }),
  });
}

export function useDeleteArticle() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: articlesApi.delete,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.articles.all }),
  });
}
