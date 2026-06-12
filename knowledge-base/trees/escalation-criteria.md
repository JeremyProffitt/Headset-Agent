---
section: trees
topic: escalation
applies_to: all_symptoms
---

# When to Escalate to a Human / Tier 2

**Scope:** Escalation criteria for all headset diagnostic trees. Escalate (warm-transfer to Tier 2 or open a hardware/RMA case as noted) when **any** of the following is true. Don't keep the user looping on steps past these thresholds.

- **Hardware fault suspected.** The symptom follows the headset across **multiple USB ports and/or a second computer**, audio cuts in/out with **physical movement** (wiggle test), one earcup or the mic is **consistently dead** after balance/Mono/source/port checks, or detection fails everywhere. → **Open an RMA / defective-unit case.**
- **Verified-failed fix after completing a tree.** You reached the end of the relevant tree — including a **reboot and a driver update/reinstall** — and the symptom remains. Escalate with notes on what you already tried so Tier 2 doesn't repeat it.
- **The "≥ 2 reboots / no progress" rule.** If you've rebooted **twice** and reinstalled the driver with **no change**, stop and escalate rather than re-running the same steps.
- **User can't perform the steps.** The user is unable or uncomfortable doing Device Manager / power-settings / driver changes, lacks admin rights, or it's a **managed/locked-down machine** where policy blocks the change (USB whitelisting, no admin). → **Escalate to Tier 2 / IT** who can act on the endpoint.
- **Accessibility need.** The user requires an accommodation (e.g., mono audio, specific amplification, a specific assistive device, or can't follow standard-pace steps). Route to a path/agent equipped to support it rather than forcing the default flow.
- **Suspected defective unit / RMA.** Out-of-box failure, physical damage, intermittent fault that survives every software fix, or the unit fails on a **known-good** machine. → **Initiate RMA / replacement.**
- **Beyond-the-desk Genesys/platform issues.** Symptom persists on a **good wired connection** with the headset confirmed working at the Windows layer (robotic/choppy WebRTC audio, call-control unsupported-or-broken on a *supported* model, persistent echo with a proper headset, mute desync on a supported model). → Escalate via §4 to **Tier 2 / Genesys admin**.

**When you escalate, hand off cleanly:** state the symptom, the **exact tree and step you stopped at**, every fix already attempted (especially reboots, driver reinstall, ports tried, and whether it failed on a second machine), the **headset make/model**, and whether the machine is **managed/restricted**. This prevents Tier 2 from restarting at step 1.
