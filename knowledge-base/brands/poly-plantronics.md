---
section: brands
brand: poly_plantronics
products: [Blackwire, Voyager, Savi, BT700]
---

# Poly / Plantronics USB Headset Troubleshooting (Blackwire, Voyager, Savi; Poly Lens vs Plantronics Hub; BT700)

**Scope:** Poly/Plantronics-specific diagnosis for USB connection, call-control, firmware, sidetone, and dongle pairing issues.

**The single most important Poly fact: there are two apps, and newer gear uses Poly Lens, not Plantronics Hub.**
- **Poly Lens Desktop App** (download from **poly.com/lens**) is the current, recommended app and the official **replacement for the deprecated Plantronics Hub**. It manages firmware, device settings (sidetone, mute tones/alerts, language packs), call-control button config, and **target softphone** selection.
- **Plantronics Hub** is **legacy/headset-only** — keep it only for old hardware (e.g. Savi W740, CS500/CS540) or old OSes where Lens won't install.

## Which App for Which Product (Use Lens Unless Told Otherwise)

| Product | App |
|---|---|
| Voyager Focus 2, Voyager 4300 (4310/4320), Voyager Free 60/60+ | **Poly Lens** |
| Blackwire 8225 | **Poly Lens** (settings also in Hub) |
| Blackwire 3200/5200 | **Poly Lens** for firmware (settings in both) |
| Savi 7200/8200 DECT | **Poly Lens** for firmware (settings also in Hub) |
| BT700 adapter | **Poly Lens** (re-pairing) |
| Legacy Savi W740, CS500/CS540 | **Plantronics Hub** |

## Firmware Update (Poly Lens)

Connect device → open Lens → select it → Lens auto-checks; click **Update**. For a specific version (or rollback) use **Manage → Software Versions**. *Gotcha:* the **Voyager Free 60+ touch-screen charging case firmware updates only via Poly Lens Desktop over USB**.

## Common USB Issues

**Device not detected.**
- *Steps:* replug **directly into the PC** (not a dock/hub); in **Windows Sound → Manage sound devices** ensure the Poly device is **Enabled** for Playback and Recording; update/roll back the audio driver in Device Manager (especially after a Windows update); reinstall Poly Lens.
- **Prefer the BT700 dongle over the PC's built-in Bluetooth.**

**Call-control buttons not working.**
- *Steps:* install **Poly Lens**; in Lens select the headset → **Buttons** tab → ensure Attend/End Call enabled; set the correct **target softphone** (wrong target causes "answer button hangs up").

**BT700 pairing/re-pairing.**
- Replug the BT700 directly → Poly Lens → select **Poly BT700** → **Pair New Device** (adapter LED flashes **red/blue**) → put headset in pair mode → success when LED goes **solid** and you hear "PC connected."

**Sidetone gotcha:** on the **Blackwire 8225, sidetone is currently disabled** in firmware in **both** Lens and Hub.

## Gotchas

- **BT700 comes in USB-A and USB-C** — order the matching connector; the dongle (not raw PC Bluetooth) gives full call control and longer range.
- **Teams vs UC SKUs:** Teams SKUs map a dedicated Teams button; UC SKUs are platform-neutral.
- **Plantronics Hub is being retired** — pointing a Voyager Focus 2 / 4300 / Free 60 / Blackwire 8225 user at Hub is the wrong path.

## Genesys Cloud with Poly

**Plantronics Hub software must be installed** for Poly/Plantronics call control in Genesys Cloud; connect via **USB**. Works in browser client, desktop app, and embedded clients. See `genesys/gc-4.6-call-control.md`.
