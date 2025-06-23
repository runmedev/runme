# Cloud Assistant Server

This is the server for the Cloud Assistant project. The server performs two functions

1. It acts as a "proxy" between the Web Client and Runme.
2. It serves the Web Client.
3. It is the "Agent"  orchestrating calls to the OpenAI server and post processing the results

[Runme](https://github.com/runmedev/runme) provides a gRPC serve and relies on bidirectional streaming
for the [Execute Request](https://github.com/runmedev/runme/blob/35cb336a37a4e81d3a4623644dfbe39916a2e824/pkg/api/proto/runme/runner/v2/runner.proto#L438).
Since BIDI streaming isn't supported in the browser, we use websockets to allow bidirectional streaming.

The server provides a WebsocketHandler for Execute requests that handles requests by invoking Runme.
Runme is used by linking it in; rather than running it as a separate process and communicating via gRPC.

This currently depends on a forked version of Runme available in
[runmedev/runme#767](https://github.com/runmedev/runme/pull/767).
