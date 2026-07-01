# Update Minor Dependencies Eval

If this Docker-backed task fails during environment startup with
`EnvironmentStartTimeoutError`, build the task image outside Harbor first so the
layers are cached before the eval starts:

```sh
DOCKER_BUILDKIT=1 docker build \
  --progress=plain \
  -f evals/tasks/update-minor-deps/environment/Dockerfile \
  evals/tasks/update-minor-deps/environment
```

Optionally tag the local image while warming the same Docker cache:

```sh {"name":"tag-update-minor-deps-image"}
DOCKER_BUILDKIT=1 docker build \
  --progress=plain \
  -t runme-update-minor-deps-eval:local \
  -f evals/tasks/update-minor-deps/environment/Dockerfile \
  evals/tasks/update-minor-deps/environment
```
