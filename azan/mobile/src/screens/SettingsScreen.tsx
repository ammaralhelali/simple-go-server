import React, { useState } from 'react';
import {
  Alert,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import * as Location from 'expo-location';
import {
  useHomeLocation,
  useNotificationMode,
  usePreferences,
  useSetHomeLocation,
  useSetPreferences,
} from '../api/hooks';
import { startGeofencing } from '../services/geofence';
import { colors } from '../theme';

const METHODS = ['MWL', 'ISNA', 'Egypt', 'Makkah', 'Karachi'];
const ASR_METHODS = ['Standard', 'Hanafi'];

export default function SettingsScreen({ deviceId }: { deviceId: string | null }) {
  const [radius, setRadius] = useState('5');

  const { data: home } = useHomeLocation(deviceId);
  const { data: state } = useNotificationMode(deviceId);
  const { data: prefs } = usePreferences(deviceId);
  const setHome = useSetHomeLocation(deviceId);
  const setPrefs = useSetPreferences(deviceId);

  const markHomeHere = async () => {
    const { status } = await Location.requestForegroundPermissionsAsync();
    if (status !== 'granted') {
      Alert.alert('Permission needed', 'Location permission is required to set your home.');
      return;
    }
    const pos = await Location.getCurrentPositionAsync({
      accuracy: Location.Accuracy.BestForNavigation,
    });
    const radiusM = Math.max(1, parseFloat(radius) || 5);
    setHome.mutate(
      { lat: pos.coords.latitude, lng: pos.coords.longitude, radius_m: radiusM },
      {
        onSuccess: async () => {
          const ok = await startGeofencing();
          Alert.alert(
            'Home saved',
            ok
              ? `Azan sound plays within ${radiusM} m of here; a normal notification is used beyond that.`
              : 'Home saved, but background location permission was denied — the mode will only update while the app is open.',
          );
        },
        onError: (e) => Alert.alert('Error', String(e)),
      },
    );
  };

  return (
    <ScrollView style={styles.container} contentContainerStyle={{ padding: 16 }}>
      <Text style={styles.section}>Home & azan sound</Text>
      <View style={styles.card}>
        <Text style={styles.text}>
          {home
            ? `Home set at ${home.lat.toFixed(5)}, ${home.lng.toFixed(5)} (radius ${home.radius_m} m)`
            : 'Home is not set yet. The full azan sound is used everywhere until you set it.'}
        </Text>
        <Text style={styles.muted}>
          Current mode:{' '}
          {state?.notification_mode === 'normal' ? 'normal notification (away)' : 'azan sound (home)'}
          {state?.distance_m != null && ` · ${state.distance_m.toFixed(1)} m from home`}
        </Text>
        <View style={styles.rowInline}>
          <Text style={styles.text}>Radius (m):</Text>
          <TextInput
            style={styles.input}
            value={radius}
            onChangeText={setRadius}
            keyboardType="numeric"
          />
        </View>
        <Text style={styles.mutedSmall}>
          Tip: 5 m is at the edge of GPS accuracy; raise the radius if the mode flips while you are
          still inside the house.
        </Text>
        <Pressable style={styles.button} onPress={markHomeHere}>
          <Text style={styles.buttonText}>📍 Set current location as Home</Text>
        </Pressable>
      </View>

      <Text style={styles.section}>Calculation method</Text>
      <View style={styles.card}>
        <View style={styles.chips}>
          {METHODS.map((m) => (
            <Pressable
              key={m}
              onPress={() => setPrefs.mutate({ method: m, asr_method: prefs?.asr_method ?? 'Standard' })}
              style={[styles.chip, prefs?.method === m && styles.chipActive]}
            >
              <Text style={styles.text}>{m}</Text>
            </Pressable>
          ))}
        </View>
        <Text style={[styles.section, { marginTop: 12 }]}>Asr</Text>
        <View style={styles.chips}>
          {ASR_METHODS.map((m) => (
            <Pressable
              key={m}
              onPress={() => setPrefs.mutate({ method: prefs?.method ?? 'MWL', asr_method: m })}
              style={[styles.chip, prefs?.asr_method === m && styles.chipActive]}
            >
              <Text style={styles.text}>{m}</Text>
            </Pressable>
          ))}
        </View>
      </View>

      <Text style={styles.mutedSmall}>Device ID: {deviceId ?? 'registering…'}</Text>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: colors.background },
  section: {
    color: colors.textMuted,
    fontSize: 13,
    textTransform: 'uppercase',
    marginBottom: 8,
    marginTop: 16,
  },
  card: { backgroundColor: colors.card, borderRadius: 12, padding: 16 },
  text: { color: colors.text, fontSize: 15, marginBottom: 4 },
  muted: { color: colors.textMuted, fontSize: 14, marginBottom: 8 },
  mutedSmall: { color: colors.textMuted, fontSize: 12, marginTop: 8 },
  rowInline: { flexDirection: 'row', alignItems: 'center', gap: 8, marginTop: 8 },
  input: {
    backgroundColor: colors.background,
    color: colors.text,
    borderRadius: 8,
    paddingHorizontal: 12,
    paddingVertical: 6,
    width: 80,
  },
  button: {
    backgroundColor: colors.primary,
    borderRadius: 10,
    padding: 14,
    alignItems: 'center',
    marginTop: 12,
  },
  buttonText: { color: colors.text, fontWeight: '600' },
  chips: { flexDirection: 'row', flexWrap: 'wrap', gap: 8 },
  chip: {
    borderRadius: 999,
    paddingHorizontal: 14,
    paddingVertical: 8,
    backgroundColor: colors.background,
  },
  chipActive: { backgroundColor: colors.primary },
});
