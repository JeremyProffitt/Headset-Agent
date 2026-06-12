---
section: brands
brand: jabra
products: [Evolve, Evolve2, Engage, Link 380, Link 390]
---

# Jabra USB Headset Troubleshooting (Evolve / Evolve2, Engage; Jabra Direct / Xpress; Link 380/390)

**Scope:** Jabra-specific diagnosis for USB connection, call-control, mute sync, firmware, sidetone, and busylight issues. Identify the **brand and exact model** first (look on the headset, the boom arm, or the USB dongle).

**Companion app — Jabra Direct** (end-user desktop app; download from jabra.com). It runs in the background and bridges the headset's buttons to the PC softphone. It handles firmware updates (headset **and** dongle), device settings (sidetone, audio bandwidth, busylight, button mapping), call control, and **preferred-softphone** selection. **Jabra Xpress** is the IT mass-deployment counterpart (builds an MSI for SCCM rollout, can *lock* settings so they appear greyed-out in Direct on the user's PC). If a user can't change a setting in Direct, IT may have locked it via Xpress.

## Firmware Update (Jabra Direct)

Connect device → select it in Direct → click **Update now** → **Update** next to the device → pick language → **Update** → do **not** disconnect until done. For wireless models, the **Link 380/390 dongle has its own firmware** — update both; a headset/dongle firmware mismatch is a top cause of mute-sync and Teams-button failures.

## Common USB Issues

**Device not detected.**
- *Steps:* move the dongle to a **different port** (older Jabra dongles like the Link 370 are flaky in USB 3.x — try a **USB 2.0** port); reinstall Jabra Direct; in Device Manager uninstall the Jabra device and reboot.
- *Verify:* device shows in Direct and in Windows Sound.

**Call-control buttons do nothing.**
- *Check:* Direct → **Device settings → Softphone (PC)**.
- *Steps:* turn **Call control with softphone** ON; set **Preferred softphone** to the app in use; also select Jabra as the audio device *inside* the softphone.
- *Verify:* pressing answer/end on the headset drives the call.
- *Note:* the **Engage 50/50 II headset has no on-headset buttons** — call control lives on the optional **Engage Link** controller or the softphone UI.

**Mute LED out of sync with Teams.**
- *Steps:* update Teams + Jabra firmware/Direct (headset **and** dongle); in Teams **Settings → Devices** toggle device-button sync off/on; restart Teams; if persistent, factory-reset the headset (typically hold Mute ~10 s until the LED flashes).
- *Verify:* headset mute and Teams mute icon move together.

**Sidetone.**
- Direct → **Device settings → Headset → Sidetone**; default 0 dB, raise to +3/+6 dB to hear more of yourself.

**Busylight not reflecting status.**
- Direct → **Device settings → Headset → Headset busylight** (on), and **Settings → Softphone Integration → Presence Synchronization** ON (softphone must support Jabra integration).

## Gotchas

- **MS vs UC variant.** Evolve2/Engage and the dongles ship in **MS (Teams-certified)** and **UC** SKUs. MS = auto-default device, dedicated **Teams button**, Teams LED notifications. **UC = no Teams button/LED with Teams**, but certified for Cisco/Mitel/Avaya/Zoom. The variant is firmware-baked and **not field-switchable** — order the right SKU.
- **Link 380 vs 390.** Both are USB Bluetooth dongles managed in Direct. **390** is newer (Bluetooth 5.3, HD Voice). Each comes as **a (USB-A)** or **c (USB-C)**, in MS and UC — e.g. 380a, 390c. Match the laptop's port.
- **Replacement dongle must be re-paired** to the headset in Jabra Direct, and its variant (MS/UC) and connector (A/C) must match.
- **Launch order matters:** start Jabra Direct *before* the softphone. If Direct loads after the softphone, call control can break; restart Direct first.

## Genesys Cloud with Jabra

Jabra call control in Genesys requires Chrome/Edge, the desktop app, or embedded clients. Connect by **wired USB or a Jabra USB Bluetooth adapter/dongle**. On first use, approve the prompts and pick the device in Chrome's **WebHID "Connect"** dialog, confirm it as default device, and save a profile. **Not** supported on Firefox/Safari. See `genesys/gc-4.6-call-control.md`.
