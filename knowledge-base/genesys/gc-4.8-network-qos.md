---
section: genesys
subsection: "4.8"
topic: network_qos
application: genesys_cloud
---

# §4.8 Network / QoS Notes (Act on What You Can, Escalate the Rest)

**Scope:** Network-layer actions a Tier-1 agent can take to improve Genesys Cloud WebRTC audio quality, and when to escalate to IT/admin.

WebRTC audio is real-time and unforgiving of network problems.

## What a Tier-1 Agent Can Do

- **Use a wired Ethernet connection** instead of Wi‑Fi when possible.
- **Disconnect from VPN** if call audio is choppy — VPNs/proxies add latency and can mangle UDP media. Genesys explicitly recommends dropping the VPN to test.
- **Disconnect and re-place the call**, **clear browser cache**, and **log out/in** (or restart the desktop app).

## When to Escalate to IT/Admin

Escalate when **Run Diagnostics** (see `gc-4.5-diagnostics.md`) shows:
- Packet loss ≥ 1%
- Latency ≥ 150 ms
- Jitter ≥ 30 ms
- MOS below ~3.5
- The **Network Test** can't reach AWS/Genesys media

Hand IT a screenshot of **Test Results**.

## Admin-Level Actions (Escalation Only)

Admins should:
- Ensure **QoS prioritization** of Genesys voice traffic.
- Allow **32–128 Kbps bidirectional per concurrent call**.
- Run the **Genesys Cloud Network Readiness Assessment**.
- Confirm firewall rules match Genesys's recommended ranges.
