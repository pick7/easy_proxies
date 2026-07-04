# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Stable per-node ports**: in `multi-port`/`hybrid` mode, each node keeps the same local port across subscription refreshes and process restarts
  - Ports are preserved by a stable node identity derived from the URI (ignoring the display name and query-parameter order), so renamed or reordered subscription nodes keep their port
  - Assignments are persisted to `node_ports.json` next to `config.yaml` and restored on startup
- **Shadowsocks-compatible link format**: support for additional Shadowsocks URI variants (#28)
- **WebUI: export all nodes**: new "全部导出" button and `GET /api/export?all=true` that exports every node regardless of health (dead or alive); the default export still returns only healthy/available nodes

### Changed
- Improved configuration persistence diagnostics and error handling
- `entrypoint.sh` now detects the "bind-mount of a non-existent file → Docker creates a directory" foot-gun for `config.yaml`/`nodes.txt` and exits with an actionable fix instead of a vague runtime crash
- Removed `start.sh` and `diagnose.sh` helper scripts; `docker compose up -d` (with a directory mount) is now the documented path. README/docs updated to inline the equivalent checks

### Fixed
- **Error messages now match actual mount configuration**: entrypoint.sh error messages previously hardcoded `./data/` paths, causing confusion when using file-mount mode (`-v ./nodes.txt:/etc/easy_proxies/nodes.txt`). Now displays correct fix instructions for both directory-mount and file-mount configurations

### Fixed
- **WebUI: dashboard blacklist/abnormal count stuck at 0**: `GET /api/nodes` returned only the filtered healthy set, so the frontend never saw blacklisted/unavailable nodes and their count always showed 0. It now returns the full node set, restoring the count and making blacklisted nodes visible in the table with a working "解封" (release) button
- WebUI: long node names and URIs are now truncated so they no longer break the table layout
- Prevent crash from malformed VLESS `packetEncoding` nodes
- Preserve inline nodes when a subscription update occurs

## [3.0.1] - 2026-06-17

### Added
- WebUI: sticky proxy settings are now editable from the dashboard

## [3.0.0] - 2026-06-17

### Added
- **Sticky Proxy**: Optional dedicated entry port (default `listener.port + 1`, e.g. `2324`) that pins each client to a single upstream node by source IP, keeping the egress IP stable instead of rotating per connection
  - Coexists with the regular non-sticky pool entry (`2323`) — choose per port
  - Pin is permanent until the node is blacklisted/removed, then re-selects automatically
  - New `sticky` config section (`enabled`, `port`); listen address and credentials inherited from `listener`
  - Pool/hybrid mode only
- **Log Rotation**: Configurable log file rotation with size limits, backup count, and compression
  - New `log` section in config with `output`, `file`, `max_size`, `max_backups`, `max_age`, `compress` options
  - Uses lumberjack for automatic log rotation
  - Defaults: 50MB max size, 3 backups, 7 days retention
- **WebUI Console**: Real-time log streaming in the dashboard
  - In-memory ring buffer captures last 1000 log lines
  - WebSocket-based live log streaming to browser
  - Console tab in WebUI for instant log viewing
- **AnyTLS Protocol**: Support for AnyTLS outbound protocol
  - Parse `anytls://` URIs from subscriptions
  - Full TLS configuration support
- **TUIC Protocol**: Support for TUIC outbound protocol
  - Parse `tuic://` URIs with UUID and password authentication
  - Congestion control and UDP relay mode configuration
  - Full TLS/ALPN support
- **Clash API Integration**: Embedded Clash API controller
  - Internal controller at `127.0.0.1:9092`
  - Enables Clash-compatible tooling integration

### Changed
- **Subscription Parsing**: Improved Clash YAML format detection
  - User-Agent changed to `clash-verge/v2.2.3` for better compatibility
  - YAML detection sample size increased from 200 to 16384 characters
  - Better support for modern proxy types (AnyTLS, TUIC) in Clash format
- **Docker Entrypoint**: Fixed bind-mount permission issues
  - Uses gosu for privilege dropping
  - Ensures proper file ownership for nodes.txt and logs

### Fixed
- Docker nodes.txt permission denied on bind-mount
- VMess node name extraction from base64 payload
- Cross-platform file locking for Windows support

## [2.0.0] - 2025-01-XX

### Added
- SOCKS5 inbound protocol support via Mixed type
- Cross-platform file locking for Windows support
- GeoIP database auto-download and hot-reload
- Hysteria2 (hy2://) protocol support
- Comprehensive security and performance improvements

### Changed
- Major protocol, performance and UI overhaul
- Improved subscription parsing with better error handling
- Enhanced dashboard with real-time statistics

## [1.1.0] - 2024-12-XX

### Added
- GeoIP region routing and dashboard statistics
- Global skip_cert_verify option
- Node port assignment persistence across reloads
- ARM64 support for Docker image

### Fixed
- Hybrid mode export credentials
- Settings save permission issues
- Health check timing after node registration

## [1.0.0] - 2024-11-XX

### Added
- Initial release
- Pool, multi-port, and hybrid runtime modes
- Support for vmess, vless, trojan, ss, hysteria2, socks5, http protocols
- Subscription support (Base64/plain text/Clash YAML)
- Web dashboard with node management
- Automatic health checks and blacklist recovery
- Configurable DNS resolver