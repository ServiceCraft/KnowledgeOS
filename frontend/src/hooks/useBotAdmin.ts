import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { botAdminApi, type SetSecretRequest } from '@/api/botAdmin';
import { queryKeys } from '@/lib/queryKeys';
import type { UpdateBotSettingsRequest } from '@/types';

export function useBotSettings() {
  return useQuery({
    queryKey: queryKeys.botAdmin.settings,
    queryFn: botAdminApi.getSettings,
  });
}

export function useUpdateBotSettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateBotSettingsRequest) => botAdminApi.updateSettings(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.botAdmin.settings }),
  });
}

export function useBotSecrets() {
  return useQuery({
    queryKey: queryKeys.botAdmin.secrets,
    queryFn: botAdminApi.listSecrets,
  });
}

export function useSetBotSecret() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ kind, data }: { kind: string; data: SetSecretRequest }) =>
      botAdminApi.setSecret(kind, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.botAdmin.secrets }),
  });
}

export function useDeleteBotSecret() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (kind: string) => botAdminApi.deleteSecret(kind),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.botAdmin.secrets }),
  });
}
