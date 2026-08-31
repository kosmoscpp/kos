# KOS - Portable, Encrypted Version Control

A zero-dependency version control system built for portability, encryption, and simplicity.

KOS isn't trying to replace Git. It's a different tool for different needs:
- Portable - Single binary, works from USB drives, no installation required
- Encrypted - Built-in AES-256-CFB encryption with HMAC integrity verification
- Shareable - Export snapshots as tiny encrypted files (400 bytes), share via email/USB/cloud
- Simple - No staging area, no complex branching, just save and restore
- Global Store - Work on projects from any directory, UUID-based collision prevention
- Zero Dependencies - 100% Go standard library, compiles anywhere

## 🚀 Why KOS?

Git is powerful, but sometimes you need:
- A VCS that fits on a USB stick
- Encrypted snapshots you can share securely
- A tool that works without any setup or configuration
- Simple mental model: save → view → restore
- Portability across machines without complex sync

KOS gives you all of that in a single binary.

## 📦 Installation

### Linux/Mac
git clone <your-repo>
cd kos
make install
### Windows
git clone <your-repo>
cd kos
build.bat
After installation, run kos from anywhere!

### Uninstall
make uninstall  # Linux/Mac
## 🎯 Quick Start

# Initialize a project
kos init

# Create a snapshot
kos save "Initial commit"

# View timeline
kos view

# Tag a release
kos tag v1.0

# Export as portable file
kos share v1.0
## 🔐 Encryption

Share encrypted projects via USB, email, or cloud storage:

# Save with encryption
kos save --encrypt "Secret commit"
# Enter password when prompted (masked input)

# Export encrypted snapshot
kos share HEAD
# Creates: snap_abc123.tar.gz.enc (400 bytes)

# Load encrypted file
kos load snap_abc123.tar.gz.enc
# Enter password to decrypt
File format: [KENC][version][IV][encrypted data][HMAC]
- AES-256-CFB stream cipher (zero RAM spike, works on 2GB+ files)
- HMAC-SHA256 integrity verification (fail-fast on wrong password)
- Password verification before extraction

## 🛠️ Commands

| Command | Description |
|---------|-------------|
| init | Initialize KOS repository |
| save <msg> | Create snapshot |
| save --encrypt <msg> | Create encrypted snapshot |
| view | View timeline with relative timestamps |
| status | Show modified files |
| checkout <ref> [file] | Restore snapshot or single file |
| diff <id1> <id2> [file] | Compare snapshots |
| tag <name> [id] | Tag a snapshot |
| share <id> | Export as portable .tar.gz |
| load <file/project> | Load from file or global store |
| projects | List all projects in global store |
| ignore <pattern> | Add to .kosignore |
| search <keyword> | Search commit messages |
| stats | Project statistics |
| ks | Quick save with timestamp |

## 🏗️ Architecture
Project Folder/
├── .kos-project          # Project UUID reference
├── .kosignore           # Ignore patterns
└── [your files]
Global Store (~/.kos-global/):
├── /
│   ├── manifest.json    # Snapshot metadata
│   ├── snap_*.tar.gz    # Compressed snapshots
│   └── .kosignore
├── /
└── projects.json        # UUID → name index

Key features:
- UUID-based projects - No folder name conflicts, no ghost projects
- Embedded manifests - Every snapshot is self-contained
- Streaming encryption - O(1) memory usage, works on large files
- **Smart .kosignore** - O(1) directory skippinHEAD navigationon** - HEAD, HEAD~1, tags work everywhere

## 🎓 Built For

- Students working on USB drives
- Developers who need encrypted snapshots
- Quick prototyping without complex setup
- Secure project sharing via email/USB
- Learning version control concepts
- Air-gapped environments

## 📊 Technical DetailsSingle file:e:** Entire project is main.go (~2000 linesZero dependencies:s:** 100% Go standard librarReproducible builds:s:** Byte-identical output with deterministic flagCross-platform:m:** Works on Linux, Mac, WindowEncryption:n:** AES-256-CFB + HMAC-SHA256 (stdlib cryptoPassword masking:g:** Pure syscall implementation (no golang.org/x/term)

See [STDLIB.md](STDLIB.md) for the complete list of stdlib-for-package substitutions.

## 🏆 Bonuses Achieved

Single File (+5)5)** - Entire project in one main.go file
Reproducible Build (+5)5)** - Byte-identical output, see [REPRODUCIBLE.md](REPRODUCIBLE.md)
STDLIB Log (+3)3)** - 10 real substitutions in [STDLIB.md](STDLIB.md)

## 📄 License

MIT License - Built for Hackathon 2026

-KOS: Portable, encrypted version control that **just works**
