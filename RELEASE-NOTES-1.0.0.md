# OpenAvionicsBridge 1.0.0

OpenAvionicsBridge 1.0.0 is the first stable public release of the Windows bridge for Microsoft Flight Simulator avionics display data.

The initial supported integration is the **Just Flight F70/F100 Professional for MSFS 2024**, with CDU display output to **WinWing MCDU hardware through MobiFlight**.

## Highlights

- Native Windows x64 application with no separate runtime required.
- Captain or First Officer CDU output.
- 24 × 14 CDU reconstruction with large/small font information and mapped Fokker symbols.
- Local MobiFlight/WinWing WebSocket output.
- Read-only simulator process access.
- Portable runtime discovery that does not depend on one generated DLL filename.
- Dynamic resolution of the WASM linear-memory export.
- Known-hash verification plus structural runtime validation for compatible unknown hashes.
- Automatic reconnect after MobiFlight restarts or aircraft/runtime reloads.
- Live CDU preview.
- Hardware **Test display** mode that works without MSFS.
- **Report** and **Copy diag.** controls for compatibility/support reports.
- Single-instance protection.
- Lightweight GitHub Releases update check.
- Daily log and compatibility-report generation.

## Supported aircraft

| Aircraft | Simulator | Status |
| --- | --- | --- |
| Just Flight F100 Professional | MSFS 2024 | Tested with the analyzed 1.3-compatible layout |
| Just Flight F70 Professional | MSFS 2024 | Tested with the shared 1.3-compatible layout |

Aircraft support remains build-specific. If a future aircraft update changes the CDU memory layout or relevant WASM exports, a profile update may be required.

## Runtime detection

OpenAvionicsBridge does not assume that the generated MSFS runtime DLL filename is identical on every computer.

The F70/F100 adapter:

1. finds the running simulator process,
2. enumerates loaded modules,
3. inspects in-memory PE exports,
4. matches the required WASM export suffixes,
5. resolves the linear-memory export RVA dynamically,
6. reads the current WASM linear-memory pointer,
7. validates the configured Captain and First Officer CDU memory regions before attaching.

A known SHA-256 is reported as a verified build. An unknown hash can still be accepted when the runtime signature and configured CDU display-memory layout validate.

## Release-test results

Before 1.0.0, the current build was exercised for:

- Just Flight F70 attachment and CDU output,
- Just Flight F100 operation on the development system,
- MobiFlight disconnect/reconnect,
- aircraft reload and re-attachment,
- hardware Test display,
- automatic-connect/Test display mutual exclusion,
- Copy diagnostics,
- live-preview page changes.

Broader machine-to-machine compatibility will continue to be validated through public use of 1.0.0.

## Diagnostics

Runtime data is stored below:

```text
%LOCALAPPDATA%\OpenAvionicsBridge
```

Compatibility report:

```text
%LOCALAPPDATA%\OpenAvionicsBridge\logs\compatibility-report.txt
```

Use **Copy diag.** to collect the most useful status information for a bug report. Personal installation paths may be redacted before posting publicly.

## Windows SmartScreen

The 1.0.0 executable is currently unsigned. Windows SmartScreen may therefore show a reputation warning, especially on systems that have not previously run OpenAvionicsBridge.

## Scope

Version 1.0.0 ships with the MobiFlight/WinWing output transport and the F70/F100 WASM linear-memory adapter/profile. The adapter/profile architecture is intended to support additional aircraft and output transports in later releases.

OpenAvionicsBridge is an independent community project and is not affiliated with, endorsed by, or supported by Microsoft, Asobo Studio, Just Flight, WinWing, or MobiFlight.

© 2026 leboerF · MIT License
