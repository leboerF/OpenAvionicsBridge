# OpenAvionicsBridge 1.0.0-rc3

This is the first public release candidate intended for testing on systems other than the development machine.

## Main change: portable aircraft runtime detection

The bridge no longer requires one exact MSFS-generated DLL filename. It discovers the F70/F100 runtime by its WASM export signature and resolves the linear-memory export dynamically.

A known SHA-256 still marks a tested build as verified. A different hash can also be accepted when the required runtime exports and both CDU memory layouts pass validation.

## Tester diagnostics

When an aircraft is detected, OpenAvionicsBridge writes:

`%LOCALAPPDATA%\OpenAvionicsBridge\logs\compatibility-report.txt`

Please attach that file together with the normal bridge log when reporting compatibility problems. You may remove personal path information before posting it.

## Current compatibility

- Just Flight F100 Professional / analyzed 1.3-compatible layout: verified on the development system.
- Just Flight F70 Professional: shares the profile and is pending final hardware verification.
- WinWing MCDU output through MobiFlight.

## Notes

The executable is currently unsigned, so Windows SmartScreen may display a reputation warning for the new binary.
