export type NotificationMode = 'azan' | 'normal';

export interface PrayerTimes {
  date: string;
  fajr: string;
  sunrise: string;
  dhuhr: string;
  asr: string;
  maghrib: string;
  isha: string;
  method: string;
  qibla: number;
}

export const PRAYER_NAMES = ['fajr', 'dhuhr', 'asr', 'maghrib', 'isha'] as const;
export type PrayerName = (typeof PRAYER_NAMES)[number];

export interface QiblaResult {
  bearing: number;
  distance_to_kaaba_m: number;
}

export interface DeviceState {
  device_id: string;
  notification_mode: NotificationMode;
  distance_m: number | null;
  updated_at?: string;
}

export interface ReportLocationResult {
  notification_mode: NotificationMode;
  distance_m: number | null;
  mode_changed: boolean;
  home_set: boolean;
}

export interface HomeLocation {
  device_id: string;
  lat: number;
  lng: number;
  radius_m: number;
}

export interface TasbihCounter {
  device_id: string;
  dhikr: string;
  count: number;
  target: number;
  total: number;
}

export interface Preferences {
  device_id: string;
  method: string;
  asr_method: string;
}

export interface Coords {
  lat: number;
  lng: number;
}
