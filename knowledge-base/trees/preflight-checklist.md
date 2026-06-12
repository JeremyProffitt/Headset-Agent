---
section: trees
topic: preflight
applies_to: all_symptoms
---

# Universal Pre-Flight Checklist

**Scope:** Run this checklist before entering any diagnostic tree. It resolves the large majority of headset tickets in under two minutes.

Read each item aloud and confirm with the user before moving on.

1. **Reseat the USB connector.** Unplug the headset and plug it **directly into a USB port on the computer itself** — not into a monitor, keyboard, docking station, or USB hub. (Docks and hubs are the #1 cause of "it worked yesterday" issues.) Wait ~10 seconds for Windows to chime/recognize it.
2. **Confirm the headset is the selected/default device** in both **Windows Sound settings** and **the softphone's own device picker**. Many machines silently default to laptop speakers or a webcam mic.
3. **Check the hardware mute.** Look for a physical mute button, an inline mute switch on the cable, or "mute on boom-arm up" — confirm the headset is **not** muted at the hardware level. Many headsets have a mute LED (often red = muted).
4. **Turn the volume up** — both on the headset's inline volume control/dial **and** in Windows. Confirm the Windows volume mixer for the app isn't at zero.
5. **Make sure only one app is using the microphone.** Close Teams, Zoom, Discord, other browser tabs in a call, voice recorders, etc. Two apps fighting for the mic is a top cause of "no mic" and "device busy."
6. **Reboot if needed.** If anything above looked wrong or the user has been fighting this for a while, a full restart (then re-plug the headset) clears most stuck-driver and stuck-device states. In Genesys, also try **refreshing the browser tab** or **restarting the desktop app**.

If the pre-flight resolved it, **stop here** — no tree needed. If not, use the Symptom-to-Tree Quick Index (see `common/symptom-index.md`) to jump to the matching tree.
