import { Preferences } from '@capacitor/preferences';
import { isNativePlatform } from './platform';

export interface ConnectionConfig {
  serverUrl: string;
  username: string;
  accessToken?: string;
}

export async function saveConnectionConfig(config: ConnectionConfig) {
  if (!isNativePlatform()) return;
  
  await Preferences.set({
    key: 'connection_config',
    value: JSON.stringify(config)
  });
}

export async function getConnectionConfig(): Promise<ConnectionConfig | null> {
  if (!isNativePlatform()) {
    // For web platform, return null or your default server configuration
    return null;
  }
  
  const { value } = await Preferences.get({ key: 'connection_config' });
  return value ? JSON.parse(value) : null;
}

export async function clearConnectionConfig() {
  if (!isNativePlatform()) return;
  
  await Preferences.remove({ key: 'connection_config' });
}
