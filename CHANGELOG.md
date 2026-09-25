# Changelog

## 1.0.1 — 2026-09-25

Maintenance release preparing the display core for aircraft with multi-color CDU/FMS output while preserving current F70/F100 behavior.

- Extended the normalized 336-cell display model with per-cell WinCtrl color and reverse/inverse-video state.
- MobiFlight serialization now preserves supported cell colors and emits the optional inverse style when required.
- Existing F70/F100 output remains green by default for backward-compatible behavior.
- Added tests for all supported WinCtrl color identifiers, green fallback, inverse serialization, and the Fokker default color.


## 1.0.0 — 2026-09-21

First stable public release.

- Promoted the F70/F100 bridge from release-candidate testing to the 1.0.0 stable line.
- Confirmed Just Flight F70 attachment and CDU output with the shared 1.3-compatible profile.
- Retains tested Just Flight F100 support for the analyzed 1.3-compatible layout.
- Confirmed MobiFlight reconnect and aircraft reload/re-attachment behavior during release testing.
- Confirmed the hardware Test display and Copy diagnostics workflows.
- Fixed live-preview repaint artifacts that could leave text from previous CDU pages visible.
- Retains portable WASM runtime detection, dynamic linear-memory export resolution and runtime CDU-memory validation.
- Includes Captain/First Officer output selection, compatibility reports, single-instance protection and update checking.
- Release remains unsigned; Windows SmartScreen may show a reputation warning on new systems.

## 1.0.0-rc4 — 2026-09-21

Quality-of-life release candidate for public testing.

- Added one-click **Copy diagnostics** support for GitHub issue reports.
- Added a direct **Report** button for `compatibility-report.txt`.
- Added a **Test display** mode that sends a deterministic CDU pattern to MobiFlight without requiring MSFS.
- Added single-instance protection to prevent multiple bridge processes from competing for the same hardware endpoint.
- Improved profile/build status with profile, adapter, verification state and short SHA-256 information.
- Added an automatic GitHub release update check with a clickable status in the header.
- Added unit coverage for version comparison and the test display.

## 1.0.0-rc3 — 2026-09-21

First public release candidate focused on portability and tester diagnostics.

- Reworked aircraft detection so the generated MSFS runtime DLL filename is no longer a hard requirement.
- Added in-memory PE export inspection and suffix-based WASM runtime identification.
- Resolves the WASM linear-memory export RVA dynamically instead of using a fixed RVA.
- Known module SHA-256 values are retained as a verified-build signal, but are no longer the sole compatibility gate.
- Unknown hashes can attach only after required exports, the linear-memory pointer, and both CDU memory layouts pass runtime validation.
- Added `%LOCALAPPDATA%\OpenAvionicsBridge\logs\compatibility-report.txt` for tester reports.
- Introduced the `aircraftAdapter` boundary and generic aircraft profiles to allow future aircraft and manufacturers.
- Added optional merged `profiles.json` support and `profiles.example.json`.
- Generalized the GUI aircraft status instead of hard-coding an F100 status row.
- Added architecture and aircraft-integration documentation.

## 1.0.0-rc2 — 2026-09-21

- Renamed the application and repository branding to **OpenAvionicsBridge**.
- Added the redesigned native Windows GUI.
- Embedded the supplied multi-resolution application icon.
- Added Windows VERSIONINFO metadata with `leboerF` as publisher/copyright identifier.
- Added `© 2026 leboerF` to the application footer.
- Project license changed to the **MIT License**.
- Migrates settings from the previous `F100WinCtrlBridge` local-data directory on first launch.
- Retains the proven F70/F100 bridge core from the rc1 build.
- Uses 100 ms sampling, immediate change delivery and a 2-second safety refresh.
- Small square CDU markers are rendered using the compact font size.

## 1.0.0-rc1

- Stable F100 CDU bridge for Captain and First Officer output on the development system.
- Build-profile validation for the analyzed Just Flight F70/F100 layout.
- MobiFlight/WinWing WebSocket output.
- CDU preview and diagnostics.
