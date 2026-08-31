# Reproducible Build Verification

Built twice with deterministic flags: go build -trimpath -buildvcs=false -ldflags="-s -w"

Build Environment:
- Go version: 1.18.1
- OS: Linux amd64
- Build flags: -trimpath -buildvcs=false -ldflags="-s -w"

Build 1 Hash:
9d8ec0644c1ced3370752e5e887286317e08ca6c041683f4cc5b8f53e218f0f4 kos-build1

Build 2 Hash:
9d8ec0644c1ced3370752e5e887286317e08ca6c041683f4cc5b8f53e218f0f4 kos-build2

Result: Byte-identical output. Determinism achieved.
