---
section: windows
subsection: "2.3"
topic: volume_mixer
platforms: [windows10, windows11]
---

# §2.3 Volume Mixer and Per-App Volume — Software Mute vs Hardware Mute

**Scope:** How to check and correct volume levels in the Windows Volume Mixer, including per-app volume and the difference between hardware and software mute.

**Likely cause:** The correct device is selected, but audio is muted or turned all the way down — either system-wide, for that one app, or by a physical button on the headset.

**What to check:** The per-app **Volume Mixer**, the system volume, and any inline mute switch/button on the headset cord or earcup.

## Exact Steps

1. **Hardware first:** Many call-center headsets have an **inline mute button or a mic boom that mutes when flipped up**. Ask the caller to check for a physical mute switch and a mute LED, and to confirm the mic boom (if any) is rotated **down** toward the mouth. A hardware mute overrides everything in software, and Windows often **cannot** see or override it.
2. **System volume:** Click the **speaker icon** in the taskbar and make sure the slider isn't at zero and the speaker isn't showing a "muted" (crossed-out) symbol.
3. **Per-app (Volume Mixer):**
   - **Win11:** **Start > Settings > System > Sound > Volume mixer.** Confirm the specific app (the softphone/browser) is **not muted** and its slider is up, **and** that the app's output is pointed at the headset (each app row lets you pick its own output device in Win11).
   - **Win10:** right-click the **speaker icon** in the taskbar > **Open Volume Mixer**, and check each app's column.

## Additional: Sidetone / "Listen to this device"

If the user complains of hearing their own voice too loudly, check the Recording tab in **mmsys.cpl**: right-click the headset microphone > **Properties** > **Listen** tab. Make sure **"Listen to this device" is UNCHECKED**. When this is accidentally enabled, the mic audio is looped back to the speakers/headset in real time — classic cause of "I can hear myself."

## How to Verify It's Fixed

With the app playing audio, its bar in Volume Mixer moves and sound is audible in the headset at a comfortable level, with no crossed-out/muted icons.
