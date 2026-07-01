FROM ghcr.io/runmedev/runme-build-env:latest

ENV PATH="/usr/local/go/bin:/root/go/bin:${PATH}"
ENV GOTOOLCHAIN=auto
ENV RUNME_TEST_ENV=docker
ENV BUILD_OUTPUT=/usr/local/bin/runme

WORKDIR /app/evals/tasks/update-minor-deps/workdir

RUN git clone https://github.com/runmedev/runme.git .
RUN make install/dev
RUN make build
