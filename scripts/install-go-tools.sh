#!/usr/bin/env sh

go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8
go install mvdan.cc/gofumpt@v0.7.0
go install honnef.co/go/tools/cmd/staticcheck@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/daixiang0/gci@latest
go install github.com/bufbuild/buf/cmd/buf@v1.55.0
go install github.com/bufbuild/buf/cmd/buf@v1.55.0
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.20.0