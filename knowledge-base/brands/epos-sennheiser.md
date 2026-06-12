---
section: brands
brand: epos_sennheiser
products: [ADAPT, IMPACT, BTD 800]
---

# EPOS / Sennheiser USB Headset Troubleshooting (ADAPT / IMPACT; EPOS Connect; BTD 800)

**Scope:** EPOS/Sennheiser-specific diagnosis for USB connection, call-control, firmware, sidetone, busylight, and dongle pairing issues.

**Brand history (the gotcha):** Sennheiser Communications' enterprise line became **EPOS** in 2020; older units may still say "Sennheiser." The old **HeadSetup / HeadSetup Pro** software was merged into **EPOS Connect** (desktop) and **EPOS Manager** (cloud/IT). **Use EPOS Connect for all current ADAPT/IMPACT gear; uninstall legacy HeadSetup Pro so two clients don't fight over the device.**

**Companion app — EPOS Connect** (Windows/Mac, plus mobile, plus an **EPOS Connect for Web** that does firmware updates without installing software). Handles firmware, ANC/sidetone, **call control** (answer/end/volume/mute with mute sync), **default-softphone** selection, and busylight.

## Firmware Update (EPOS Connect)

Connect device → open EPOS Connect → **Update Overview** tab → **Check for Updates** → click the update icon → don't disconnect.

## Common USB Issues

**Device not detected.**
- Try a different USB port; **avoid KVMs, port replicators, docking stations, and hubs** — connect directly.
- In Windows Sound, set the EPOS device as **Default Communication Device** on both Playback and Recording.
- Restart the EPOS background service and relaunch.

**Call-control buttons not working.**
- EPOS Connect **must be installed and running**.
- *Known failure:* the **default-softphone field goes blank**.
- *Fix:* close EPOS Connect + softphone → reinstall the softphone plugin/connector → restart EPOS Connect → **set the default softphone** → then launch the softphone.

**Sidetone/Busylight/ANC** are all in EPOS Connect device settings.

**BTD 800 dongle pairing.**
- Usually pre-paired. To pair: plug in, **hold the button ~3 s** (LED alternates **blue/red**). LED: dimmed blue = connected, **purple = Microsoft Teams**. Clear pairing list: in pair mode, **double-press** the button.

## Gotchas

- **Teams variant** (often a **"T"** suffix) has a dedicated Teams button and Teams signaling (BTD 800 lights **purple**); the standard UC unit lacks the certified Teams button.
- **IMPACT 5000 = the SDW 5000 DECT system** (base + headset over DECT; SDW D1 USB dongle for PC).
- Uninstall legacy **HeadSetup Pro** before installing EPOS Connect — two clients fighting over the device causes intermittent connection failures.

## Genesys Cloud with EPOS

Install **EPOS Connect** software first before expecting call-control buttons to work in Genesys Cloud. See `genesys/gc-4.6-call-control.md`.
