---
section: brands
topic: cross_manufacturer
applies_to: all_brands
---

# Cross-Manufacturer Topics — Call Control, HID, Firmware, and Docks

**Scope:** Topics that apply across all headset brands: call-control/HID troubleshooting, firmware update best practices, and docking-station/USB-hub gotchas.

## Call-Control / HID — Answer/End/Mute/Volume Not Working

**Mental model:** the headset only emits a raw HID event ("hook off," "mute toggle"). Something on the PC must translate it into a softphone action — either the **softphone itself** (Teams/Zoom/Webex natively, for certified devices) or the **vendor companion app** acting as a bridge. Buttons fail when the bridge is missing, misconfigured, losing an arbitration fight, or the wrong device variant is in use.

**Diagnostic checklist (in order):**
1. **Install and run the companion app** — and where it matters (Jabra especially), **launch the app *before* the softphone**. If Jabra Direct loads after the softphone, call control can break; restart Direct first.
2. **Enable softphone integration + mute sync** in the app, and **select the correct target/preferred softphone** (wrong target = "button does nothing" or "answer hangs up").
3. **Set the headset as the default communication device** both in **Windows Sound** and **inside the softphone** (Settings → Devices/Audio).
4. **Enable HID / "use headset buttons" in the softphone.** In **Teams** this is **Settings → Devices → "Sync device buttons."** Turn it **OFF** when you need another UC app to co-exist with the headset's HID; ON to let Teams sync the buttons.
5. **Match MS vs UC variant to the platform.**
6. **Resolve multi-app HID conflicts.** Use Teams' **"Sync device buttons"** toggle to co-exist; for **Teams + Skype for Business 2016 (Islands mode)** apply the **EnableTeamsHIDInterop** Group Policy (`HKCU\Software\Policies\Microsoft\Office\16.0\Lync\EnableTeamsHIDInterop = 1`). Note Teams' mute-sync can **override a physical mute after a few seconds** — a known "my mute un-mutes itself" cause.
7. **Platform-specific limits exist** — e.g. Five9 may not honor a headset's HID mute. Clean-reinstall the companion app if buttons silently stop after an app update.

## Microsoft / Generic Certified USB-HID Headsets

**Why some buttons "just work" with no app at all — USB HID.** A standard USB headset enumerates as USB Audio Class (no driver) plus a **HID interface**. Telephony controls live on the **HID Telephony page (0x0B)** and **Consumer page (0x0C)** — hook switch (off-hook = answer, on-hook = end), mute toggle, volume. Windows/macOS understand these natively, so on a well-implemented device **answer/end/mute and the mute LED work without any vendor software**. Vendor apps add firmware, sidetone, EQ, and busylight config — not basic call control.

**What Microsoft Teams certification guarantees:** plug-and-play with no config, auto-selection as default device, basic call control (answer/hang-up, mute, volume) on Windows and Mac, and firmware-update capability. **New-Teams** certification adds a dedicated **Teams button + LED**. **Critical gotcha (verbatim):** *"Teams doesn't support button controls on connected certified peripherals if third-party collaboration and conferencing apps are running at the same time."*

**When there's no companion app** (e.g. basic Microsoft Modern USB Headset, OEM units): call control still works via the OS HID stack, but you can't tune sidetone/EQ/busylight/firmware. If a generic device's buttons don't work, it's almost always an unsupported HID usage in that specific app or an app conflict — **not** a missing driver.

## Firmware Update Best Practices and Risks

Firmware writes to the device's flash; **interrupting it can brick the device or force a vendor recovery/reflash.** Manufacturer guidance converges on:
1. **Never disconnect/unplug** the headset, dongle, or base mid-update.
2. **Plug directly into a PC USB port** — not a hub, dock, monitor-USB, or KVM.
3. **Close softphone/conferencing apps** during the update.
4. **Wireless/DECT/dongle headsets: keep them docked and charged** for the whole update; don't undock.
5. **Don't update across a KVM or virtual-desktop USB redirection** — the device re-enumerates and drops.
6. **Don't close the companion app or sleep/shut down** until it reports success.

## Docking Stations / USB Hubs / Monitors-with-USB — The #1 Cause of Intermittent USB Audio

USB audio is **isochronous and power-sensitive**, so it's unusually intolerant of the bandwidth contention, power limits, and re-enumeration churn that docks and hubs add. Classic symptoms: audio **stutters/drops on a ~30–60 second cycle**, the headset **disappears and won't re-enumerate** after a dock power-sequence, or resource-allocation errors from insufficient bus power.

**Practical rule:** For any intermittent USB-audio dropout, re-enumeration, or "headset disappears," **move the headset or its dongle to a USB port directly on the PC** (prefer a rear/native USB 3.x port), bypassing docks, monitor-USB ports, hubs, and KVMs. Direct connection is both the diagnostic test and, very often, the fix. If the dock must stay: try a **powered** hub, update the **dock firmware**, and check **USB selective-suspend / port power-management** settings (see `windows/win-2.6-audio-services.md`).
