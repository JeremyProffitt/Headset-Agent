---
section: brands
brand: yealink
products: [UH series, WH series]
---

# Yealink USB Headset Troubleshooting (UH / WH Series; Yealink USB Connect; DECT Bases)

**Scope:** Yealink-specific diagnosis for USB connection, call-control, DECT pairing, firmware, sidetone, and busylight issues.

**Companion app — Yealink USB Connect** (Windows/Mac). Manages firmware, device status/battery, **sidetone**, EQ, busylight, button functions, and call control (mute + hook-state sync). **Hard limit: USB Connect only works over a direct USB connection** — a Bluetooth-only connection is invisible to it. For IT at scale: **YMCS** (cloud) or **YDMP** (on-prem).

## Firmware Update (USB Connect)

**Wired UH / BT-via-dongle:** connect via USB → open USB Connect → select device → **Update device → Update now**.

**DECT WH6x (two-stage):** connect the **base** to power + PC USB → **dock the headset on the base** → **Update device → Update Now** (up to ~5 min). The **base updates first, then the docked headset**. Don't unplug or undock mid-update. **DECT base and headset have separate firmware** — a "Teams button won't connect" symptom is often a stale **base** that needs updating and re-docking. (Desk-phone use needs phone firmware **≥ 86**.)

## Common USB Issues

**Not detected.**
- Try a different port; on DECT bases note **two micro-USB ports** (one PC, one desk phone — use the right one); power-cycle the base (unplug power 10 s); update USB Connect; set as default in Windows Sound.

**Call control / Teams button dead (audio fine).**
- *Likely cause:* a **conflicting softphone/legacy client running alongside Teams.**
- Microsoft: *"Teams doesn't support button controls on connected certified peripherals if third-party collaboration and conferencing apps are running at the same time."*
- **Uninstall the conflicting/old client, reboot.**
- Also update base + headset firmware; ensure the UC↔Teams platform switch in USB Connect (UC SKU only) matches the app.

**Sidetone:** raise it in **Yealink USB Connect** (the only place to set it).

## DECT Pairing / Subscription

- Pair by **docking the headset** (LED green ~5 s).
- Subscription **persists** through power-off/undock; clearing it needs a factory reset.
- Add up to **4 headsets** per base (hold the base **PC button ~5 s**, then press the primary headset's call-control button).
- Factory reset: hold **Computer + Desk Phone buttons together 6 s**, or via USB Connect → **Device Support → Restore factory settings**.

## Busylight

Built-in (UH38) or external **BLT60** for WH6x — it shows **one** device's presence, so set the intended app as the default audio/dialer device.

## Gotchas

- Teams vs UC SKUs differ in hardware/firmware.
- USB Connect is required for firmware/tuning and only over USB — Bluetooth connections are invisible to it.
- Two micro-USB ports on DECT bases: one for PC, one for desk phone — use the PC port for USB Connect management.

## Genesys Cloud with Yealink

Yealink call control works in Genesys Cloud. See `genesys/gc-4.6-call-control.md` for supported vendors and setup.
