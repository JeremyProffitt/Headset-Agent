---
section: genesys
subsection: "4.5"
topic: webrtc_diagnostics
application: genesys_cloud
---

# §4.5 Built-In WebRTC Diagnostics / Test Call — and What Good vs Bad Looks Like

**Scope:** How to run the Genesys Cloud built-in WebRTC diagnostics and interpret the results.

## Run It

1. Select **Calls > Phone Settings**.
2. Click **Run Diagnostics**. (Prereqs: logged in, voicemail configured, no persistent phone connection active.)
3. Wait for the tests to finish, then click **Test Results** for metrics.

## What It Tests

Streaming Connection, WebRTC Station, Call Connected, Call Quality, plus a Network Test (DNS + connectivity to AWS / Genesys media). A quick in-settings check is also available: the **Speaker** button plays test tones (Chrome) and **Test Settings** runs the mic diagnostic.

## Good vs Bad (Target Thresholds)

| Metric | Good (pass) | Bad (investigate/escalate) |
|---|---|---|
| MOS (Mean Opinion Score) | 4–5 | below ~3.5 |
| Packet loss | < 1% | ≥ 1% |
| Round-trip time (latency) | < 150 ms | ≥ 150 ms |
| Jitter | < 30 ms | ≥ 30 ms |

## Interpreting Results

If all tests pass but audio is still bad, the problem is local (device selection, mic settings, or the headset itself). If the **Network Test** or quality metrics fail, it's a network/firewall issue — escalate to IT with the **Test Results** screenshot.
