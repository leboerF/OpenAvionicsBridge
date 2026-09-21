# Troubleshooting

This page covers common OpenAvionicsBridge 1.0.0-rc3 problems.

## MSFS shows "Not running"

Check that Microsoft Flight Simulator is already running.

The Windows process layer searches known simulator process names. Detection is automatic and does not depend on the MSFS installation path.

If MSFS is running with elevated Administrator permissions, try launching OpenAvionicsBridge as Administrator as well.

## "Cannot enumerate MSFS modules"

This normally indicates a Windows process-permission problem.

Typical cause:

```text
MSFS: Administrator
OpenAvionicsBridge: normal user
```

Run both applications at the same elevation level.

## "No compatible aircraft detected"

This means the simulator was found, but no configured aircraft adapter/profile passed detection.

Possible reasons:

- the supported aircraft has not finished loading,
- the current aircraft build has changed,
- the expected WASM runtime exports are different,
- the CDU memory layout changed,
- the loaded aircraft is not currently supported.

Wait until the aircraft avionics are initialized and try again.

If the problem remains, include the compatibility report and bridge log in a GitHub issue.

## The DLL filename is different from the development system

That alone should no longer be a problem in rc3.

The current WASM adapter does not require one exact generated DLL filename. Module names are only scanning hints.

The adapter identifies a compatible runtime using its PE exports and resolves the linear-memory export dynamically.

## The SHA-256 is different

A different SHA-256 does **not** automatically mean the build is unsupported.

If the required export signature and both CDU layouts validate, the GUI should show:

```text
Runtime validated · ...
```

instead of:

```text
Verified build · ...
```

If runtime validation fails, the bridge should reject the profile rather than reading from unverified offsets.

## "MobiFlight not reachable - retrying"

Check that:

- MobiFlight is running,
- the WinWing MCDU is detected by MobiFlight,
- no local security software is blocking the connection,
- port 8320 is available locally.

The current endpoints are:

```text
ws://localhost:8320/winwing/cdu-captain
ws://localhost:8320/winwing/cdu-co-pilot
```

OpenAvionicsBridge retries automatically.

## The display works, but updates stop

If the local WebSocket connection fails, the bridge closes the stale connection and retries.

The bridge also sends the current display again after two seconds even when no display change has occurred. This safety refresh is separate from the normal 100 ms display sampling.

If the problem persists, inspect the normal bridge log for WebSocket errors.

## The display is scrambled or text is in the wrong rows

Do not assume this is a MobiFlight font problem.

The normalized CDU layout depends on correctly interpreting the aircraft memory blocks.

For the current Fokker profile:

- strings are 24 characters wide,
- characters are stored as 32-bit little-endian values,
- each 0x480-byte layer contains 12 strings,
- those strings represent six left-side and six right-side fields,
- the fields are combined into six label/data row pairs.

A changed aircraft layout requires a profile/code update rather than arbitrary offset adjustments.

## Captain works but First Officer does not

The runtime profile validates both Captain and First Officer memory regions before attachment.

If the bridge attaches but one side renders incorrectly, include:

- the selected side,
- screenshots of the in-sim CDU and WinWing CDU,
- compatibility report,
- bridge log,
- aircraft version.

## Compatibility report location

```text
%LOCALAPPDATA%\OpenAvionicsBridge\logs\compatibility-report.txt
```

Normal logs are in the same `logs` directory.

## What to include in a GitHub issue

Please include:

- OpenAvionicsBridge version,
- MSFS version,
- aircraft name/version,
- Captain or First Officer,
- MobiFlight version,
- `compatibility-report.txt`,
- relevant bridge-log excerpt,
- clear reproduction steps.

If the report contains a personal installation path, you may redact that path before posting it publicly.
