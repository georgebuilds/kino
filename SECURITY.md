# Security policy

## Reporting a vulnerability

Please email **tc.pf.rc [at] by9 [dot] pw** (replace [at] with @ and [dot] with .) with details. Do **not** open a public GitHub issue for security-sensitive reports.

Include, where possible:

- A description of the issue and its impact.
- Steps to reproduce (or a proof-of-concept).
- The commit / release you tested against.
- Any suggested mitigation.

## What to expect

- Acknowledgement within **~7 days**.
- A follow-up with an assessment and remediation plan, or a reasoned decline.
- Credit in release notes if you'd like it (let us know).

## Attack surface

Kino is a local desktop app with no network server. All data is read from and written to a `.kino` SQLite file on the user's filesystem. The most relevant attack surfaces are:

- **File parsing** — CSV and OFX/QFX import (`internal/importer/`). Malformed or malicious input files could trigger unexpected behavior.
- **SQLite file handling** — opening an untrusted `.kino` file. SQLite itself is the parser; pragmas are fixed at open time.
- **Wails webview** — the embedded WebView renders the Vue frontend. XSS in the UI could call Go methods via the Wails RPC bridge.

Kino makes no outbound network connections and binds no listening sockets.
