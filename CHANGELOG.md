# Changelog

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
