# OpenAvionicsBridge Wiki

Welcome to the OpenAvionicsBridge technical documentation.

OpenAvionicsBridge is a Windows bridge for Microsoft Flight Simulator that reads avionics display data from supported aircraft, converts it into a normalized CDU/MCDU representation, and forwards that data to external cockpit hardware.

The current public release candidate is **1.0.0-rc3**.

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

- [Architecture Overview](ARCHITECTURE.md)  
  High-level separation between simulator access, aircraft adapters, profiles, the normalized display model and output transports.

- [Adding Aircraft Support](ADDING-AIRCRAFT.md)  
  How to add another build, another aircraft using the existing WASM adapter, or a completely new acquisition adapter.

## Current support status

| Aircraft | Simulator | Status |
| --- | --- | --- |
| Just Flight F100 Professional | MSFS 2024 | Verified with the analyzed 1.3-compatible layout |
| Just Flight F70 Professional | MSFS 2024 | Uses the same profile; final hardware verification is pending |

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

OpenAvionicsBridge is currently in release-candidate testing. The main focus of rc3 is portability across different systems and MSFS runtime-module naming differences.

© 2026 leboerF
