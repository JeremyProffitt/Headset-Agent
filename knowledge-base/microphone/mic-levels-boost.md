---
section: microphone
topic: mic_levels_boost
platforms: [windows10, windows11]
---

# Adjusting Microphone Input Level and Gain in Windows

**Scope:** How to find and adjust the microphone's input level in Windows 10 and Windows 11, plus automatic gain and app-level gain settings. Use this when the mic works but the level needs tuning — too quiet, too loud, or fluctuating. This supports Tree 2.

## Likely Cause

The Windows input volume for the headset mic is set too low (or too high), an **automatic gain control** is pumping the level up and down, or the softphone's own mic setting is overriding what Windows shows.

## Check

1. Confirm the headset's microphone is the selected **Input** device in Windows Sound settings — adjusting the level of the wrong mic changes nothing.
2. Speak at a normal volume and watch the **input level meter** in Sound settings. A healthy setup shows clear movement as you talk, without slamming to the top.

## Steps

1. **Set the input level in Settings.**
   - **Windows 11:** Start > Settings > System > Sound. Under **Input**, select the headset's microphone and adjust the input **volume slider**. About 80% is a good starting point.
   - **Windows 10:** Start > Settings > System > Sound. Under **Input** ("Choose your input device"), select the headset mic and adjust its volume slider.
2. **Classic Levels tab (works on Windows 10 and 11):** press Windows key + R, type **mmsys.cpl**, press Enter. Go to the **Recording** tab, right-click the headset microphone, choose **Properties**, and open the **Levels** tab. Adjust the microphone level there. If the device exposes an extra boost/gain slider on this tab, raise it gently — large boosts amplify background noise along with the voice.
3. **While you're in Properties, check the Listen tab:** make sure **"Listen to this device" is unchecked**. When accidentally enabled, the mic loops back to the headset in real time — the classic "I can hear myself" complaint.
4. **Automatic gain (AGC):** if the mic level keeps fluctuating up and down on its own, in Genesys Cloud on **Chrome** go to **Menu > More > Settings > WebRTC > Advanced Mic Settings** and disable **Automatic Mic Gain (AGC)**. This menu only appears in Chrome — if you don't see it, you may be in the desktop app or a non-Chrome browser.
5. **App-level gain:** set the softphone's microphone to the headset **by name** (not "Default"), and use the app's mic test — in Genesys Cloud, **Test your microphone** — to confirm the level inside the app. Some manufacturers' companion apps (for example **Logi Tune** for Logitech Zone headsets) also offer their own mic gain control and mic test; check there if installed.

## Verify

Speak normally and confirm the input level meter moves well — strong, steady movement without pinning at maximum. Place a test call and confirm the other party hears you at a comfortable, consistent volume with no pumping up and down. If the level is still wrong at the extremes of every slider, go back to Tree 2 for device-selection and driver checks.
