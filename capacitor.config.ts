import type { CapacitorConfig } from '@capacitor/cli';
import { SafeArea } from '@capacitor-community/safe-area';

const config: CapacitorConfig = {
  appId: 'com.scriberr.app',
  appName: 'scriberr',
  webDir: 'build',
   server: {
    cleartext: true,
    allowNavigation: ['*'],
    androidScheme: 'http',
    iosScheme: 'http'
  },
  ios: {
    // Disable full screen
    // contentInset: 'always',
    scrollEnabled: true,
    limitsNavigationsToAppBoundDomains: false
   },
   loggingBehavior: 'debug',
   plugins: {
    "SafeArea": {
      "enabled": true,
      "customColorsForSystemBars": true,
      "statusBarColor": '#000000',
      "statusBarContent": 'light',
      "navigationBarColor": '#000000',
      "navigationBarContent": 'light',
    }
  }
};


export default config;
