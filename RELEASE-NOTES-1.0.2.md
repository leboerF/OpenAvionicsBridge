# OpenAvionicsBridge 1.0.2

Release date: 2026-09-25

OpenAvionicsBridge 1.0.2 fixes FMS-generated scratchpad messages for the Just Flight F70/F100 integration.

## Fixed

- FMS-generated messages can now override the normal scratchpad when the aircraft exposes a non-zero message selector.
- Verified standard messages include `NOT IN DATABASE` and `NOT ALLOWED`.
- SimBrief/company-route selector states 1001–1009 are mapped to the observed UTF-32 message strings.
- `CLR*` and manually entered scratchpad text remain on the existing normal scratchpad path when the selector is zero.
- Unknown selector values fall back safely to the normal scratchpad.
- A transient WASM memory read is retried once before the bridge re-attaches.

## Compatibility

- Microsoft Flight Simulator 2024
- Just Flight F70/F100 Professional: analyzed 1.3-compatible layout
- MobiFlight / WinWing CDU output
- Captain and First Officer display endpoints

## Notes

This release does not add Avro RJ support. It is intentionally limited to the F70/F100 scratchpad-message fix and the existing 1.0.1 display-core behavior.

OpenAvionicsBridge remains read-only on the simulator/aircraft side.

© 2026 leboerF · MIT License
