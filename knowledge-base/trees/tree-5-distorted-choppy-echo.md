---
section: trees
topic: tree_5
symptom: distorted_choppy_echo
---

# Tree 5 — Distorted, Choppy, Robotic, Crackling, or Echoing Audio

**Scope:** Diagnostic tree for audio quality problems. "Robotic/choppy" usually = **network or CPU** (WebRTC). "Crackling/static" usually = **USB/driver/port**. "Echo" usually = **acoustic loop or sidetone**. Branch on which one the user describes.

> Always run the **Universal Pre-Flight Checklist** (`preflight-checklist.md`) before entering this tree.

1. **Which best describes it?**
   - **Robotic / choppy / cutting in and out / "underwater" →** go to step 2 (network/CPU path).
   - **Crackling / static / popping / buzzing →** go to step 3 (USB/driver path).
   - **Echo (I or they hear a repeat of the voice) →** go to step 4 (echo path).
2. **Robotic/choppy (network/CPU).** Most often this is the connection, not the headset. In order: check the user's **network** (prefer **wired Ethernet over Wi‑Fi**, move off VPN if possible, close bandwidth-heavy apps/downloads); **close heavy apps** and extra browser tabs to free CPU; for **Genesys Cloud**, run **Test your media settings**, and try **restarting the desktop app or refreshing/switching the browser**. See `genesys/gc-4.5-diagnostics.md` and `genesys/gc-4.8-network-qos.md`.
   - **If switching to wired / freeing resources cleared it → resolved.**
   - **If it persists on a good wired connection → escalate to `genesys/` docs / Tier 2** (possible WebRTC, codec, or network-path issue beyond the desk).
3. **Crackling/static (USB/driver).** In order: **reseat directly into the PC** (off the dock/hub — a weak/over-loaded USB path causes crackle); try a **different USB port**; see `windows/win-2.5-enhancements-exclusive-samplerate.md` — try **disabling audio enhancements/effects** on the device and **lowering the sample rate** (e.g., 24-bit/48000 Hz) in the device's Advanced properties; **update/reinstall the audio driver** (see `windows/win-2.7-drivers.md`).
   - **If a port change / disabling enhancements / driver update cleared it → resolved.**
   - **If crackle persists on multiple ports after a driver reinstall → suspect hardware/cable.** Go to `trees/escalation-criteria.md` (RMA).
4. **Echo.** Determine **who hears the echo.**
   - **The user hears their *own* voice repeated back too strongly → that's sidetone.** Go to **Tree 6** (`tree-6-volume-sidetone.md`).
   - **The *other party* hears an echo → it's usually the *other* side's speakers/mic loop, but on the user's end:** confirm they're on a **headset (not open speakers)**, lower **speaker volume**, and ensure **echo cancellation isn't disabled**. In Genesys, confirm the correct headset mic+speaker are selected (so AEC works). See `genesys/gc-4.7-advanced-mic.md`.
   - **If isolating to a headset and normalizing volume cleared it → resolved.**
   - **If echo persists with a proper headset → escalate to `genesys/` docs / Tier 2.**
