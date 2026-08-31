# KOS - Zero-Dependency Portable Version Control

A lightweight, offline-first, encrypted version control system built for disconnected environments.

## 🚀 The Problem

Git is amazing, but it requires:
- Internet access (for remote repos)
- Complex setup (SSH keys, GitHub accounts)
- Server infrastructure

What if you're on a plane? Working on a USB drive? In a remote area with no internet?

## 💡 The Solution: KOS

KOS gives you full version control with:
- Zero dependencies - Single binary, no installation
- Portable snapshots - Every save is a self-contained .tar.gz file
- Built-in encryption - AES-128-CFB + HMAC-SHA256
- Global project store - Work from any folder, any device
- Collision-proof UUIDs - Never lose work to folder conflicts

## 🎯 Quick Start

# Initialize
kos init

# Save with message
kos save "Initial commit"

# View timeline
kos view

# Encrypt and share
kos save --encrypt "Secret commit"
kos share HEAD
## 🔐 Encryption Demo

# Save encrypted snapshot
kos save --encrypt "Top secret"
Password: ******
Confirm password: ******
🔐 Saved encrypted snapshot abc12345: Top secret (3 files)

# Export
kos share HEAD
✅ Snapshot exported: snap_abc12345.tar.gz.enc (400 bytes)

# Load on another machine
kos load snap_abc12345.tar.gz.enc
Password: ******
✅ Decryption successful!
That 400-byte file contains your entire encrypted project. Email it, put it on a USB, share it anywhere.

## 🛠️ Commands

| Command | Description |
|---------|-------------|
| init | Initialize repository |
| save <msg> | Create snapshot |
| save --encrypt <msg> | Encrypted snapshot |
| view | Timeline with relative timestamps |
| status | Show modified files |
| checkout <ref> [file] | Restore snapshot or file |
| diff <id1> <id2> | Compare snapshots |
| tag <name> | Tag a snapshot |
| share <id> | Export portable file |
| load <file> | Load from file or global store |
| projects | List all projects |

## 🏗️ Architecture
Project Folder/
├── .kos-project          # UUID reference
├── .kosignore           # Ignore patterns
└── [your files]
Global Store (~/.kos-global/):
├── /
│   ├── manifest.json    # Snapshot metadata
│   └── snap_*.tar.gz    # Compressed snapshots
└── projects.json        # UUID → name index

## 🎓 Built For

- Students working on USB drives
- Developers in offline environments
- Quick prototyping without cloud setup
- Secure project sharing
- Learning version control concepts

## 📄 License

MIT License - Built with ❤️  for Hackathon 2026
