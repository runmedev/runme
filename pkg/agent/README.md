# Runme Agent

This is a quickstart to get you up and running to work on Runme Agent.

./pkg/agent contains the golang server
The client side web application intended to be served by the golang server lives here https://github.com/runmedev/web and distributed via npm.

## Quickstart Setup

### Configure OpenAI

Create a minimal configuration file `~/.runme-agent/config.yaml`

```yaml
apiVersion: ""
kind: ""
logging:
  level: debug
  sinks:
    - path: stderr
openai:
  apiKeyFile: /Users/${USER}/.runme-agent/openai_key_file
cloudAssistant:
  vectorStores:
    - ${VSID} # e.g. vs_67e5xxxxcabxfakexxxe13b2fcd7e612, get your own from OpenAI
assistantServer:
  port: 8080
  httpMaxReadTimeout: 0s
  httpMaxWriteTimeout: 0s
  staticAssets: /workspaces/runme-web/packages/react-components/dist/app # bundled version of https://github.com/runmedev/web
  runnerService: true
  corsOrigins:
    - "http://localhost:5173"
    - "http://localhost:3000"
    - "http://localhost:8080"
```

- set **apiKeyFile** to the path of your OpenAI API key
- set **vectoreStores** to contain the ID of your OpenAI API vector store
- Change the path to the static assets to the location where you checked out the repository

```sh
runme agent config set assistantServer.staticAssets=$(PWD)/web/dist
```

### Build the static assets

```sh
git clone http://github.com/runmedev/web runme-web
cd runme-web
runme run setup clean build
```

### Start the server

```bash {"name":"serve"}
runme agent serve
```

Open up `https://localhost:8443`.

### Development Mode

If you make changes to the UI you need to rerun `npm run build` to recompile the static assets.
However, you don't need to restart the GoLang server; it is sufficient to refresh the page to pick up the
latest static assets.

## Local Tracing

It's handy to have local tracing for debugging. Make sure to configure the OTLP
endpoint in the config.yaml file.

```yaml
telemetry:
  otlpHTTPEndpoint: localhost:4318
```

### Run Jaeger locally

```sh {"name":"jaeger"}
docker run --rm --name jaeger \
  -p 16686:16686 \
  -p 4317:4317 \
  -p 4318:4318 \
  -p 5778:5778 \
  -p 9411:9411 \
  jaegertracing/jaeger:2.6.0
```
