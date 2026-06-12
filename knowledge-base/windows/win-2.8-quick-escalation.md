---
section: windows
subsection: "2.8"
topic: quick_escalation
platforms: [windows10, windows11]
---

# §2.8 Quick Escalation Checklist (If All Windows Steps Fail)

**Scope:** Final checklist to run before escalating a Windows-layer audio issue to Tier 2, including the information to gather for a clean handoff.

Run this checklist if all of §2.1–§2.7 have been attempted (device recognized, defaults set, volume checked, privacy confirmed, enhancements disabled, services restarted, driver reinstalled) and the symptom remains.

## Checklist

1. Test the **same headset on a second computer** — if it fails there too, suspect the **hardware/cable**.
2. Test a **second, known-good headset** on the caller's PC — if that one works, the original headset is faulty.
3. Note the **Windows version** (run **winver**), the **Device Manager error code**, and whether the issue is **playback, recording, or both** — and escalate to Tier 2 / the headset vendor with those details.

## Information to Capture Before Escalating

- Headset make and model (check the headset body, boom arm, or USB dongle label)
- Windows version (run **winver** in the search bar)
- Device Manager error code (if any yellow triangle was present)
- Whether the symptom is playback only, recording only, or both
- Steps already attempted (ports tried, reboots done, driver reinstall outcome)
- Whether the headset was tested on a second computer and the result
- Whether the machine is managed/corporate (group policy, restricted USB)

See `trees/escalation-criteria.md` for the full escalation thresholds and handoff script.
