# KOS - Zero-Dependency Portable Version Control

A lightweight, offline-first, encrypted version control system built for disconnected environments.

KOS solves a real problem: Git requires internet access and complex setup. What if you're on a plane, in a remote area, or working on a USB drive? KOS gives you full version control with zero dependencies, portable snapshots, and built-in AES encryption.

## 🚀 Features

- Zero Dependencies - Single binary, no installation needed
- Portable Snapshots - Every save is a self-contained .tar.gz file
- Global Project Store - Work on projects from any folder, any device
- Collision-Proof UUIDs - Never lose work to folder name conflicts
- Built-in Encryption - AES-128-CFB + HMAC-SHA256 for secure sharing
- **Smart .kosignore** - Instantly skips massive directories (O(1) optimizationBeautiful CLILI** - ASCII timeline with relative timestampHEAD Navigationon** - HEAD, HEAD~1, tags work everywherSingle-File Checkoutut** - Restore individual files from historCode Diffff** - Line-by-line LCS diff for any two snapshots

## 📦 Quick Start

# Initialize a project
kos init

# Create a snapshot
kos save "Initial commit"

# View history
kos view

# Tag a release
kos tag v1.0

# Export as portable file
kos share v1.0
## 🔐 Encryption

Share encrypted projects via USB, email, or cloud storage:

# Save with encryption
kos save --encrypt "Secret commit"
# Enter password when prompted

# Load encrypted file
kos load snap_*.tar.gz.enc
# Enter password to decrypt
**File format:** [KENC][version][IV][encrypted data][HMAC]
- AES-128-CFB stream cipher (zero RAM spike)
- HMAC-SHA256 integrity verification
- Password verification before extraction

## 🎯 WhGit is amazing.azing.** But it's not built for:
- Working offline on a plane
- Sharing projects via USB drive
- Quick prototyping without GitHub setup
- Air-gapped enviroKOS is built for exactly that. that.** It's Git's little brother for the disconnected world.

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

## 🎓 Built For

- Students working on USB drives
- Developers in offline environments
- Quick prototyping without cloud setup
- Secure project sharing
- Learning version control concepts

## 📄 License

MIT License - Built with ❤️  by Krishna for the Hackathon 2026

---

KOS: Version control that just works, anywhere.
