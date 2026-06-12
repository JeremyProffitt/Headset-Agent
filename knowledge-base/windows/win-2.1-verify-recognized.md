---
section: windows
subsection: "2.1"
topic: verify_usb_recognized
platforms: [windows10, windows11]
---

# §2.1 Verify Windows Actually Recognizes the USB Headset

**Scope:** How to confirm the headset is enumerated on the USB bus, appears in Device Manager, and shows up as both a playback and recording device.

**Likely cause:** The headset isn't enumerating on the USB bus — bad port, unpowered hub, or a missing/failed driver — so no app can use it.

**What to check:** Whether the headset appears in Device Manager without a warning icon, and whether it shows up as both a playback and a recording device.

## Exact Steps

1. Ask the caller to unplug the headset and plug it **directly into a USB port on the computer itself** — not into a monitor, keyboard, docking station, or unpowered USB hub. Many headsets draw more power than a passive hub can supply, which causes them to drop out or fail to enumerate.
2. Prefer a port on the **back of a desktop** (those are wired directly to the motherboard). If the first port doesn't work, **try a different physical port.**
3. **USB 2.0 vs 3.0:** A USB headset is a low-bandwidth device and works fine in either a USB 2.0 (usually black) or USB 3.0/3.1 (usually blue) port. If one port type misbehaves, have them try the other — occasionally a flaky USB 3.0 controller or a power-saving setting on a 3.0 port causes audio dropouts.
4. Open **Device Manager**: in the taskbar search box type **device manager** and select it (or press **Windows key + X** and choose **Device Manager**).
5. Expand **Sound, video and game controllers** — the headset (or "USB Audio" / "USB Audio Class 2 Device") should be listed there.
6. Also expand **Audio inputs and outputs** — you should see the headset listed as both a **speaker/headphone** and a **microphone**.

## Reading the Icons

- **A yellow triangle with an exclamation mark** on the device = Windows sees the hardware but the driver isn't working. Right-click the device > **Properties** > the **General** tab shows a "device status" message and an error **Code**. **Code 28** specifically means *"The drivers for this device are not installed"* — Windows can't find a usable driver. (See driver steps in `win-2.7-drivers.md`.)
- **An "Unknown device" or "Unknown USB Device" entry** (often under **Other devices** or **Universal Serial Bus controllers**, sometimes "Port Reset Failed") = the headset failed to identify itself on the bus. This usually points to a bad cable, a bad/underpowered port, or a hardware fault — not a software setting.

## First Fixes for a Yellow Triangle / Unknown Device

1. Right-click the problem device > **Uninstall device**. If a checkbox **"Attempt to remove the driver for this device"** appears, leave it **checked**.
2. **Unplug the headset, restart the PC**, then plug the headset back in. Windows re-enumerates and reinstalls the in-box driver.
3. If it still doesn't appear, in Device Manager click the **Action** menu > **Scan for hardware changes**.

## How to Verify It's Fixed

The headset appears under both **Sound, video and game controllers** and **Audio inputs and outputs** with **no yellow triangle**, and the device status reads **"This device is working properly."**
