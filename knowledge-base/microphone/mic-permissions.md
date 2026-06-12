---
section: microphone
topic: mic_permissions
platforms: [windows10, windows11]
---

# Microphone Permissions — Windows Privacy Settings and Apps Holding the Mic

**Scope:** Windows microphone privacy permissions (the system-wide switch and per-app access) plus apps holding the mic exclusively. This is the #1 cause of "the agent/customer can't hear me" and supports Tree 2.

## Likely Cause

Windows privacy controls are blocking the app's access to the microphone — extremely common after Windows updates and on managed or corporate machines. It produces the classic symptom: **the headset works for listening, but no one can hear the caller.** The other frequent culprit: **another app is holding the mic**, or exclusive mode let one app seize the device and lock everyone else out.

## Check

Three nested toggles must all be **On**: the master mic switch, "let apps access," and — for installed desktop softphones — "let desktop apps access." Also check how many apps are open that could be using the mic: Teams, Zoom, Discord, other browser tabs in a call, voice recorders.

## Steps

1. **Windows 11 privacy settings:**
   1. Start > Settings > **Privacy & security**, then under **App permissions** select **Microphone**.
   2. Turn **Microphone access** **On** — this is the master switch for the whole PC; if it's off, nothing can use the mic.
   3. Turn **Let apps access your microphone** **On**.
   4. Under that, scroll the app list and turn **On** the specific softphone or conferencing app if it's listed.
   5. If the softphone is a traditional installed program (most call-center clients, Teams desktop, Zoom), scroll down and turn **On** **Let desktop apps access your microphone**. Desktop apps don't each get their own toggle — this single switch governs all of them. **This is the one people miss.**
2. **Windows 10 privacy settings:**
   1. Start > Settings > **Privacy**, then under **App permissions** select **Microphone**.
   2. Click **Change** and ensure **Microphone access for this device** is **On**.
   3. Turn **Allow apps to access your microphone** **On**.
   4. Under **Choose which Microsoft Store apps can access your microphone**, turn the relevant app **On**.
   5. Scroll down and turn **On** **Allow desktop apps to access your microphone**.
3. **Browser-based softphones:** the browser also keeps its own permission. For Genesys Cloud in a browser, confirm the browser has granted **microphone permission** to the Genesys site — check the site-permission prompt or the lock icon in the address bar.
4. **Close apps holding the mic.** Make sure only one app is using the microphone: close Teams, Zoom, Discord, other browser tabs in a call, voice recorders, and so on. Two apps fighting for the mic is a top cause of "no mic" and "device busy."
5. **Stop exclusive locks.** Press Windows key + R, type **mmsys.cpl**, press Enter. On the **Recording** tab, right-click the headset microphone > **Properties** > **Advanced** tab, and **uncheck** both "Allow applications to take exclusive control of this device" and "Give exclusive mode applications priority." This stops one app from seizing the mic and locking the softphone out.

## Verify

Go to **Settings > System > Sound** (Windows 11) or the **Recording** tab of **mmsys.cpl**, speak into the headset, and watch the **input level meter move** as you talk. Then place a quick test call in the softphone and confirm the other party can hear you.
