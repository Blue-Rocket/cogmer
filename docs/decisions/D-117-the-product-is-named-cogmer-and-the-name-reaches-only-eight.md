# D-117 — The product is named `cogmer`, and the name reaches only eight places

**Date:** 2026-09-22 · **Status:** active · **Areas:** commands, release, project

**Decision.** The name `cogmer` reaches the module path, the binary, the command directory, the
plugin manifest, the state directory, the local API header, the environment variable prefix and the
release path, and nowhere else. Nothing cryptographic carries it.

**Support.**
- Signing uses `protocolNamespace`, which is `peer-room` and arbitrary on purpose, so a rename moved
  nothing a signature covers. D-069 (every signing tag uses one namespace that never changes).
- The module path in `go.mod` carries the organization's actual casing, which a path that had never
  resolved was free to leave and one that names a real repository is not.
  `plugin/hooks-handlers/install.sh`.
- The plugin manifest name is what a user types before every colon, so it is the command namespace.
  D-118 (the plugin manifest name is the command namespace, and commands keep their prefixes under
  it).
