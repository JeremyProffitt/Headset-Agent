# Headset Troubleshooting Guide — USB Headsets on Windows

**Audience:** Contact-center / customer-service agents (Tier 1) helping an end user troubleshoot a **USB-connected headset on Windows 10/11**, primarily for use with softphones — especially **Genesys Cloud** (WebRTC).
**Purpose:** A read-aloud, branch-driven troubleshooting reference. It is also the **domain knowledge base** the AI Headset Support Agent retrieves against (see `PRD.md` → CC-1, DM-1).
**Scope (v1):** USB connection · Windows 10/11 · Genesys Cloud + general softphones · top business headset brands (Jabra, Poly/Plantronics, Logitech, EPOS/Sennheiser, Yealink). *Out of scope for this version: Bluetooth-only pairing, macOS/Linux/mobile, DECT deep-dives — tracked as future work in `PRD.md`.*

> **How to use this guide**
> 1. Always run the **Universal Pre-Flight Checklist** (Section 1) first — it resolves most tickets in under two minutes.
> 2. Use the **Symptom-to-Tree Quick Index** to jump to the right decision tree.
> 3. Each tree branches with **If YES / If NO** and points you to a deeper reference section (**Windows/OS** = §2, **Manufacturer-Specific** = §3, **Genesys** = §4) for click-by-click steps.
> 4. Every fix is framed **Likely cause → What to check → Exact steps → How to verify.**
> 5. Stop and escalate when you hit the criteria in **"When to Escalate to a Human / Tier 2"** (end of Section 1).

