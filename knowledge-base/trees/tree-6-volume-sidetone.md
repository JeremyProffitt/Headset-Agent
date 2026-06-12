---
section: trees
topic: tree_6
symptom: volume_sidetone
---

# Tree 6 — Volume Too Low / Too Loud / Sidetone

**Scope:** Diagnostic tree for volume problems and sidetone (hearing your own voice). Two different problems live here: (a) **call volume** (what the user hears of the other party), and (b) **sidetone** (how loudly the user hears *their own* voice). Branch first.

> Always run the **Universal Pre-Flight Checklist** (`preflight-checklist.md`) before entering this tree.

1. **Is this about the other person's volume, or hearing your own voice?**
   - **Other person too quiet/loud →** go to step 2.
   - **Hearing my own voice (too much, too little, or echo of myself) →** go to step 3 (sidetone).
2. **Call volume (the other party).** In order: adjust the **headset inline volume dial**; adjust **Windows volume** and the **Volume Mixer** for the app (see `windows/win-2.3-volume-mixer.md`); in the device's playback **Properties > Levels**, raise the **output level**; check **Loudness Equalization** if available for taming loud spikes. If too quiet only in calls, set the headset as the app's output and check the app's call volume. See `windows/win-2.3-volume-mixer.md` and `genesys/gc-4.2-device-selection.md`.
   - **If adjusting these fixed it → resolved.**
   - **If max volume is still too quiet on all apps → suspect hardware/driver;** update the driver (see `windows/win-2.7-drivers.md`), then go to `trees/escalation-criteria.md` if unresolved.
3. **Sidetone (hearing your own voice).** Clarify what the user wants:
   - **"I hear myself too loudly / it echoes back at me" →** Sidetone is too high. If the headset/vendor app has a **sidetone control** (e.g., Poly Lens, Logitech Logi Tune, Jabra Direct), lower it there — see brand docs in `brands/`. Then check the Windows culprit: in the **Recording** tab > headset mic > **Properties > Listen tab**, make sure **"Listen to this device" is UNCHECKED** (a frequent accidental cause of hearing yourself). See `windows/win-2.3-volume-mixer.md`.
   - **"I can't hear myself at all and it feels dead/muffled" →** Some users want *some* sidetone for comfort. Raise it in the **vendor app's sidetone setting** if the model supports it (see `brands/`). If the model has no sidetone feature, that's a hardware limitation — explain it's expected.
   - **If adjusting sidetone (vendor app) or the Listen toggle resolved it → resolved.**
   - **If the model has the feature but the control has no effect → escalate to brand-specific docs / Tier 2** (possible firmware/driver issue).
