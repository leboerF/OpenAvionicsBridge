# OpenAvionicsBridge

OpenAvionicsBridge is an open-source Windows bridge for Microsoft Flight Simulator that exposes avionics display data from supported aircraft to external cockpit hardware and custom integrations.

The first supported integration is the **Just Flight F70/F100 Professional** for **MSFS 2024**, with CDU output to **WinWing MCDU hardware through MobiFlight**.

> **Current release:** `1.0.0-rc2`  
> **License:** MIT  
> **Author:** `leboerF`

## Current features

- Reads the Captain or First Officer F70/F100 CDU display from the running simulator.
- Reconstructs the 24×14 CDU character layout, including small/large text information and mapped special symbols.
- Sends CDU display data to MobiFlight's WinWing CDU WebSocket interface.
- Native Windows GUI with connection/status information, Start/Stop controls and a live CDU preview.
- 100 ms display sampling with immediate updates and a 2-second safety refresh.
- Read-only simulator access: the bridge does not modify the installed aircraft files.
- Build-profile verification before known memory layouts are used.

## Supported aircraft

| Aircraft | Simulator | Output | Status |
| --- | --- | --- | --- |
| Just Flight F70/F100 Professional | MSFS 2024 | WinWing MCDU via MobiFlight | Supported for the verified 1.3 build profile |

Support for additional aircraft and more generic output formats is planned. Aircraft support is build-specific and may need updating after aircraft updates.

## Requirements

- Windows 10 or Windows 11, x64
- Microsoft Flight Simulator 2024
- A supported aircraft/build
- MobiFlight running locally when using WinWing output
- WinWing MCDU configured in MobiFlight

## Usage

1. Start MobiFlight and make sure the WinWing MCDU is available.
2. Start Microsoft Flight Simulator 2024 and load a supported aircraft.
3. Run `OpenAvionicsBridge.exe`.
4. Select **Captain** or **First Officer**.
5. Start the bridge, or leave automatic connection enabled.
6. Check the status indicators and CDU preview in the application.

Application settings and logs are stored below `%LOCALAPPDATA%\OpenAvionicsBridge`. On first launch, the bridge imports existing settings from the older `%LOCALAPPDATA%\F100WinCtrlBridge` location when available.

## Building from source

The project currently uses Go and the native Win32 API. There are no runtime third-party Go dependencies.

### Build requirements

- Go 1.23 or later
- Python 3
- `clang` with an `x86_64-w64-windows-gnu` target, only when rebuilding the Windows resource file

### Build

```bash
python tools/build-windows-resources.py
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath \
  -ldflags="-s -w -H=windowsgui -X main.appVersion=1.0.0-rc2" \
  -o OpenAvionicsBridge.exe .
```

The repository already contains the generated `resource_windows_amd64.syso`, so rebuilding the resource is only required after changing the application icon or version-resource generator.

Run the unit tests with:

```bash
go test ./...
```

## Windows SmartScreen and code signing

Release builds are currently unsigned. Windows SmartScreen may therefore show a reputation warning, especially for new releases. The application icon and Windows VERSIONINFO metadata are embedded, but these do not replace Authenticode code signing.

## Project scope

OpenAvionicsBridge is intended to become a common bridge for multiple simulator aircraft rather than an F100-only application. The current F70/F100 implementation is the first aircraft integration. Future work can separate aircraft-specific readers from generic output transports such as MobiFlight, WebSocket, JSON or plain-text consumers.

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the current architecture and extension direction.

## License

OpenAvionicsBridge is released under the **MIT License**. You may use, copy, modify, merge, publish, distribute, sublicense and sell copies of the software, provided that the copyright and license notice are retained as required by the license.

See [LICENSE](LICENSE).

## Disclaimer

This is an independent community project and is not affiliated with, endorsed by, or supported by Microsoft, Asobo Studio, Just Flight, WinWing, or MobiFlight. Product and company names are used only to identify compatibility. No Just Flight aircraft files, assets, code, or proprietary content are distributed with this repository.

© 2026 leboerF
