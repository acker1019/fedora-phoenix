🌊 fedora-trisolaran
Nuke your OS. Keep your soul.

A single-binary, zero-dependency provisioning tool dedicated to Fedora Linux.

👤 About
Built by a Fedora enthusiast who insists on running bleeding-edge Linux as their main productivity environment — which comes with frequent updates and just as frequent backups. fedora-trisolaran exists to make that cycle painless.

📖 The "Why"
I break my Fedora setup—often. Whether it's kernel experiments or messing with runtimes, I need the ability to wipe my root partition (/) and start fresh without hesitation.

I tried Shell Scripts: brittle, hard to handle errors, spaghetti code. I tried Ansible: slow, requires installing Python/pip first, YAML hell, and feels like killing a fly with a bazooka.

fedora-trisolaran is the answer. It treats your laptop infrastructure as code, but real code (Go), not configuration files.

✨ Key Features (The Selling Points)
🚀 Zero Dependency (Single Binary)
No Python. No Pip. No Git required to start. Just wget the binary, chmod +x, and run. It carries its own logic and "playbooks" (compiled Go code). It bootstraps the system from a fresh install to a fully operational dev environment.

🛡️ LUKS-Aware & Secure
Designed for the Framework Laptop (and similar setups) with a split partition strategy:

Root (/): Ephemeral, formatted on every reinstall.

Work (/dev/nvme0n1p4): LUKS encrypted, persistent. Trisolaran handles the decryption (securely in RAM), unlocking, and mounting of your persistent data, seamlessly bind-mounting it back to your $HOME.

🐧 Fedora Native & Unapologetic
We don't support Ubuntu, Arch, or macOS. By hardcoding logic for dnf, systemd, and GNOME, we achieve blistering speed and absolute reliability. No abstraction layers, no "cross-platform" bloat.

⚡ Pure Go DSL
Why write YAML when you can write type-safe Go? Instead of obscure Ansible modules, we use a custom, lightweight internal library (internal/ops) to handle state:

Go

// Clean, readable, and compiles to a binary.
ops.EnsurePackages("docker", "zsh", "fprintd")
ops.EnsureService("docker")
ops.EnsureSymlink(persistentData+"/Workspace", home+"/Workspace")
workflow: The "Trisolaran" Protocol (see ADR-0009)
Nuke: Install a fresh Fedora from Live USB. Format /. Keep LUKS partition untouched.

Download: Pull the latest tri binary from your S3/EC2.

Execute:

Bash

sudo ./tri rehydra
Unlock: Enter your LUKS password once.

Relax: Go grab a coffee. Trisolaran will:

Decrypt and mount your work data.

Install all system/dev packages (DNF).

Configure Systemd services & User shell.

Restore user space from an artifact (see `tri dehydra` below).

Reborn: Return to a "World Line Restored" environment.

🛠️ Tech Stack
Language: Go (Golang) 1.23+

Dependencies: Standard Library only (mostly).

Target OS: Fedora Linux Workstation (latest)

📋 Usage

### Build

```bash
go build -o bin/tri cmd/trisolaran/trisolaran.go
```

### Commands

#### `tri rehydra`

Start the full rehydration protocol.

**Required Flags:**

- `-s, --secrets <path>`: Path to secrets YAML file (required)

**Optional Flags:**

- `-b, --blueprint <path>`: Path to blueprint YAML file (default: `trisolaran.yml`)
- `-a, --artifact <path>`: Path to artifact tgz produced by `tri dehydra` (see [ADR-0008](docs/adr/adr-0008-artifact-storage-format.md))

**Example:**

```bash
sudo ./tri rehydra \
  --secrets=/path/to/secrets.yml \
  --blueprint=trisolaran.yml \
  --artifact=trisolaran-backup-20260203.tgz
```

**What it does:**

1. Unlocks LUKS encrypted partition
2. Mounts persistent data
3. Installs system packages, ensures users/groups
4. Restores user space from the artifact (`--artifact`)

#### `tri dehydra`

Collect the paths configured under `userspace.dehydration.paths` in the blueprint into a single artifact.

**Optional Flags:**

- `-o, --output <path>`: Path to write the artifact (default: `trisolaran-backup.tgz`)

**Example:**

```bash
./tri dehydra --blueprint=trisolaran.yml --output=trisolaran-backup-20260203.tgz
```

#### `tri stargazing`

Observe system state and drift. Not yet implemented — see [ADR-0009](docs/adr/adr-0009-trisolaran-rebranding.md).

#### `tri version`

Print version information and VCS commit hash.

**Example:**

```bash
./tri version
# Output: Fedora Trisolaran dev (commit: a1b2c3d)
```

⚠️ Disclaimer
TODO
