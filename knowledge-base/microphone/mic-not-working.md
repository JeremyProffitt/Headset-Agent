---
section: microphone
topic: mic_not_working
platforms: [windows10, windows11]
---

# Microphone Not Working — Other Party Can't Hear You at All

**Scope:** The headset works for listening, but the microphone produces no input — the customer or agent on the other end hears nothing. This supports Tree 2, the #1 contact-center complaint.

## Likely Cause

In order of how often they happen: the headset is **muted at the hardware** (mute button, inline switch, or boom flipped up); Windows has the **wrong microphone selected** as the input device (a webcam mic or laptop array instead of the headset); Windows **microphone privacy settings** are blocking app access; or **another app is holding the mic**.

## Check

1. Look for a physical mute button, an inline mute switch on the cable, or "mute on boom-arm up." Check any mute LED — often red means muted.
2. Confirm the headset microphone is the selected **Input** device in Windows Sound settings, not a webcam or built-in mic.
3. Speak into the headset and watch the **input level meter** in Windows Sound settings. If the bar moves, Windows hears the mic and the problem is in the softphone or its permissions. If the bar never moves, Windows can't hear it.

## Steps

1. **Un-mute at the hardware.** Press the mute button or flip the inline switch, and rotate the mic boom down toward the mouth. A hardware mute overrides everything in software, and Windows often cannot see or override it.
2. **Select the headset mic as the input device.**
   - **Windows 11:** Start > Settings > System > Sound. Under **Input**, select the headset's microphone.
   - **Windows 10:** Start > Settings > System > Sound. Use the **Input** dropdown ("Choose your input device") and select the headset microphone.
   - To also set the **Default Communication Device** (which softphones follow), press Windows key + R, type **mmsys.cpl**, press Enter, go to the **Recording** tab, highlight the headset mic, and use the arrow next to **Set Default** to set both **Default Device** and **Default Communication Device**.
3. **Test:** speak and watch the input level bar move in Sound settings.
4. **If the bar moves but the other party still can't hear you:** fix the softphone side. Set the app's **Microphone** to the headset. For Genesys Cloud, select the headset mic in the WebRTC phone, run **Test your microphone**, confirm the **browser** has granted **microphone permission** to Genesys (the site-permission prompt or lock icon), and confirm you are not muted inside the call controls.
5. **If the bar does NOT move:** in order — confirm **Microphone access** is On in Settings > Privacy & security > Microphone (Windows 11) or Settings > Privacy > Microphone (Windows 10), including desktop app access; close every other app that might hold the mic (Teams, Zoom, Discord, other browser tabs in a call, voice recorders); plug the headset into a **different USB port directly on the computer** (not a dock, hub, or monitor); **reboot**; then update or reinstall the audio driver via Device Manager.
6. Note: some headsets expose a separate "hands-free" or "communications" device — make sure the correct one is selected.

## Verify

Speak into the headset and confirm the input level meter in Windows Sound settings moves as you talk. Then place a quick test call in the softphone and confirm the other party can hear you. If the level meter still never moves after a reboot, a driver reinstall, and a known-good USB port, suspect a failed mic or boom and escalate.
