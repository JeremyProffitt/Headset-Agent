---
application: genesys_cloud
platform: chrome
category: setup
difficulty: intermediate
---

# Genesys Cloud Desktop - Chrome Setup

## Overview
Genesys Cloud WebRTC phone runs in Chrome browser and requires specific settings for audio to work correctly.

## Prerequisites
- Google Chrome (version 89+)
- Genesys Cloud account with WebRTC phone enabled
- USB or Bluetooth headset connected to computer
- Windows audio correctly configured

## Initial Setup

### Step 1: Chrome Microphone Permissions
Chrome must have permission to access your microphone.

**Check Site Permissions**:
1. Navigate to Genesys Cloud in Chrome
2. Click the lock/tune icon (🔒) left of the URL
3. Click "Site settings"
4. Find "Microphone" - ensure it says "Allow"

**User Prompt**: "In Chrome, look at the address bar where you see the website address. There's a little lock or tune icon on the left side. Click that, then 'Site settings'. Find 'Microphone' and make sure it says 'Allow'."

**If blocked**:
1. Change "Microphone" to "Allow"
2. Refresh the page (Ctrl + F5)

### Step 2: Verify Headset Selected in Chrome
Chrome may have its own audio device settings.

**Check Chrome Settings**:
1. Click the three dots menu (⋮) in Chrome
2. Settings > Privacy and security > Site settings
3. Scroll to "Microphone"
4. Verify your headset is selected in the dropdown

### Step 3: Genesys Cloud WebRTC Phone Settings
Configure the phone within Genesys Cloud.

**Navigation**:
1. In Genesys Cloud, click "Calls" in the left sidebar
2. Click the Settings icon (gear) in the Phone Details panel
3. Review the "Audio Controls" section

**Required Settings**:
- **Microphone**: Select your headset microphone
- **Speaker**: Select your headset speakers
- **Ringer**: Can be headset or computer speakers

**User Prompt**: "In Genesys Cloud, click on 'Calls' over on the left. Then look for a little gear icon for settings. In there, we need to make sure your headset is selected for both the microphone and the speaker."

## Audio Profile Setup

### Creating an Audio Profile
When you first connect a USB headset, Genesys Cloud may prompt you.

**If prompted "Create a device profile?"**:
1. Click "Yes"
2. Verify the device names are correct
3. Enter a profile name (e.g., "Jabra Evolve2")
4. Click "Save"

**Manual Profile Creation**:
1. In Phone Settings, click "Create profile"
2. Select your output device (headset speakers)
3. Select your input device (headset microphone)
4. Name the profile
5. Save

**User Prompt**: "Did you see a popup asking to create a profile for your headset? If you did, click 'Yes' and give it a name you'll remember."

## Running Diagnostics

### WebRTC Diagnostics Test
Genesys Cloud includes a built-in diagnostic tool.

**Navigation**:
1. In Phone Settings, find "Run Diagnostics" or "Test Connection"
2. Click to start the test
3. Wait for all tests to complete
4. Review any failures

**What It Tests**:
- Microphone access and level
- Speaker output
- Network connectivity
- WebRTC capability
- Codec support

**User Prompt**: "There's a test we can run to check everything. In those phone settings, look for 'Run Diagnostics' or 'Test Connection'. Click that and let it run through all the checks."

### Interpreting Results

| Test | Pass | Fail |
|------|------|------|
| Microphone | Green check | Check Chrome permissions |
| Speaker | Green check | Check output device selection |
| Network | Green check | Check firewall/proxy |
| WebRTC | Green check | Update Chrome |

## Advanced Settings

### Pop WebRTC Phone Window
For embedded clients (Salesforce, Zendesk) where audio doesn't work.

**Why**: Embedded iframes have limited microphone access in Chrome.

**Navigation**:
1. In Phone Settings, find "Pop WebRTC Phone window"
2. Enable this option
3. When receiving calls, a separate window opens with full permissions

**User Prompt**: "If you're using Genesys inside another app like Salesforce, there's a special setting we need. Look for 'Pop WebRTC Phone window' and turn that on. It'll open calls in a separate window that has full microphone access."

### Advanced Microphone Settings
Fine-tune audio quality settings.

**Navigation**:
1. In Phone Settings, click "Advanced Mic Settings"
2. Available options:
   - **Automatic Mic Gain**: Turn OFF if volume fluctuates
   - **Echo Cancellation**: Keep ON with speakers, OFF with closed headset
   - **Noise Suppression**: Turn OFF if voice sounds robotic

**User Prompt**: "If your audio sounds weird - like robotic or cutting out - there are some advanced settings we can tweak. Look for 'Advanced Mic Settings'."

## Headset-Specific Configuration

### Jabra Headsets
- Works natively with WebRTC
- Call controls work with Chrome WebHID (Chrome 89+)
- Jabra Direct optional but recommended

### Poly/Plantronics
- **Plantronics Hub REQUIRED** for call controls
- Poly Lens NOT supported
- Must select specific device name (not "Default")

### Yealink
- Works via USB or DECT dongle
- Call controls work in Chrome browser only
- NOT supported in desktop app

## Troubleshooting

### No Audio in Calls
1. Check Chrome microphone permissions
2. Verify headset selected in Genesys phone settings
3. Run diagnostics
4. Check Windows audio settings

### Microphone Not Working
1. Check Chrome site permissions for microphone
2. Check Windows privacy settings for microphone
3. Ensure headset microphone selected (not laptop mic)
4. Run diagnostics and check microphone test

### Audio Quality Issues
1. Check Advanced Mic Settings
2. Ensure using correct audio profile (not Stereo for BT headsets)
3. Check network connectivity
4. Close other applications using audio

### Call Controls Not Working
1. Verify headset brand compatibility
2. Check for required software (Plantronics Hub, etc.)
3. Ensure Chrome is up to date (89+)
4. Try disconnecting and reconnecting headset
