---
section: genesys
subsection: "4.4"
topic: embedded_iframe_popout
application: genesys_cloud
platforms: [salesforce, zendesk]
---

# §4.4 Embedded / iFrame Scenarios (Salesforce, Zendesk) — You Usually Must Pop Out

**Scope:** How to handle Genesys Cloud embedded inside another app (Salesforce, Zendesk) where the microphone or call-control buttons don't work due to browser iframe restrictions.

**Likely cause:** Genesys is embedded inside another app (Salesforce, Zendesk) as an **iframe**. Browsers restrict mic and **HID** (headset-button) access inside iframes, so the mic prompt may never appear or call-control buttons don't work.

## Exact Steps

1. **Pop out the phone:** **Menu > More > Settings > WebRTC > select "Pop WebRTC Phone window."** This opens a standalone window that *can* prompt for and hold the microphone permission and HID access. For Salesforce specifically, this pop-out is required to use Jabra/Yealink headset buttons.
2. **Allow pop-ups** for the embedding site, or the window won't appear (common error: "WebRTC Phone window unable to display").
3. Grant the mic permission in the popped-out window (lock icon > Microphone > Allow), then reload.
4. **For administrators** (escalate if you can't self-serve): the embedding iframe must include `allow="camera *; microphone *; autoplay *"`, and Embeddable Framework deployments may need additional **HID** permissions for headset call control.

## How to Verify

In the popped-out window, run **Test Settings**; the mic test passes and (for supported headsets) the answer/mute buttons respond.
