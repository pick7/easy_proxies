#!/bin/sh
# Prepare /etc/easy_proxies (auto-generate missing files, guard against
# Docker bind-mount foot-guns), then exec easy_proxies.

CONFIG_DIR="/etc/easy_proxies"
CONFIG_FILE="$CONFIG_DIR/config.yaml"
NODES_FILE="$CONFIG_DIR/nodes.txt"
EXAMPLE_CONFIG="/app/config.example.yaml"

# When a host bind-mount targets a single file that does not exist yet
# (e.g. -v ./data/nodes.txt:/etc/easy_proxies/nodes.txt), Docker creates a
# HOST DIRECTORY at that path and mounts it. The container then sees a
# directory where a file is expected and the app fails opaquely. Detect
# this and exit with a concrete fix instead of a confusing runtime crash.
die() {
    echo "[easy_proxies] ERROR: $1" >&2
    echo "$2" >&2
    exit 1
}

fix_perm_hint="Ensure the mounted directory is writable by the container user:
    mkdir -p data && chown -R $(id -u):$(id -g) data"

# --- config.yaml: reject directory, generate if absent ---
if [ -d "$CONFIG_FILE" ]; then
    die "$CONFIG_FILE is a directory, not a file." \
"This happens when you bind-mount a host file that does not exist yet
(e.g. '-v ./data/config.yaml:/etc/easy_proxies/config.yaml'); Docker then
creates a host directory named config.yaml. Fix on the host:
    rm -rf ./data/config.yaml
    cp config.example.yaml ./data/config.yaml
    docker compose up -d"
fi
if [ ! -e "$CONFIG_FILE" ]; then
    [ -w "$CONFIG_DIR" ] || die "Cannot create $CONFIG_FILE ($CONFIG_DIR not writable)." "$fix_perm_hint"
    cp "$EXAMPLE_CONFIG" "$CONFIG_FILE"
    echo "[easy_proxies] Generated default config from $EXAMPLE_CONFIG"
fi

# --- nodes.txt: reject directory, create empty file if absent ---
if [ -d "$NODES_FILE" ]; then
    die "$NODES_FILE is a directory, not a file." \
"This happens when you bind-mount a host file that does not exist yet
(e.g. '-v ./data/nodes.txt:/etc/easy_proxies/nodes.txt'); Docker then
creates a host directory named nodes.txt. Fix on the host:
    rm -rf ./data/nodes.txt
    touch ./data/nodes.txt
    docker compose up -d"
fi
if [ ! -e "$NODES_FILE" ]; then
    [ -w "$CONFIG_DIR" ] || die "Cannot create $NODES_FILE ($CONFIG_DIR not writable)." "$fix_perm_hint"
    touch "$NODES_FILE"
    echo "[easy_proxies] Created empty nodes.txt"
fi

# --- final readability check ---
if [ ! -r "$CONFIG_FILE" ]; then
    die "Cannot read $CONFIG_FILE (permission denied)." \
"Fix on the host:
    chown -R $(id -u):$(id -g) data
    chmod 644 data/config.yaml data/nodes.txt"
fi

exec /usr/local/bin/easy_proxies "$@"