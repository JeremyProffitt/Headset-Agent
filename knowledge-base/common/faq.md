---
category: faq
type: general
---

# Frequently Asked Questions

## General Headset Issues

### Q: My headset isn't producing any sound. What should I check first?
**A**: Start with the basics:
1. Check if the headset is properly connected (USB firmly seated, Bluetooth showing "Connected")
2. Look for a mute switch or volume wheel on the headset itself
3. In Windows Sound Settings, make sure your headset is selected as the output device
4. Check that the volume isn't muted or set very low

### Q: My headset works for audio but the microphone isn't working.
**A**: Microphone issues are common and usually related to settings:
1. In Windows Sound Settings, check the Input section - select your headset microphone
2. For Bluetooth headsets, make sure you're using "Hands-Free" mode, not "Stereo"
3. Check app permissions - some apps need explicit microphone access
4. Try speaking and watch the microphone level indicator move

### Q: Why does my Bluetooth headset show twice in Windows?
**A**: This is normal! Bluetooth audio has two modes:
- **Stereo/A2DP**: High quality audio for music, but NO microphone
- **Hands-Free/HFP**: Lower quality audio, but INCLUDES microphone

For calls, you need to select the "Hands-Free" version. For music, use "Stereo".

### Q: My headset keeps disconnecting randomly.
**A**: This could be several things:
1. Low battery on wireless headset - charge it
2. USB power saving - disable "Allow computer to turn off this device" in Device Manager
3. Interference - move away from WiFi routers, USB 3.0 devices
4. Driver issues - update or reinstall audio drivers

## Genesys Cloud Specific

### Q: Genesys Cloud can't access my microphone.
**A**: This is usually a Chrome permission issue:
1. Click the lock icon next to the URL in Chrome
2. Click "Site settings"
3. Set "Microphone" to "Allow"
4. Refresh the page

### Q: The diagnostics pass but I still can't hear callers.
**A**: Check these settings:
1. In Genesys Phone Settings, verify the correct speaker is selected
2. Make sure the headset volume is up
3. Check Windows Volume Mixer - Genesys/Chrome may be muted
4. Try the "Pop WebRTC window" option if using embedded version

### Q: My Poly headset buttons don't control calls.
**A**: Poly headsets require Plantronics Hub software:
1. Download Plantronics Hub from the Poly website
2. Install and restart Chrome
3. Reconnect your headset
4. Call controls should now work

Note: Poly Lens is NOT supported - you must use Plantronics Hub.

### Q: Voice sounds robotic or keeps cutting out.
**A**: Try these audio quality fixes:
1. In Genesys Advanced Mic Settings, try turning OFF Noise Suppression
2. Check your internet connection - video calls need stable bandwidth
3. Close other applications that might be using the microphone
4. Turn off Automatic Mic Gain if your volume fluctuates

## Windows Settings

### Q: How do I set my headset as the default audio device?
**A**:
1. Right-click the speaker icon in the taskbar
2. Click "Sound settings"
3. Under "Output", click the dropdown and select your headset
4. Under "Input", do the same for the microphone

### Q: Windows doesn't recognize my USB headset at all.
**A**: The headset might not be detected:
1. Try a different USB port (back ports are often more reliable)
2. Check Device Manager for any devices with yellow warning triangles
3. Try the headset on a different computer to rule out hardware issues
4. Check manufacturer website for required drivers

### Q: I can't find sound settings on Windows 11.
**A**: Windows 11 moved some things:
1. Right-click the speaker icon and choose "Sound settings"
2. Or: Settings > System > Sound
3. For more options, click "More sound settings" at the bottom to access the classic panel

## Brand-Specific

### Q: My Jabra headset isn't working properly.
**A**: For Jabra headsets:
1. Download Jabra Direct from jabra.com/direct for updates and settings
2. Make sure firmware is up to date
3. For Bluetooth issues, use the Jabra Link adapter if available
4. Restart the headset by turning it off and on

### Q: My Logitech headset has no audio.
**A**: Logitech troubleshooting:
1. Try installing Logitech G HUB (for gaming headsets) or Logi Tune
2. Check the sidetone/volume controls on the headset
3. Make sure the mute switch isn't engaged
4. Update drivers through the Logitech software

### Q: Can I use AirPods with Genesys Cloud?
**A**: Yes, but with limitations:
1. Connect AirPods via Windows Bluetooth settings
2. Select them in Genesys phone settings
3. Note: Call controls on the AirPods won't work with Genesys
4. Audio quality may be lower than dedicated headsets

## Escalation

### Q: When should I ask for a human agent?
**A**: I'll help as much as I can, but it's time for a human when:
- The same issue keeps happening despite trying multiple solutions
- You need physical hardware inspection or replacement
- The issue involves account settings or permissions I can't access
- You've been troubleshooting for a while and prefer human assistance
- You just want to talk to a person - that's always okay!

Just say "I'd like to speak to someone" and I'll connect you right away.
