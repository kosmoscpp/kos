# Reproducible Build Verification

Built twice with deterministic flags: go build -trimpath -buildvcs=false -ldflags="-s -w"

Build Environment:
- Go version: 1.18.1
- OS: Linux amd64
- Build flags: -trimpath -buildvcs=false -ldflags="-s -w"

Build 1 Hash:
11e4c1c412a5715b968d19410b608724bb52ead2ce712ea643e3fd8d7181a0aa  kos-build1

Build 2 Hash:
11e4c1c412a5715b968d19410b608724bb52ead2ce712ea643e3fd8d7181a0aa  kos-build2

Result: Byte-identical output. Determinism achieved.
