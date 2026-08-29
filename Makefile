BINARY_NAME=kos

# Reproducible Build: Fixes timestamps and strips debug info for a tiny, identical binary
build:
 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BINARY_NAME) main.go

clean:
 rm -f $(BINARY_NAME)

.PHONY: build clean
