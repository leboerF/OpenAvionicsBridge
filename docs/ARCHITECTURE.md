# Architecture

OpenAvionicsBridge is split into four responsibilities so aircraft-specific code does not leak into the rest of the application.

```text
Microsoft Flight Simulator
        │
        ▼
Aircraft adapter / profile
        │
        ▼
Normalized CDU frame (24 × 14)
        │
        ▼
Output transport
        │
        ▼
MobiFlight / WinWing MCDU
```

## 1. Simulator process layer

`winproc_windows.go` provides the read-only Windows process primitives used by adapters:

- locate the MSFS process,
- enumerate loaded modules,
- open the simulator with `PROCESS_VM_READ | PROCESS_QUERY_INFORMATION`,
- read memory with `ReadProcessMemory`,
- inspect PE exports directly from the module image loaded in the simulator,
- optionally calculate a module SHA-256 from its on-disk path.

The application never writes into the simulator process.

## 2. Aircraft adapter layer

`aircraftAdapter` is the extension boundary for aircraft integrations.

The first implementation is `wasm-linear-memory-cdu-v1` in `adapter_wasm_windows.go`.

An adapter receives:

- the read-only simulator process handle,
- an aircraft profile,
- the current module list.

It either returns a validated `aircraftAttachment` or reports that the profile is not currently present.

This allows future aircraft from other manufacturers to use a completely different discovery mechanism without rewriting the GUI, CDU renderer or output code.

## 3. Profile layer

`profiles.go` contains build-specific data such as:

- manufacturer and aircraft name,
- adapter ID,
- optional module-name hints,
- known module hashes,
- required export suffixes,
- linear-memory export suffix,
- Captain/First Officer CDU memory offsets,
- output endpoint for each CDU.

Module names are deliberately hints, not requirements.

Built-in profiles are always available. An optional `profiles.json` next to the executable is merged by profile ID. See `profiles.example.json`.

## 4. Normalized display model

`renderer.go` converts aircraft-specific text buffers into a common `displayFrame`:

- 14 rows,
- 24 columns,
- 336 cells,
- one character and font-size state per cell.

The current Fokker source uses 32-bit little-endian character values in the WASM linear memory. Three 0x480-byte layer blocks contain six left and six right 24-character fields each. The renderer combines those layers into the final CDU grid and maps known special glyphs.

Output transports only consume this normalized frame; they do not need to understand the aircraft memory layout.

## 5. Output transport

`websocket.go` implements the local WebSocket transport currently used by MobiFlight/WinWing. `frameJSON` serializes all 336 cells into the message shape expected by the MobiFlight WinWing CDU endpoint.

A changed frame is transmitted immediately. The current frame is retransmitted after two seconds even if unchanged, which acts as a lightweight safety refresh without delaying normal changes.

## Portable runtime detection in rc3

The public rc3 build no longer requires one fixed MSFS-generated DLL name or a fixed RVA for the WASM linear-memory export.

The WASM adapter performs this sequence:

```text
Enumerate MSFS modules
        │
        ├─ prioritize known module-name hints
        │
        ▼
Read PE export directory from process memory
        │
        ▼
Match required export suffixes
        │
        ▼
Resolve *_WASM_linearmemory0 dynamically
        │
        ▼
Read WASM linear-memory pointer
        │
        ▼
Validate Captain + First Officer CDU blocks
        │
        ├─ known SHA-256 → verified build
        │
        └─ other SHA-256 → runtime-validated compatible build
```

The generated prefix of exports can change; profiles match the stable suffix. For example, a profile asks for `_WASM_linearmemory0`, not the full generated symbol name.

## Compatibility safety

A different module hash is **not** accepted merely because a similarly named DLL exists. For an unknown hash the adapter still requires:

1. all profile-defined runtime export signatures,
2. a plausible linear-memory pointer,
3. readable Captain and First Officer CDU regions,
4. plausible UTF-32 display data in every configured text block.

This is intended to reduce false negatives across machines while still preventing the bridge from blindly applying offsets to an unrelated module.

If a future aircraft update keeps the same runtime signature but changes the CDU layout, runtime validation should reject the old profile. A new profile can then be added for that build.

## Future adapters

Examples of future adapter types could include:

- another WASM-memory layout,
- a SimConnect-backed aircraft interface,
- a local vendor SDK/API,
- an HTML/JS avionics integration,
- another simulator process interface.

Those are architectural possibilities, not current supported features.

## Future outputs

The normalized frame can later feed additional sinks, for example:

- generic local WebSocket JSON,
- plain 24×14 text,
- TCP/UDP,
- serial DIY hardware,
- other commercial MCDU hardware.

`1.0.0-rc3` currently ships only the MobiFlight/WinWing output.
