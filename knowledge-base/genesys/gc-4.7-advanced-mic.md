---
section: genesys
subsection: "4.7"
topic: advanced_mic_settings
application: genesys_cloud
platforms: [chrome]
---

# §4.7 Known Limitations and Advanced Mic Tuning

**Scope:** Genesys Cloud known headset limitations and Chrome-only Advanced Mic Settings (AGC, echo cancellation, noise suppression).

## Known Limitations

- **AirPods / consumer Bluetooth earbuds:** Not in the supported call-control list and prone to WebRTC quirks (mono "hands-free" mode kills audio quality). Prefer a wired USB or vendor-dongle headset.
- **Desktop app vs Chrome differences:** **Advanced Mic Settings only appears in Chrome.** If a setting you read about isn't visible, you may be in the desktop app or a non-Chrome browser.

## Advanced Mic Settings (Chrome Only)

Navigate to: **Menu > More > Settings > WebRTC > Advanced Mic Settings**

Clear (disable) these only to fix a specific symptom:

| Setting | When to Disable |
|---|---|
| **Automatic Mic Gain (AGC)** | Disable if mic volume keeps fluctuating up and down |
| **Echo Cancellation** | Disable only if you don't use open speakers (pointless in a headset-only setup) |
| **Noise Suppression** | Before disabling, first **move the mic boom closer and speak up** — over-aggressive suppression can clip a quiet voice |

## General Cleanup Steps

- Close other apps that grab the mic (Teams, Zoom, Webex).
- Select your headset *by name* (not "Default").
- Use **Refresh** after plugging in.
