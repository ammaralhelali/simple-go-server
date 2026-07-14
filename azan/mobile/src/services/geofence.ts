import * as Location from 'expo-location';
import * as TaskManager from 'expo-task-manager';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { api } from '../api/client';
import { schedulePrayerNotifications } from './notifications';
import type { NotificationMode, PrayerTimes, ReportLocationResult } from '../types';

const LOCATION_TASK = 'azan-home-geofence';
const MODE_KEY = 'azan.notification_mode';
const TIMES_KEY = 'azan.cached_prayer_times';
const DEVICE_KEY = 'azan.device_id';

export async function getCachedMode(): Promise<NotificationMode> {
  return ((await AsyncStorage.getItem(MODE_KEY)) as NotificationMode) ?? 'azan';
}

export async function cachePrayerTimes(times: PrayerTimes) {
  await AsyncStorage.setItem(TIMES_KEY, JSON.stringify(times));
}

/**
 * Background task: every location fix is reported to the backend, which owns
 * the geofence rule (azan inside the home radius, normal >= 5 m away). When
 * the mode flips, today's remaining prayer notifications are rescheduled on
 * the matching sound channel.
 */
TaskManager.defineTask(LOCATION_TASK, async ({ data, error }) => {
  if (error || !data) return;
  const { locations } = data as { locations: Location.LocationObject[] };
  const last = locations[locations.length - 1];
  if (!last) return;

  const deviceId = await AsyncStorage.getItem(DEVICE_KEY);
  if (!deviceId) return;

  try {
    const result = await api<ReportLocationResult>(`/devices/${deviceId}/location`, {
      method: 'POST',
      body: JSON.stringify({ lat: last.coords.latitude, lng: last.coords.longitude }),
    });

    const previous = await getCachedMode();
    if (result.notification_mode !== previous) {
      await AsyncStorage.setItem(MODE_KEY, result.notification_mode);
      const cached = await AsyncStorage.getItem(TIMES_KEY);
      if (cached) {
        await schedulePrayerNotifications(JSON.parse(cached), result.notification_mode);
      }
    }
  } catch {
    // Offline: keep the last known mode; the next fix will retry.
  }
});

/**
 * Starts background location watching. `distanceInterval: 3` gives fixes
 * roughly every 3 m of movement, tight enough to catch a 5 m geofence.
 * Note: a 5 m radius is at the edge of GPS accuracy — the radius is
 * configurable on the Settings screen if transitions feel jumpy.
 */
export async function startGeofencing(): Promise<boolean> {
  const fg = await Location.requestForegroundPermissionsAsync();
  if (fg.status !== 'granted') return false;
  const bg = await Location.requestBackgroundPermissionsAsync();
  if (bg.status !== 'granted') return false;

  const already = await Location.hasStartedLocationUpdatesAsync(LOCATION_TASK).catch(() => false);
  if (already) return true;

  await Location.startLocationUpdatesAsync(LOCATION_TASK, {
    accuracy: Location.Accuracy.BestForNavigation,
    distanceInterval: 3,
    timeInterval: 15000,
    showsBackgroundLocationIndicator: true,
    foregroundService: {
      notificationTitle: 'Azan is watching your prayer location',
      notificationBody: 'Switches to the full azan sound when you are home.',
    },
  });
  return true;
}

export async function stopGeofencing() {
  const started = await Location.hasStartedLocationUpdatesAsync(LOCATION_TASK).catch(() => false);
  if (started) await Location.stopLocationUpdatesAsync(LOCATION_TASK);
}