## Table of Contents
1. [Symptom-Based Diagnostic Trees](#1-symptom-based-diagnostic-trees) — the navigational spine (start here)
2. [Windows & OS-Level USB Headset Troubleshooting](#2-windows--os-level-usb-headset-troubleshooting)
3. [Manufacturer-Specific USB Headset Troubleshooting](#3-manufacturer-specific-usb-headset-troubleshooting)
4. [Genesys Cloud (WebRTC) USB Headset Troubleshooting](#4-genesys-cloud-webrtc-usb-headset-troubleshooting)
5. [Consolidated Sources](#5-consolidated-sources)

---

# 1. Symptom-Based Diagnostic Trees

This section is the navigational spine of the guide. Each tree is an ordered triage flow designed to be read aloud to the end user. Steps are sequenced **cheapest and most-common fix first**. At each decision point, follow the **If YES** or **If NO** branch. When a branch tells you to jump to a deeper section (e.g., *Windows/OS Audio Settings* = §2, *Manufacturer-Specific Notes* = §3, or *Genesys Cloud Configuration* = §4), that section contains the detailed click-by-click steps.

> **Agent note on terminology:** "The softphone" means the calling app in front of the user — most often **Genesys Cloud** (browser-based WebRTC phone or desktop app). When the tree says "the app's audio/device settings," in Genesys that lives under the **call control / gear / Audio menu** and the WebRTC phone's device picker. See §4 for exact locations.

## First, always check these — Universal Pre-Flight Checklist

Run this **before** entering any tree. It resolves the large majority of headset tickets in under two minutes. Read each item aloud and confirm with the user before moving on.

1. **Reseat the USB connector.** Unplug the headset and plug it **directly into a USB port on the computer itself** — not into a monitor, keyboard, docking station, or USB hub. (Docks and hubs are the #1 cause of "it worked yesterday" issues.) Wait ~10 seconds for Windows to chime/recognize it.
2. **Confirm the headset is the selected/default device** in both **Windows Sound settings** and **the softphone's own device picker**. Many machines silently default to laptop speakers or a webcam mic.
3. **Check the hardware mute.** Look for a physical mute button, an inline mute switch on the cable, or "mute on boom-arm up" — confirm the headset is **not** muted at the hardware level. Many headsets have a mute LED (often red = muted).
4. **Turn the volume up** — both on the headset's inline volume control/dial **and** in Windows. Confirm the Windows volume mixer for the app isn't at zero.
5. **Make sure only one app is using the microphone.** Close Teams, Zoom, Discord, other browser tabs in a call, voice recorders, etc. Two apps fighting for the mic is a top cause of "no mic" and "device busy."
6. **Reboot if needed.** If anything above looked wrong or the user has been fighting this for a while, a full restart (then re-plug the headset) clears most stuck-driver and stuck-device states. In Genesys, also try **refreshing the browser tab** or **restarting the desktop app**.

If the pre-flight resolved it, **stop here** — no tree needed. If not, use the quick index below to jump to the matching tree.

## Symptom-to-Tree Quick Index

| # | What the user says | Go to Tree |
|---|---|---|
| 1 | "I can't hear anything / there's no sound in my headset." | **Tree 1 — No Audio Output** |
| 2 | "They can't hear me / my mic isn't working." (most common) | **Tree 2 — Microphone / Other Party Can't Hear Me** |
| 3 | "My headset isn't showing up at all / not detected." | **Tree 3 — Headset Not Detected** |
| 4 | "Sound's only in one ear / it's mono." | **Tree 4 — One-Sided / Mono Audio** |
| 5 | "Audio is choppy, robotic, crackly, or echoing." | **Tree 5 — Distorted / Choppy / Echo Audio** |
| 6 | "It's too quiet / too loud / I can't (or can too loudly) hear myself." | **Tree 6 — Volume & Sidetone** |
| 7 | "Mute is out of sync / my answer/end/mute buttons don't work." | **Tree 7 — Mute Sync & Call-Control Buttons** |
| 8 | "It keeps cutting out / dropping / disconnecting." | **Tree 8 — Intermittent Disconnects / Audio Drops** |

## Tree 1 — No Audio Output / Can't Hear Anything

> Goal: get sound into the user's ears. Work from "is it even selected and turned up" outward.

1. **Confirm the headset is plugged directly in and powered/recognized** (pre-flight #1). Did Windows make a connect chime, or does the headset show up?
   - **If NO →** go to **Tree 3 (Not Detected)** — you have a detection problem, not an output problem.
   - **If YES →** continue.
2. **Is the headset set as the Windows output (playback) device?** In §2, set the headset as the default **Output** device.
   - **If it was wrong and fixing it restored sound → resolved.**
   - **If already correct →** continue.
3. **Volume and mute check.** Confirm Windows volume is up and not muted; confirm the **inline/headset volume dial** is up; confirm no hardware mute. Open the **Windows Volume Mixer** and confirm the app (browser/Genesys) isn't muted or at zero.
   - **If a volume/mute control was the cause → resolved.**
   - **If all up and unmuted →** continue.
4. **Quick playback test outside the softphone.** Have the user play any sound (a video clip, the Windows "Test" button on the device's properties, or a system sound).
   - **If they hear the test sound → the headset/Windows side is fine. The problem is inside the softphone.** Go to step 5.
   - **If they hear NOTHING even in the test →** the problem is at the Windows/hardware layer. Go to step 6.
5. **Softphone device selection.** In the app's audio settings, confirm the **output/speaker** is set to the headset (not "Default" pointing elsewhere, not speakers). For **Genesys Cloud**, set the WebRTC phone's **Speaker** to the headset and use **Test your media settings** to play a test tone. See §4.
   - **If selecting the headset restored call audio → resolved.**
   - **If still silent in calls only →** escalate to §4 (WebRTC phone / browser permissions / desktop app reset).
6. **Windows-layer remediation** (no sound anywhere). In order: run the **Windows Audio troubleshooter**; check that the audio device isn't **Disabled** in Sound > Manage sound devices; try a **different USB port**; **reboot**; then **update/reinstall the audio driver**. Full steps in §2.
   - **If restored → resolved.**
   - **If still no sound on any app after a reboot and driver reinstall →** suspect hardware. Go to **When to Escalate**.

## Tree 2 — Other Party Can't Hear Me / Microphone Not Working

> This is the #1 contact-center complaint. The most common real causes, in order: **hardware mute, wrong mic selected, app mic permission off, another app holding the mic.**

1. **Hardware mute first** (it's the most common). Check the physical mute button / inline mute switch / mute-on-boom-up, and any mute LED. Have the user explicitly un-mute at the hardware.
   - **If un-muting fixed it → resolved.**
   - **If not muted →** continue.
2. **Is the headset selected as the Windows Input (recording) device?** In §2, set the headset mic as the default **Input** device — make sure it's not pointed at a webcam mic or laptop array mic.
   - **If selecting the right mic fixed it → resolved.**
   - **If already correct →** continue.
3. **Watch the input level meter.** In Windows Sound settings, have the user **speak** and watch the mic test/level bar move.
   - **If the bar moves → Windows hears the mic. The problem is in the softphone or its permissions.** Go to step 4.
   - **If the bar does NOT move →** continue to step 5 (Windows can't hear it).
4. **Softphone mic selection + permissions.** In the app's audio settings, set the **Microphone** to the headset. For **Genesys Cloud**: select the headset mic in the WebRTC phone, run **Test your microphone**, and confirm the **browser** has granted **microphone permission** to Genesys (the browser site-permission prompt/lock-icon). Confirm the user isn't **muted inside the call/in Genesys' call controls**. See §4.
   - **If correcting the in-app mic / permission / in-call mute fixed it → resolved.**
   - **If still silent to the other party →** escalate to §4 (WebRTC phone reset, browser switch, clear cache).
5. **Windows can't hear the mic.** In order: confirm **Settings > Privacy & security > Microphone** has **Microphone access ON** and **desktop/app access allowed**; close every **other app that might hold the mic** (pre-flight #5); try a **different USB port**; **reboot**; then **update/reinstall the audio driver**. Full steps in §2. Note: some headsets expose a separate "hands-free/communications" device — make sure you selected the correct one (see §3).
   - **If restored → resolved.**
   - **If the level meter still never moves after reboot + driver reinstall + a known-good USB port →** suspect a failed mic/boom. Go to **When to Escalate**.

## Tree 3 — Headset Not Detected (Not in Windows or the Softphone At All)

> Goal: get the device to physically appear. Until Windows sees it, no app can.

1. **Reseat directly into the computer** (pre-flight #1) — bypass any dock/hub/monitor. Listen for the connect chime.
   - **If it now appears → return to the symptom the user originally had (Tree 1 or 2).**
   - **If still nothing →** continue.
2. **Try a different USB port** — ideally a port on a different side/controller of the machine, and a USB-A vs USB-C port if available.
   - **If it appears → resolved (note the bad port for the user).**
   - **If not →** continue.
3. **Try a different computer or a different cable** (if the headset is detachable-cable or the user has a second machine handy).
   - **If it works on another machine → the headset is fine; the problem is this PC.** Go to step 4.
   - **If it fails on every port and every machine → strongly suspect a dead headset/cable.** Go to **When to Escalate (RMA)**.
4. **Check Device Manager.** In §2, open **Device Manager** and look under **Sound, video and game controllers** and **Audio inputs and outputs**. Look for the headset, a **yellow ! warning**, or an **Unknown/Other device**.
   - **If present but with an error/yellow ! →** uninstall the device, then **scan for hardware changes** or reboot to force driver reinstall. Update the driver if needed.
   - **If completely absent →** uninstall the **USB controllers** (or reboot to re-enumerate USB), update **chipset/USB drivers**; if a managed/corporate machine, group policy or USB-device restrictions may be blocking it — see step 5.
5. **Managed-machine / policy check.** On locked-down call-center machines, USB audio devices can be blocked by endpoint policy, or the model may not be on an approved list.
   - **If you suspect policy/whitelisting → escalate to Tier 2 / IT** (the agent can't override policy from the desk). Go to **When to Escalate**.
   - **If detection still fails on a known-good machine with no policy → hardware fault.** Go to **When to Escalate (RMA)**.

## Tree 4 — One-Sided Audio / Only One Ear / Mono

1. **Check the Windows balance.** In §2, open the headset's playback **Properties > Levels > Balance** and confirm **left and right are equal** (both not at 0). A skewed balance slider is the most common cause.
   - **If balancing the channels fixed it → resolved.**
   - **If already balanced →** continue.
2. **Check the "Mono audio" accessibility toggle.** In **Settings > Accessibility > Audio**, confirm **Mono audio is OFF** (unless the user needs it on for an accessibility reason — if so, that's expected behavior, not a fault).
   - **If turning off Mono audio fixed it → resolved.**
   - **If already off →** continue.
3. **Isolate hardware vs. signal.** Play a known **stereo** test (e.g., a left/right channel test clip) outside the softphone.
   - **If both ears work on the stereo test → the source/call audio was mono or the app is mono — expected for a phone call (voice calls are typically mono).** Reassure the user; if it only happens in one app, check that app's audio settings.
   - **If still one-sided on a stereo test →** continue (hardware suspected).
4. **Wiggle test / cable + connector.** Have the user gently move the cable near the plug and the earcup joint while audio plays. Reseat the USB. Try a different USB port.
   - **If audio cuts in/out with movement → a failing cable/connector.** Go to **When to Escalate (RMA)**.
   - **If consistently dead in one ear regardless of balance, Mono toggle, source, and port → a failed driver/speaker in that earcup.** Go to **When to Escalate (RMA)**.

## Tree 5 — Distorted, Choppy, Robotic, Crackling, or Echoing Audio

> "Robotic/choppy" usually = **network or CPU** (WebRTC). "Crackling/static" usually = **USB/driver/port**. "Echo" usually = **acoustic loop or sidetone**. Branch on which one the user describes.

1. **Which best describes it?**
   - **Robotic / choppy / cutting in and out / "underwater" →** go to step 2 (network/CPU path).
   - **Crackling / static / popping / buzzing →** go to step 3 (USB/driver path).
   - **Echo (I or they hear a repeat of the voice) →** go to step 4 (echo path).
2. **Robotic/choppy (network/CPU).** Most often this is the connection, not the headset. In order: check the user's **network** (prefer **wired Ethernet over Wi‑Fi**, move off VPN if possible, close bandwidth-heavy apps/downloads); **close heavy apps** and extra browser tabs to free CPU; for **Genesys Cloud**, run **Test your media settings**, and try **restarting the desktop app or refreshing/switching the browser**. See §4.
   - **If switching to wired / freeing resources cleared it → resolved.**
   - **If it persists on a good wired connection → escalate to §4 / Tier 2** (possible WebRTC, codec, or network-path issue beyond the desk).
3. **Crackling/static (USB/driver).** In order: **reseat directly into the PC** (off the dock/hub — a weak/over-loaded USB path causes crackle); try a **different USB port**; in §2 try **disabling audio enhancements/effects** on the device and **lowering the sample rate** (e.g., 24-bit/48000 Hz) in the device's Advanced properties; **update/reinstall the audio driver**.
   - **If a port change / disabling enhancements / driver update cleared it → resolved.**
   - **If crackle persists on multiple ports after a driver reinstall → suspect hardware/cable.** Go to **When to Escalate (RMA)**.
4. **Echo.** Determine **who hears the echo.**
   - **The user hears their *own* voice repeated back too strongly → that's sidetone.** Go to **Tree 6** (Volume & Sidetone).
   - **The *other party* hears an echo → it's usually the *other* side's speakers/mic loop, but on the user's end: confirm they're on a **headset (not open speakers)**, lower **speaker volume**, and ensure **echo cancellation isn't disabled**. In Genesys, confirm the correct headset mic+speaker are selected (so AEC works). See §4.
   - **If isolating to a headset and normalizing volume cleared it → resolved.**
   - **If echo persists with a proper headset → escalate to §4 / Tier 2.**

## Tree 6 — Volume Too Low / Too Loud / Sidetone

> Two different problems live here: (a) **call volume** (what the user hears of the other party), and (b) **sidetone** (how loudly the user hears *their own* voice). Branch first.

1. **Is this about the other person's volume, or hearing your own voice?**
   - **Other person too quiet/loud →** go to step 2.
   - **Hearing my own voice (too much, too little, or echo of myself) →** go to step 3 (sidetone).
2. **Call volume (the other party).** In order: adjust the **headset inline volume dial**; adjust **Windows volume** and the **Volume Mixer** for the app; in the device's playback **Properties > Levels**, raise the **output level**; check **Loudness Equalization** if available for taming loud spikes. If too quiet only in calls, set the headset as the app's output and check the app's call volume. See §2 and §4.
   - **If adjusting these fixed it → resolved.**
   - **If max volume is still too quiet on all apps → suspect hardware/driver;** update the driver, then go to **When to Escalate** if unresolved.
3. **Sidetone (hearing your own voice).** Clarify what the user wants:
   - **"I hear myself too loudly / it echoes back at me" →** Sidetone is too high. If the headset/vendor app has a **sidetone control** (e.g., Poly Lens, Logitech Logi Tune, Jabra Direct), lower it there — see §3. Then check the Windows culprit: in the **Recording** tab > headset mic > **Properties > Listen tab**, make sure **"Listen to this device" is UNCHECKED** (a frequent accidental cause of hearing yourself). See §2.
   - **"I can't hear myself at all and it feels dead/muffled" →** Some users want *some* sidetone for comfort. Raise it in the **vendor app's sidetone setting** if the model supports it (§3). If the model has no sidetone feature, that's a hardware limitation — explain it's expected.
   - **If adjusting sidetone (vendor app) or the Listen toggle resolved it → resolved.**
   - **If the model has the feature but the control has no effect → escalate to §3 / Tier 2** (possible firmware/driver issue).

## Tree 7 — Mute Out of Sync / Call-Control Buttons Not Working

> Key fact to set expectations: **Genesys Cloud gives audio to *any* USB headset, but only supports the built-in call-control buttons (answer/end/mute) on *specific* models from *certain* manufacturers.** Confirm support before chasing a "bug."

1. **Is this about the buttons (answer/end/mute) not working, or mute being out of sync between headset and app?**
   - **Buttons do nothing →** go to step 2.
   - **Mute state disagrees (muted on headset but app shows unmuted, or vice-versa) →** go to step 3.
2. **Call-control buttons not working.**
   1. **Is this headset model on the supported call-control list for Genesys Cloud?** (See §4 and §3.)
      - **If NOT supported → buttons-as-call-control won't work; this is expected.** The user must answer/mute **in the app**. Audio still works. Done (set expectations).
      - **If supported →** continue.
   2. **Is the vendor software installed and running?** Supported call control usually requires the manufacturer's helper app/middleware (e.g., **Poly Lens / Plantronics Hub, Jabra Direct, Logitech Logi Tune, EPOS Connect**). Confirm it's installed, running, and the headset shows as connected in it. See §3.
      - **If installing/launching it enabled the buttons → resolved.**
      - **If still not working →** continue.
   3. **Restart the link.** Re-plug the headset directly, **restart the vendor app**, then **restart the Genesys desktop app / refresh the browser tab** so it re-handshakes call control. In Genesys, confirm the headset is selected as the active call-control device. See §4.
      - **If restored → resolved.**
      - **If supported model + vendor app present + restarted, still dead → escalate to §4 / §3 / Tier 2.**
3. **Mute out of sync.** This is almost always a **handshake desync** between the headset's mute and the app's mute.
   1. **Re-sync by toggling once on each side** so they match a known state (un-mute on the headset, then un-mute in the app), and **end/rejoin or place a fresh test call**.
      - **If they track together again → resolved.**
      - **If they keep drifting →** continue.
   2. Confirm the **vendor app is installed/running** (it's what relays mute state) and the model is **call-control supported** (step 2.i). Then **restart the vendor app + Genesys app/browser tab** to rebuild the link. See §3 and §4.
      - **If restored → resolved.**
      - **If sync still drifts on a supported model with current vendor software → escalate to Tier 2** (likely a firmware/middleware compatibility issue). **Safety call-out for the agent:** warn the user that until it's fixed, they should **trust the in-app mute indicator**, since hardware/app disagreement risks hot-mic privacy.

## Tree 8 — Intermittent Disconnects / Audio Drops

> The classic root cause is the **dock/hub/USB port + Windows power management putting the USB device to sleep.** Work that first.

1. **Get off the dock/hub.** Plug the headset **directly into a USB port on the computer** (pre-flight #1). Docking stations and unpowered hubs are the most common cause of cut-outs.
   - **If drops stop on a direct port → resolved** (advise the user to keep it off the dock, or use a **powered** hub).
   - **If still dropping →** continue.
2. **Try a different direct USB port** (different side/controller; try USB-A vs USB-C). A flaky single port is common.
   - **If a different port is stable → resolved.**
   - **If still dropping →** continue.
3. **Disable USB power management (selective suspend).** This is the highest-value fix for intermittent USB audio drops. In order, in §2:
   - In **Device Manager**, for the headset device **and** for each **USB Root Hub / Generic USB Hub**, open **Properties > Power Management** and **uncheck "Allow the computer to turn off this device to save power."**
   - In **Power Options > advanced settings**, set **USB selective suspend** to **Disabled**.
   - **Update USB/chipset drivers** and, if used, the **dock's firmware**.
   - **If drops stop → resolved.**
   - **If still dropping →** continue.
4. **Rule out the cable/headset.** Do the **wiggle test** near the plug and earcup; try a **different cable** (if detachable) and, if possible, a **different computer**.
   - **If movement triggers drops, or it drops on another machine too → hardware/cable fault.** Go to **When to Escalate (RMA)**.
   - **If it only drops on this one machine after all the above → escalate to Tier 2 / IT** (host USB controller, dock firmware, or driver issue beyond the desk).

## When to Escalate to a Human / Tier 2

Escalate (warm-transfer to Tier 2 or open a hardware/RMA case as noted) when **any** of the following is true. Don't keep the user looping on steps past these thresholds.

- **Hardware fault suspected.** The symptom follows the headset across **multiple USB ports and/or a second computer**, audio cuts in/out with **physical movement** (wiggle test), one earcup or the mic is **consistently dead** after balance/Mono/source/port checks, or detection fails everywhere. → **Open an RMA / defective-unit case.**
- **Verified-failed fix after completing a tree.** You reached the end of the relevant tree — including a **reboot and a driver update/reinstall** — and the symptom remains. Escalate with notes on what you already tried so Tier 2 doesn't repeat it.
- **The "≥ 2 reboots / no progress" rule.** If you've rebooted **twice** and reinstalled the driver with **no change**, stop and escalate rather than re-running the same steps.
- **User can't perform the steps.** The user is unable or uncomfortable doing Device Manager / power-settings / driver changes, lacks admin rights, or it's a **managed/locked-down machine** where policy blocks the change (USB whitelisting, no admin). → **Escalate to Tier 2 / IT** who can act on the endpoint.
- **Accessibility need.** The user requires an accommodation (e.g., mono audio, specific amplification, a specific assistive device, or can't follow standard-pace steps). Route to a path/agent equipped to support it rather than forcing the default flow.
- **Suspected defective unit / RMA.** Out-of-box failure, physical damage, intermittent fault that survives every software fix, or the unit fails on a **known-good** machine. → **Initiate RMA / replacement.**
- **Beyond-the-desk Genesys/platform issues.** Symptom persists on a **good wired connection** with the headset confirmed working at the Windows layer (robotic/choppy WebRTC audio, call-control unsupported-or-broken on a *supported* model, persistent echo with a proper headset, mute desync on a supported model). → Escalate via §4 to **Tier 2 / Genesys admin**.

**When you escalate, hand off cleanly:** state the symptom, the **exact tree and step you stopped at**, every fix already attempted (especially reboots, driver reinstall, ports tried, and whether it failed on a second machine), the **headset make/model**, and whether the machine is **managed/restricted**. This prevents Tier 2 from restarting at step 1.

---

# 2. Windows & OS-Level USB Headset Troubleshooting

> **How to use this section (for the agent):** Each symptom below is laid out as **Likely cause → What to check → Exact steps → How to verify it's fixed.** Read the steps aloud slowly, one at a time, and wait for the caller to confirm each before moving on. Most "no audio" or "they can't hear me" cases are solved by the first three subsections (device recognized → correct default device → mic privacy). Work top to bottom.
>
> **Quick orientation:** In Windows 11, Settings is **Start > Settings** and most sound options live under **System > Sound**. In Windows 10 the layout is similar but a few menus are named differently — those differences are called out as **(Win10)** throughout. Tell the caller to right-click the **Start** button if they can't find the Settings gear.

## 2.1 Verify Windows actually recognizes the USB headset

**Likely cause:** The headset isn't enumerating on the USB bus — bad port, unpowered hub, or a missing/failed driver — so no app can use it.

**What to check:** Whether the headset appears in Device Manager without a warning icon, and whether it shows up as both a playback and a recording device.

**Exact steps:**
1. Ask the caller to unplug the headset and plug it **directly into a USB port on the computer itself** — not into a monitor, keyboard, docking station, or unpowered USB hub. Many headsets draw more power than a passive hub can supply, which causes them to drop out or fail to enumerate.
2. Prefer a port on the **back of a desktop** (those are wired directly to the motherboard). If the first port doesn't work, **try a different physical port.**
3. **USB 2.0 vs 3.0:** A USB headset is a low-bandwidth device and works fine in either a USB 2.0 (usually black) or USB 3.0/3.1 (usually blue) port. If one port type misbehaves, have them try the other — occasionally a flaky USB 3.0 controller or a power-saving setting on a 3.0 port causes audio dropouts.
4. Open **Device Manager**: in the taskbar search box type **device manager** and select it (or press **Windows key + X** and choose **Device Manager**).
5. Expand **Sound, video and game controllers** — the headset (or "USB Audio" / "USB Audio Class 2 Device") should be listed there.
6. Also expand **Audio inputs and outputs** — you should see the headset listed as both a **speaker/headphone** and a **microphone**.

**Reading the icons:**
- **A yellow triangle with an exclamation mark** on the device = Windows sees the hardware but the driver isn't working. Right-click the device > **Properties** > the **General** tab shows a "device status" message and an error **Code**. **Code 28** specifically means *"The drivers for this device are not installed"* — Windows can't find a usable driver. (See driver steps in §2.7.)
- **An "Unknown device" or "Unknown USB Device" entry** (often under **Other devices** or **Universal Serial Bus controllers**, sometimes "Port Reset Failed") = the headset failed to identify itself on the bus. This usually points to a bad cable, a bad/underpowered port, or a hardware fault — not a software setting.

**First fixes for a yellow triangle / unknown device:**
1. Right-click the problem device > **Uninstall device**. If a checkbox **"Attempt to remove the driver for this device"** appears, leave it **checked**.
2. **Unplug the headset, restart the PC**, then plug the headset back in. Windows re-enumerates and reinstalls the in-box driver.
3. If it still doesn't appear, in Device Manager click the **Action** menu > **Scan for hardware changes**.

**How to verify it's fixed:** The headset appears under both **Sound, video and game controllers** and **Audio inputs and outputs** with **no yellow triangle**, and the device status reads **"This device is working properly."**

## 2.2 Select the headset as the default playback, communication, AND recording device

**Likely cause:** Windows is still routing audio to the laptop speakers / built-in mic, or the softphone is grabbing a different "communication" device than the one the caller expects.

**Why this matters (read this to yourself, not the caller):** Windows keeps **two** separate default-device roles:
- **Default Device** — used by general apps (media players, browsers, music).
- **Default Communication Device** — used specifically by apps that detect a phone/VoIP call (softphones, Teams, Zoom, many call-center clients).

A softphone will often follow the **Default Communication Device**, so if that role still points at the built-in speaker or mic, the caller hears nothing in the headset (or the customer can't hear them) **even though the headset looks selected** for everything else. **Set the headset for both roles** to be safe.

**Exact steps — Windows 11 (Settings):**
1. **Start > Settings > System > Sound.**
2. Under **Output**, select the headset to make it the default playback device.
3. Under **Input**, select the headset's microphone as the default recording device.

**Exact steps — the legacy Sound control panel (works on Win10 and Win11, and is the only place to split "Device" vs "Communication"):**
1. Press **Windows key + R**, type **mmsys.cpl**, press **Enter**. (Or, Win11: **Settings > System > Sound > Advanced > More sound settings**. Win10: right-click the speaker icon in the taskbar > **Sounds** > **Playback** tab.)
2. On the **Playback** tab, click the headset once to highlight it.
3. Click the **small arrow next to the "Set Default" button** and choose **Default Device**. Then click the arrow again and choose **Default Communication Device**. (When one device holds both roles you'll see a green check **and** a green phone icon on it.)
4. Switch to the **Recording** tab and do the same for the headset's microphone: set it as both **Default Device** and **Default Communication Device**.
5. Click **OK**.

**(Win10):** Settings path is **Start > Settings > System > Sound**, with **Output** and **Input** dropdowns instead of the radio-button list; the **mmsys.cpl** control panel is identical.

**How to verify it's fixed:** In **mmsys.cpl > Playback**, the headset shows both the green check and the green phone icon. As a live test, play any sound — it should come out of the headset. For the mic, see §2.4's verification.

## 2.3 Volume Mixer and per-app volume — software mute vs hardware mute

**Likely cause:** The correct device is selected, but audio is muted or turned all the way down — either system-wide, for that one app, or by a physical button on the headset.

**What to check:** The per-app **Volume Mixer**, the system volume, and any inline mute switch/button on the headset cord or earcup.

**Exact steps:**
1. **Hardware first:** Many call-center headsets have an **inline mute button or a mic boom that mutes when flipped up**. Ask the caller to check for a physical mute switch and a mute LED, and to confirm the mic boom (if any) is rotated **down** toward the mouth. A hardware mute overrides everything in software, and Windows often **cannot** see or override it.
2. **System volume:** Click the **speaker icon** in the taskbar and make sure the slider isn't at zero and the speaker isn't showing a "muted" (crossed-out) symbol.
3. **Per-app (Volume Mixer):**
   - **Win11:** **Start > Settings > System > Sound > Volume mixer.** Confirm the specific app (the softphone/browser) is **not muted** and its slider is up, **and** that the app's output is pointed at the headset (each app row lets you pick its own output device in Win11).
   - **Win10:** right-click the **speaker icon** in the taskbar > **Open Volume Mixer**, and check each app's column.

**How to verify it's fixed:** With the app playing audio, its bar in Volume Mixer moves and sound is audible in the headset at a comfortable level, with no crossed-out/muted icons.

## 2.4 Microphone privacy settings — the #1 cause of "the agent/customer can't hear me"

**Likely cause:** Windows privacy controls are blocking the app's access to the microphone. This is extremely common after Windows updates and on managed/corporate machines, and it produces the classic symptom: **the headset works for listening, but no one can hear the caller.**

**What to check:** Three nested toggles must all be **On**: the master mic switch, "let apps access," and (for installed/desktop softphones) "let desktop apps access."

**Exact steps — Windows 11:**
1. **Start > Settings > Privacy & security**, then under **App permissions** select **Microphone**.
2. Turn **Microphone access** **On** — this is the master switch for the whole PC; if it's off, nothing can use the mic.
3. Turn **Let apps access your microphone** **On.**
4. Under that, scroll the app list and turn **On** the specific softphone/conferencing app if it's listed.
5. If the softphone is a traditional installed program (most call-center clients, Teams desktop, Zoom), scroll down and turn **On** **Let desktop apps access your microphone.** Desktop apps don't each get their own toggle — this single switch governs all of them. **This is the one people miss.**

**Exact steps — Windows 10:**
1. **Start > Settings > Privacy**, then under **App permissions** select **Microphone.**
2. Click **Change** and ensure **Microphone access for this device** is **On.**
3. Turn **Allow apps to access your microphone** **On.**
4. Under **Choose which Microsoft Store apps can access your microphone**, turn the relevant app **On.**
5. Scroll down and turn **On** **Allow desktop apps to access your microphone.**

**How to verify it's fixed:** Go to **Settings > System > Sound** (Win11) or the **Recording** tab of **mmsys.cpl**, speak into the headset, and watch the **input level / volume meter move** as you talk. Then place a quick test call in the softphone and confirm the customer can hear them.

## 2.5 Audio enhancements, exclusive mode, and sample-rate/format mismatches

**Likely cause:** A signal-processing "enhancement," an "exclusive control" setting, or a default audio format the headset doesn't support — causing distortion, crackling, robotic/garbled audio, intermittent dropouts, or an app that "can't open the device."

**Exact steps — turn off enhancements:**
- **Win11:** **Start > Settings > System > Sound** > click the headset (output device) > scroll to **Audio enhancements** and set it to **Off.**
- **Win10:** open **mmsys.cpl** > **Playback** tab > right-click the headset > **Properties** > **Enhancements** tab > check **Disable all enhancements** (or "Disable all sound effects") > **OK.**

**Exact steps — exclusive mode (the "an app grabbed the device" fix):**
1. Open **mmsys.cpl** (Windows key + R > **mmsys.cpl** > Enter).
2. **Playback** tab > right-click the headset > **Properties** > **Advanced** tab.
3. Under **Exclusive Mode**, **uncheck** both:
   - **Allow applications to take exclusive control of this device**
   - **Give exclusive mode applications priority**
4. Click **OK**. Repeat on the **Recording** tab for the headset's microphone.

> Why: when exclusive mode is on, one app can seize the headset and lock every other app out — so the softphone goes silent the moment another program (a media player, a recorder, a meeting app) is using audio. Clearing these checkboxes lets apps share the device.

**Exact steps — fix the Default Format (sample rate) for distortion / no audio:**
1. In **mmsys.cpl > Playback > [headset] > Properties > Advanced**, find the **Default Format** dropdown.
2. Set it to a standard, widely supported value: **16 bit, 48000 Hz (DVD Quality)** or **24 bit, 48000 Hz.** Avoid unusually high rates.
3. Click **Apply** > **Test** (Windows plays a tone) and confirm it's clean. Do the same on the **Recording** tab for the mic.

**How to verify it's fixed:** Play a test tone or short clip — no crackle/distortion — and make a test call where both directions sound clear. If a specific app previously errored with "couldn't open the audio device," it should now connect.

## 2.6 Restart the Windows Audio services and run the built-in troubleshooter

**Likely cause:** The Windows audio engine has gotten into a bad state (common symptoms: a red X on the speaker icon, "Audio service is not running," or audio vanishing system-wide).

**Restart the two audio services:**
1. In the taskbar search box type **services** and open the **Services** app (or **Windows key + R** > **services.msc** > Enter).
2. Find **Windows Audio**, right-click it > **Restart.**
3. Find **Windows Audio Endpoint Builder**, right-click it > **Restart.** (This service builds the list of audio endpoints; if it's stuck, devices won't appear. Restarting it also restarts Windows Audio, which depends on it.)
4. While there, confirm each service's **Startup type** is **Automatic**: right-click > **Properties** > set **Startup type: Automatic** > **Apply** > **OK**. The dependency **Remote Procedure Call (RPC)** must also be running (it normally is by default).

**Run the built-in audio troubleshooter:**
- **Win11:** **Start > Settings > System > Troubleshoot > Other troubleshooters** > find **Audio** > **Run.** (Or open the **Get Help** app and run the automated audio troubleshooter.)
- **Win10:** **Start > Settings > Update & Security > Troubleshoot** > **Additional troubleshooters** > **Playing Audio** > **Run the troubleshooter** (and **Recording Audio** for mic issues).

**When to reboot or reinstall the driver:** If restarting the services and running the troubleshooter doesn't restore audio — or the device still shows a yellow triangle — do a **full reboot**. If problems persist after reboot, move to the driver reinstall in §2.7.

**How to verify it's fixed:** Both services show status **Running**, the troubleshooter reports problems fixed or "no issues," the speaker icon no longer shows a red X, and audio plays/records normally.

## 2.7 Drivers — generic USB Audio class driver vs. vendor driver, updating and reinstalling

**Likely cause:** A missing, corrupt, or mismatched driver (often the source of a Code 28 / yellow triangle, or of missing vendor features like a busylight or sidetone).

**Background (so the agent understands what they're looking at):**
- Most USB headsets are **"class-compliant"** and work **plug-and-play** using a driver that **ships inside Windows** — no download needed. Windows includes **usbaudio.sys** (USB Audio 1.0) and, **starting with Windows 10 version 1703, usbaudio2.sys** (USB Audio 2.0). A class-compliant headset shows up automatically as something like **"USB Audio Class 2 Device"** (replaced by the product name when the device reports one).
- A **vendor driver** (from the headset maker) provides extra features and can fix model-specific bugs. **If a vendor/partner driver is present on the PC or offered via Windows Update, Windows installs it and it overrides the in-box class driver.** So for plain "no audio" issues the in-box driver is usually fine; install the vendor driver when you need the vendor's features or the maker's support team recommends it.

**Update the driver via Device Manager:**
1. Open **Device Manager**.
2. Expand **Sound, video and game controllers**, right-click the headset > **Update driver** > **Search automatically for drivers.**
3. Follow the prompts and **restart** the PC if asked.

**Reinstall the driver (clears a corrupt driver / Code 28):**
1. In Device Manager, right-click the headset > **Uninstall device.** If **"Attempt to remove the driver for this device"** appears, **check it.**
2. **Restart the PC** (and re-plug the headset). Windows automatically reinstalls the in-box USB Audio driver on the next connection.
3. If it doesn't reappear, click **Action > Scan for hardware changes.**

**Windows Update optional driver updates:**
1. **Start > Settings > Windows Update > Advanced options.**
2. Under **Additional options**, select **Optional updates.**
3. Expand **Driver updates.** If a headset/audio/USB driver is listed, check it and click **Download & install.**

> **Caution to convey:** Optional driver updates are **not** installed automatically and **shouldn't** be installed unless there's a specific problem with that device — a wrong optional driver can introduce new issues. If the vendor publishes its own installer/app, that's usually the better source for vendor-specific drivers.

**Vendor driver/app:** If features like a busy-light, hook-switch/call-control buttons, or sidetone don't work, install the manufacturer's headset app/driver from the maker's official support site, then reboot.

**How to verify it's fixed:** Device Manager shows the headset with **no warning icon** and status **"This device is working properly"**; audio plays and records; and (if a vendor app was installed) the headset's special buttons/lights work in the softphone.

## 2.8 Quick escalation checklist (if all of the above fail)
1. Test the **same headset on a second computer** — if it fails there too, suspect the **hardware/cable**.
2. Test a **second, known-good headset** on the caller's PC — if that one works, the original headset is faulty.
3. Note the **Windows version** (run **winver**), the **Device Manager error code**, and whether the issue is **playback, recording, or both** — and escalate to Tier 2 / the headset vendor with those details.

---

# 3. Manufacturer-Specific USB Headset Troubleshooting

> **For the agent:** A USB headset is two things at once — a plug-and-play **audio device** (sound in/out, which almost always "just works") and a **call-control device** (the answer/end/mute/volume buttons and mute LED, which depend on a vendor companion app *plus* the right softphone settings). Most "my headset is broken" tickets are really call-control, firmware, or USB-path problems, not dead hardware. Identify the **brand and exact model** first (look on the headset, the boom arm, or the USB dongle), because the right companion app and the "Teams vs UC variant" question both hinge on it.

Each fix below is framed as **Likely cause → Check → Steps → Verify.**

## 3.1 Jabra (Evolve / Evolve2, Engage; Jabra Direct / Xpress; Link 380/390)

**Companion app — Jabra Direct** (end-user desktop app; download from jabra.com). It runs in the background and bridges the headset's buttons to the PC softphone. It handles firmware updates (headset **and** dongle), device settings (sidetone, audio bandwidth, busylight, button mapping), call control, and **preferred-softphone** selection. **Jabra Xpress** is the IT mass-deployment counterpart (builds an MSI for SCCM rollout, can *lock* settings so they appear greyed-out in Direct on the user's PC). If a user can't change a setting in Direct, IT may have locked it via Xpress.

**Firmware update (Jabra Direct):** Connect device → select it in Direct → click **Update now** → **Update** next to the device → pick language → **Update** → do **not** disconnect until done. For wireless models, the **Link 380/390 dongle has its own firmware** — update both; a headset/dongle firmware mismatch is a top cause of mute-sync and Teams-button failures.

**Common USB issues**
- **Device not detected.** *Steps:* move the dongle to a **different port** (older Jabra dongles like the Link 370 are flaky in USB 3.x — try a **USB 2.0** port); reinstall Jabra Direct; in Device Manager uninstall the Jabra device and reboot. *Verify:* device shows in Direct and in Windows Sound.
- **Call-control buttons do nothing.** *Check:* Direct → **Device settings → Softphone (PC)**. *Steps:* turn **Call control with softphone** ON; set **Preferred softphone** to the app in use; also select Jabra as the audio device *inside* the softphone. *Verify:* pressing answer/end on the headset drives the call. (Note: the **Engage 50/50 II headset has no on-headset buttons** — call control lives on the optional **Engage Link** controller or the softphone UI.)
- **Mute LED out of sync with Teams.** *Steps:* update Teams + Jabra firmware/Direct (headset **and** dongle); in Teams **Settings → Devices** toggle device-button sync off/on; restart Teams; if persistent, factory-reset the headset (typically hold Mute ~10 s until the LED flashes). *Verify:* headset mute and Teams mute icon move together.
- **Sidetone.** Direct → **Device settings → Headset → Sidetone**; default 0 dB, raise to +3/+6 dB to hear more of yourself.
- **Busylight not reflecting status.** Direct → **Device settings → Headset → Headset busylight** (on), and **Settings → Softphone Integration → Presence Synchronization** ON (softphone must support Jabra integration).

**Gotchas**
- **MS vs UC variant.** Evolve2/Engage and the dongles ship in **MS (Teams-certified)** and **UC** SKUs. MS = auto-default device, dedicated **Teams button**, Teams LED notifications. **UC = no Teams button/LED with Teams**, but certified for Cisco/Mitel/Avaya/Zoom. The variant is firmware-baked and **not field-switchable** — order the right SKU.
- **Link 380 vs 390.** Both are USB Bluetooth dongles managed in Direct. **390** is newer (Bluetooth 5.3, HD Voice). Each comes as **a (USB-A)** or **c (USB-C)**, in MS and UC — e.g. 380a, 390c. Match the laptop's port.
- **Replacement dongle must be re-paired** to the headset in Jabra Direct, and its variant (MS/UC) and connector (A/C) must match.

## 3.2 Poly / Plantronics (Blackwire, Voyager, Savi; Poly Lens vs Plantronics Hub; BT700)

**The single most important Poly fact: there are two apps, and newer gear uses Poly Lens, not Plantronics Hub.**
- **Poly Lens Desktop App** (download from **poly.com/lens**) is the current, recommended app and the official **replacement for the deprecated Plantronics Hub**. It manages firmware, device settings (sidetone, mute tones/alerts, language packs), call-control button config, and **target softphone** selection.
- **Plantronics Hub** is **legacy/headset-only** — keep it only for old hardware (e.g. Savi W740, CS500/CS540) or old OSes where Lens won't install.

**Which app for which product (use Lens unless told otherwise):**

| Product | App |
|---|---|
| Voyager Focus 2, Voyager 4300 (4310/4320), Voyager Free 60/60+ | **Poly Lens** |
| Blackwire 8225 | **Poly Lens** (settings also in Hub) |
| Blackwire 3200/5200 | **Poly Lens** for firmware (settings in both) |
| Savi 7200/8200 DECT | **Poly Lens** for firmware (settings also in Hub) |
| BT700 adapter | **Poly Lens** (re-pairing) |
| Legacy Savi W740, CS500/CS540 | **Plantronics Hub** |

**Firmware update (Poly Lens):** Connect device → open Lens → select it → Lens auto-checks; click **Update**. For a specific version (or rollback) use **Manage → Software Versions**. *Gotcha:* the **Voyager Free 60+ touch-screen charging case firmware updates only via Poly Lens Desktop over USB**.

**Common USB issues**
- **Device not detected.** *Steps:* replug **directly into the PC** (not a dock/hub); in **Windows Sound → Manage sound devices** ensure the Poly device is **Enabled** for Playback and Recording; update/roll back the audio driver in Device Manager (especially after a Windows update); reinstall Poly Lens. **Prefer the BT700 dongle over the PC's built-in Bluetooth.**
- **Call-control buttons not working.** *Steps:* install **Poly Lens**; in Lens select the headset → **Buttons** tab → ensure Attend/End Call enabled; set the correct **target softphone** (wrong target causes "answer button hangs up").
- **BT700 pairing/re-pairing.** Replug the BT700 directly → Poly Lens → select **Poly BT700** → **Pair New Device** (adapter LED flashes **red/blue**) → put headset in pair mode → success when LED goes **solid** and you hear "PC connected."
- **Sidetone gotcha:** on the **Blackwire 8225, sidetone is currently disabled** in firmware in **both** Lens and Hub.

**Gotchas**
- **BT700 comes in USB-A and USB-C** — order the matching connector; the dongle (not raw PC Bluetooth) gives full call control and longer range.
- **Teams vs UC SKUs:** Teams SKUs map a dedicated Teams button; UC SKUs are platform-neutral.
- **Plantronics Hub is being retired** — pointing a Voyager Focus 2 / 4300 / Free 60 / Blackwire 8225 user at Hub is the wrong path.

## 3.3 Logitech (Zone series; Logi Tune)

**Companion app — Logi Tune** (Desktop + mobile; download from logitech.com/tune). The **only** correct app for Zone business headsets. Controls EQ, **sidetone** (dial: higher = more of your own voice), mic gain/test, ANC (e.g. Zone 950), the **busy light** (off by default — enable in Tune), voice prompts, and firmware.

**Firmware update (Logi Tune):** Connect → open Logi Tune → select the device under **MY DEVICES** → click the **(i) info icon** → **Check for update** → **UPDATE** → **Done**. Keep it connected throughout. **Logitech's standalone Firmware Update Tool is retired for Zone — use Logi Tune.**

**Common USB issues**
- **Not detected / no audio.** Plug **directly into the PC, not a hub/adapter**; try another port; for wired earbuds make sure the 3.5 mm plug is fully seated in the controller; for receivers, replug or re-pair.
- **Call-control buttons not working.** *Steps:* **use the included USB receiver, not Bluetooth** — some apps don't support full mute/call control over BT; quit/reopen the call client; reselect the audio device. Logitech states answer/end behavior **varies by softphone** and the button won't *start* a call. (E.g. the physical mute button does **not** sync with **Five9** due to HID limitations.)
- **Receiver pairing (Zone Wireless 2):** plug the USB-C receiver → power on headset → Logi Tune → **Zone Receiver** → **Pair headset** → put headset in pairing (slide power to Bluetooth icon, hold ~2 s, flashes blue) → both LEDs go **solid white**.

**Gotchas**
- **Use Logi Tune — NOT G HUB (gaming) and NOT Logi Options+ (mice/keyboards).** Wrong app = no controls/firmware shown. The single most common Zone mistake.
- **Teams vs UC variants have different part numbers.** Teams has the dedicated Teams button; UC omits it.
- **USB-A vs USB-C receivers:** a **replacement receiver must be re-paired in Logi Tune**.

## 3.4 EPOS / Sennheiser (ADAPT / IMPACT; EPOS Connect; BTD 800)

**Brand history (the gotcha):** Sennheiser Communications' enterprise line became **EPOS** in 2020; older units may still say "Sennheiser." The old **HeadSetup / HeadSetup Pro** software was merged into **EPOS Connect** (desktop) and **EPOS Manager** (cloud/IT). **Use EPOS Connect for all current ADAPT/IMPACT gear; uninstall legacy HeadSetup Pro so two clients don't fight over the device.**

**Companion app — EPOS Connect** (Windows/Mac, plus mobile, plus an **EPOS Connect for Web** that does firmware updates without installing software). Handles firmware, ANC/sidetone, **call control** (answer/end/volume/mute with mute sync), **default-softphone** selection, and busylight.

**Firmware update (EPOS Connect):** Connect device → open EPOS Connect → **Update Overview** tab → **Check for Updates** → click the update icon → don't disconnect.

**Common USB issues**
- **Device not detected.** Try a different USB port; **avoid KVMs, port replicators, docking stations, and hubs** — connect directly. In Windows Sound, set the EPOS device as **Default Communication Device** on both Playback and Recording. Restart the EPOS background service and relaunch.
- **Call-control buttons not working.** EPOS Connect **must be installed and running**. *Known failure:* the **default-softphone field goes blank**. *Fix:* close EPOS Connect + softphone → reinstall the softphone plugin/connector → restart EPOS Connect → **set the default softphone** → then launch the softphone.
- **Sidetone/Busylight/ANC** are all in EPOS Connect device settings.
- **BTD 800 dongle pairing.** Usually pre-paired. To pair: plug in, **hold the button ~3 s** (LED alternates **blue/red**). LED: dimmed blue = connected, **purple = Microsoft Teams**. Clear pairing list: in pair mode, **double-press** the button.

**Gotchas**
- **Teams variant** (often a **"T"** suffix) has a dedicated Teams button and Teams signaling (BTD 800 lights **purple**); the standard UC unit lacks the certified Teams button.
- **IMPACT 5000 = the SDW 5000 DECT system** (base + headset over DECT; SDW D1 USB dongle for PC).

## 3.5 Yealink (UH / WH series; Yealink USB Connect; DECT bases)

**Companion app — Yealink USB Connect** (Windows/Mac). Manages firmware, device status/battery, **sidetone**, EQ, busylight, button functions, and call control (mute + hook-state sync). **Hard limit: USB Connect only works over a direct USB connection** — a Bluetooth-only connection is invisible to it. For IT at scale: **YMCS** (cloud) or **YDMP** (on-prem).

**Firmware update (USB Connect):**
- **Wired UH / BT-via-dongle:** connect via USB → open USB Connect → select device → **Update device → Update now**.
- **DECT WH6x (two-stage):** connect the **base** to power + PC USB → **dock the headset on the base** → **Update device → Update Now** (up to ~5 min). The **base updates first, then the docked headset**. Don't unplug or undock mid-update. **DECT base and headset have separate firmware** — a "Teams button won't connect" symptom is often a stale **base** that needs updating and re-docking. (Desk-phone use needs phone firmware **≥ 86**.)

**Common USB issues**
- **Not detected.** Try a different port; on DECT bases note **two micro-USB ports** (one PC, one desk phone — use the right one); power-cycle the base (unplug power 10 s); update USB Connect; set as default in Windows Sound.
- **Call control / Teams button dead (audio fine).** *Likely cause:* a **conflicting softphone/legacy client running alongside Teams.** Microsoft: *"Teams doesn't support button controls on connected certified peripherals if third-party collaboration and conferencing apps are running at the same time."* **Uninstall the conflicting/old client, reboot.** Also update base + headset firmware; ensure the UC↔Teams platform switch in USB Connect (UC SKU only) matches the app.
- **Sidetone:** raise it in **Yealink USB Connect** (the only place to set it).
- **DECT pairing/subscription.** Pair by **docking the headset** (LED green ~5 s). Subscription **persists** through power-off/undock; clearing it needs a factory reset. Add up to **4 headsets** per base (hold the base **PC button ~5 s**, then press the primary headset's call-control button). Factory reset: hold **Computer + Desk Phone buttons together 6 s**, or via USB Connect → **Device Support → Restore factory settings**.
- **Busylight:** built-in (UH38) or external **BLT60** for WH6x — it shows **one** device's presence, so set the intended app as the default audio/dialer device.

**Gotchas:** Teams vs UC SKUs differ in hardware/firmware. USB Connect is required for firmware/tuning and only over USB.

## 3.6 Microsoft / generic certified USB-HID headsets

**Why some buttons "just work" with no app at all — USB HID.** A standard USB headset enumerates as USB Audio Class (no driver) plus a **HID interface**. Telephony controls live on the **HID Telephony page (0x0B)** and **Consumer page (0x0C)** — hook switch (off-hook = answer, on-hook = end), mute toggle, volume. Windows/macOS understand these natively, so on a well-implemented device **answer/end/mute and the mute LED work without any vendor software**. Vendor apps add firmware, sidetone, EQ, and busylight config — not basic call control.

**What Microsoft Teams certification guarantees:** plug-and-play with no config, auto-selection as default device, basic call control (answer/hang-up, mute, volume) on Windows and Mac, and firmware-update capability. **New-Teams** certification adds a dedicated **Teams button + LED**. **Critical gotcha (verbatim):** *"Teams doesn't support button controls on connected certified peripherals if third-party collaboration and conferencing apps are running at the same time."*

**When there's no companion app** (e.g. basic Microsoft Modern USB Headset, OEM units): call control still works via the OS HID stack, but you can't tune sidetone/EQ/busylight/firmware. If a generic device's buttons don't work, it's almost always an unsupported HID usage in that specific app or an app conflict — **not** a missing driver.

## 3.7 Cross-Manufacturer Topics

### Call-control / HID — answer/end/mute/volume not working
**Mental model:** the headset only emits a raw HID event ("hook off," "mute toggle"). Something on the PC must translate it into a softphone action — either the **softphone itself** (Teams/Zoom/Webex natively, for certified devices) or the **vendor companion app** acting as a bridge. Buttons fail when the bridge is missing, misconfigured, losing an arbitration fight, or the wrong device variant is in use.

**Diagnostic checklist (in order):**
1. **Install and run the companion app** — and where it matters (Jabra especially), **launch the app *before* the softphone**. If Jabra Direct loads after the softphone, call control can break; restart Direct first.
2. **Enable softphone integration + mute sync** in the app, and **select the correct target/preferred softphone** (wrong target = "button does nothing" or "answer hangs up").
3. **Set the headset as the default communication device** both in **Windows Sound** and **inside the softphone** (Settings → Devices/Audio).
4. **Enable HID / "use headset buttons" in the softphone.** In **Teams** this is **Settings → Devices → "Sync device buttons."** Turn it **OFF** when you need another UC app to co-exist with the headset's HID; ON to let Teams sync the buttons.
5. **Match MS vs UC variant to the platform.**
6. **Resolve multi-app HID conflicts.** Use Teams' **"Sync device buttons"** toggle to co-exist; for **Teams + Skype for Business 2016 (Islands mode)** apply the **EnableTeamsHIDInterop** Group Policy (`HKCU\Software\Policies\Microsoft\Office\16.0\Lync\EnableTeamsHIDInterop = 1`). Note Teams' mute-sync can **override a physical mute after a few seconds** — a known "my mute un-mutes itself" cause.
7. **Platform-specific limits exist** — e.g. Five9 may not honor a headset's HID mute. Clean-reinstall the companion app if buttons silently stop after an app update.

### Firmware update best practices & risks
Firmware writes to the device's flash; **interrupting it can brick the device or force a vendor recovery/reflash.** Manufacturer guidance converges on:
1. **Never disconnect/unplug** the headset, dongle, or base mid-update.
2. **Plug directly into a PC USB port** — not a hub, dock, monitor-USB, or KVM.
3. **Close softphone/conferencing apps** during the update.
4. **Wireless/DECT/dongle headsets: keep them docked and charged** for the whole update; don't undock.
5. **Don't update across a KVM or virtual-desktop USB redirection** — the device re-enumerates and drops.
6. **Don't close the companion app or sleep/shut down** until it reports success.

### Docking stations / USB hubs / monitors-with-USB — the #1 cause of intermittent USB audio
USB audio is **isochronous and power-sensitive**, so it's unusually intolerant of the bandwidth contention, power limits, and re-enumeration churn that docks and hubs add. Classic symptoms: audio **stutters/drops on a ~30–60 second cycle**, the headset **disappears and won't re-enumerate** after a dock power-sequence, or resource-allocation errors from insufficient bus power.

**Practical rule:** For any intermittent USB-audio dropout, re-enumeration, or "headset disappears," **move the headset or its dongle to a USB port directly on the PC** (prefer a rear/native USB 3.x port), bypassing docks, monitor-USB ports, hubs, and KVMs. Direct connection is both the diagnostic test and, very often, the fix. If the dock must stay: try a **powered** hub, update the **dock firmware**, and check **USB selective-suspend / port power-management** settings.

---

# 4. Genesys Cloud (WebRTC) USB Headset Troubleshooting

This section is for the agent at the desk. Work top to bottom. Most "my headset doesn't work in Genesys" cases are solved in the first three subsections (device selection + Chrome mic permission + the pop-out window). Each issue is framed as **likely cause → what to check → exact steps → how to verify**.

## 4.1 How Genesys Cloud uses WebRTC, the browser, and your headset

Genesys Cloud's WebRTC phone carries call audio directly through your **browser** using WebRTC and the **OPUS** audio codec — there is no desk phone. Genesys recommends a Chromium browser (**Google Chrome** or **Microsoft Edge**) or the **Genesys Cloud desktop app**. Firefox works but is limited (no Advanced Mic Settings, no Jabra/Poly call-control). Safari is not supported for headset call control.

There are **two layers of device selection**, and both must point at your USB headset:
1. **Windows OS layer** — Windows decides which device is "Default." Genesys uses your Windows **default communication device** for incoming-call *alerts/ringtone* unless you tell it otherwise.
2. **Genesys WebRTC layer** — Inside Genesys you separately pick the **microphone** and **speaker** used for the actual *call conversation*, plus an optional separate **ringer** device.

They interact like this: the **ring** can come out of your computer speakers (Windows default) while the **call** goes to your headset — that is intentional and configurable. If audio is "wrong," it is almost always because one of these two layers is pointed at the wrong device.

**Verify which device Genesys is using:** open the WebRTC phone settings (below) — the microphone/speaker dropdowns should show your headset by name (e.g., "Jabra Evolve2 65"), not "Default - Communications" or "Realtek."

## 4.2 Selecting microphone, speaker, and ringer in Genesys Cloud

**Likely cause:** Genesys is sending/receiving audio on the laptop's built-in mic/speakers instead of the USB headset.

**Exact steps (Chrome or Edge):**
1. In the Genesys Cloud client, go to **Menu (top-right) > More > Settings > WebRTC**. (Or, from the popped-out WebRTC Phone window: click the **Arrow > Open WebRTC Settings > Settings**.)
2. Under **Audio Controls**, set **Microphone** to your headset by name.
3. Set **Speaker** to your headset by name.
4. Click **Refresh** next to the device list if your headset just got plugged in or is missing — Refresh also restarts the headset vendor software (Jabra Direct / Plantronics Hub).
5. Click **Speaker** (blue speaker icon) to play test tones — you should hear them in the headset.

**Ringer / "play ringtone on separate device":** To make the **ring play on computer speakers while the call stays in the headset**: enable **Play ringtone on separate device** and select your computer speakers; leave the call **Speaker** set to the headset.

**Device profiles:** Once you select a headset and (for call control) approve it, Genesys saves a named **profile** and **automatically re-recognizes that headset on future connections** — so you don't reconfigure every shift. If you swap headsets you may need to pick the new device and create a new profile.

**How to verify:** Run a station/test call (§4.5). You should hear the test tone in the headset and see your mic level move when you speak.

## 4.3 Browser microphone permissions (Chrome site settings) — the #1 "no mic" cause

**Likely cause:** When Chrome first asked "Allow microphone?", **Allow** was not clicked (or it was blocked). WebRTC then has no mic access regardless of which device is selected. The classic symptom is *"…does not have permission to access microphone"* or your voice not being heard.

**Exact steps:**
1. With the Genesys Cloud tab open, click the **lock / tune icon** at the left of the Chrome address bar.
2. Open **Site settings** and set **Microphone** to **Allow**.
3. Alternatively go to `chrome://settings/content/microphone`, confirm the Genesys URL (e.g., `apps.mypurecloud.com`) is in **Allowed**, not **Blocked**, and that the **default microphone** at the top is your headset.
4. **Reload** the Genesys tab so the new permission takes effect.

**Firefox equivalent:** if it keeps re-prompting, check **Remember this decision** then **Allow**. If you previously chose "Don't Allow," change the URL's Microphone from **Block** to **Allow** in site permissions.

**How to verify:** Reload, open WebRTC settings, click **Test Settings** — the mic test should pass and the level meter should respond to your voice.

## 4.4 Embedded / iframe scenarios (Salesforce, Zendesk) — you usually must pop out

**Likely cause:** Genesys is embedded inside another app (Salesforce, Zendesk) as an **iframe**. Browsers restrict mic and **HID** (headset-button) access inside iframes, so the mic prompt may never appear or call-control buttons don't work.

**Exact steps:**
1. **Pop out the phone:** **Menu > More > Settings > WebRTC > select "Pop WebRTC Phone window."** This opens a standalone window that *can* prompt for and hold the microphone permission and HID access. For Salesforce specifically, this pop-out is required to use Jabra/Yealink headset buttons.
2. **Allow pop-ups** for the embedding site, or the window won't appear (common error: "WebRTC Phone window unable to display").
3. Grant the mic permission in the popped-out window (lock icon > Microphone > Allow), then reload.
4. **For administrators** (escalate if you can't self-serve): the embedding iframe must include `allow="camera *; microphone *; autoplay *"`, and Embeddable Framework deployments may need additional **HID** permissions for headset call control.

**How to verify:** In the popped-out window, run **Test Settings**; the mic test passes and (for supported headsets) the answer/mute buttons respond.

## 4.5 Built-in WebRTC diagnostics / test call — and what good vs bad looks like

**Run it:**
1. Select **Calls > Phone Settings**.
2. Click **Run Diagnostics**. (Prereqs: logged in, voicemail configured, no persistent phone connection active.)
3. Wait for the tests to finish, then click **Test Results** for metrics.

**What it tests:** Streaming Connection, WebRTC Station, Call Connected, Call Quality, plus a Network Test (DNS + connectivity to AWS / Genesys media). A quick in-settings check is also available: the **Speaker** button plays test tones (Chrome) and **Test Settings** runs the mic diagnostic.

**Good vs bad (target thresholds):**

| Metric | Good (pass) | Bad (investigate/escalate) |
|---|---|---|
| MOS (Mean Opinion Score) | 4–5 | below ~3.5 |
| Packet loss | < 1% | ≥ 1% |
| Round-trip time (latency) | < 150 ms | ≥ 150 ms |
| Jitter | < 30 ms | ≥ 30 ms |

If all tests pass but audio is still bad, the problem is local (device selection, mic settings, or the headset itself). If the **Network Test** or quality metrics fail, it's a network/firewall issue — escalate to IT with the **Test Results** screenshot.

## 4.6 Headset call control (answer / end / mute / hold from the headset)

**What's supported:** Genesys Cloud passes audio for *any* USB headset, but **built-in button call control** (answer, hang up, hold/resume, mute/unmute, reject) is only supported on specific vendors: **Jabra, Poly/Plantronics, Sennheiser/EPOS, Yealink, and Cyber Acoustics**.

**Vendor requirements:**
- **Jabra:** Chrome/Edge, the desktop app, or embedded clients. Connect by **wired USB or a Jabra USB Bluetooth adapter/dongle**. On first use, approve the prompts and pick the device in Chrome's **WebHID "Connect"** dialog, confirm it as default device, and save a profile. Jabra Direct recommended. **Not** supported on Firefox/Safari.
- **Poly/Plantronics:** **Plantronics Hub software must be installed**; connect via **USB**. Works in browser client, desktop app, and embedded clients.
- **Sennheiser/EPOS:** install **EPOS Connect** software first.

**Key limitation — Bluetooth:** A headset paired through the computer's **internal/built-in Bluetooth** adapter does **not** get call control. You must use a wired USB connection or the vendor's own USB Bluetooth dongle.

**Desktop app vs browser:** Call control works in both for the supported vendors. The browser path requires **WebHID** (Chromium only). The desktop app needs the **latest version**.

**How to verify:** Place a test call; pressing the headset's hook/mute button should answer/mute in the Genesys call controls (the on-screen Mute toggles in sync).

## 4.7 Known limitations and advanced mic tuning

- **AirPods / consumer Bluetooth earbuds:** Not in the supported call-control list and prone to WebRTC quirks (mono "hands-free" mode kills audio quality). Prefer a wired USB or vendor-dongle headset.
- **Desktop app vs Chrome differences:** **Advanced Mic Settings only appears in Chrome.** If a setting you read about isn't visible, you may be in the desktop app or a non-Chrome browser.
- **Advanced Mic Settings** (Chrome only): **Menu > More > Settings > WebRTC > Advanced Mic Settings**. Clear (disable) these only to fix a specific symptom:
  - **Automatic Mic Gain (AGC)** — **disable** if mic volume keeps fluctuating up and down.
  - **Echo Cancellation** — **disable** only if you don't use open speakers (pointless in a headset-only setup).
  - **Noise Suppression** — before disabling, first **move the mic boom closer and speak up** — over-aggressive suppression can clip a quiet voice.
- Close other apps that grab the mic (Teams, Zoom, Webex), select your headset *by name* (not "Default"), and use **Refresh** after plugging in.

## 4.8 Network / QoS notes (act on what you can, escalate the rest)

WebRTC audio is real-time and unforgiving of network problems. As a tier-1 agent you can:
- **Use a wired Ethernet connection** instead of Wi‑Fi when possible.
- **Disconnect from VPN** if call audio is choppy — VPNs/proxies add latency and can mangle UDP media. Genesys explicitly recommends dropping the VPN to test.
- **Disconnect and re-place the call**, **clear browser cache**, and **log out/in** (or restart the desktop app).

Escalate to IT/admin when **Run Diagnostics** shows packet loss ≥ 1%, latency ≥ 150 ms, jitter ≥ 30 ms, or MOS below ~3.5, or the **Network Test** can't reach AWS/Genesys media. Hand IT a screenshot of **Test Results**. Admins should ensure **QoS prioritization** of Genesys voice traffic, allow **32–128 Kbps bidirectional per concurrent call**, run the **Genesys Cloud Network Readiness Assessment**, and confirm firewall rules match Genesys's recommended ranges.

---

# 5. Consolidated Sources

### Windows / Microsoft
- [Fix sound or audio problems in Windows — Microsoft Support](https://support.microsoft.com/en-us/windows/fix-sound-or-audio-problems-in-windows-73025246-b61c-40fb-671a-2535c7cd56c8)
- [Fix audio issues when no sound plays — Microsoft Support](https://support.microsoft.com/en-us/windows/fix-audio-issues-when-no-sound-plays-from-speakers-or-headphones-in-windows-684eb0bb-824e-4003-9755-f263067341fa)
- [Fix missing or undetected audio output device in Windows — Microsoft Support](https://support.microsoft.com/en-us/windows/fix-missing-or-undetected-audio-output-device-in-windows-5504aed3-2c01-4214-89d1-9e8dbe6828e8)
- [Turn on app permissions for your microphone in Windows — Microsoft Support](https://support.microsoft.com/en-us/windows/turn-on-app-permissions-for-your-microphone-in-windows-94991183-f69d-b4cf-4679-c98ca45f577a)
- [Fix microphone problems — Microsoft Support](https://support.microsoft.com/en-us/windows/fix-microphone-problems-5f230348-106d-bfa4-1db5-336f35576011)
- [Error codes in Device Manager in Windows — Microsoft Support](https://support.microsoft.com/en-us/topic/error-codes-in-device-manager-in-windows-524e9e89-4dee-8883-0afa-6bca0456324e)
- [USB Audio 2.0 Drivers — Microsoft Learn](https://learn.microsoft.com/en-us/windows-hardware/drivers/audio/usb-2-0-audio-drivers)
- [USB Audio Class System Driver (Usbaudio.sys) — Microsoft Learn](https://learn.microsoft.com/en-us/windows-hardware/drivers/audio/usb-audio-class-system-driver--usbaudio-sys-)
- [Automatically get recommended and updated hardware drivers — Microsoft Support](https://support.microsoft.com/en-us/windows/automatically-get-recommended-and-updated-hardware-drivers-0549a8d9-4842-8acb-75fa-a6faadb62507)
- [Phones and Devices for Microsoft Teams (USB devices) — Microsoft Learn](https://learn.microsoft.com/en-us/microsoftteams/devices/usb-devices)
- [EnableTeamsHIDInterop (Teams + Skype for Business Islands mode) — Microsoft Support](https://support.microsoft.com/en-us/topic/enableteamshidinterop-for-coordination-of-hid-device-usage-in-microsoft-teams-and-skype-for-business-2016-in-islands-mode-5119085d-fd0f-030b-9d83-9f6f93c73de1)
- [Disable USB Selective Suspend in Windows 11 — PCNMobile](https://pcnmobile.com/disable-usb-selective-suspend-settings-in-windows-11/)
- [I fixed random USB disconnects in Windows by changing these settings — How-To Geek](https://www.howtogeek.com/i-fixed-random-usb-disconnects-in-windows-by-changing-these-settings/)

### Manufacturers
- [Jabra Direct support](https://www.jabra.com/supportpages/jabra-direct) · [MS vs UC variants](https://www.jabra.com/supportpages/jabra-link-380/14208-24/faq/how-do-the-microsoft-teams-certified-devices-differ-from-the-uc-variants) · [Firmware update via Jabra Direct](https://www.jabra.com/supportpages/jabra-evolve-75/7599-832-199/faq/how-do-i-manually-update-the-firmware-on-my-jabra-device-using-jabra-direct) · [Jabra Xpress whitepaper](https://www.jabra.com/-/media/Images/SA-Pages/Microsoft/Practical-Guidance/PDF/Jabra-Xpress_Whitepaper_DEC17_UPDATE.pdf)
- [Poly Lens supported devices](https://info.lens.poly.com/docs/begin/supported-devices) · [Updating device software (Poly Lens)](https://docs.poly.com/bundle/poly-lens-da/page/updating-device-software.html) · [Pairing with BT700](https://docs.poly.com/bundle/poly-lens-da/page/pairing-devices-with-the-poly-bt700-bluetooth-usb-adapter.html) · [Plantronics Hub vs Poly Lens — HeadsetAdvisor](https://headsetadvisor.com/blogs/headset/plantronics-hub-vs-poly-lens)
- [Logi Tune App](https://www.logitech.com/en-us/video-collaboration/software/logi-tune-software.html) · [Logitech firmware update guide](https://www.logitech.com/en-us/discover/a/update-firmware-on-devices) · [Pair Zone Wireless 2 to a replacement receiver](https://prosupport.logi.com/hc/en-001/articles/17916253912215-Pair-your-Zone-Wireless-2-headset-to-a-replacement-receiver)
- [EPOS Connect](https://www.eposaudio.com/en/us/software/epos-connect) · [Updating EPOS firmware](https://www.eposaudio.com/en/us/support/knowledge-base/software/epos-connect/epos-connect---updating-your-epos-products-firmware) · [USB headsets connectivity troubleshooting](https://www.eposaudio.com/en/us/gaming/support/knowledge-base/headsets/usb-headsets/usb-headsets---connectivity-troubleshooting) · [BTD 800 dongle](https://www.eposaudio.com/en/us/support/knowledge-base/adapt-line/btd-800-dongle-series)
- [Yealink USB Connect](https://www.yealink.com/en/product-detail/usb-connect-management) · [Update WH6X firmware](https://www.yealink.com/en/blog/how-to-update-yealink-wh6x-firmware-through-yealink-usb-connect) · [WH62/WH63 troubleshooting (Intermedia)](https://support.intermedia.com/app/articles/detail/a_id/28224/~/yealink-wh62-/-wh63-troubleshooting-guide)
- [Use Microsoft Modern USB Headset in Teams — Microsoft Support](https://support.microsoft.com/en-gb/topic/use-microsoft-modern-usb-headset-in-microsoft-teams-2e906422-2766-48ad-954c-1ebdd94c5f18)

### Genesys Cloud
- [Change your WebRTC phone settings](https://help.genesys.cloud/articles/change-your-webrtc-phone-settings/) · [Audio issues with WebRTC phones](https://help.genesys.cloud/articles/audio-issues-with-webrtc-phones/) · [Troubleshoot the WebRTC phone](https://help.genesys.cloud/articles/troubleshoot-genesys-cloud-webrtc-phone/) · [Error messages with WebRTC phones](https://help.genesys.cloud/articles/error-messages-with-webrtc-phones/)
- [Test your WebRTC phone settings](https://help.genesys.cloud/articles/test-your-webrtc-phone-settings/) · [Test your microphone](https://help.genesys.cloud/articles/test-your-microphone/) · [Test your media settings](https://help.genesys.cloud/articles/test-media-settings/) · [Run the built-in WebRTC Diagnostics app](https://help.genesys.cloud/articles/run-the-built-in-genesys-cloud-webrtc-diagnostics-app/)
- [Configure advanced microphone settings](https://help.genesys.cloud/articles/configure-advanced-microphone-settings-for-webrtc-phones/) · [Configure headsets for embedded clients](https://help.genesys.cloud/articles/configure-headsets-for-embedded-clients/) · [Configure a Jabra headset](https://help.genesys.cloud/articles/configure-a-jabra-headset/) · [Configure a Plantronics/Poly headset](https://help.genesys.cloud/articles/configure-a-plantronics-headset/)
- [Customer network readiness](https://help.genesys.cloud/articles/customer-network-readiness/) · [Run the Network Readiness Assessment](https://help.genesys.cloud/articles/run-the-genesys-cloud-network-readiness-assessment/)

---

> **Source-verification note:** Microsoft Learn / Support, EPOS, Logitech, and Genesys Resource Center pages were fetched directly and are first-party. Several manufacturer pages (jabra.com, docs.poly.com, support.yealink.com) are JavaScript-rendered and were corroborated via search extracts plus reputable IT/reseller KBs and community threads; exact UI menu paths (which vendors rename between app versions) should be confirmed against the live cited URLs before publishing externally.

*Companion documents: `PRD.md` (product requirements), `TODO.md` (original build plan).*
