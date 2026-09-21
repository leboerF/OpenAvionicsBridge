# Installation and Quick Start

> Current version: **OpenAvionicsBridge 1.0.0-rc3**

This page describes the normal setup for the current WinWing/MobiFlight output path.

## Requirements

- Windows 10 or Windows 11, x64
- Microsoft Flight Simulator 2024
- a supported aircraft/build
- MobiFlight running locally
- a WinWing MCDU configured and available in MobiFlight
- OpenAvionicsBridge 1.0.0-rc3

No separate Go runtime is required for the release executable.

## Recommended start order

1. Start MobiFlight.
2. Verify that the WinWing MCDU is available in MobiFlight.
3. Start Microsoft Flight Simulator 2024.
4. Load a supported aircraft.
5. Wait until the aircraft avionics are initialized.
6. Start `OpenAvionicsBridge.exe`.
7. Select **Captain** or **First Officer**.
8. Click **Start bridge**, or enable automatic connection.

## GUI status fields

The current rc3 GUI reports four main states.

### Microsoft Flight Simulator

Typical states:

- `Checking...`
- `Not running`
- `Running (PID ...)`

If MSFS is running but cannot be enumerated, see [Troubleshooting](TROUBLESHOOTING.md).

### Aircraft adapter

Typical states:

- `Waiting for aircraft`
- `No compatible aircraft detected`
- `Supported aircraft detected`

The adapter status is intentionally generic. OpenAvionicsBridge is no longer structured as an F70/F100-only application.

### Build status

After a successful attachment, the build line normally shows either:

```text
Verified build · <profile name>
```

or:

```text
Runtime validated · <profile name>
```

A **Verified build** means the loaded module hash is already known to the profile and the runtime validation also passed.

A **Runtime validated** build means the module hash differs from the currently known hash, but the required WASM export signature and configured CDU memory layout passed validation.

### MobiFlight / WinCtrl

Typical states:

- `Not connected`
- `MobiFlight not reachable - retrying`
- `Connected`
- `Connection lost - retrying`

OpenAvionicsBridge retries the local WebSocket connection automatically.

## Captain vs First Officer

The selected CDU determines which configured memory layout and MobiFlight endpoint is used.

Current Fokker endpoints:

```text
Captain:
ws://localhost:8320/winwing/cdu-captain

First Officer:
ws://localhost:8320/winwing/cdu-co-pilot
```

The CDU selection is disabled while the bridge is actively running. Stop the bridge before switching sides.

## Local files

OpenAvionicsBridge stores its runtime files below:

```text
%LOCALAPPDATA%\OpenAvionicsBridge
```

Logs are stored below:

```text
%LOCALAPPDATA%\OpenAvionicsBridge\logs
```

The compatibility report is:

```text
%LOCALAPPDATA%\OpenAvionicsBridge\logs\compatibility-report.txt
```

On first launch, the application can import settings from the older development path:

```text
%LOCALAPPDATA%\F100WinCtrlBridge
```

## Administrator permissions

OpenAvionicsBridge opens the MSFS process with read/query permissions only.

If MSFS itself is started **as Administrator**, Windows can prevent a normally launched OpenAvionicsBridge process from enumerating modules or reading process memory.

In that case, launch OpenAvionicsBridge with the same elevation level.

## Windows SmartScreen

Current release builds are unsigned. Windows SmartScreen may therefore display a reputation warning for a new executable.

This is independent of the embedded icon and VERSIONINFO metadata.

## What the bridge does not do

The current rc3 build does not:

- modify aircraft package files,
- write into MSFS process memory,
- inject a DLL into MSFS,
- render the CDU using OCR or screenshots,
- emulate MobiFlight,
- provide a generic external JSON API yet.

The current public output path is WinWing MCDU through MobiFlight.
