---
section: genesys
subsection: "4.6"
topic: headset_call_control
application: genesys_cloud
---

# §4.6 Headset Call Control (Answer / End / Mute / Hold from the Headset)

**Scope:** Which headset brands support call-control buttons in Genesys Cloud, and how to set them up.

**What's supported:** Genesys Cloud passes audio for *any* USB headset, but **built-in button call control** (answer, hang up, hold/resume, mute/unmute, reject) is only supported on specific vendors: **Jabra, Poly/Plantronics, Sennheiser/EPOS, Yealink, and Cyber Acoustics**.

## Vendor Requirements

- **Jabra:** Chrome/Edge, the desktop app, or embedded clients. Connect by **wired USB or a Jabra USB Bluetooth adapter/dongle**. On first use, approve the prompts and pick the device in Chrome's **WebHID "Connect"** dialog, confirm it as default device, and save a profile. Jabra Direct recommended. **Not** supported on Firefox/Safari.
- **Poly/Plantronics:** **Plantronics Hub software must be installed**; connect via **USB**. Works in browser client, desktop app, and embedded clients.
- **Sennheiser/EPOS:** install **EPOS Connect** software first.
- **Yealink:** works in Genesys Cloud. Use USB Connect for firmware/tuning.
- **Cyber Acoustics:** supported; no special companion app noted.

## Key Limitation — Bluetooth

A headset paired through the computer's **internal/built-in Bluetooth** adapter does **not** get call control. You must use a wired USB connection or the vendor's own USB Bluetooth dongle.

## Desktop App vs Browser

Call control works in both for the supported vendors. The browser path requires **WebHID** (Chromium only). The desktop app needs the **latest version**.

## How to Verify

Place a test call; pressing the headset's hook/mute button should answer/mute in the Genesys call controls (the on-screen Mute toggles in sync).
