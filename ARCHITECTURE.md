# Architecture

OpenAvionicsBridge currently consists of three practical layers:

1. **Aircraft reader** — detects Microsoft Flight Simulator 2024, verifies a supported aircraft build and reads the CDU display data required for reconstruction.
2. **Renderer/model** — reconstructs the 24×14 CDU character grid, font-size state and mapped special characters.
3. **Output transport** — serializes the reconstructed display for the MobiFlight WinWing CDU WebSocket endpoint.

The current implementation is optimized for the Just Flight F70/F100 Professional and therefore still contains aircraft-specific code in the bridge engine. The repository name is intentionally broader because future aircraft should be implemented as separate profiles/readers while reusing the UI, display model and output transports.

## Extension direction

A future adapter boundary should expose a normalized CDU frame independent of the aircraft source. Output modules can then consume the same frame for:

- MobiFlight / WinWing
- generic local WebSocket clients
- JSON consumers
- plain-text output
- DIY cockpit hardware

No generic output API is claimed as stable in `1.0.0-rc2` yet.
