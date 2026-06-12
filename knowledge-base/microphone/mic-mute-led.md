---
section: microphone
topic: mic_mute_led
platforms: [windows10, windows11]
---

# Mute LED On, or Mute Out of Sync Between Headset and App

**Scope:** The headset's mute light is on and the caller can't be heard, or the headset's mute state disagrees with the softphone (muted on the headset but the app shows unmuted, or the reverse). This supports Tree 2 and Tree 7.

## Likely Cause

Either the headset is simply **muted at the hardware** — the most common cause of "they can't hear me" — or the headset and the app have had a **handshake desync**, where the hardware mute and the in-app mute have drifted out of agreement. Hardware mute overrides everything in software, and Windows often cannot see or override it.

## Check

1. Find every mute control on the headset: a physical mute button, an inline mute switch on the cable, or a boom arm that **mutes when flipped up**. Check the mute LED — often red means muted.
2. Ask whether the problem is "the light is on and they can't hear me" (simple hardware mute) or "the headset says one thing and the app says another" (mute out of sync).
3. Key expectation to set: Genesys Cloud gives audio to **any** USB headset, but only supports the built-in call-control buttons — answer, end, **mute** — on **specific models from certain manufacturers**. If the model isn't on the supported list, the mute button syncing with the app is not expected to work; the user must mute and un-mute **in the app**.

## Steps

1. **Physical mute first.** Press the mute button or flip the inline switch to un-mute, and rotate the mic boom **down toward the mouth** (boom-up mutes on many models). Confirm the mute LED turns off.
2. **Software mute second.** Confirm the user isn't muted inside the call — check the softphone's in-call mute control (for Genesys Cloud, the call controls), and check Windows: click the speaker icon in the taskbar and confirm nothing shows a crossed-out muted symbol.
3. **If mute is out of sync:** re-sync by toggling once on each side to a known state — un-mute on the headset, then un-mute in the app — and place a fresh test call or end and rejoin.
4. **If they keep drifting apart:** confirm the manufacturer's helper app is installed and running, since it is what relays mute state — for example **Poly Lens / Plantronics Hub, Jabra Direct, Logitech Logi Tune, EPOS Connect** — and confirm the headset shows as connected in it. Then restart the vendor app and restart the Genesys desktop app or refresh the browser tab so it re-handshakes call control.
5. **If sync still drifts** on a supported model with current vendor software, escalate to Tier 2 — this is likely a firmware or middleware compatibility issue.

## Verify

Toggle mute on the headset and confirm the app's mute indicator follows it (on supported models with the vendor app running), and toggle in the app and confirm the headset LED follows. Place a test call and confirm the other party hears you when un-muted and does not when muted. **Safety note while any desync persists:** tell the user to **trust the in-app mute indicator**, since hardware/app disagreement risks a hot mic.
