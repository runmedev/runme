# Runme Agent

The `pkg/agent` package contains the Go server used by the Runme web
application. It provides runner and parser services, optional authentication,
and static asset hosting. WebMCP is the supported integration path for browser
automation.

Despite its historical package name, this is not an AI-only server. Runme Web
depends on it for normal notebook and cell execution.

## Server responsibilities

The server provides the runtime services required by Runme Web:

- `/ws` is the bidirectional WebSocket transport used to execute cells and
  stream terminal input and output. It is available when
  `assistantServer.runnerService` is enabled.
- The Runner service manages execution sessions and related runner operations.
- The Parser service parses and serializes notebook content when
  `assistantServer.parserService` is enabled.
- The Jupyter proxy manages Jupyter servers and forwards kernel channel
  WebSockets.
- Optional OIDC authentication, authorization, telemetry, and static web asset
  hosting are provided by the same HTTP server.

These services remain supported. The legacy AI messages, ChatKit, and
app-server bridge endpoints have been removed independently of the execution
server.

## Quickstart

Create a minimal configuration file at `~/.runme-agent/config.yaml`:

```yaml
apiVersion: ""
kind: ""
logging:
  level: debug
  sinks:
    - path: stderr
assistantServer:
  port: 8080
  staticAssets: /workspaces/runme-web/app/dist
  runnerService: true
  parserService: true
  corsOrigins:
    - "http://localhost:5173"
    - "http://localhost:8080"
```

Set `assistantServer.staticAssets` to the built web application:

```sh
runme agent config set assistantServer.staticAssets=$(PWD)/web/app/dist
```

Build the static assets:

```sh
git clone https://github.com/runmedev/web runme-web
cd runme-web
runme run setup clean build
```

Start the server:

```bash {"name":"serve"}
runme agent serve
```

Open `http://localhost:8080`.

## Deprecated configuration compatibility

The `openai`, `cloudAssistant`, and `assistantServer.agentService` keys are
accepted for a transition release so existing configuration files continue to
load. The server ignores these keys and logs a deprecation notice when they are
present. Remove them from personal and deployment configuration files before a
later release removes the compatibility fields.

Published legacy agent protobuf schemas and generated clients remain available
during the same transition, but the server no longer registers their services.
Consumers should stop calling those endpoints before the schemas are removed
in a later breaking release.

## Development mode

Rebuild the web application after UI changes. The Go server does not need to be
restarted; refresh the page to load the updated assets.

## Local tracing

Configure an OTLP endpoint in `config.yaml`:

```yaml
telemetry:
  otlpHTTPEndpoint: localhost:4318
```

Start Jaeger locally:

```sh {"name":"jaeger"}
docker run --rm --name jaeger \
  -p 16686:16686 \
  -p 4317:4317 \
  -p 4318:4318 \
  -p 5778:5778 \
  -p 9411:9411 \
  jaegertracing/jaeger:2.6.0
```
