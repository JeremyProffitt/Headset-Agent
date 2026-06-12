---
section: windows
subsection: "2.5"
topic: enhancements_exclusive_samplerate
platforms: [windows10, windows11]
---

# §2.5 Audio Enhancements, Exclusive Mode, and Sample-Rate/Format Mismatches

**Scope:** How to disable audio enhancements, exclusive-mode locks, and fix sample-rate mismatches that cause distortion, crackling, robotic audio, or "can't open device" errors.

**Likely cause:** A signal-processing "enhancement," an "exclusive control" setting, or a default audio format the headset doesn't support — causing distortion, crackling, robotic/garbled audio, intermittent dropouts, or an app that "can't open the device."

## Exact Steps — Turn Off Enhancements

- **Win11:** **Start > Settings > System > Sound** > click the headset (output device) > scroll to **Audio enhancements** and set it to **Off.**
- **Win10:** open **mmsys.cpl** > **Playback** tab > right-click the headset > **Properties** > **Enhancements** tab > check **Disable all enhancements** (or "Disable all sound effects") > **OK.**

## Exact Steps — Exclusive Mode (The "An App Grabbed the Device" Fix)

1. Open **mmsys.cpl** (Windows key + R > **mmsys.cpl** > Enter).
2. **Playback** tab > right-click the headset > **Properties** > **Advanced** tab.
3. Under **Exclusive Mode**, **uncheck** both:
   - **Allow applications to take exclusive control of this device**
   - **Give exclusive mode applications priority**
4. Click **OK**. Repeat on the **Recording** tab for the headset's microphone.

> Why: when exclusive mode is on, one app can seize the headset and lock every other app out — so the softphone goes silent the moment another program (a media player, a recorder, a meeting app) is using audio. Clearing these checkboxes lets apps share the device.

## Exact Steps — Fix the Default Format (Sample Rate) for Distortion / No Audio

1. In **mmsys.cpl > Playback > [headset] > Properties > Advanced**, find the **Default Format** dropdown.
2. Set it to a standard, widely supported value: **16 bit, 48000 Hz (DVD Quality)** or **24 bit, 48000 Hz.** Avoid unusually high rates.
3. Click **Apply** > **Test** (Windows plays a tone) and confirm it's clean. Do the same on the **Recording** tab for the mic.

## How to Verify It's Fixed

Play a test tone or short clip — no crackle/distortion — and make a test call where both directions sound clear. If a specific app previously errored with "couldn't open the audio device," it should now connect.
