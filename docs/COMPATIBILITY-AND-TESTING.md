# Compatibility and Testing

This page explains how OpenAvionicsBridge 1.0.0-rc3 decides whether an aircraft runtime is compatible.

## Supported aircraft

Current built-in profile:

```text
justflight-f70-f100-1.3
```

Profile display name:

```text
Just Flight F70/F100 Professional - 1.3 compatible layout
```

Current status:

| Aircraft | Status |
| --- | --- |
| Just Flight F100 Professional | Verified on the development system |
| Just Flight F70 Professional | Same profile; final hardware verification pending |

The profile is build-specific. A future aircraft update can require a new profile if the CDU memory layout changes.

## Why rc3 does not require one fixed DLL filename

During development, the aircraft WASM runtime was exposed by MSFS using a generated native DLL name.

A public build must not assume that the generated filename is identical on every system.

In rc3:

- configured module names are only **discovery hints**,
- all loaded MSFS modules can be inspected,
- the adapter reads the in-memory PE export table,
- required WASM export suffixes identify the correct runtime,
- the linear-memory export RVA is resolved dynamically.

The current profile requires:

```text
_WASM_linearmemory0
_WASM_CDU_DISP_gauge_callback
_WASM_CDU_DISP_2_gauge_callback
```

The generated symbol prefix is ignored.

## Hash handling

The current profile contains a known SHA-256 value for the already analyzed runtime build.

A matching hash is useful evidence, but it is not the only compatibility condition.

### Verified build

A build is reported as **Verified build** when:

1. the runtime export signature matches,
2. the WASM linear-memory pointer can be resolved,
3. the Captain CDU layout validates,
4. the First Officer CDU layout validates,
5. the module SHA-256 matches a known profile hash.

### Runtime validated build

A build is reported as **Runtime validated** when:

1. the runtime export signature matches,
2. the WASM linear-memory pointer can be resolved,
3. the Captain CDU layout validates,
4. the First Officer CDU layout validates,
5. the module SHA-256 is not currently listed as a known hash.

This allows equivalent builds on another system to work without weakening the memory-layout checks.

## What is validated

For both Captain and First Officer, the bridge checks that the following configured regions are readable:

- title large
- title small
- label layer
- large-text layer
- small-text layer
- scratchpad

The contents are also checked for plausible 32-bit display characters.

This validation is intended to prevent an unrelated module or a changed aircraft build from being accepted just because it contains similarly named exports.

## Compatibility report

After a successful attachment, OpenAvionicsBridge writes:

```text
%LOCALAPPDATA%\OpenAvionicsBridge\logs\compatibility-report.txt
```

The report includes:

- OpenAvionicsBridge version
- selected profile
- selected adapter
- MSFS PID
- actual runtime-module filename
- module path
- module SHA-256, if available
- whether the hash is already known
- matched linear-memory export name
- dynamically resolved export RVA
- current linear-memory address
- validation result

If a close candidate is rejected, the report can also contain the rejection reason.

## How testers should report a result

For a useful compatibility report, include:

1. aircraft name,
2. aircraft add-on version/build,
3. MSFS version,
4. OpenAvionicsBridge version,
5. whether Captain and First Officer were tested,
6. whether the CDU display matched the in-sim display,
7. `compatibility-report.txt`,
8. the normal bridge log around the detection attempt.

You may remove personal directory information from file paths before posting logs publicly.

## Suggested F70/F100 verification sequence

For a new machine or aircraft build, check at least:

- initial CDU page,
- INIT,
- F-PLN,
- PROG,
- PERF,
- scratchpad entry,
- left and right line-select labels,
- small-font labels,
- large-font values,
- arrows and selectable-field square symbols,
- Captain CDU,
- First Officer CDU,
- reconnection after restarting MobiFlight.

A build should not be marked verified based only on successful process attachment. The rendered CDU should also be visually compared with the aircraft display.
