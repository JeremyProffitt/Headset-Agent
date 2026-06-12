---
section: windows
subsection: "2.2"
topic: default_device_selection
platforms: [windows10, windows11]
---

# §2.2 Select the Headset as the Default Playback, Communication, AND Recording Device

**Scope:** How to set the USB headset as the Windows default for both general audio and communication (VoIP) roles on both playback and recording.

**Likely cause:** Windows is still routing audio to the laptop speakers / built-in mic, or the softphone is grabbing a different "communication" device than the one the caller expects.

**Why this matters (read this to yourself, not the caller):** Windows keeps **two** separate default-device roles:
- **Default Device** — used by general apps (media players, browsers, music).
- **Default Communication Device** — used specifically by apps that detect a phone/VoIP call (softphones, Teams, Zoom, many call-center clients).

A softphone will often follow the **Default Communication Device**, so if that role still points at the built-in speaker or mic, the caller hears nothing in the headset (or the customer can't hear them) **even though the headset looks selected** for everything else. **Set the headset for both roles** to be safe.

## Exact Steps — Windows 11 (Settings)

1. **Start > Settings > System > Sound.**
2. Under **Output**, select the headset to make it the default playback device.
3. Under **Input**, select the headset's microphone as the default recording device.

## Exact Steps — Legacy Sound Control Panel (Works on Win10 and Win11; Only Place to Split "Device" vs "Communication")

1. Press **Windows key + R**, type **mmsys.cpl**, press **Enter**. (Or, Win11: **Settings > System > Sound > Advanced > More sound settings**. Win10: right-click the speaker icon in the taskbar > **Sounds** > **Playback** tab.)
2. On the **Playback** tab, click the headset once to highlight it.
3. Click the **small arrow next to the "Set Default" button** and choose **Default Device**. Then click the arrow again and choose **Default Communication Device**. (When one device holds both roles you'll see a green check **and** a green phone icon on it.)
4. Switch to the **Recording** tab and do the same for the headset's microphone: set it as both **Default Device** and **Default Communication Device**.
5. Click **OK**.

**(Win10):** Settings path is **Start > Settings > System > Sound**, with **Output** and **Input** dropdowns instead of the radio-button list; the **mmsys.cpl** control panel is identical.

## How to Verify It's Fixed

In **mmsys.cpl > Playback**, the headset shows both the green check and the green phone icon. As a live test, play any sound — it should come out of the headset. For the mic, see `win-2.4-mic-privacy.md` verification step.
