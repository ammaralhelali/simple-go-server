import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from './client';
import type {
  Coords,
  DeviceState,
  HomeLocation,
  PrayerTimes,
  Preferences,
  QiblaResult,
  ReportLocationResult,
  TasbihCounter,
} from '../types';

const tzOffsetHours = -new Date().getTimezoneOffset() / 60;

export function usePrayerTimes(coords: Coords | null, method = 'MWL', asr = 'Standard') {
  return useQuery({
    queryKey: ['prayer-times', coords, method, asr],
    enabled: !!coords,
    staleTime: 1000 * 60 * 30,
    queryFn: () =>
      api<PrayerTimes>(
        `/prayer-times?lat=${coords!.lat}&lng=${coords!.lng}&tz=${tzOffsetHours}&method=${method}&asr=${asr}`,
      ),
  });
}

export function useQibla(coords: Coords | null) {
  return useQuery({
    queryKey: ['qibla', coords],
    enabled: !!coords,
    staleTime: Infinity,
    queryFn: () => api<QiblaResult>(`/qibla?lat=${coords!.lat}&lng=${coords!.lng}`),
  });
}

export function useNotificationMode(deviceId: string | null) {
  return useQuery({
    queryKey: ['notification-mode', deviceId],
    enabled: !!deviceId,
    refetchInterval: 1000 * 60, // keep the mode fresh while the app is open
    queryFn: () => api<DeviceState>(`/devices/${deviceId}/notification-mode`),
  });
}

export function useHomeLocation(deviceId: string | null) {
  return useQuery({
    queryKey: ['home', deviceId],
    enabled: !!deviceId,
    retry: false, // 404 simply means home isn't set yet
    queryFn: () => api<HomeLocation>(`/devices/${deviceId}/home`),
  });
}

export function useSetHomeLocation(deviceId: string | null) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (home: { lat: number; lng: number; radius_m: number }) =>
      api<HomeLocation>(`/devices/${deviceId}/home`, {
        method: 'PUT',
        body: JSON.stringify(home),
      }),
    onSuccess: (data) => {
      qc.setQueryData(['home', deviceId], data);
      qc.invalidateQueries({ queryKey: ['notification-mode', deviceId] });
    },
  });
}

export function useReportLocation(deviceId: string | null) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (coords: Coords) =>
      api<ReportLocationResult>(`/devices/${deviceId}/location`, {
        method: 'POST',
        body: JSON.stringify(coords),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['notification-mode', deviceId] }),
  });
}

export function useTasbih(deviceId: string | null) {
  return useQuery({
    queryKey: ['tasbih', deviceId],
    enabled: !!deviceId,
    queryFn: () => api<TasbihCounter[]>(`/devices/${deviceId}/tasbih`),
  });
}

export function useIncrementTasbih(deviceId: string | null) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ dhikr, target }: { dhikr: string; target: number }) =>
      api<TasbihCounter>(`/devices/${deviceId}/tasbih/${encodeURIComponent(dhikr)}/increment`, {
        method: 'POST',
        body: JSON.stringify({ target }),
      }),
    // Optimistic update so the counter feels instant even on a slow network.
    onMutate: async ({ dhikr, target }) => {
      await qc.cancelQueries({ queryKey: ['tasbih', deviceId] });
      const previous = qc.getQueryData<TasbihCounter[]>(['tasbih', deviceId]);
      qc.setQueryData<TasbihCounter[]>(['tasbih', deviceId], (old = []) => {
        const existing = old.find((t) => t.dhikr === dhikr);
        if (!existing) {
          return [...old, { device_id: deviceId ?? '', dhikr, count: 1, target, total: 1 }];
        }
        return old.map((t) =>
          t.dhikr === dhikr
            ? { ...t, count: t.count + 1 > t.target ? 1 : t.count + 1, total: t.total + 1 }
            : t,
        );
      });
      return { previous };
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) qc.setQueryData(['tasbih', deviceId], context.previous);
    },
    onSettled: () => qc.invalidateQueries({ queryKey: ['tasbih', deviceId] }),
  });
}

export function useResetTasbih(deviceId: string | null) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (dhikr: string) =>
      api(`/devices/${deviceId}/tasbih/${encodeURIComponent(dhikr)}/reset`, { method: 'POST' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tasbih', deviceId] }),
  });
}

export function usePreferences(deviceId: string | null) {
  return useQuery({
    queryKey: ['preferences', deviceId],
    enabled: !!deviceId,
    queryFn: () => api<Preferences>(`/devices/${deviceId}/preferences`),
  });
}

export function useSetPreferences(deviceId: string | null) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (prefs: { method: string; asr_method: string }) =>
      api<Preferences>(`/devices/${deviceId}/preferences`, {
        method: 'PUT',
        body: JSON.stringify(prefs),
      }),
    onSuccess: (data) => {
      qc.setQueryData(['preferences', deviceId], data);
      qc.invalidateQueries({ queryKey: ['prayer-times'] });
    },
  });
}
