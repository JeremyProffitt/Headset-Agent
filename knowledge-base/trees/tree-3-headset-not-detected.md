---
section: trees
topic: tree_3
symptom: headset_not_detected
---

# Tree 3 — Headset Not Detected (Not in Windows or the Softphone At All)

**Scope:** Diagnostic tree for when the headset doesn't appear anywhere in Windows or the softphone. Goal: get the device to physically appear. Until Windows sees it, no app can.

> Always run the **Universal Pre-Flight Checklist** (`preflight-checklist.md`) before entering this tree.

1. **Reseat directly into the computer** (pre-flight #1) — bypass any dock/hub/monitor. Listen for the connect chime.
   - **If it now appears → return to the symptom the user originally had (Tree 1 or 2).**
   - **If still nothing →** continue.
2. **Try a different USB port** — ideally a port on a different side/controller of the machine, and a USB-A vs USB-C port if available.
   - **If it appears → resolved (note the bad port for the user).**
   - **If not →** continue.
3. **Try a different computer or a different cable** (if the headset is detachable-cable or the user has a second machine handy).
   - **If it works on another machine → the headset is fine; the problem is this PC.** Go to step 4.
   - **If it fails on every port and every machine → strongly suspect a dead headset/cable.** Go to `trees/escalation-criteria.md` (RMA).
4. **Check Device Manager.** See `windows/win-2.1-verify-recognized.md`. Open **Device Manager** and look under **Sound, video and game controllers** and **Audio inputs and outputs**. Look for the headset, a **yellow ! warning**, or an **Unknown/Other device**.
   - **If present but with an error/yellow ! →** uninstall the device, then **scan for hardware changes** or reboot to force driver reinstall. Update the driver if needed.
   - **If completely absent →** uninstall the **USB controllers** (or reboot to re-enumerate USB), update **chipset/USB drivers**; if a managed/corporate machine, group policy or USB-device restrictions may be blocking it — see step 5.
5. **Managed-machine / policy check.** On locked-down call-center machines, USB audio devices can be blocked by endpoint policy, or the model may not be on an approved list.
   - **If you suspect policy/whitelisting → escalate to Tier 2 / IT** (the agent can't override policy from the desk). Go to `trees/escalation-criteria.md`.
   - **If detection still fails on a known-good machine with no policy → hardware fault.** Go to `trees/escalation-criteria.md` (RMA).
