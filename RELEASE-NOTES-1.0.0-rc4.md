# OpenAvionicsBridge 1.0.0-rc4

This release candidate focuses on quality-of-life improvements for public testing while retaining the portable aircraft-runtime detection introduced in rc3.

## New in rc4

- **Copy diagnostics** — copies version, current status and the compatibility report into the clipboard for issue reports.
- **Open compatibility report** — opens `compatibility-report.txt` directly from the GUI.
- **Test display** — sends a deterministic CDU test pattern to the selected Captain/First Officer MobiFlight endpoint without requiring MSFS. The pattern exercises large/small text and special symbols.
- **Single-instance protection** — prevents multiple OpenAvionicsBridge processes from competing for the same local hardware endpoint.
- **Clearer profile/build information** — the GUI now shows the selected profile/adapter, verification state and a short module SHA-256 value.
- **Update check** — the app checks the public GitHub Releases API on startup and shows whether a newer release is available. The status can be clicked to recheck or open the release page.

## Compatibility

The rc3 runtime-detection design is unchanged in principle:

- generated DLL names are discovery hints, not hard requirements,
- WASM PE exports are inspected directly from MSFS process memory,
- the linear-memory export RVA is resolved dynamically,
- known hashes are treated as verified builds,
- unknown hashes can still attach only after runtime-export and CDU-memory validation.

## Notes for testers

If something does not work, use **Copy diag.** in the Activity section and include the copied text in the GitHub issue.

The **Test display** button is intended to isolate hardware/MobiFlight problems from aircraft-detection problems. Stop the live bridge before using it.

The executable remains unsigned, so Windows SmartScreen may show a reputation warning.
