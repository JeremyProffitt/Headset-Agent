---
section: windows
subsection: "2.7"
topic: drivers
platforms: [windows10, windows11]
---

# §2.7 Drivers — Generic USB Audio Class Driver vs. Vendor Driver, Updating and Reinstalling

**Scope:** How to understand, update, and reinstall audio drivers for USB headsets — including when to use the in-box Windows driver vs. a vendor-specific driver.

**Likely cause:** A missing, corrupt, or mismatched driver (often the source of a Code 28 / yellow triangle, or of missing vendor features like a busylight or sidetone).

## Background

- Most USB headsets are **"class-compliant"** and work **plug-and-play** using a driver that **ships inside Windows** — no download needed. Windows includes **usbaudio.sys** (USB Audio 1.0) and, **starting with Windows 10 version 1703, usbaudio2.sys** (USB Audio 2.0). A class-compliant headset shows up automatically as something like **"USB Audio Class 2 Device"** (replaced by the product name when the device reports one).
- A **vendor driver** (from the headset maker) provides extra features and can fix model-specific bugs. **If a vendor/partner driver is present on the PC or offered via Windows Update, Windows installs it and it overrides the in-box class driver.** So for plain "no audio" issues the in-box driver is usually fine; install the vendor driver when you need the vendor's features or the maker's support team recommends it.

## Update the Driver via Device Manager

1. Open **Device Manager**.
2. Expand **Sound, video and game controllers**, right-click the headset > **Update driver** > **Search automatically for drivers.**
3. Follow the prompts and **restart** the PC if asked.

## Reinstall the Driver (Clears a Corrupt Driver / Code 28)

1. In Device Manager, right-click the headset > **Uninstall device.** If **"Attempt to remove the driver for this device"** appears, **check it.**
2. **Restart the PC** (and re-plug the headset). Windows automatically reinstalls the in-box USB Audio driver on the next connection.
3. If it doesn't reappear, click **Action > Scan for hardware changes.**

## Windows Update Optional Driver Updates

1. **Start > Settings > Windows Update > Advanced options.**
2. Under **Additional options**, select **Optional updates.**
3. Expand **Driver updates.** If a headset/audio/USB driver is listed, check it and click **Download & install.**

> **Caution to convey:** Optional driver updates are **not** installed automatically and **shouldn't** be installed unless there's a specific problem with that device — a wrong optional driver can introduce new issues. If the vendor publishes its own installer/app, that's usually the better source for vendor-specific drivers.

## Vendor Driver/App

If features like a busy-light, hook-switch/call-control buttons, or sidetone don't work, install the manufacturer's headset app/driver from the maker's official support site, then reboot. See brand-specific docs in `brands/`.

## How to Verify It's Fixed

Device Manager shows the headset with **no warning icon** and status **"This device is working properly"**; audio plays and records; and (if a vendor app was installed) the headset's special buttons/lights work in the softphone.
