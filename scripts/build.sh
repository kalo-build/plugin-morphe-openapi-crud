#!/bin/bash
GOOS=wasip1 GOARCH=wasm go build -o ../dist/morphe-go-struct-v1.0.0.wasm ../cmd/plugin/main.go
