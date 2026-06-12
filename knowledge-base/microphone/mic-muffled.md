---
section: microphone
topic: mic_muffled
platforms: [windows10, windows11]
---

# Microphone Sounds Muffled, Distorted, or Robotic

**Scope:** The other party can hear the caller, but the voice sounds muffled, garbled, robotic, crackly, or distorted. This supports Tree 2 and overlaps Tree 5 (audio quality). Rule of thumb from Tree 5: **robotic or choppy usually means network or CPU; crackling or static usually means USB, driver, or port.**

## Likely Cause

The mic boom is positioned poorly; a Windows audio **enhancement**, an **exclusive-mode** lock, or an unsupported **default format (sample rate)** is mangling the signal; the USB connection is going through a weak dock or hub; or — for robotic/choppy audio — the network or CPU is struggling, not the headset.

## Check

1. Confirm the mic boom is rotated **down toward the mouth** and the headset microphone (not a webcam or laptop mic) is the selected input device.
2. Ask which word fits best: **robotic/choppy/cutting out** points to network or CPU; **crackling/static/popping** points to USB, driver, or port; **muffled** points to mic position, device selection, or signal processing.
3. Confirm the headset is plugged **directly into a USB port on the computer** — not a monitor, keyboard, docking station, or unpowered hub.

## Steps

1. **Reseat the USB connector directly into the PC** and, if needed, try a different physical port. A weak or overloaded USB path causes crackle.
2. **Turn off audio enhancements.**
   - **Windows 11:** Start > Settings > System > Sound, click the headset device, scroll to **Audio enhancements**, and set it to **Off**.
   - **Windows 10:** press Windows key + R, type **mmsys.cpl**, press Enter. On the **Playback** tab, right-click the headset, choose **Properties**, open the **Enhancements** tab, check **Disable all enhancements** (or "Disable all sound effects"), and click **OK**.
3. **Turn off Exclusive Mode** (works on Windows 10 and 11): in **mmsys.cpl**, on the **Playback** tab, right-click the headset > **Properties** > **Advanced** tab, and **uncheck** both "Allow applications to take exclusive control of this device" and "Give exclusive mode applications priority." Click OK, then repeat on the **Recording** tab for the headset's microphone.
4. **Fix the default format (sample rate):** in **mmsys.cpl > Properties > Advanced**, set **Default Format** to a standard value such as **16 bit, 48000 Hz (DVD Quality)** or **24 bit, 48000 Hz**. Avoid unusually high rates. Do this on both the Playback and Recording tabs.
5. **If the voice is robotic or choppy:** treat it as network or CPU. Prefer **wired Ethernet over Wi-Fi**, move off VPN if possible, close bandwidth-heavy apps and extra browser tabs. In Genesys Cloud, run **Test your media settings**, and try restarting the desktop app or refreshing the browser.
6. **If using consumer Bluetooth earbuds** (such as AirPods): these are prone to WebRTC quirks — the mono "hands-free" mode kills audio quality. Prefer a wired USB or vendor-dongle headset.
7. **Still crackly?** Update or reinstall the audio driver: Device Manager > Sound, video and game controllers > right-click the headset > **Update driver** > **Search automatically for drivers**, or **Uninstall device** then restart so Windows reinstalls the in-box driver.

## Verify

Make a test call and confirm both directions sound clear, with no muffling, crackle, or robotic artifacts. If robotic/choppy audio persists on a good wired connection, escalate (possible WebRTC, codec, or network-path issue beyond the desk). If crackle persists on multiple ports after a driver reinstall, suspect hardware or cable and escalate for replacement.
