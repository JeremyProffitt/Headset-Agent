---
section: windows
subsection: "2.6"
topic: audio_services_troubleshooter
platforms: [windows10, windows11]
---

# §2.6 Restart the Windows Audio Services and Run the Built-In Troubleshooter

**Scope:** How to restart the Windows Audio and Windows Audio Endpoint Builder services, and how to run the built-in audio troubleshooter to fix a "stuck" audio engine.

**Likely cause:** The Windows audio engine has gotten into a bad state (common symptoms: a red X on the speaker icon, "Audio service is not running," or audio vanishing system-wide).

## Restart the Two Audio Services

1. In the taskbar search box type **services** and open the **Services** app (or **Windows key + R** > **services.msc** > Enter).
2. Find **Windows Audio**, right-click it > **Restart.**
3. Find **Windows Audio Endpoint Builder**, right-click it > **Restart.** (This service builds the list of audio endpoints; if it's stuck, devices won't appear. Restarting it also restarts Windows Audio, which depends on it.)
4. While there, confirm each service's **Startup type** is **Automatic**: right-click > **Properties** > set **Startup type: Automatic** > **Apply** > **OK**. The dependency **Remote Procedure Call (RPC)** must also be running (it normally is by default).

## Run the Built-In Audio Troubleshooter

- **Win11:** **Start > Settings > System > Troubleshoot > Other troubleshooters** > find **Audio** > **Run.** (Or open the **Get Help** app and run the automated audio troubleshooter.)
- **Win10:** **Start > Settings > Update & Security > Troubleshoot** > **Additional troubleshooters** > **Playing Audio** > **Run the troubleshooter** (and **Recording Audio** for mic issues).

## USB Power Management — Disable Selective Suspend (for Intermittent Drops)

This is the highest-value fix for intermittent USB audio drops:
- In **Device Manager**, for the headset device **and** for each **USB Root Hub / Generic USB Hub**, open **Properties > Power Management** and **uncheck "Allow the computer to turn off this device to save power."**
- In **Power Options > advanced settings**, set **USB selective suspend** to **Disabled**.

## When to Reboot or Reinstall the Driver

If restarting the services and running the troubleshooter doesn't restore audio — or the device still shows a yellow triangle — do a **full reboot**. If problems persist after reboot, move to the driver reinstall in `win-2.7-drivers.md`.

## How to Verify It's Fixed

Both services show status **Running**, the troubleshooter reports problems fixed or "no issues," the speaker icon no longer shows a red X, and audio plays/records normally.
