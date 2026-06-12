// Prompt catalog for the triage engine (B-05). The B-04 trees reference copy
// by PromptRef key (triage.Step.ReadAloudKey / triage.Terminal.ReadAloudKey);
// this file holds the actual spoken text, condensed for voice from
// troubleshooting.md §1. Keeping the copy here (not in the tree data) lets
// WS-E swap pacing/phrasing without touching navigation.
//
// Completeness is enforced by TestPromptCatalogCoversAllTreeKeys in
// triage_turn_test.go: every ReadAloudKey reachable from BuildSymptomFork(),
// every engine-fired handoff.* key, and the engine's tree3.s1.resolved key
// must have an entry.
package handlers

import "github.com/headset-support-agent/internal/triage"

// PromptText returns the spoken copy for a triage prompt key.
func PromptText(key triage.PromptRef) (string, bool) {
	text, ok := promptCatalog[key]
	return text, ok
}

// promptOrDefault returns the catalog copy for key, or def when absent.
func promptOrDefault(key triage.PromptRef, def string) string {
	if text, ok := PromptText(key); ok {
		return text
	}
	return def
}

var promptCatalog = map[triage.PromptRef]string{
	// -----------------------------------------------------------------------
	// Universal Pre-Flight Checklist
	// -----------------------------------------------------------------------
	"preflight.s1": "Let's start with the quickest fix. Unplug your headset and plug it directly into a USB port on the computer itself — not into a monitor, keyboard, dock, or USB hub. Wait about ten seconds for Windows to recognize it. Did that fix the problem?",
	"preflight.s2": "Next, let's make sure the headset is the selected device. In Windows Sound settings and in your calling app's device picker, confirm the headset is the chosen device — computers often silently default to the laptop speakers or a webcam mic. Did selecting the headset fix it?",
	"preflight.s3": "Now check the hardware mute. Look for a physical mute button, an inline mute switch on the cable, or mute-when-the-boom-arm-is-up — many headsets show a red light when muted. Make sure it's unmuted. Did that fix it?",
	"preflight.s4": "Let's turn the volume up — both on the headset's inline volume control and in Windows. Also open the Windows volume mixer and make sure your calling app isn't at zero. Did that fix it?",
	"preflight.s5": "Make sure only one app is using the microphone. Close Teams, Zoom, Discord, voice recorders, and any other browser tabs that might be in a call — two apps fighting over the mic is a top cause of trouble. Did closing the other apps fix it?",
	"preflight.s6": "Let's do a full restart. Reboot the computer, then plug the headset back in. If you're using Genesys, also refresh the browser tab or restart the desktop app. Once you're back up, is the problem fixed?",

	"preflight.s1.resolved":   "Perfect — going directly into the computer did it. Docks and hubs cause a lot of these issues, so keep it plugged straight in. You're all set!",
	"preflight.s2.resolved":   "Great — it was just pointing at the wrong device. You're all set now!",
	"preflight.s3.resolved":   "There we go — it was the hardware mute. Easy to hit by accident. You're all set!",
	"preflight.s4.resolved":   "Great — just a volume setting. You're good to go!",
	"preflight.s5.resolved":   "That was it — two apps fighting over the microphone. You're all set!",
	"preflight.s6.resolved":   "A fresh restart clears most stuck audio states. Glad it's working — you're all set!",
	"preflight.route_to_tree": "The quick checks didn't catch it, so let's dig into the specific symptom.",

	// -----------------------------------------------------------------------
	// Tree 1 — No Audio Output
	// -----------------------------------------------------------------------
	"tree1.s1": "Is the headset plugged directly into the computer and recognized — did Windows make its connect chime, or does the headset show up in the sound devices?",
	"tree1.s2": "In Windows Sound settings, check whether the headset is set as the output — the playback — device. If it wasn't, select it now. Did that bring the sound back?",
	"tree1.s3": "Let's check volume and mute everywhere: Windows volume up and unmuted, the inline dial on the headset turned up, no hardware mute, and in the Windows volume mixer make sure your calling app isn't muted or at zero. Did that bring the sound back?",
	"tree1.s4": "Let's test sound outside the calling app. Play any sound — a video clip, or the Test button in the device's properties. Do you hear the test sound in the headset?",
	"tree1.s5": "Windows is fine, so the problem is inside the calling app. In its audio settings, set the output or speaker to the headset — not 'Default' and not the speakers. In Genesys Cloud, set the WebRTC phone's speaker to the headset and run 'Test your media settings' to play a test tone. Did that restore the call audio?",
	"tree1.s6": "Let's work the Windows layer, in order: run the Windows audio troubleshooter; make sure the device isn't disabled under 'Manage sound devices'; try a different USB port; reboot; then update or reinstall the audio driver. Did any of that bring the sound back?",

	"tree1.s1.route_tree3": "It sounds like the computer isn't seeing the headset at all — let's fix detection first.",
	"tree1.s2.resolved":    "That did it — Windows was sending sound to the wrong device. You're all set!",
	"tree1.s3.resolved":    "There we go — a volume or mute control was the culprit. You're all set!",
	"tree1.s5.resolved":    "That fixed it — the app was pointing at the wrong output. You're all set!",
	"tree1.s5.escalate":    "The headset works fine in Windows but stays silent in calls, so this looks like a platform problem beyond your desk — likely the WebRTC phone or browser side. I'm connecting you with a specialist and passing along everything we verified.",
	"tree1.s6.resolved":    "The Windows-layer fix did it. You're all set!",
	"tree1.s6.escalate":    "Even after a reboot and a driver reinstall there's still no sound anywhere, so I suspect the headset hardware. I'm connecting you with a specialist, with notes on everything we tried.",

	// -----------------------------------------------------------------------
	// Tree 2 — Microphone / Other Party Can't Hear Me
	// -----------------------------------------------------------------------
	"tree2.s1": "First, the most common cause: hardware mute. Check the physical mute button, the inline mute switch on the cable, and whether the boom arm mutes when raised — many headsets show a red light when muted. Unmute at the headset and try again. Can they hear you now?",
	"tree2.s2": "In Windows Sound settings, make sure the headset's microphone is the input — the recording — device, not a webcam mic or the laptop's built-in mic. Did selecting the right mic fix it?",
	"tree2.s3": "Let's see whether Windows hears the mic at all. In Windows Sound settings, speak normally and watch the microphone level bar. Does the bar move when you talk?",
	"tree2.s4": "Windows hears you, so the problem is in the calling app. Set its microphone to the headset. In Genesys Cloud: pick the headset mic in the WebRTC phone, run 'Test your microphone', make sure the browser has granted microphone permission to Genesys, and check that you're not muted in the call controls. Did that fix it?",
	"tree2.s5": "Let's check the Windows side. In Settings, under Privacy and security, Microphone: make sure microphone access is on and apps are allowed to use it. Close every other app that might be holding the mic, try a different USB port, reboot, and then update or reinstall the audio driver. Does the level bar move now?",

	"tree2.s1.resolved": "That was it — the hardware mute. It's the single most common cause. You're all set!",
	"tree2.s2.resolved": "Fixed — Windows was listening to the wrong microphone. You're all set!",
	"tree2.s4.resolved": "That did it — the app's mic selection or permission was the problem. You're all set!",
	"tree2.s4.escalate": "Windows hears your mic fine, but the calling app still doesn't — that points to a platform issue beyond your desk. I'm connecting you with a specialist, with everything we checked.",
	"tree2.s5.resolved": "There we go — Windows can hear the mic again. You're all set!",
	"tree2.s5.rma":      "After a reboot, a driver reinstall, and a known-good USB port, the microphone still picks up nothing — that points to a failed mic or boom. I'm opening a hardware replacement case and connecting you with someone to arrange it.",

	// -----------------------------------------------------------------------
	// Tree 3 — Headset Not Detected
	// -----------------------------------------------------------------------
	"tree3.s1": "Unplug the headset and plug it directly into a USB port on the computer — bypass any dock, hub, or monitor — and listen for the Windows connect chime. Does the headset show up now?",
	"tree3.s2": "Try a different USB port — ideally on the other side of the machine, and try USB-A versus USB-C if you have both. Does it appear now?",
	"tree3.s3": "If you can, try a different computer, or a different cable if the cable detaches. Does the headset work on the other machine?",
	"tree3.s4": "Let's check Device Manager. Look under 'Sound, video and game controllers' and 'Audio inputs and outputs' for the headset, a yellow warning icon, or an unknown device. If it shows an error, uninstall the device, then scan for hardware changes or reboot, and update the driver if needed. Does the headset show up properly now?",
	"tree3.s5": "One more thing — is this a managed or locked-down work machine? Company policy can block USB audio devices entirely. Do you think policy or USB restrictions could be blocking it?",

	"tree3.s1.route_original": "Good news — it's detected now. Let's get back to the problem you originally called about.",
	"tree3.s1.resolved":       "It's showing up now — plugging directly into the computer did it. You're all set!",
	"tree3.s2.resolved":       "It's detected on the new port — the original port is likely faulty, so stick with this one. You're all set!",
	"tree3.s3.rma":            "It fails on every port and every machine, which strongly suggests a dead headset or cable. I'm opening a hardware replacement case and connecting you with someone to arrange it.",
	"tree3.s4.resolved":       "The driver reinstall brought it back. You're all set!",
	"tree3.s5.escalate":       "Since this is a managed machine, a policy block has to be lifted by IT — that can't be overridden from the desk. I'm connecting you with Tier 2 and IT, with notes on everything we checked.",
	"tree3.s5.rma":            "Detection fails even on a known-good machine with no policy in the way — that's a hardware fault. I'm opening a replacement case and connecting you with someone to arrange it.",

	// -----------------------------------------------------------------------
	// Tree 4 — One-Sided Audio / Mono
	// -----------------------------------------------------------------------
	"tree4.s1": "In the headset's playback properties, open Levels and check the balance — make sure left and right are equal and neither is at zero. A skewed balance slider is the most common cause. Did fixing the balance bring both ears back?",
	"tree4.s2": "In Settings, under Accessibility, Audio, check the 'Mono audio' toggle — it should be off unless you need it on. Did turning it off fix it?",
	"tree4.s3": "Let's isolate it. Play a known stereo test — a clip with separate left and right channels — outside the calling app. Do both ears work on the stereo test?",
	"tree4.s4": "Gently wiggle the cable near the plug and the earcup joint while audio plays, reseat the USB, and try a different port. Does the audio cut in and out when you move the cable?",

	"tree4.s1.resolved":   "That did it — the balance slider was skewed. You're all set!",
	"tree4.s2.resolved":   "Fixed — the Mono audio toggle was on. You're all set!",
	"tree4.s3.resolved":   "Good news — the headset is fine. Voice calls are typically mono by design, so hearing the same thing in both ears on calls is expected. If it only happens in one specific app, check that app's audio settings.",
	"tree4.s4.rma_cable":  "Audio cutting in and out as the cable moves means a failing cable or connector. I'm opening a hardware replacement case and connecting you with someone to arrange it.",
	"tree4.s4.rma_earcup": "One ear staying dead regardless of balance, settings, source, and port points to a failed speaker in that earcup. I'm opening a hardware replacement case and connecting you with someone to arrange it.",

	// -----------------------------------------------------------------------
	// Tree 5 — Distorted / Choppy / Echo
	// -----------------------------------------------------------------------
	"tree5.s1":  "Which best describes it: robotic or choppy, like it's cutting in and out; crackling or static; or an echo, where you or the other person hear a repeat of the voice?",
	"tree5.s2":  "Robotic or choppy audio is usually the connection, not the headset. Prefer wired Ethernet over Wi-Fi, get off the VPN if you can, and close bandwidth-heavy apps and extra tabs to free up the network and CPU. In Genesys Cloud, run 'Test your media settings', and try restarting the desktop app or refreshing the browser. Did that clear it up?",
	"tree5.s3":  "Crackling usually means USB or driver. Plug the headset directly into the PC — off any dock or hub — and try a different port. Then in the device's Advanced properties, disable audio enhancements and try lowering the sample rate, and update or reinstall the audio driver. Did the crackling stop?",
	"tree5.s4":  "Who hears the echo — do you hear your own voice repeated back, or does the other person hear an echo?",
	"tree5.s4b": "For echo on the other side: make sure you're on the headset, not open speakers; lower the speaker volume; make sure echo cancellation isn't disabled; and in Genesys confirm the correct headset mic and speaker are selected so echo cancellation can work. Did that stop the echo?",

	"tree5.s2.resolved":  "Great — a cleaner connection or freeing up the computer cleared it. You're all set!",
	"tree5.s2.escalate":  "It persists even on a good wired connection, so this looks like a network or platform issue beyond your desk. I'm connecting you with a specialist, with everything we verified.",
	"tree5.s3.resolved":  "The crackling is gone — it was the USB path or the driver. You're all set!",
	"tree5.s3.rma":       "The crackle survives multiple ports and a driver reinstall, so I suspect the hardware or cable. I'm opening a replacement case and connecting you with someone to arrange it.",
	"tree5.s4b.resolved": "That stopped the echo. You're all set!",
	"tree5.s4b.escalate": "The echo persists even with a proper headset setup, so I'm bringing in a specialist to look at the platform side, with notes on what we checked.",

	// -----------------------------------------------------------------------
	// Tree 6 — Volume / Sidetone
	// -----------------------------------------------------------------------
	"tree6.s1": "Is this about how loud the other person sounds, or about hearing your own voice in the headset?",
	"tree6.s2": "Let's adjust call volume, in order: the headset's inline volume dial; Windows volume and the volume mixer for the app; then the output level under the device's Properties and Levels — and check Loudness Equalization if it's available. If it's only quiet in calls, also check the app's own call volume. Is the volume good now?",
	"tree6.s3": "That's called sidetone — how much of your own voice the headset plays back. If your headset's vendor app — Poly Lens, Logi Tune, or Jabra Direct — has a sidetone control, adjust it there. Also check the Windows culprit: in the Recording tab, under the mic's Properties, Listen tab, make sure 'Listen to this device' is unchecked. Did that sort it out?",

	"tree6.s2.resolved": "Volume sorted. You're all set!",
	"tree6.s2.escalate": "Maximum volume is still too quiet everywhere, even after a driver update, so I suspect the hardware or driver. I'm connecting you with a specialist, with notes on what we tried.",
	"tree6.s3.resolved": "Sidetone sorted. You're all set!",
	"tree6.s3.escalate": "Your model has the sidetone feature but the control isn't taking effect, which points to firmware or driver. I'm connecting you with a specialist, with notes on what we tried.",

	// -----------------------------------------------------------------------
	// Tree 7 — Mute Sync / Call-Control Buttons
	// -----------------------------------------------------------------------
	"tree7.s1": "Is this about the buttons — answer, end, or mute — doing nothing at all, or about the mute state disagreeing between the headset and the app?",
	"tree7.s2": "Quick expectation check: Genesys Cloud gives audio to any USB headset, but the built-in call-control buttons only work on specific supported models. Is your headset model on the Genesys supported call-control list?",
	"tree7.s3": "Supported call control needs the manufacturer's helper app — Poly Lens or Plantronics Hub, Jabra Direct, Logi Tune, or EPOS Connect. Make sure it's installed and running, with the headset showing as connected in it. Did that enable the buttons?",
	"tree7.s4": "Let's restart the link: re-plug the headset directly, restart the vendor app, then restart the Genesys desktop app or refresh the browser tab so call control re-handshakes. Are the buttons working now?",
	"tree7.s5": "Let's re-sync the mute. Toggle mute once on each side so they match a known state — unmute on the headset, then unmute in the app — and place a fresh test call. Do the headset and app mute track together now?",
	"tree7.s6": "Make sure the vendor app is installed and running — it's what relays the mute state — then restart the vendor app and the Genesys app or browser tab to rebuild the link. Is mute staying in sync now?",

	"tree7.s2.resolved": "Mystery solved — that model isn't on the supported call-control list, so the buttons not controlling calls is expected behavior, not a fault. Answer and mute in the app itself; your audio still works normally.",
	"tree7.s3.resolved": "The vendor app did it — the buttons are alive. You're all set!",
	"tree7.s4.resolved": "Restarting the link re-handshook call control. You're all set!",
	"tree7.s4.escalate": "Supported model, vendor app running, links restarted — and the buttons are still dead. I'm escalating to Tier 2, with everything we verified.",
	"tree7.s5.resolved": "They're tracking together again. You're all set!",
	"tree7.s6.resolved": "Rebuilding the link fixed the sync. You're all set!",
	"tree7.s6.escalate": "Mute keeps drifting on a supported model with current vendor software, which points to a firmware or middleware issue — I'm escalating to Tier 2. One safety note until it's fixed: trust the in-app mute indicator, because when the headset and app disagree you risk an open microphone.",

	// -----------------------------------------------------------------------
	// Tree 8 — Intermittent Disconnects / Drops
	// -----------------------------------------------------------------------
	"tree8.s1": "Plug the headset directly into a USB port on the computer itself — docking stations and unpowered hubs are the most common cause of cut-outs. Did the drops stop on a direct port?",
	"tree8.s2": "Try a different direct USB port — a different side of the machine, or USB-A versus USB-C. Is it stable on the other port?",
	"tree8.s3": "Now the highest-value fix: USB power management. In Device Manager, for the headset and each USB Root Hub, open Properties, Power Management, and uncheck 'Allow the computer to turn off this device to save power'. In Power Options, set USB selective suspend to Disabled. Then update the USB and chipset drivers, and the dock's firmware if you use one. Have the drops stopped?",
	"tree8.s4": "Let's rule out the hardware: wiggle the cable near the plug while audio plays, try a different cable if it detaches, and a different computer if you can. Does movement trigger the drops, or does it also drop on another machine?",

	"tree8.s1.resolved": "The drops stopped on a direct port — the dock or hub was the culprit. Keep it plugged straight into the computer, or use a powered hub. You're all set!",
	"tree8.s2.resolved": "Stable on the new port — the old one is flaky, so avoid it. You're all set!",
	"tree8.s3.resolved": "That did it — Windows was putting the USB device to sleep. You're all set!",
	"tree8.s4.rma":      "Movement triggering drops, or drops following the headset to another machine, means a hardware or cable fault. I'm opening a replacement case and connecting you with someone to arrange it.",
	"tree8.s4.escalate": "The headset tests fine elsewhere, so the problem is in this machine's USB path — that needs Tier 2 or IT. I'm escalating, with the full list of what we tried.",

	// -----------------------------------------------------------------------
	// Engine-fired handoffs (triage.EscalationTerminal: "handoff." + reason)
	// -----------------------------------------------------------------------
	"handoff.user_requested":            "Of course — let me get you to a person right away. I'll pass along everything we've covered so you won't have to repeat yourself.",
	"handoff.user_frustrated":           "I'm sorry this has been such a hassle. Let me connect you with a specialist who can take over — I'll hand off everything we've tried so far.",
	"handoff.troubleshooting_exhausted": "We've tried quite a few things without luck, so rather than keep you looping, I'm bringing in a specialist. I'll pass along exactly what we've done so they won't repeat it.",
	"handoff.hardware_fault":            "Everything points to a fault in the headset itself. I'm opening a hardware replacement case and connecting you with someone to arrange it.",
	"handoff.reboot_limit":              "We've rebooted twice and reinstalled the driver with no change, so I won't keep you re-running the same steps. Let me hand this to a specialist, with notes on everything we tried.",
	"handoff.tree_exhausted":            "We've completed the full set of checks for this symptom, including a reboot and a driver reinstall, and it's still happening. I'm escalating to a specialist with the complete list of what we tried.",
	"handoff.managed_machine_policy":    "This looks like a managed-machine policy block, which has to be handled by IT. I'm connecting you with Tier 2, with notes on what we found.",
	"handoff.user_cannot_perform":       "No problem — these steps can be fiddly. Let me connect you with someone who can take it from here.",
	"handoff.accessibility_need":        "Let me route you to a specialist who can support that accommodation properly.",
	"handoff.genesys_platform":          "The headset checks out at the Windows layer, so this looks like a platform issue beyond your desk. I'm escalating to Tier 2 and the Genesys admin, with everything we verified.",
}
