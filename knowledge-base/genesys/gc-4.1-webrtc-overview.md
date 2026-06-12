---
section: genesys
subsection: "4.1"
topic: webrtc_overview
application: genesys_cloud
---

# §4.1 How Genesys Cloud Uses WebRTC, the Browser, and Your Headset

**Scope:** Architecture overview of how Genesys Cloud's WebRTC phone interacts with the browser and USB headset — two-layer device selection model.

Genesys Cloud's WebRTC phone carries call audio directly through your **browser** using WebRTC and the **OPUS** audio codec — there is no desk phone. Genesys recommends a Chromium browser (**Google Chrome** or **Microsoft Edge**) or the **Genesys Cloud desktop app**. Firefox works but is limited (no Advanced Mic Settings, no Jabra/Poly call-control). Safari is not supported for headset call control.

## Two Layers of Device Selection

There are **two layers of device selection**, and both must point at your USB headset:

1. **Windows OS layer** — Windows decides which device is "Default." Genesys uses your Windows **default communication device** for incoming-call *alerts/ringtone* unless you tell it otherwise.
2. **Genesys WebRTC layer** — Inside Genesys you separately pick the **microphone** and **speaker** used for the actual *call conversation*, plus an optional separate **ringer** device.

They interact like this: the **ring** can come out of your computer speakers (Windows default) while the **call** goes to your headset — that is intentional and configurable. If audio is "wrong," it is almost always because one of these two layers is pointed at the wrong device.

**Verify which device Genesys is using:** open the WebRTC phone settings (see `gc-4.2-device-selection.md`) — the microphone/speaker dropdowns should show your headset by name (e.g., "Jabra Evolve2 65"), not "Default - Communications" or "Realtek."

## Browser Compatibility Summary

| Browser | Audio | Advanced Mic Settings | Jabra/Poly call control |
|---|---|---|---|
| Chrome | Yes | Yes | Yes (WebHID) |
| Edge | Yes | Yes | Yes (WebHID) |
| Firefox | Yes | No | No |
| Safari | Yes | No | No |
| Genesys desktop app | Yes | No | Yes |
