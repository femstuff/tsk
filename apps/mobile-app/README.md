# TSK Mobile MVP

`apps/mobile-app` is a minimal Expo client for the local MVP flow (**Expo SDK 54**, совместим с актуальным **Expo Go** из Play Маркета).

## What works

- fetches templates and request status from the new backend
- records a voice note from the device microphone
- uploads the audio as part of a document request
- lets the user add manual notes/transcript text
- lets the user attach a minimal Bitrix24/email task command intent
- shows recent mobile jobs, uploaded source documents, and task command statuses

## What is still simplified

- no authentication yet
- no speech-to-text pipeline yet; the operator enters notes manually
- Bitrix24 is webhook-based only when `BITRIX_WEBHOOK_URL` is configured
- email approval is recorded as an approval flow/status, not sent by real SMTP

## Local start

Install dependencies and run Expo from the host machine:

```bash
cd apps/mobile-app
npm install
npm run start
```

Set the backend URL through `EXPO_PUBLIC_API_BASE_URL` when the device/emulator
cannot reach `localhost` directly.

Examples:

```powershell
$env:EXPO_PUBLIC_API_BASE_URL="http://10.0.2.2:8080"
npm run start
```

```powershell
$env:EXPO_PUBLIC_API_BASE_URL="http://192.168.1.50:8080"
npm run start
```

- Android emulator: usually `http://10.0.2.2:8080`
- iOS simulator / Expo web on same machine: usually `http://localhost:8080`
- physical device: use your host machine LAN IP

The Docker `mobile-workspace` service is optional and remains only as a helper
workspace container. The working MVP scenario is host-based Expo.

## Local end-to-end check

1. Start the backend/admin/infra stack from the repo root.
2. Confirm backend health JSON in the browser:
   - `http://localhost:8080/health`
   - or `http://localhost:8080/api/v1/health`
3. Make sure the mobile app points to the reachable backend URL:
   - Android emulator: usually `http://10.0.2.2:8080`
   - iOS simulator / Expo web on the same machine: usually `http://localhost:8080`
   - physical device: use your host LAN IP, for example `http://192.168.1.50:8080`
4. Start Expo and open the app on the device or emulator.
5. Verify the app shows backend status `ok` and loads available templates.
6. Select a template, record a short voice note, add request notes, and submit.
7. Open `http://localhost:5173` and verify that the new job, source document, processing events, and generated document appear.
8. Open Grafana at `http://localhost:3000` to confirm request and job metrics after the flow.
