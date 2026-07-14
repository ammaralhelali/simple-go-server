import { useEffect, useState } from 'react';
import * as Location from 'expo-location';
import type { Coords } from '../types';

export function useCurrentLocation() {
  const [coords, setCoords] = useState<Coords | null>(null);
  const [denied, setDenied] = useState(false);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      const { status } = await Location.requestForegroundPermissionsAsync();
      if (status !== 'granted') {
        if (!cancelled) setDenied(true);
        return;
      }
      const pos = await Location.getCurrentPositionAsync({
        accuracy: Location.Accuracy.Balanced,
      });
      if (!cancelled) {
        setCoords({ lat: pos.coords.latitude, lng: pos.coords.longitude });
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  return { coords, denied };
}
