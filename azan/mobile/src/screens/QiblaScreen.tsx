import React, { useEffect, useState } from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { Magnetometer } from 'expo-sensors';
import { useCurrentLocation } from '../hooks/useCurrentLocation';
import { useQibla } from '../api/hooks';
import { colors } from '../theme';

/** Converts a magnetometer vector to a compass heading in degrees. */
function vectorToHeading(x: number, y: number): number {
  let angle = Math.atan2(y, x) * (180 / Math.PI);
  angle = angle - 90; // sensor x-axis points right of the screen; north is up
  return (angle + 360) % 360;
}

export default function QiblaScreen() {
  const { coords, denied } = useCurrentLocation();
  const { data: qibla } = useQibla(coords);
  const [heading, setHeading] = useState(0);

  useEffect(() => {
    Magnetometer.setUpdateInterval(100);
    const sub = Magnetometer.addListener(({ x, y }) => {
      setHeading(vectorToHeading(x, y));
    });
    return () => sub.remove();
  }, []);

  if (denied) {
    return (
      <View style={styles.center}>
        <Text style={styles.muted}>Location permission is required to find the Qibla.</Text>
      </View>
    );
  }
  if (!qibla) {
    return (
      <View style={styles.center}>
        <Text style={styles.muted}>Finding the Qibla…</Text>
      </View>
    );
  }

  // How far to rotate the needle so it points at the Kaaba.
  const rotation = (qibla.bearing - heading + 360) % 360;
  const aligned = rotation < 5 || rotation > 355;
  const distanceKm = Math.round(qibla.distance_to_kaaba_m / 1000);

  return (
    <View style={styles.center}>
      <Text style={styles.title}>Qibla Compass</Text>
      <View style={[styles.dial, aligned && styles.dialAligned]}>
        <Text style={styles.north}>N</Text>
        <View style={{ transform: [{ rotate: `${rotation}deg` }] }}>
          <Text style={styles.needle}>🕋</Text>
          <View style={styles.needleLine} />
        </View>
      </View>
      <Text style={[styles.status, aligned && { color: colors.primary }]}>
        {aligned ? 'Facing the Qibla ✓' : 'Rotate until the Kaaba is at the top'}
      </Text>
      <Text style={styles.muted}>
        Qibla bearing: {qibla.bearing.toFixed(1)}° · Heading: {heading.toFixed(0)}°
      </Text>
      <Text style={styles.muted}>{distanceKm.toLocaleString()} km to Makkah</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  center: {
    flex: 1,
    backgroundColor: colors.background,
    alignItems: 'center',
    justifyContent: 'center',
    padding: 24,
  },
  title: { color: colors.text, fontSize: 24, fontWeight: '700', marginBottom: 24 },
  dial: {
    width: 260,
    height: 260,
    borderRadius: 130,
    borderWidth: 3,
    borderColor: colors.card,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: colors.card,
  },
  dialAligned: { borderColor: colors.primary },
  north: { position: 'absolute', top: 10, color: colors.textMuted, fontWeight: '700' },
  needle: { fontSize: 44, textAlign: 'center' },
  needleLine: {
    width: 4,
    height: 70,
    backgroundColor: colors.accent,
    alignSelf: 'center',
    borderRadius: 2,
    marginTop: 4,
  },
  status: { color: colors.text, fontSize: 16, marginTop: 24, marginBottom: 8 },
  muted: { color: colors.textMuted, marginTop: 4, textAlign: 'center' },
});
