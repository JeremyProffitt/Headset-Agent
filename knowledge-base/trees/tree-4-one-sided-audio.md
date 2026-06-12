---
section: trees
topic: tree_4
symptom: one_sided_mono_audio
---

# Tree 4 — One-Sided Audio / Only One Ear / Mono

**Scope:** Diagnostic tree for when sound is only coming from one ear, or audio is unexpectedly mono.

> Always run the **Universal Pre-Flight Checklist** (`preflight-checklist.md`) before entering this tree.

1. **Check the Windows balance.** See `windows/win-2.2-default-device.md`. Open the headset's playback **Properties > Levels > Balance** and confirm **left and right are equal** (both not at 0). A skewed balance slider is the most common cause.
   - **If balancing the channels fixed it → resolved.**
   - **If already balanced →** continue.
2. **Check the "Mono audio" accessibility toggle.** In **Settings > Accessibility > Audio**, confirm **Mono audio is OFF** (unless the user needs it on for an accessibility reason — if so, that's expected behavior, not a fault).
   - **If turning off Mono audio fixed it → resolved.**
   - **If already off →** continue.
3. **Isolate hardware vs. signal.** Play a known **stereo** test (e.g., a left/right channel test clip) outside the softphone.
   - **If both ears work on the stereo test → the source/call audio was mono or the app is mono — expected for a phone call (voice calls are typically mono).** Reassure the user; if it only happens in one app, check that app's audio settings.
   - **If still one-sided on a stereo test →** continue (hardware suspected).
4. **Wiggle test / cable + connector.** Have the user gently move the cable near the plug and the earcup joint while audio plays. Reseat the USB. Try a different USB port.
   - **If audio cuts in/out with movement → a failing cable/connector.** Go to `trees/escalation-criteria.md` (RMA).
   - **If consistently dead in one ear regardless of balance, Mono toggle, source, and port → a failed driver/speaker in that earcup.** Go to `trees/escalation-criteria.md` (RMA).
