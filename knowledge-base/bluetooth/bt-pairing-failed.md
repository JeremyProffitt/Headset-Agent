---
headset_type: Bluetooth
issue_category: pairing_failed
platforms: [windows]
brands: [all]
difficulty: beginner
---

# Bluetooth Headset - Pairing Failed

## Problem Description
User cannot pair their Bluetooth headset with their Windows computer. The device either doesn't appear in the Bluetooth list or pairing fails when attempted.

## Prerequisites
- Bluetooth is available on the computer
- Headset is charged
- Headset is not currently connected to another device

## Quick Diagnosis Questions
1. "Is your headset charged and turned on?"
2. "Is the Bluetooth on your computer turned on?"
3. "Is the headset currently connected to your phone or another device?"

## Troubleshooting Steps

### Step 1: Verify Bluetooth is Enabled
Check that Bluetooth is turned on in Windows.

**Navigation**:
1. Press Windows + I to open Settings
2. Click "Bluetooth & devices"
3. Ensure the Bluetooth toggle is ON

**User Prompt**: "Let's make sure Bluetooth is on. Press the Windows key and the letter I at the same time to open Settings. Then click on 'Bluetooth & devices'."

**If Bluetooth is off**: "Toggle that switch to turn Bluetooth on."

**If no Bluetooth option**: Computer may not have Bluetooth or adapter may be disabled in Device Manager.

### Step 2: Put Headset in Pairing Mode
Each brand has a different method to enter pairing mode.

**User Prompt**: "What brand is your headset?"

**Common Pairing Methods**:

| Brand | Method |
|-------|--------|
| **Jabra** | Press and hold the Answer/End button for 3-5 seconds until LED flashes blue |
| **Poly/Plantronics** | Press and hold the Call button until you hear "pairing" or see flashing LED |
| **Logitech** | Press and hold the Bluetooth button until LED blinks fast |
| **Sony** | Press and hold power button 7 seconds until you hear "Bluetooth pairing" |
| **Bose** | Slide power switch to Bluetooth symbol and hold |
| **Generic** | Usually hold power button 5+ seconds until rapid blinking |

**User Prompt**: "Now we need to put your headset into pairing mode. [Brand-specific instructions]. You should see a rapidly blinking light - usually blue."

### Step 3: Add Device in Windows
Pair the headset from Windows settings.

**Navigation**:
1. In Bluetooth settings, click "Add device"
2. Select "Bluetooth"
3. Wait for your headset to appear
4. Click on your headset to pair

**User Prompt**: "In that Bluetooth settings screen, click 'Add device', then choose 'Bluetooth'. Your computer will start searching. Your headset should appear in a few seconds."

**If headset appears**: "Click on it to start pairing."

**If headset doesn't appear**: Proceed to Step 4.

### Step 4: Troubleshoot Discovery Issues

**Check if headset is connected elsewhere**:
"Is your headset connected to your phone right now? Bluetooth headsets can usually only connect to one device at a time. If it's connected to your phone, disconnect it there first."

**Reset Bluetooth on computer**:
1. Turn Bluetooth off in Settings
2. Wait 10 seconds
3. Turn Bluetooth back on
4. Try adding device again

**User Prompt**: "Let's try resetting the Bluetooth. Turn it off, count to 10, then turn it back on. Then try 'Add device' again."

### Step 5: Forget and Re-pair
If previously paired but failing to reconnect.

**Navigation**:
1. In Bluetooth settings, find the headset under "Paired devices"
2. Click the three dots (⋮) next to it
3. Click "Remove device"
4. Put headset back in pairing mode
5. Add device again

**User Prompt**: "Sometimes we need to start fresh. Look for your headset under 'Paired devices'. Click those three dots next to it and choose 'Remove device'. Then we'll pair it again from scratch."

### Step 6: Check for Interference
Bluetooth can be affected by interference.

**Common sources**:
- USB 3.0 devices (especially on front ports)
- WiFi routers
- Microwave ovens
- Other Bluetooth devices

**User Prompt**: "Is your computer near a WiFi router or using USB 3.0 devices? Sometimes these can interfere with Bluetooth. Try moving a bit further from the router or unplugging any USB devices temporarily."

## Post-Pairing Configuration

### Audio Profile Selection
When Bluetooth headset connects, Windows may show two devices:
- **Stereo** - High quality audio, NO microphone
- **Hands-Free** - Lower quality audio WITH microphone

**User Prompt**: "Now that it's paired, you might see your headset listed twice in Sound settings - one as 'Stereo' and one as 'Hands-Free'. For phone calls where you need the microphone, you'll want to use the 'Hands-Free' one."

## Resolution Confirmation
**Ask**: "Is your headset now showing as 'Connected' in the Bluetooth settings?"

## If Unresolved
- Check if headset pairs with a phone (to rule out headset issue)
- Update Bluetooth drivers in Device Manager
- Check for Windows updates
- Consider Bluetooth adapter issues if built-in Bluetooth

## Common Issues

### "Paired but not connected"
- Click on the device and choose "Connect"
- May need to select as audio output device separately

### "Connection keeps dropping"
- Check battery level on headset
- Update Bluetooth drivers
- Check power management settings (prevent Windows from turning off Bluetooth)
