---
section: common
topic: symptom_index
applies_to: all_symptoms
---

# Symptom-to-Tree Quick Index

**Scope:** Jump table mapping user-reported symptoms to the correct diagnostic tree. Always run the **Universal Pre-Flight Checklist** (`trees/preflight-checklist.md`) first, then use this index to navigate.

## How to Use

> "The softphone" means the calling app in front of the user — most often **Genesys Cloud** (browser-based WebRTC phone or desktop app). When the tree says "the app's audio/device settings," in Genesys that lives under the **call control / gear / Audio menu** and the WebRTC phone's device picker. See `genesys/gc-4.2-device-selection.md` for exact locations.

| # | What the user says | Go to Tree |
|---|---|---|
| 1 | "I can't hear anything / there's no sound in my headset." | `trees/tree-1-no-audio-output.md` |
| 2 | "They can't hear me / my mic isn't working." (most common) | `trees/tree-2-mic-not-working.md` |
| 3 | "My headset isn't showing up at all / not detected." | `trees/tree-3-headset-not-detected.md` |
| 4 | "Sound's only in one ear / it's mono." | `trees/tree-4-one-sided-audio.md` |
| 5 | "Audio is choppy, robotic, crackly, or echoing." | `trees/tree-5-distorted-choppy-echo.md` |
| 6 | "It's too quiet / too loud / I can (or can too loudly) hear myself." | `trees/tree-6-volume-sidetone.md` |
| 7 | "Mute is out of sync / my answer/end/mute buttons don't work." | `trees/tree-7-mute-sync-buttons.md` |
| 8 | "It keeps cutting out / dropping / disconnecting." | `trees/tree-8-intermittent-disconnects.md` |

## Quick Reference: Section Cross-Links

| Area | Docs |
|---|---|
| Windows OS steps | `windows/win-2.1-verify-recognized.md` through `win-2.8-quick-escalation.md` |
| Brand-specific | `brands/jabra.md`, `brands/poly-plantronics.md`, `brands/logitech.md`, `brands/epos-sennheiser.md`, `brands/yealink.md`, `brands/cross-manufacturer.md` |
| Genesys Cloud | `genesys/gc-4.1-webrtc-overview.md` through `gc-4.8-network-qos.md` |
| Escalation | `trees/escalation-criteria.md` |
