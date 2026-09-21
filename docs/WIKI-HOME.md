# OpenAvionicsBridge Wiki

Welcome to the OpenAvionicsBridge technical documentation.

OpenAvionicsBridge is a Windows bridge for Microsoft Flight Simulator that reads avionics display data from supported aircraft, converts it into a normalized CDU/MCDU representation, and forwards that data to external cockpit hardware.

The current stable release is **1.0.0**.

## Current implementation

The first production-targeted integration supports the Just Flight F70/F100 Professional family in MSFS 2024 and forwards the CDU display to WinWing MCDU hardware through MobiFlight.

The current data path is:

```text
Microsoft Flight Simulator
        ↓
aircraft adapter + build profile
        ↓
validated avionics display source
        ↓
normalized 24 × 14 CDU frame
        ↓
MobiFlight WebSocket output
        ↓
WinWing MCDU
```

For the F70/F100 implementation, the display source is the aircraft WASM runtime memory. The bridge does not use OCR or screenshots.

## Documentation

### For users

- [Installation and Quick Start](INSTALLATION-AND-QUICK-START.md)  
  Requirements, first launch, normal start order and GUI status information.

- [Compatibility and Testing](COMPATIBILITY-AND-TESTING.md)  
  Supported aircraft/builds, verified vs runtime-validated builds, and how to report a new compatible system.

- [Troubleshooting](TROUBLESHOOTING.md)  
  Common problems with MSFS detection, aircraft detection, MobiFlight, permissions and compatibility reports.

### For developers

- [Technical Architecture](TECHNICAL-ARCHITECTURE.md)  
  Detailed explanation of process discovery, PE export scanning, WASM linear memory, CDU reconstruction and WebSocket serialization.

- [Adding Aircraft Support](ADDING-AIRCRAFT.md)  
  How to add another build, another aircraft using the existing WASM adapter, or a completely new acquisition adapter.

## Current support status

| Aircraft | Simulator | Status |
| --- | --- | --- |
| Just Flight F100 Professional | MSFS 2024 | Verified with the analyzed 1.3-compatible layout |
| Just Flight F70 Professional | MSFS 2024 | Tested with the shared 1.3-compatible layout |

Support is build-specific. If an aircraft update changes the CDU memory layout, a new or updated profile may be required.

## Important design goals

OpenAvionicsBridge is designed around the following principles:

- **Read-only simulator access.** The current implementation requests process read/query access and does not write into MSFS.
- **No fixed runtime DLL filename requirement.** rc3 discovers the current Fokker runtime by its exported WASM signature.
- **Strict compatibility validation.** Unknown hashes are not trusted solely by filename; the expected runtime exports and CDU memory layout must also validate.
- **Aircraft-independent core.** Aircraft-specific acquisition is isolated behind adapters and profiles.
- **Output-independent display model.** The normalized CDU frame is kept separate from aircraft-specific memory structures.
- **Useful diagnostics.** Compatibility reports are generated so testers can report unknown builds without manually reverse engineering them.

## Project status

OpenAvionicsBridge 1.0.0 is the first stable public release. It retains the portable runtime detection developed during release-candidate testing and includes direct diagnostics/report access, a hardware test display, single-instance protection, clearer profile/build status, reconnect handling, and an automatic update check.

© 2026 leboerF


## 1.0.0 quality-of-life features

- **Test display** sends a deterministic 24 × 14 pattern directly to the selected MobiFlight/WinWing endpoint without MSFS.
- **Report** opens the latest compatibility report.
- **Copy diag.** copies current status and compatibility data for GitHub issues.
- A clearer profile/build line shows the active adapter, verification state and short module hash.
- Single-instance protection prevents two bridge processes from using the same endpoint concurrently.
- The header performs a lightweight check against the public GitHub Releases API and can open a newer release when available.
