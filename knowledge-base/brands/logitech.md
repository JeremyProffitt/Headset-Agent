---
section: brands
brand: logitech
products: [Zone series]
---

# Logitech USB Headset Troubleshooting (Zone Series; Logi Tune)

**Scope:** Logitech-specific diagnosis for USB connection, call-control, firmware, sidetone, and receiver pairing issues.

**Companion app — Logi Tune** (Desktop + mobile; download from logitech.com/tune). The **only** correct app for Zone business headsets. Controls EQ, **sidetone** (dial: higher = more of your own voice), mic gain/test, ANC (e.g. Zone 950), the **busy light** (off by default — enable in Tune), voice prompts, and firmware.

> **Critical:** Use Logi Tune — **NOT G HUB (gaming)** and **NOT Logi Options+ (mice/keyboards).** Wrong app = no controls/firmware shown. The single most common Zone mistake.

## Firmware Update (Logi Tune)

Connect → open Logi Tune → select the device under **MY DEVICES** → click the **(i) info icon** → **Check for update** → **UPDATE** → **Done**. Keep it connected throughout. **Logitech's standalone Firmware Update Tool is retired for Zone — use Logi Tune.**

## Common USB Issues

**Not detected / no audio.**
- Plug **directly into the PC, not a hub/adapter**; try another port; for wired earbuds make sure the 3.5 mm plug is fully seated in the controller; for receivers, replug or re-pair.

**Call-control buttons not working.**
- *Steps:* **use the included USB receiver, not Bluetooth** — some apps don't support full mute/call control over BT; quit/reopen the call client; reselect the audio device.
- Logitech states answer/end behavior **varies by softphone** and the button won't *start* a call.
- *Known limitation:* the physical mute button does **not** sync with **Five9** due to HID limitations.

**Receiver pairing (Zone Wireless 2):**
- Plug the USB-C receiver → power on headset → Logi Tune → **Zone Receiver** → **Pair headset** → put headset in pairing (slide power to Bluetooth icon, hold ~2 s, flashes blue) → both LEDs go **solid white**.

## Gotchas

- **Teams vs UC variants have different part numbers.** Teams has the dedicated Teams button; UC omits it.
- **USB-A vs USB-C receivers:** a **replacement receiver must be re-paired in Logi Tune**.
- **Sidetone** is controlled exclusively in Logi Tune (higher = more of your own voice heard).
- **Busylight** is off by default — enable it in Logi Tune.
