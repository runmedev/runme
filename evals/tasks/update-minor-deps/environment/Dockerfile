FROM ghcr.io/runmedev/runme-build-env:latest

WORKDIR /app/evals/tasks/update-minor-deps/workdir

RUN git clone --depth=1 https://github.com/runmedev/runme.git .
RUN make install/dev
