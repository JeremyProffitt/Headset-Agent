---
section: trees
topic: tree_8
symptom: intermittent_disconnects
---

# Tree 8 — Intermittent Disconnects / Audio Drops

**Scope:** Diagnostic tree for headsets that keep cutting out, dropping, or disconnecting. The classic root cause is the **dock/hub/USB port + Windows power management putting the USB device to sleep.** Work that first.

> Always run the **Universal Pre-Flight Checklist** (`preflight-checklist.md`) before entering this tree.

1. **Get off the dock/hub.** Plug the headset **directly into a USB port on the computer** (pre-flight #1). Docking stations and unpowered hubs are the most common cause of cut-outs.
   - **If drops stop on a direct port → resolved** (advise the user to keep it off the dock, or use a **powered** hub).
   - **If still dropping →** continue.
2. **Try a different direct USB port** (different side/controller; try USB-A vs USB-C). A flaky single port is common.
   - **If a different port is stable → resolved.**
   - **If still dropping →** continue.
3. **Disable USB power management (selective suspend).** This is the highest-value fix for intermittent USB audio drops. In order, see `windows/win-2.6-audio-services.md` and Device Manager:
   - In **Device Manager**, for the headset device **and** for each **USB Root Hub / Generic USB Hub**, open **Properties > Power Management** and **uncheck "Allow the computer to turn off this device to save power."**
   - In **Power Options > advanced settings**, set **USB selective suspend** to **Disabled**.
   - **Update USB/chipset drivers** and, if used, the **dock's firmware**.
   - **If drops stop → resolved.**
   - **If still dropping →** continue.
4. **Rule out the cable/headset.** Do the **wiggle test** near the plug and earcup; try a **different cable** (if detachable) and, if possible, a **different computer**.
   - **If movement triggers drops, or it drops on another machine too → hardware/cable fault.** Go to `trees/escalation-criteria.md` (RMA).
   - **If it only drops on this one machine after all the above → escalate to Tier 2 / IT** (host USB controller, dock firmware, or driver issue beyond the desk).
