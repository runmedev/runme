#!/usr/bin/env sh
set -eu

mkdir -p /logs/verifier
go run /tests/score.go > /logs/verifier/test-stdout.txt
