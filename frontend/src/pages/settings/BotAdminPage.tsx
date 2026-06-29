import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { SettingsTab } from '@/components/bot-admin/SettingsTab';
import { ChannelsAndSecretsTab } from '@/components/bot-admin/ChannelsAndSecretsTab';
import { RagTab } from '@/components/bot-admin/RagTab';

export function BotAdminPage() {
  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">Администрирование бота</h1>
      <Tabs defaultValue="settings">
        <TabsList>
          <TabsTrigger value="settings">Настройки</TabsTrigger>
          <TabsTrigger value="channels">Каналы и секреты</TabsTrigger>
          <TabsTrigger value="rag">База знаний</TabsTrigger>
        </TabsList>
        <TabsContent value="settings" className="mt-4">
          <SettingsTab />
        </TabsContent>
        <TabsContent value="channels" className="mt-4">
          <ChannelsAndSecretsTab />
        </TabsContent>
        <TabsContent value="rag" className="mt-4">
          <RagTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}
