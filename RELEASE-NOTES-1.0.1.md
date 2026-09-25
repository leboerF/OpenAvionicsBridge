# OpenAvionicsBridge 1.0.1

Release date: 2026-09-25

OpenAvionicsBridge 1.0.1 is a maintenance release that extends the normalized CDU display model with per-cell color and reverse/inverse-video support.

## What's new

- Added per-cell WinCtrl color state to the normalized 24×14 / 336-cell display frame.
- Added optional reverse/inverse-video state per display cell.
- MobiFlight serialization now preserves supported cell colors instead of forcing one global color.
- Supported WinCtrl color identifiers: amber, white, cyan, green, magenta, red, yellow, blue, grey, and khaki.
- Unknown or unset colors fall back safely to green.
- Existing Just Flight F70/F100 output remains green by default, so the current integration keeps its established visual behavior.
- Added automated tests covering supported colors, fallback behavior, inverse serialization, and the F70/F100 green default.

## Why this release matters

The richer display-cell model keeps the current F70/F100 integration backward-compatible while preparing the common output layer for aircraft whose CDU/FMS displays use multiple colors.

This is an output-model change only. It does not add Avro RJ support to the stable release yet.

## Compatibility

- Microsoft Flight Simulator 2024
- Just Flight F70/F100 Professional: existing supported 1.3-compatible layout
- MobiFlight / WinWing CDU output
- Captain and First Officer display endpoints

## Notes

OpenAvionicsBridge remains read-only on the simulator/aircraft side. The release is unsigned, so Windows SmartScreen may show a reputation warning on new systems.

© 2026 leboerF · MIT License
