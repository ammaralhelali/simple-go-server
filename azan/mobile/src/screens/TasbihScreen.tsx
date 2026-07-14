import React, { useState } from 'react';
import { Pressable, StyleSheet, Text, Vibration, View } from 'react-native';
import { useIncrementTasbih, useResetTasbih, useTasbih } from '../api/hooks';
import { colors } from '../theme';

const DHIKR_LIST = [
  { key: 'subhanallah', label: 'سبحان الله', latin: 'SubhanAllah' },
  { key: 'alhamdulillah', label: 'الحمد لله', latin: 'Alhamdulillah' },
  { key: 'allahuakbar', label: 'الله أكبر', latin: 'Allahu Akbar' },
];
const TARGETS = [33, 66, 99];

export default function TasbihScreen({ deviceId }: { deviceId: string | null }) {
  const [dhikr, setDhikr] = useState(DHIKR_LIST[0]);
  const [target, setTarget] = useState(33);

  const { data: counters } = useTasbih(deviceId);
  const increment = useIncrementTasbih(deviceId);
  const reset = useResetTasbih(deviceId);

  const current = counters?.find((c) => c.dhikr === dhikr.key);
  const count = current?.count ?? 0;
  const total = current?.total ?? 0;

  const onTap = () => {
    Vibration.vibrate(count + 1 >= target ? [0, 120, 80, 120] : 15);
    increment.mutate({ dhikr: dhikr.key, target });
  };

  return (
    <View style={styles.container}>
      <View style={styles.selector}>
        {DHIKR_LIST.map((d) => (
          <Pressable
            key={d.key}
            onPress={() => setDhikr(d)}
            style={[styles.chip, dhikr.key === d.key && styles.chipActive]}
          >
            <Text style={styles.chipText}>{d.latin}</Text>
          </Pressable>
        ))}
      </View>

      <Text style={styles.arabic}>{dhikr.label}</Text>

      <Pressable onPress={onTap} style={({ pressed }) => [styles.counter, pressed && styles.counterPressed]}>
        <Text style={styles.count}>{count}</Text>
        <Text style={styles.target}>of {target}</Text>
      </Pressable>

      <View style={styles.selector}>
        {TARGETS.map((t) => (
          <Pressable
            key={t}
            onPress={() => setTarget(t)}
            style={[styles.chip, target === t && styles.chipActive]}
          >
            <Text style={styles.chipText}>{t}</Text>
          </Pressable>
        ))}
      </View>

      <Text style={styles.total}>Lifetime total: {total.toLocaleString()}</Text>

      <Pressable onPress={() => reset.mutate(dhikr.key)} style={styles.reset}>
        <Text style={styles.resetText}>Reset round</Text>
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: colors.background,
    alignItems: 'center',
    justifyContent: 'center',
    padding: 24,
  },
  selector: { flexDirection: 'row', gap: 8, marginVertical: 12 },
  chip: {
    borderRadius: 999,
    paddingHorizontal: 16,
    paddingVertical: 8,
    backgroundColor: colors.card,
  },
  chipActive: { backgroundColor: colors.primary },
  chipText: { color: colors.text, fontSize: 14 },
  arabic: { color: colors.accent, fontSize: 36, marginVertical: 8 },
  counter: {
    width: 220,
    height: 220,
    borderRadius: 110,
    backgroundColor: colors.card,
    borderWidth: 4,
    borderColor: colors.primary,
    alignItems: 'center',
    justifyContent: 'center',
    marginVertical: 16,
  },
  counterPressed: { backgroundColor: '#1d2f25' },
  count: { color: colors.text, fontSize: 72, fontWeight: '700', fontVariant: ['tabular-nums'] },
  target: { color: colors.textMuted, fontSize: 16 },
  total: { color: colors.textMuted, marginTop: 8 },
  reset: { marginTop: 16, padding: 12 },
  resetText: { color: colors.danger, fontSize: 15 },
});
