# Load variables from .env if present
set dotenv-load

# Connection settings — override in .env or via environment variables
PI_USER := env_var_or_default("PI_USER", "pi")
PI_HOST := env_var_or_default("PI_HOST", "monitor.local")
PI      := PI_USER + "@" + PI_HOST
REMOTE  := "/home/" + PI_USER + "/monitor"
# Pi's LAN IP — used to populate WebRTC ICE candidates in mediamtx.yml
PI_IP   := env_var_or_default("PI_IP", "192.168.1.100")

# Generate mediamtx.yml from the example template, substituting PI_IP
setup:
    @sed 's/YOUR_PI_IP/{{PI_IP}}/' config/mediamtx.yml.example > mediamtx.yml

# Generate JS/TS types from Go API structs
gen:
    $(go env GOPATH)/bin/tygo generate
    sed -i '' 's/detected_at?: string/detected_at: string | null/' server/web/src/api_types.d.ts

# Cross-compile the Go HTTP server for linux/arm64 (Raspberry Pi)
build: gen
    npx tsc
    GOOS=linux GOARCH=arm64 go build -o bin/monitor ./server/cmd/

# Sync project files to the Pi (runs setup first to generate mediamtx.yml)
sync: setup
    rsync -av \
        --exclude bin/ \
        --exclude cmd \
        --exclude mediamtx \
        --exclude node_modules/ \
        --exclude .git/ \
        --exclude .github/ \
        --exclude server/web/src/ \
        --exclude server/cmd/ \
        --exclude server/internal/ \
        --exclude tsconfig.json \
        --exclude package.json \
        --exclude package-lock.json \
        --exclude tygo.yaml \
        --exclude go.mod \
        --exclude CLAUDE.md \
        --exclude README.md \
        --exclude justfile \
        ./ {{PI}}:{{REMOTE}}/

# Build binary, sync all files, then deploy the binary atomically and restart HTTP service
deploy: build sync
    scp bin/monitor {{PI}}:/tmp/monitor-new
    ssh {{PI}} "mv /tmp/monitor-new {{REMOTE}}/monitor && chmod +x {{REMOTE}}/monitor && sudo systemctl restart monitor-http.service"

# Full install: sync files, download mediamtx, register systemd services
install: sync
    ssh {{PI}} "{{REMOTE}}/setup/install.sh"

# Start all three services on the Pi (stream, mediamtx, monitor-http)
start:
    ssh {{PI}} "{{REMOTE}}/services/control.sh start" < /dev/null

# Stop all services on the Pi
stop:
    ssh {{PI}} "{{REMOTE}}/services/control.sh stop"

# Restart all services on the Pi
restart:
    ssh {{PI}} "{{REMOTE}}/services/control.sh stop && {{REMOTE}}/services/control.sh start" < /dev/null

# Show systemd status for all services
status:
    ssh {{PI}} "{{REMOTE}}/services/control.sh status"

# Tail the FFmpeg stream log
logs:
    ssh {{PI}} "tail -f {{REMOTE}}/logs/ffmpeg.log"

# Tail the Go HTTP server log
logs-http:
    ssh {{PI}} "tail -f {{REMOTE}}/logs/monitor.log"

# Tail the sound detection log
logs-detect:
    ssh {{PI}} "tail -f {{REMOTE}}/logs/detect.log"

# Run cry detection unit tests on the Pi
test:
    ssh {{PI}} "{{REMOTE}}/venv/bin/python3 {{REMOTE}}/services/detect/test_cry.py"
