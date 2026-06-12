---
section: microphone
topic: mic_too_quiet
platforms: [windows10, windows11]
---

# Microphone Too Quiet — Others Say You Sound Faint

**Scope:** The other party can hear the caller, but only faintly. The mic works — its level is just too low. This supports Tree 2; for "the *other person* is too quiet in my ear," that's output volume — see Tree 6.

## Likely Cause

The mic boom is positioned too far from the mouth, the Windows **input volume** is set low, over-aggressive **noise suppression** is clipping a quiet voice, or the softphone's own mic level or automatic gain is misbehaving.

## Check

1. Confirm the mic boom (if any) is rotated **down toward the mouth**, not up or away.
2. Confirm the headset microphone — not a distant webcam or laptop mic — is the selected **Input** device in Windows Sound settings. A far-away mic picks you up faintly.
3. Speak at a normal volume and watch the **input level meter** in Windows Sound settings. A bar that barely moves confirms a low input level.

## Steps

1. **Position first.** Move the mic boom closer and speak up — this alone fixes many "you're faint" complaints, especially when noise suppression is clipping a quiet voice.
2. **Raise the Windows input volume.**
   - **Windows 11:** Start > Settings > System > Sound. Under **Input**, select the headset's microphone and raise the input **volume slider** — about 80% is a good starting point.
   - **Windows 10:** Start > Settings > System > Sound. Under **Input**, confirm the headset mic is selected, then raise the input volume slider.
   - For the classic view on either version: press Windows key + R, type **mmsys.cpl**, press Enter, go to the **Recording** tab, right-click the headset microphone, choose **Properties**, and check the **Levels** tab — raise the microphone level there.
3. **Check the softphone's mic level.** In the app's audio settings, select the headset mic by name (not "Default"). For Genesys Cloud, run **Test your microphone** to confirm the level.
4. **If the volume keeps fluctuating up and down** (loud, then faint, then loud): in Genesys Cloud on Chrome, go to **Menu > More > Settings > WebRTC > Advanced Mic Settings** and disable **Automatic Mic Gain (AGC)**. Note this menu only appears in Chrome, not the desktop app.
5. **If a quiet voice keeps getting cut off:** before disabling **Noise Suppression** in those same Advanced Mic Settings, first try moving the boom closer and speaking up — over-aggressive suppression clips a quiet voice.
6. **Vendor app gain:** some manufacturers' companion apps include a mic gain setting and mic test (for example, Logi Tune for Logitech Zone headsets). If installed, check the mic gain there.

## Verify

Speak at a normal volume and confirm the input level meter in Windows Sound settings moves well as you talk. Then place a test call and confirm the other party hears you at a comfortable, steady level. If you are still faint at maximum input volume with the boom positioned correctly, suspect hardware — update the driver, and escalate if unresolved.
