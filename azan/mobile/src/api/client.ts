import AsyncStorage from '@react-native-async-storage/async-storage';

/**
 * Point this at the Go backend. For a device on the same network use your
 * machine's LAN IP; the Android emulator maps the host to 10.0.2.2.
 */
export const API_BASE_URL = 'http://10.0.2.2:8081/api/v1';

const DEVICE_ID_KEY = 'azan.device_id';

export async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(`${API_BASE_URL}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`API ${res.status}: ${body}`);
  }
  return res.json() as Promise<T>;
}

/**
 * Returns the persisted device id, registering the device with the backend on
 * first launch.
 */
export async function getDeviceId(): Promise<string> {
  const existing = await AsyncStorage.getItem(DEVICE_ID_KEY);
  if (existing) return existing;

  const { device_id } = await api<{ device_id: string }>('/devices', {
    method: 'POST',
    body: JSON.stringify({ platform: 'mobile' }),
  });
  await AsyncStorage.setItem(DEVICE_ID_KEY, device_id);
  return device_id;
}
