---
cwd: ../..
runme:
  id: 01JAGWRDZSF2NHJA774WHDDD1V
  version: v3
shell: bash
skipPrompts: true
---

## CI/CD

Run all tests with coverage reports.

```sh {"id":"01J5XTG2WKVR4WG7B2FNPF6VZT","name":"ci-test","promptEnv":"no"}
unset RUNME_SESSION_STRATEGY RUNME_TLS_DIR RUNME_SERVER_ADDR
export SHELL="/bin/bash"
export TZ="UTC"
export GOCOVERDIR="."
export TAGS="test_with_docker"
make test/coverage
make test/coverage/func
```

Run parser/serializer against a large quantity of markdown files.

```sh {"id":"01J5XXFEGPJ5ZJZERQ5YGBBRN8","name":"ci-test-parser","promptEnv":"no"}
export GOPATH="$(go env GOPATH)"
export NUM_OF_FILES=$(find "$GOPATH/pkg/mod/github.com" -name "*.md" | grep -v "\/\." | grep -v glamour | xargs dirname | uniq | wc -l | tr -d "    ")
echo "Checking $NUM_OF_FILES files inside GOPATH=$GOPATH"
make test/parser
```
