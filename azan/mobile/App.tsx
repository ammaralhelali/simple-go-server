import React, { useEffect, useState } from 'react';
import { Text } from 'react-native';
import { NavigationContainer, DarkTheme } from '@react-navigation/native';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { StatusBar } from 'expo-status-bar';

import { getDeviceId } from './src/api/client';
import { setupNotifications } from './src/services/notifications';
import { startGeofencing } from './src/services/geofence';
import PrayerTimesScreen from './src/screens/PrayerTimesScreen';
import QiblaScreen from './src/screens/QiblaScreen';
import TasbihScreen from './src/screens/TasbihScreen';
import SettingsScreen from './src/screens/SettingsScreen';
import { colors } from './src/theme';

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: 1 } },
});

const Tab = createBottomTabNavigator();

const theme = {
  ...DarkTheme,
  colors: {
    ...DarkTheme.colors,
    background: colors.background,
    card: colors.card,
    primary: colors.primary,
    text: colors.text,
  },
};

const TAB_ICONS: Record<string, string> = {
  Prayers: '🕌',
  Qibla: '🧭',
  Tasbih: '📿',
  Settings: '⚙️',
};

export default function App() {
  const [deviceId, setDeviceId] = useState<string | null>(null);

  useEffect(() => {
    (async () => {
      await setupNotifications();
      try {
        const id = await getDeviceId();
        setDeviceId(id);
        // Resume the home geofence watcher if permissions are already granted.
        await startGeofencing();
      } catch (e) {
        console.warn('startup:', e);
      }
    })();
  }, []);

  return (
    <QueryClientProvider client={queryClient}>
      <NavigationContainer theme={theme}>
        <StatusBar style="light" />
        <Tab.Navigator
          screenOptions={({ route }) => ({
            headerShown: false,
            tabBarActiveTintColor: colors.primary,
            tabBarIcon: () => <Text style={{ fontSize: 18 }}>{TAB_ICONS[route.name]}</Text>,
          })}
        >
          <Tab.Screen name="Prayers">
            {() => <PrayerTimesScreen deviceId={deviceId} />}
          </Tab.Screen>
          <Tab.Screen name="Qibla" component={QiblaScreen} />
          <Tab.Screen name="Tasbih">
            {() => <TasbihScreen deviceId={deviceId} />}
          </Tab.Screen>
          <Tab.Screen name="Settings">
            {() => <SettingsScreen deviceId={deviceId} />}
          </Tab.Screen>
        </Tab.Navigator>
      </NavigationContainer>
    </QueryClientProvider>
  );
}
