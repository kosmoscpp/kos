# KOS Standard Library Usage

Zero External Dependencies. 100% Go Standard Library.

KOS is built entirely using Go's standard library. No third-party packages. No go get required. No dependency management headaches.

## 📦 Packages Used (17 total)

### Core I/O & File Operations
| Package | Purpose in KOS |
|---------|----------------|
| os | File system operations, directory creation, file reading/writing |
| io | Stream copying, EOF handling, tar/gzip readers |
| path/filepath | Cross-platform path manipulation, absolute path resolution |
| archive/tar | Creating and extracting tar archives for snapshots |
| compress/gzip | Compressing/decompressing snapshot archives |

### Cryptography & Security
| Package | Purpose in KOS |
|---------|----------------|
| crypto/aes | AES-256-CFB encryption for encrypted snapshots |
| crypto/cipher | Stream cipher modes (CFB) for streaming encryption |
| crypto/hmac | HMAC-SHA256 integrity verification (Encrypt-then-MAC) |
| crypto/rand | Generating cryptographically secure random IVs |
| crypto/sha256 | SHA-256 hashing for file checksums and key derivation |

### Data Serialization
| Package | Purpose in KOS |
|---------|----------------|
| encoding/json | Parsing and writing manifest.json and projects.json |

### System-Level Operations
| Package | Purpose in KOS |
|---------|----------------|
| syscall | Low-level terminal control for password masking (TCGETS, TCSETS, ECHO) |
| unsafe | Pointer operations for syscall terminal state manipulation |

### String & Text Processing
| Package | Purpose in KOS |
|---------|----------------|
| strings | String manipulation, pattern matching, path cleaning |
| bytes | Buffer operations for tar file extraction |

### Utilities
| Package | Purpose in KOS |
|---------|----------------|
| fmt | Formatted output, user interaction, error messages |
| time | Timestamps, relative time formatting, snapshot IDs |

## 🔒 Security Implementation

### Encryption Pipeline
Password → SHA-256 Hash → 32-byte Key → AES-256-CFB + Random IV → Ciphertext
↓
HMAC-SHA256 Tag
↓
Final .enc File: [KENC][Version][IV][Data][HMAC]

Packages involved: crypto/sha256, crypto/aes, crypto/cipher, crypto/rand, crypto/hmac

### Password Masking (Unix/Linux)
go
// Pure syscall implementation - no golang.org/x/term
oldState, _ := getTerminalState(fd)
newState := oldState
newState.Lflag &^= syscall.ECHO  // Disable echo
setTerminalState(fd, newState)
// Read password character by character, print '*' for each
setTerminalState(fd, oldState)  // Restore
Packages involved: syscall, unsafe, os
## 🎯 Why Zero Dependencies Matters

1. Portability: Single binary, no runtime dependencies
2. Security: No supply chain attacks from third-party packages
3. Simplicity: go build works out of the box, no go mod tidy needed
4. Reliability: Standard library is stable and well-tested
5. Performance: No overhead from external dependencies

## 📊 Verification

Run this to verify zero dependencies:
bash
go mod graph
Expected output: Only shows your module, no external packages.
bash
go list -m all
Expected output: Only github.com/yourusername/kos (or similar), no golang.org/x/... packages.

## 🏆 Achievement

KOS demonstrates that professional-grade software can be built with only the Go standard library:
- ✅ 15 functional commands
- ✅ AES-256-CFB encryption with HMAC integrity
- ✅ Cross-platform compatibility (Windows, Mac, Linux)
- ✅ System-level terminal control
- ✅ Streaming encryption (O(1) memory usage)
- ✅ UUID-based collision prevention
- ✅ Beautiful CLI with ASCII timeline

All of this... in ~2000 lines of code. Zero external dependencies.

---

Built for Hackathon 2026 by Hackathon Raptors 🦖