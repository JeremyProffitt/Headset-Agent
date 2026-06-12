---
section: trees
topic: tree_2
symptom: microphone_not_working
---

# Tree 2 — Other Party Can't Hear Me / Microphone Not Working

**Scope:** Diagnostic tree for when the other party can't hear the user. This is the #1 contact-center complaint. The most common real causes, in order: **hardware mute, wrong mic selected, app mic permission off, another app holding the mic.**

> Always run the **Universal Pre-Flight Checklist** (`preflight-checklist.md`) before entering this tree.

1. **Hardware mute first** (it's the most common). Check the physical mute button / inline mute switch / mute-on-boom-up, and any mute LED. Have the user explicitly un-mute at the hardware.
   - **If un-muting fixed it → resolved.**
   - **If not muted →** continue.
2. **Is the headset selected as the Windows Input (recording) device?** See `windows/win-2.2-default-device.md`, set the headset mic as the default **Input** device — make sure it's not pointed at a webcam mic or laptop array mic.
   - **If selecting the right mic fixed it → resolved.**
   - **If already correct →** continue.
3. **Watch the input level meter.** In Windows Sound settings, have the user **speak** and watch the mic test/level bar move.
   - **If the bar moves → Windows hears the mic. The problem is in the softphone or its permissions.** Go to step 4.
   - **If the bar does NOT move →** continue to step 5 (Windows can't hear it).
4. **Softphone mic selection + permissions.** In the app's audio settings, set the **Microphone** to the headset. For **Genesys Cloud**: select the headset mic in the WebRTC phone, run **Test your microphone**, and confirm the **browser** has granted **microphone permission** to Genesys (the browser site-permission prompt/lock-icon). Confirm the user isn't **muted inside the call/in Genesys' call controls**. See `genesys/gc-4.2-device-selection.md` and `genesys/gc-4.3-browser-permissions.md`.
   - **If correcting the in-app mic / permission / in-call mute fixed it → resolved.**
   - **If still silent to the other party →** escalate to Genesys docs (WebRTC phone reset, browser switch, clear cache).
5. **Windows can't hear the mic.** In order: confirm **Settings > Privacy & security > Microphone** has **Microphone access ON** and **desktop/app access allowed** (see `windows/win-2.4-mic-privacy.md`); close every **other app that might hold the mic** (pre-flight #5); try a **different USB port**; **reboot**; then **update/reinstall the audio driver** (see `windows/win-2.7-drivers.md`). Note: some headsets expose a separate "hands-free/communications" device — make sure you selected the correct one (see brand-specific docs in `brands/`).
   - **If restored → resolved.**
   - **If the level meter still never moves after reboot + driver reinstall + a known-good USB port →** suspect a failed mic/boom. Go to `trees/escalation-criteria.md`.
