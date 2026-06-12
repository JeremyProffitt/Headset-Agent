---
section: trees
topic: tree_7
symptom: mute_sync_buttons
---

# Tree 7 — Mute Out of Sync / Call-Control Buttons Not Working

**Scope:** Diagnostic tree for call-control button failures and mute state disagreement between headset and app. Key fact to set expectations: **Genesys Cloud gives audio to *any* USB headset, but only supports the built-in call-control buttons (answer/end/mute) on *specific* models from *certain* manufacturers.** Confirm support before chasing a "bug."

> Always run the **Universal Pre-Flight Checklist** (`preflight-checklist.md`) before entering this tree.

1. **Is this about the buttons (answer/end/mute) not working, or mute being out of sync between headset and app?**
   - **Buttons do nothing →** go to step 2.
   - **Mute state disagrees (muted on headset but app shows unmuted, or vice-versa) →** go to step 3.
2. **Call-control buttons not working.**
   1. **Is this headset model on the supported call-control list for Genesys Cloud?** See `genesys/gc-4.6-call-control.md` and brand docs in `brands/`.
      - **If NOT supported → buttons-as-call-control won't work; this is expected.** The user must answer/mute **in the app**. Audio still works. Done (set expectations).
      - **If supported →** continue.
   2. **Is the vendor software installed and running?** Supported call control usually requires the manufacturer's helper app/middleware (e.g., **Poly Lens / Plantronics Hub, Jabra Direct, Logitech Logi Tune, EPOS Connect**). Confirm it's installed, running, and the headset shows as connected in it. See brand docs in `brands/`.
      - **If installing/launching it enabled the buttons → resolved.**
      - **If still not working →** continue.
   3. **Restart the link.** Re-plug the headset directly, **restart the vendor app**, then **restart the Genesys desktop app / refresh the browser tab** so it re-handshakes call control. In Genesys, confirm the headset is selected as the active call-control device. See `genesys/gc-4.6-call-control.md`.
      - **If restored → resolved.**
      - **If supported model + vendor app present + restarted, still dead → escalate to `genesys/` docs / brand docs / Tier 2.**
3. **Mute out of sync.** This is almost always a **handshake desync** between the headset's mute and the app's mute.
   1. **Re-sync by toggling once on each side** so they match a known state (un-mute on the headset, then un-mute in the app), and **end/rejoin or place a fresh test call**.
      - **If they track together again → resolved.**
      - **If they keep drifting →** continue.
   2. Confirm the **vendor app is installed/running** (it's what relays mute state) and the model is **call-control supported** (step 2.i). Then **restart the vendor app + Genesys app/browser tab** to rebuild the link. See brand docs in `brands/` and `genesys/gc-4.6-call-control.md`.
      - **If restored → resolved.**
      - **If sync still drifts on a supported model with current vendor software → escalate to Tier 2** (likely a firmware/middleware compatibility issue). **Safety call-out for the agent:** warn the user that until it's fixed, they should **trust the in-app mute indicator**, since hardware/app disagreement risks hot-mic privacy.
