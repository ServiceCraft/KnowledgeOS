import client from './client';
import type { BotSettings, ChannelStatus, EditableTenantSecret, TenantSecretStatus, UpdateBotSettingsRequest } from '@/types';

const base = '/admin/bot';

export interface SetSecretRequest {
  value: string;
  metadata?: Record<string, unknown>;
}

export const botAdminApi = {
  getSettings: () =>
    client.get(`/admin/bot/settings`).then((r) => r.data.data as BotSettings),
  updateSettings: (data: UpdateBotSettingsRequest) =>
    client.put(`/admin/bot/settings`, data).then((r) => r.data.data as BotSettings),
  listSecrets: () =>
    client.get(`${base}/secrets`).then((r) => r.data.data as TenantSecretStatus[]),
  getSecretForEdit: (kind: string) =>
    client.get(`${base}/secrets/${kind}/edit`).then((r) => r.data.data as EditableTenantSecret),
  getChannelStatus: () =>
    client.get(`${base}/channels/status`).then((r) => r.data.data as ChannelStatus[]),
  setSecret: (kind: string, data: SetSecretRequest) =>
    client.put(`${base}/secrets/${kind}`, {
      value: data.value,
      metadata: data.metadata ?? {},
    }).then((r) => r.data.data as TenantSecretStatus),
  deleteSecret: (kind: string) =>
    client.delete(`${base}/secrets/${kind}`),
  registerWebhook: (channel: string) =>
    client
      .post(`${base}/channels/${channel}/webhook`)
      .then((r) => r.data.data as { registered: boolean }),
  getSubscriptions: (channel: string) =>
    client.get(`${base}/channels/${channel}/subscriptions`).then((r) => r.data.data as unknown),
};

