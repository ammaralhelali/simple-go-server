import * as Notifications from 'expo-notifications';
import { Platform } from 'react-native';
import type { NotificationMode, PrayerTimes } from '../types';
import { PRAYER_NAMES } from '../types';

export const AZAN_CHANNEL = 'azan';
export const NORMAL_CHANNEL = 'prayer-normal';

Notifications.setNotificationHandler({
  handleNotification: async () => ({
    shouldShowAlert: true,
    shouldPlaySound: true,
    shouldSetBadge: false,
  }),
});

/**
 * Two Android channels: one that plays the full azan sound (used at home) and
 * one with the default notification sound (used once the user is >= 5 m from
 * home). iOS picks the sound per-notification instead.
 */
export async function setupNotifications(): Promise<boolean> {
  const { status } = await Notifications.requestPermissionsAsync();
  if (status !== 'granted') return false;

  if (Platform.OS === 'android') {
    await Notifications.setNotificationChannelAsync(AZAN_CHANNEL, {
      name: 'Azan (full sound)',
      importance: Notifications.AndroidImportance.MAX,
      sound: 'azan.wav',
      vibrationPattern: [0, 250, 250, 250],
    });
    await Notifications.setNotificationChannelAsync(NORMAL_CHANNEL, {
      name: 'Prayer reminder',
      importance: Notifications.AndroidImportance.HIGH,
    });
  }
  return true;
}

const PRAYER_LABELS: Record<string, string> = {
  fajr: 'Fajr',
  dhuhr: 'Dhuhr',
  asr: 'Asr',
  maghrib: 'Maghrib',
  isha: 'Isha',
};

/**
 * (Re)schedules today's remaining prayer notifications on the channel that
 * matches the current geofence mode. Called on app start and whenever the
 * mode flips (home <-> away).
 */
export async function schedulePrayerNotifications(times: PrayerTimes, mode: NotificationMode) {
  await Notifications.cancelAllScheduledNotificationsAsync();

  const now = Date.now();
  for (const prayer of PRAYER_NAMES) {
    const at = new Date(times[prayer]);
    if (at.getTime() <= now) continue;

    await Notifications.scheduleNotificationAsync({
      content: {
        title: `${PRAYER_LABELS[prayer]} prayer time`,
        body:
          mode === 'azan'
            ? `It is time for ${PRAYER_LABELS[prayer]}. Allahu Akbar.`
            : `${PRAYER_LABELS[prayer]} time has arrived.`,
        sound: mode === 'azan' ? 'azan.wav' : true,
        data: { prayer, mode },
      },
      trigger: {
        date: at,
        channelId: mode === 'azan' ? AZAN_CHANNEL : NORMAL_CHANNEL,
      },
    });
  }
}
