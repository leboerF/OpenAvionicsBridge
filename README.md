# OpenAvionicsBridge

OpenAvionicsBridge is an open-source Windows bridge for Microsoft Flight Simulator that reads avionics display data from supported aircraft and forwards a normalized display to external cockpit hardware.

The first integration targets the **Just Flight F70/F100 Professional for MSFS 2024** and sends the CDU display to **WinWing MCDU hardware through MobiFlight**.

> **Current public release candidate:** `1.0.0-rc3`  
> **License:** MIT  
> **Author:** `leboerF`

## Current features

- Native Windows x64 application; no separate runtime is required.
- Captain or First Officer CDU output.
- Reconstructs the 24×14 CDU character grid, including large/small text and mapped special symbols.
- Sends display frames to the local MobiFlight WinWing CDU WebSocket endpoint.
- Native status GUI, Start/Stop controls, live CDU preview and diagnostic logging.
- 100 ms display sampling with immediate change delivery and a 2-second safety refresh.
- Read-only simulator access; no aircraft files are modified.
- Runtime aircraft detection does **not** depend on a fixed generated DLL filename.
- Known module hashes are recognized, but a different hash can still be accepted after export-signature and memory-layout validation.
- A compatibility report is written for testers to `%LOCALAPPDATA%\OpenAvionicsBridge\logs\compatibility-report.txt`.
- Adapter/profile architecture prepared for additional aircraft and manufacturers.

## Supported aircraft

| Aircraft | Simulator | Output | Status |
| --- | --- | --- | --- |
| Just Flight F100 Professional | MSFS 2024 | WinWing MCDU via MobiFlight | Verified with the analyzed 1.3-compatible layout |
| Just Flight F70 Professional | MSFS 2024 | WinWing MCDU via MobiFlight | Uses the same profile; final hardware verification is still pending |

Aircraft support is build-specific. An aircraft update can require a profile update if its CDU memory layout changes.

## Why rc3 is more portable

Early development builds identified the Fokker runtime using one generated DLL name and one SHA-256 value. That was safe for development but too strict for public testing.

`1.0.0-rc3` instead:

1. Finds the running Microsoft Flight Simulator process.
2. Enumerates loaded modules.
3. Uses known module names only as **discovery hints**.
4. Inspects in-memory PE exports and looks for the aircraft profile's required WASM export suffixes.
5. Resolves the linear-memory export dynamically; its RVA is no longer hard-coded.
6. Reads the candidate WASM linear-memory pointer.
7. Validates both CDU memory layouts before attaching.
8. Uses the module SHA-256 as an additional confidence signal, not as the only compatibility gate.

This means the bridge can still attach when MSFS gives the same compatible WASM runtime a different generated DLL name, and it can tolerate a different DLL hash when the runtime signature and CDU memory layout are still compatible.

## Requirements

- Windows 10 or Windows 11, x64
- Microsoft Flight Simulator 2024
- A supported aircraft/build
- MobiFlight running locally for WinWing output
- WinWing MCDU configured in MobiFlight

If MSFS is launched as Administrator, OpenAvionicsBridge may also need to be launched as Administrator so Windows allows read access to the simulator process.

## Usage

1. Start MobiFlight and verify that the WinWing MCDU is available.
2. Start MSFS 2024 and load a supported aircraft.
3. Run `OpenAvionicsBridge.exe`.
4. Select **Captain** or **First Officer**.
5. Start the bridge, or leave automatic connection enabled.
6. Check the status indicators and CDU preview.

Settings and logs are stored below `%LOCALAPPDATA%\OpenAvionicsBridge`. On first launch, the bridge imports existing settings from the older `%LOCALAPPDATA%\F100WinCtrlBridge` directory when available.

If compatibility detection fails, include `compatibility-report.txt` and the normal bridge log in a GitHub issue. Remove personal path information first if desired.

## Aircraft adapters and profiles

The core no longer treats the Fokker as a special case. Aircraft-specific discovery is routed through an adapter interface.

The first adapter is:

```text
wasm-linear-memory-cdu-v1
```

It supports profile-driven aircraft whose CDU can be reconstructed from known locations in a WASM linear-memory block. Additional aircraft from another manufacturer can therefore be added in two ways:

- **Same technical mechanism:** add another profile using the existing WASM adapter.
- **Different technical mechanism:** implement another adapter while reusing the GUI, normalized display model and output transport.

`profiles.example.json` documents the current profile format. A local `profiles.json` placed next to the executable is merged with the built-in profiles. A profile with the same `id` overrides the built-in profile; a new `id` adds another profile.

See [docs/ADDING-AIRCRAFT.md](docs/ADDING-AIRCRAFT.md) and [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Documentation

The repository includes wiki-style documentation under `docs/`:

- [Wiki Home](docs/WIKI-HOME.md)
- [Installation and Quick Start](docs/INSTALLATION-AND-QUICK-START.md)
- [Compatibility and Testing](docs/COMPATIBILITY-AND-TESTING.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [Technical Architecture](docs/TECHNICAL-ARCHITECTURE.md)
- [Adding Aircraft Support](docs/ADDING-AIRCRAFT.md)

## Building from source

The project uses Go and the native Win32 API. There are currently no runtime third-party Go dependencies.

### Build requirements

- Go 1.23 or later
- Python 3
- `clang` with an `x86_64-w64-windows-gnu` target for the Windows resource step

### Build

```bash
python tools/build-windows-resources.py
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
  -ldflags="-s -w -H=windowsgui -X main.appVersion=1.0.0-rc3" \
  -o OpenAvionicsBridge.exe .
```

The generated `resource_windows_amd64.syso` file is intentionally not committed. Run the resource build step before compiling a release build or whenever the icon/VERSIONINFO data changes.

Run unit tests with:

```bash
go test ./...
```

A Windows-target compile check can be performed from another OS with:

```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c .
```

## Windows SmartScreen and code signing

Release builds are currently unsigned. Windows SmartScreen can therefore show a reputation warning, especially for a new release. Embedded icon and VERSIONINFO metadata do not replace Authenticode code signing.

## Project scope

OpenAvionicsBridge is intended to become a common bridge for several MSFS aircraft rather than an F70/F100-only utility. The normalized 24×14 CDU frame is already separated from aircraft discovery, and the aircraft adapter registry provides the boundary for future integrations.

Generic JSON/plain-text output is a planned extension; `1.0.0-rc3` still ships only the MobiFlight/WinWing output transport.

## License

OpenAvionicsBridge is released under the **MIT License**. See [LICENSE](LICENSE).

## Disclaimer

This is an independent community project and is not affiliated with, endorsed by, or supported by Microsoft, Asobo Studio, Just Flight, WinWing, or MobiFlight. Product and company names are used only to identify compatibility. No proprietary aircraft files, assets, code, or content are distributed with this repository.

© 2026 leboerF
