---
section: genesys
subsection: "4.3"
topic: browser_mic_permissions
application: genesys_cloud
platforms: [chrome, edge, firefox]
---

# §4.3 Browser Microphone Permissions (Chrome Site Settings) — The #1 "No Mic" Cause

**Scope:** How to check and fix browser microphone permissions for Genesys Cloud WebRTC. Classic symptom: the other party can't hear you despite the headset being selected.

**Likely cause:** When Chrome first asked "Allow microphone?", **Allow** was not clicked (or it was blocked). WebRTC then has no mic access regardless of which device is selected. The classic symptom is *"…does not have permission to access microphone"* or your voice not being heard.

## Exact Steps — Chrome

1. With the Genesys Cloud tab open, click the **lock / tune icon** at the left of the Chrome address bar.
2. Open **Site settings** and set **Microphone** to **Allow**.
3. Alternatively go to `chrome://settings/content/microphone`, confirm the Genesys URL (e.g., `apps.mypurecloud.com`) is in **Allowed**, not **Blocked**, and that the **default microphone** at the top is your headset.
4. **Reload** the Genesys tab so the new permission takes effect.

## Firefox Equivalent

If it keeps re-prompting, check **Remember this decision** then **Allow**. If you previously chose "Don't Allow," change the URL's Microphone from **Block** to **Allow** in site permissions.

## How to Verify

Reload, open WebRTC settings, click **Test Settings** — the mic test should pass and the level meter should respond to your voice.
