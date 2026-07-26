.PHONY: build run inspector inspector-headless inspector-stop test tidy clean

BIN := docserver

build:
	go build -o $(BIN) .

run: build
	./$(BIN)

# Default: spawns ./docserver, opens browser automatically.
# DANGEROUSLY_OMIT_AUTH=true disables proxy auth so you can hit
# http://localhost:6274 without copying the per-session token.
# Safe for local dev only — never set this on a shared network.
inspector: build
	DANGEROUSLY_OMIT_AUTH=true npx -y @modelcontextprotocol/inspector ./$(BIN)

# Headless / server-friendly: do not auto-open browser.
# With DANGEROUSLY_OMIT_AUTH=true, just open http://localhost:6274 in your browser.
# Without it, copy the URL (including ?MCP_PROXY_AUTH_TOKEN=…) the proxy prints.
inspector-headless: build
	DANGEROUSLY_OMIT_AUTH=true BROWSER=none npx -y @modelcontextprotocol/inspector ./$(BIN)

# Kill any inspector / vite / proxy processes holding 6274 or 6277.
inspector-stop:
	-lsof -ti:6274 -ti:6277 2>/dev/null | sort -u | xargs -r kill -9 2>/dev/null
	-pkill -f "modelcontextprotocol/inspector" 2>/dev/null
	-pkill -f "vite" 2>/dev/null

test:
	go test -v ./...

tidy:
	go mod tidy

clean:
	rm -f $(BIN)
