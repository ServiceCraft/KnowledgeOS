import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { commentsApi } from '@/api/comments';
import { queryKeys } from '@/lib/queryKeys';

export function useCommentsList(entityType: string, entityId: string) {
  return useQuery({
    queryKey: queryKeys.comments.list(entityType, entityId),
    queryFn: () => commentsApi.list(entityType, entityId),
    enabled: !!entityType && !!entityId,
  });
}

export function useCreateComment(entityType: string, entityId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { body: string }) => commentsApi.create(entityType, entityId, data),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: queryKeys.comments.list(entityType, entityId) }),
  });
}

export function useUpdateComment(entityType: string, entityId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ commentId, body }: { commentId: string; body: string }) =>
      commentsApi.update(entityType, entityId, commentId, { body }),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: queryKeys.comments.list(entityType, entityId) }),
  });
}

export function useDeleteComment(entityType: string, entityId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (commentId: string) => commentsApi.delete(entityType, entityId, commentId),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: queryKeys.comments.list(entityType, entityId) }),
  });
}
