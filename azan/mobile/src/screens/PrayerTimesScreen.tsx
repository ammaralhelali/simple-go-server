import React, { useEffect, useMemo, useState } from 'react';
import { ActivityIndicator, ScrollView, StyleSheet, Text, View } from 'react-native';
import { useCurrentLocation } from '../hooks/useCurrentLocation';
import { useNotificationMode, usePrayerTimes, usePreferences } from '../api/hooks';
import { cachePrayerTimes } from '../services/geofence';
import { schedulePrayerNotifications } from '../services/notifications';
import { colors } from '../theme';
import { PRAYER_NAMES, PrayerName, PrayerTimes } from '../types';

const LABELS: Record<PrayerName | 'sunrise', string> = {
  fajr: 'Fajr',
  sunrise: 'Sunrise',
  dhuhr: 'Dhuhr',
  asr: 'Asr',
  maghrib: 'Maghrib',
  isha: 'Isha',
};

function fmt(iso: string): string {
  return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

function nextPrayer(times: PrayerTimes): { name: PrayerName; at: Date } | null {
  const now = Date.now();
  for (const name of PRAYER_NAMES) {
    const at = new Date(times[name]);
    if (at.getTime() > now) return { name, at };
  }
  return null;
}

function useCountdown(target: Date | null): string {
  const [, tick] = useState(0);
  useEffect(() => {
    const id = setInterval(() => tick((n) => n + 1), 1000);
    return () => clearInterval(id);
  }, []);
  if (!target) return '';
  const ms = Math.max(0, target.getTime() - Date.now());
  const h = Math.floor(ms / 3600000);
  const m = Math.floor((ms % 3600000) / 60000);
  const s = Math.floor((ms % 60000) / 1000);
  return `${h}h ${m}m ${s}s`;
}

export default function PrayerTimesScreen({ deviceId }: { deviceId: string | null }) {
  const { coords, denied } = useCurrentLocation();
  const { data: prefs } = usePreferences(deviceId);
  const { data: times, isLoading, error } = usePrayerTimes(
    coords,
    prefs?.method ?? 'MWL',
    prefs?.asr_method ?? 'Standard',
  );
  const { data: state } = useNotificationMode(deviceId);
  const mode = state?.notification_mode ?? 'azan';

  // Whenever fresh times or a mode change arrive, resync local notifications.
  useEffect(() => {
    if (!times) return;
    cachePrayerTimes(times);
    schedulePrayerNotifications(times, mode);
  }, [times, mode]);

  const upcoming = useMemo(() => (times ? nextPrayer(times) : null), [times]);
  const countdown = useCountdown(upcoming?.at ?? null);

  if (denied) {
    return (
      <View style={styles.center}>
        <Text style={styles.muted}>Location permission is required to compute prayer times.</Text>
      </View>
    );
  }
  if (isLoading || !times) {
    return (
      <View style={styles.center}>
        {error ? (
          <Text style={styles.muted}>Could not reach the Azan server.</Text>
        ) : (
          <ActivityIndicator color={colors.primary} size="large" />
        )}
      </View>
    );
  }

  return (
    <ScrollView style={styles.container} contentContainerStyle={{ padding: 16 }}>
      <View style={styles.hero}>
        <Text style={styles.heroLabel}>Next prayer</Text>
        <Text style={styles.heroPrayer}>{upcoming ? LABELS[upcoming.name] : 'Fajr (tomorrow)'}</Text>
        {upcoming && <Text style={styles.heroCountdown}>{countdown}</Text>}
        <View style={[styles.modeBadge, mode === 'azan' ? styles.modeAzan : styles.modeNormal]}>
          <Text style={styles.modeText}>
            {mode === 'azan' ? '🏠 At home — full azan sound' : '🚶 Away — normal notification'}
          </Text>
        </View>
      </View>

      {(['fajr', 'sunrise', 'dhuhr', 'asr', 'maghrib', 'isha'] as const).map((name) => (
        <View
          key={name}
          style={[styles.row, upcoming?.name === name && styles.rowActive]}
        >
          <Text style={styles.rowName}>{LABELS[name]}</Text>
          <Text style={styles.rowTime}>{fmt(times[name])}</Text>
        </View>
      ))}

      <Text style={styles.footer}>
        {times.date} · {times.method} method
      </Text>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: colors.background },
  center: {
    flex: 1,
    backgroundColor: colors.background,
    alignItems: 'center',
    justifyContent: 'center',
    padding: 24,
  },
  muted: { color: colors.textMuted, textAlign: 'center' },
  hero: {
    backgroundColor: colors.card,
    borderRadius: 16,
    padding: 20,
    alignItems: 'center',
    marginBottom: 16,
  },
  heroLabel: { color: colors.textMuted, fontSize: 13, textTransform: 'uppercase' },
  heroPrayer: { color: colors.accent, fontSize: 34, fontWeight: '700', marginVertical: 4 },
  heroCountdown: { color: colors.text, fontSize: 18, fontVariant: ['tabular-nums'] },
  modeBadge: { marginTop: 12, borderRadius: 999, paddingHorizontal: 12, paddingVertical: 6 },
  modeAzan: { backgroundColor: 'rgba(59,167,118,0.25)' },
  modeNormal: { backgroundColor: 'rgba(212,175,55,0.2)' },
  modeText: { color: colors.text, fontSize: 13 },
  row: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    backgroundColor: colors.card,
    borderRadius: 12,
    padding: 16,
    marginBottom: 8,
  },
  rowActive: { borderWidth: 1, borderColor: colors.primary },
  rowName: { color: colors.text, fontSize: 16 },
  rowTime: { color: colors.text, fontSize: 16, fontVariant: ['tabular-nums'] },
  footer: { color: colors.textMuted, textAlign: 'center', marginTop: 12, fontSize: 12 },
});
