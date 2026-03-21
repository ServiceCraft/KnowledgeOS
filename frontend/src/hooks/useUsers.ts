import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { usersApi, type UserFilter, type CreateUserRequest, type UpdateUserRequest } from '@/api/users';
import { queryKeys } from '@/lib/queryKeys';

export function useUsersList(filters?: UserFilter) {
  return useQuery({
    queryKey: queryKeys.users.list(filters),
    queryFn: () => usersApi.list(filters),
  });
}

export function useCreateUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateUserRequest) => usersApi.create(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.users.all }),
  });
}

export function useUpdateUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateUserRequest }) =>
      usersApi.update(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.users.all }),
  });
}

export function useDeleteUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: usersApi.delete,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.users.all }),
  });
}
