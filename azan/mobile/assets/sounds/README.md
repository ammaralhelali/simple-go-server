# Sounds

Place the azan audio file here as `azan.wav` (mono/stereo WAV, ideally under
~1 MB / 30 s so Android accepts it as a notification channel sound).

It is referenced by:

- `app.json` → `expo-notifications` plugin `sounds` array (bundles it on both platforms)
- `src/services/notifications.ts` → the Android `azan` channel and per-notification iOS sound
