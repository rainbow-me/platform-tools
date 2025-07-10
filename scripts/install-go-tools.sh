#!/usr/bin/env sh

go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8
go install mvdan.cc/gofumpt@v0.7.0
go install honnef.co/go/tools/cmd/staticcheck@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/daixiang0/gci@latest