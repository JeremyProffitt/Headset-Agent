---
headset_type: USB
issue_category: no_audio
platforms: [windows]
brands: [all]
difficulty: beginner
---

# USB Headset - No Audio Output

## Problem Description
User reports no audio output from USB headset. Headset appears to be connected but no sound is heard.

## Prerequisites
- USB headset is physically available
- User has access to Windows sound settings
- Headset was previously working or is new

## Quick Diagnosis Questions
1. "Is your headset currently plugged into a USB port on your computer?"
2. "Did you hear a chime when you plugged it in?"
3. "Are the volume controls on the headset turned up?"

## Troubleshooting Steps

### Step 1: Verify Physical Connection
Check that the USB cable is fully inserted into the computer's USB port.

**User Prompt**: "Is the USB cable firmly connected to your computer? Try giving it a gentle push to make sure it's seated properly."

**If loose**: "Go ahead and unplug it, wait about 5 seconds, then plug it back in firmly. You should hear a little chime from Windows."

**If already firm**: Proceed to Step 2.

### Step 2: Check USB Port
Try a different USB port on the computer.

**User Prompt**: "Let's try a different USB port. If you're using a port on the front of your computer, try one on the back instead - those tend to be more reliable."

**If works on different port**: Original port may be faulty. Issue resolved.

**If still no audio**: Proceed to Step 3.

### Step 3: Check Windows Sound Settings
Navigate to Windows sound settings.

**Navigation**:
1. Right-click the speaker icon in the bottom-right corner of the taskbar
2. Click "Sound settings" or "Open Sound settings"
3. Under "Output", look for your headset in the dropdown list

**User Prompt**: "Let's check your Windows sound settings. See that speaker icon down in the corner of your screen? Right-click on it and choose 'Sound settings'."

**If headset not listed**: Proceed to Step 4 (Device Manager).

**If headset is listed**: Proceed to Step 5 (Set as Default).

### Step 4: Check Device Manager
Open Device Manager to check for driver issues.

**Navigation**:
1. Right-click the Start button
2. Click "Device Manager"
3. Expand "Sound, video and game controllers"
4. Look for your headset (may show yellow warning triangle)

**User Prompt**: "Right-click on the Start button and choose 'Device Manager'. Then look for 'Sound, video and game controllers' and click the little arrow to expand it."

**If yellow triangle present**: "There's a driver issue. Right-click on the headset and choose 'Update driver', then 'Search automatically for drivers'."

**If not listed at all**: Check "Universal Serial Bus controllers" for unknown devices.

### Step 5: Set as Default Audio Device
Make the headset the default output device.

**Navigation**:
1. In Sound Settings, click the dropdown under "Choose your output device"
2. Select your USB headset from the list
3. Test by playing any audio

**User Prompt**: "In those Sound settings, find where it says 'Choose your output device' and click the dropdown. Do you see your headset in that list?"

**If yes**: "Great! Click on it to select it, then try playing some audio - maybe a YouTube video."

**If no**: The headset may not be recognized by Windows. Try Steps 2-4 again or check for specific driver requirements.

### Step 6: Check Volume Mixer
Ensure volume isn't muted in the mixer.

**Navigation**:
1. Right-click the speaker icon
2. Click "Open Volume Mixer"
3. Ensure headset volume is up and not muted

**User Prompt**: "One more check - right-click that speaker icon again and choose 'Volume Mixer'. Make sure none of the sliders are all the way down or muted."

## Resolution Confirmation
**Ask**: "Can you hear audio now? Try playing a video or some music."

## If Unresolved
If the issue persists after all steps:
- Check if headset works on a different computer
- Consider driver download from manufacturer website
- Recommend escalation if hardware defect suspected

## Common Brands - Specific Notes

### Jabra
- Jabra Direct software may be required for full functionality
- Check for firmware updates

### Poly/Plantronics
- Plantronics Hub required for call controls
- Poly Lens NOT supported with some applications

### Logitech
- Logi Tune software available for configuration
- G HUB for gaming headsets
