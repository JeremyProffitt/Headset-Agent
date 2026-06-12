---
section: genesys
subsection: "4.2"
topic: device_selection
application: genesys_cloud
---

# §4.2 Selecting Microphone, Speaker, and Ringer in Genesys Cloud

**Scope:** How to configure the microphone, speaker, and ringer device selections inside the Genesys Cloud WebRTC phone settings.

**Likely cause:** Genesys is sending/receiving audio on the laptop's built-in mic/speakers instead of the USB headset.

## Exact Steps (Chrome or Edge)

1. In the Genesys Cloud client, go to **Menu (top-right) > More > Settings > WebRTC**. (Or, from the popped-out WebRTC Phone window: click the **Arrow > Open WebRTC Settings > Settings**.)
2. Under **Audio Controls**, set **Microphone** to your headset by name.
3. Set **Speaker** to your headset by name.
4. Click **Refresh** next to the device list if your headset just got plugged in or is missing — Refresh also restarts the headset vendor software (Jabra Direct / Plantronics Hub).
5. Click **Speaker** (blue speaker icon) to play test tones — you should hear them in the headset.

## Ringer / "Play Ringtone on Separate Device"

To make the **ring play on computer speakers while the call stays in the headset**: enable **Play ringtone on separate device** and select your computer speakers; leave the call **Speaker** set to the headset.

## Device Profiles

Once you select a headset and (for call control) approve it, Genesys saves a named **profile** and **automatically re-recognizes that headset on future connections** — so you don't reconfigure every shift. If you swap headsets you may need to pick the new device and create a new profile.

## How to Verify

Run a station/test call (see `gc-4.5-diagnostics.md`). You should hear the test tone in the headset and see your mic level move when you speak.
