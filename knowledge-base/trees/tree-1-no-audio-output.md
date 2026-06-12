---
section: trees
topic: tree_1
symptom: no_audio_output
---

# Tree 1 — No Audio Output / Can't Hear Anything

**Scope:** Diagnostic tree for when the user reports no sound in their headset. Work from "is it even selected and turned up" outward.

> Always run the **Universal Pre-Flight Checklist** (`preflight-checklist.md`) before entering this tree.

1. **Confirm the headset is plugged directly in and powered/recognized** (pre-flight #1). Did Windows make a connect chime, or does the headset show up?
   - **If NO →** go to **Tree 3 (Not Detected)** — you have a detection problem, not an output problem.
   - **If YES →** continue.
2. **Is the headset set as the Windows output (playback) device?** See `windows/win-2.2-default-device.md`, set the headset as the default **Output** device.
   - **If it was wrong and fixing it restored sound → resolved.**
   - **If already correct →** continue.
3. **Volume and mute check.** Confirm Windows volume is up and not muted; confirm the **inline/headset volume dial** is up; confirm no hardware mute. Open the **Windows Volume Mixer** and confirm the app (browser/Genesys) isn't muted or at zero.
   - **If a volume/mute control was the cause → resolved.**
   - **If all up and unmuted →** continue.
4. **Quick playback test outside the softphone.** Have the user play any sound (a video clip, the Windows "Test" button on the device's properties, or a system sound).
   - **If they hear the test sound → the headset/Windows side is fine. The problem is inside the softphone.** Go to step 5.
   - **If they hear NOTHING even in the test →** the problem is at the Windows/hardware layer. Go to step 6.
5. **Softphone device selection.** In the app's audio settings, confirm the **output/speaker** is set to the headset (not "Default" pointing elsewhere, not speakers). For **Genesys Cloud**, set the WebRTC phone's **Speaker** to the headset and use **Test your media settings** to play a test tone. See `genesys/gc-4.2-device-selection.md`.
   - **If selecting the headset restored call audio → resolved.**
   - **If still silent in calls only →** escalate to `genesys/` docs (WebRTC phone / browser permissions / desktop app reset).
6. **Windows-layer remediation** (no sound anywhere). In order: run the **Windows Audio troubleshooter**; check that the audio device isn't **Disabled** in Sound > Manage sound devices; try a **different USB port**; **reboot**; then **update/reinstall the audio driver**. Full steps in `windows/win-2.6-audio-services.md` and `windows/win-2.7-drivers.md`.
   - **If restored → resolved.**
   - **If still no sound on any app after a reboot and driver reinstall →** suspect hardware. Go to `trees/escalation-criteria.md`.
