# Adding aircraft support

OpenAvionicsBridge separates **how display data is acquired** from **how the normalized CDU frame is rendered and transmitted**.

## If the aircraft fits an existing adapter

For an aircraft that uses the same `wasm-linear-memory-cdu-v1` mechanism, support can be added as another profile.

A profile specifies:

- `id`
- `adapter`
- `manufacturer`
- `aircraft`
- `displayName`
- optional `moduleNames` discovery hints
- optional known SHA-256 values
- required runtime export suffixes
- the linear-memory export suffix
- Captain and First Officer CDU buffer offsets
- output URLs

See `profiles.example.json` for a complete example.

### Module names are hints only

Do not rely on a generated MSFS runtime DLL filename as the only identity. The current adapter scans the in-memory PE export directory and identifies a compatible module using the profile's export suffixes.

### Hashes are confidence signals

Known SHA-256 values identify already tested module builds. An unknown hash can still attach if the runtime export signature and configured CDU memory layout pass validation. This is intentional so equivalent runtime DLLs on another computer do not fail solely because their generated binary differs.

### Memory offsets remain build-specific

Runtime detection does not make unknown offsets safe. Verify every CDU buffer offset for the aircraft/build before publishing a profile. If an aircraft update changes the layout, create or update a profile rather than weakening validation.

## If the aircraft needs a different acquisition method

Implement the `aircraftAdapter` interface and register the adapter in `defaultAircraftAdapters()`.

The adapter should:

1. identify its aircraft/build using evidence appropriate to that integration,
2. acquire only the information needed for the display,
3. validate compatibility before using build-specific assumptions,
4. return an `aircraftAttachment` or equivalent adapter-specific attachment state,
5. keep vendor-specific logic out of the renderer and output transport.

The current interface is intentionally small and may evolve while additional adapters are added.

## Output independence

Aircraft adapters should ultimately produce the normalized display model instead of generating MobiFlight-specific JSON themselves. The current bridge already reconstructs a common 24×14 `displayFrame`; future output sinks can consume it without knowing which aircraft produced it.

## Redistribution

Do not commit or redistribute proprietary aircraft binaries, vendor source code, assets, credentials, or extracted copyrighted content. Profiles should contain only interoperability data required by OpenAvionicsBridge.
