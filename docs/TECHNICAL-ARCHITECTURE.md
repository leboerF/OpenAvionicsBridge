# Technical architecture: MSFS → CDU → WinWing

This document describes the current `1.0.0-rc3` data path in enough detail for developers who want to understand, review, or extend OpenAvionicsBridge.

## Overview

OpenAvionicsBridge does not use OCR, screenshot capture, or image scaling for the Fokker CDU. It reads the aircraft's already-rendered textual CDU state from the WASM runtime memory inside the running Microsoft Flight Simulator process.

```text
FlightSimulator2024.exe
        │
        ├─ loaded aircraft WASM runtime module
        │        │
        │        └─ exported *_WASM_linearmemory0 variable
        │
        ▼
WASM linear memory
        │
        ├─ title buffers
        ├─ label layer
        ├─ large-font layer
        ├─ small-font layer
        └─ scratchpad
        │
        ▼
OpenAvionicsBridge renderer
        │
        └─ normalized 24 × 14 / 336-cell CDU frame
        │
        ▼
MobiFlight WebSocket JSON
        │
        ▼
WinWing MCDU
```

## 1. Locating Microsoft Flight Simulator

The Windows process layer uses a Toolhelp process snapshot and accepts the known simulator process names used by MSFS 2024/related installations.

Once a simulator process is found, the bridge opens it with read/query rights only:

```text
PROCESS_VM_READ | PROCESS_QUERY_INFORMATION
```

No write-capable process right is requested.

## 2. Aircraft runtime-module discovery

The Fokker aircraft uses a WASM module which MSFS exposes at runtime through a generated native DLL. A generated DLL filename is not treated as a stable aircraft identifier in rc3.

The adapter receives the complete loaded-module list. Known filenames in the aircraft profile are used only to prioritize scanning.

For each candidate module, the bridge reads the PE headers and export directory directly from the module image in the simulator process. It then searches export names for the suffixes defined by the aircraft profile.

The F70/F100 profile currently requires:

```text
_WASM_linearmemory0
_WASM_CDU_DISP_gauge_callback
_WASM_CDU_DISP_2_gauge_callback
```

The generated prefix is intentionally ignored. For example, both of these would satisfy the linear-memory requirement:

```text
m14f2f2d272d86b96_WASM_linearmemory0
anotherGeneratedPrefix_WASM_linearmemory0
```

This removes the dependency on one particular generated DLL filename.

## 3. Resolving WASM linear memory

The export table provides the RVA of the exported `*_WASM_linearmemory0` variable. rc3 resolves this RVA at runtime; it is no longer a fixed profile value.

The bridge then reads the 64-bit pointer stored at:

```text
module base address + resolved export RVA
```

That pointer is the base of the aircraft's WASM linear-memory allocation in the simulator process.

Because both the module base and the linear-memory allocation can move between processes/runs, the bridge never assumes an absolute address.

## 4. Build confidence and runtime validation

The profile contains one or more known SHA-256 values for module builds that have already been tested. A matching hash is reported as a verified build.

A non-matching hash is not automatically rejected. This is important because equivalent runtime binaries can theoretically differ between systems or simulator/runtime revisions.

For an unknown hash, the bridge still requires all of the following before it attaches:

1. the complete profile export signature is present,
2. the resolved linear-memory pointer is plausible,
3. every configured Captain CDU buffer is readable,
4. every configured First Officer CDU buffer is readable,
5. the buffer contents are structurally plausible 32-bit display text.

Unknown hashes that pass those checks are shown as **Runtime validated** rather than **Verified build**.

If an aircraft update changes the CDU offsets, the old profile should fail memory validation rather than being accepted only because the module name looks familiar.

## 5. F70/F100 CDU memory layout

For the currently analyzed F70/F100-compatible layout, the Captain buffers are:

```text
Title large : 0x1805F0
Title small : 0x1806B0
Labels      : 0x180770
Large text  : 0x181070
Small text  : 0x181970
Scratchpad  : 0x182270
```

First Officer:

```text
Title large : 0x180650
Title small : 0x180710
Labels      : 0x180BF0
Large text  : 0x1814F0
Small text  : 0x181DF0
Scratchpad  : 0x1822D0
```

These values are offsets from the WASM linear-memory base, not process-absolute addresses.

### Character format

The text buffers use 32-bit little-endian character values. A 24-character string occupies:

```text
24 × 4 bytes = 0x60 bytes
```

### 0x480 layer blocks

Each label/large/small layer is `0x480` bytes:

```text
0x480 / 0x60 = 12 strings
```

Those twelve strings are not twelve consecutive screen rows. They represent six left-side fields followed by six right-side fields:

```text
0..5  → left fields
6..11 → right fields
```

The bridge overlays those fields into the six CDU label/data row pairs.

## 6. Reconstructing the 24 × 14 display

`renderer.go` produces a `displayFrame` containing:

```text
14 rows × 24 columns = 336 cells
```

Each cell stores:

```text
character
font size (large/small)
```

The row mapping is:

```text
row 0       title
rows 1/2    label/data pair 1
rows 3/4    label/data pair 2
rows 5/6    label/data pair 3
rows 7/8    label/data pair 4
rows 9/10   label/data pair 5
rows 11/12  label/data pair 6
row 13      scratchpad
```

Right-side fields are right-aligned inside the 24-column row. Large and small source layers are overlaid independently so the WinWing receives the intended font-size state per character.

Known Fokker special glyphs are normalized before output, including arrow and selectable-field symbols.

## 7. MobiFlight / WinWing serialization

The current output is a local WebSocket connection to one of the standard MobiFlight WinWing CDU endpoints:

```text
ws://localhost:8320/winwing/cdu-captain
ws://localhost:8320/winwing/cdu-co-pilot
```

The 336 normalized cells are serialized as the MobiFlight display message. Conceptually each cell contains:

```json
["A", "g", 0]
```

where the final value represents the font-size state used by the current WinWing integration.

The bridge uses the MCDU's normal rendering/font path. It does not send a bitmap of the aircraft CDU.

## 8. Update cadence and connection recovery

After attachment, CDU memory is sampled every 100 ms.

A WebSocket frame is sent when:

```text
current frame != last transmitted frame
```

or when two seconds have elapsed since the last successful transmission.

The two-second transmission is a safety refresh; it does not delay normal display changes.

If MobiFlight is restarted or the local socket is lost, the bridge closes the failed connection and retries without requiring an application restart.

If MSFS/aircraft memory becomes unreadable, the attachment is discarded and normal aircraft detection starts again.

## 9. Compatibility report

Each successful attachment writes:

```text
%LOCALAPPDATA%\OpenAvionicsBridge\logs\compatibility-report.txt
```

It records the selected profile, adapter, actual runtime module name/path, SHA-256 if available, whether the hash was previously known, the resolved linear-memory export/RVA, and runtime-validation result.

This is intended to make tester reports useful even when their generated runtime DLL differs from the development machine.

## 10. Adding another manufacturer

The acquisition mechanism is selected by `aircraftProfile.Adapter`.

If another manufacturer's aircraft exposes a compatible WASM-memory structure, it can use another profile with `wasm-linear-memory-cdu-v1`.

If it requires a different mechanism, a new `aircraftAdapter` implementation can be registered while retaining:

- process/UI lifecycle,
- normalized CDU frame,
- renderer conventions,
- MobiFlight output,
- logging and recovery behavior.

This is the primary extension point for future aircraft support.
