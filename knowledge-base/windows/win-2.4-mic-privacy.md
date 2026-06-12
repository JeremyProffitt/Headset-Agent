---
section: windows
subsection: "2.4"
topic: microphone_privacy
platforms: [windows10, windows11]
---

# §2.4 Microphone Privacy Settings — The #1 Cause of "The Agent/Customer Can't Hear Me"

**Scope:** How to check and fix Windows microphone privacy settings that block app access to the mic.

**Likely cause:** Windows privacy controls are blocking the app's access to the microphone. This is extremely common after Windows updates and on managed/corporate machines, and it produces the classic symptom: **the headset works for listening, but no one can hear the caller.**

**What to check:** Three nested toggles must all be **On**: the master mic switch, "let apps access," and (for installed/desktop softphones) "let desktop apps access."

## Exact Steps — Windows 11

1. **Start > Settings > Privacy & security**, then under **App permissions** select **Microphone**.
2. Turn **Microphone access** **On** — this is the master switch for the whole PC; if it's off, nothing can use the mic.
3. Turn **Let apps access your microphone** **On.**
4. Under that, scroll the app list and turn **On** the specific softphone/conferencing app if it's listed.
5. If the softphone is a traditional installed program (most call-center clients, Teams desktop, Zoom), scroll down and turn **On** **Let desktop apps access your microphone.** Desktop apps don't each get their own toggle — this single switch governs all of them. **This is the one people miss.**

## Exact Steps — Windows 10

1. **Start > Settings > Privacy**, then under **App permissions** select **Microphone.**
2. Click **Change** and ensure **Microphone access for this device** is **On.**
3. Turn **Allow apps to access your microphone** **On.**
4. Under **Choose which Microsoft Store apps can access your microphone**, turn the relevant app **On.**
5. Scroll down and turn **On** **Allow desktop apps to access your microphone.**

## How to Verify It's Fixed

Go to **Settings > System > Sound** (Win11) or the **Recording** tab of **mmsys.cpl**, speak into the headset, and watch the **input level / volume meter move** as you talk. Then place a quick test call in the softphone and confirm the customer can hear them.
