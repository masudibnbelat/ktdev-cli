<div align="center">

# 📱 ktdev-cli

### One small CLI for any Android project

**Build it. Install it on the phone. Release it on GitHub — with one command each.**

[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20Windows%20%7C%20macOS-blue?style=flat-square)](https://github.com/masudibnbelat/ktdev-cli/releases)
[![No dependencies](https://img.shields.io/badge/Dependencies-Go%20stdlib%20only-green?style=flat-square)](./go.mod)
[![Version](https://img.shields.io/badge/Version-0.1.0-orange?style=flat-square)](https://github.com/masudibnbelat/ktdev-cli/releases)

```bash
ktdev run
ktdev release v1.5.0 "Fixed percent bug"
```

No more copy-pasting shell functions into `.zshrc`.
Install once, then `ktdev run` / `ktdev release` work in **any** Android project.

</div>

---

## ✨ What it does

| You type | It does |
|---|---|
| `ktdev run` | Finds the project, picks the right Gradle, builds `installDebug`, installs it on the connected phone, and opens the app |
| `ktdev release v1.5.0` | Tags the version, builds `assembleRelease`, checks the APK stamp, pushes the tag, and creates the GitHub Release with the APK |
| `ktdev doctor` | Checks SDK, `adb`, Java, `gh`, Gradle, the connected phone, and the project — before anything else fails |

**Built for real projects, not demos:**

- 🔍 **Dynamic Gradle** — the version is read from your project's own `gradle/wrapper/gradle-wrapper.properties`. A project with Gradle 8.9 uses 8.9, a project with 8.5 uses 8.5. Nothing is hardcoded, and the Gradle on `PATH` is never used as a wrong fallback.
- 📦 **No dependencies** — pure Go standard library. One static binary, no runtime to install.
- 🪟 **Windows + Linux** — the same commands work in PowerShell/cmd and in a Linux terminal.
- 🛡️ **Safe releases** — a dirty tree stops the release unless you pass `--yes`, and a failed build deletes its own tag so you can retry cleanly.

---

## 🚀 Quick start

### Linux / macOS — paste one line in a terminal

```bash
curl -fsSL https://raw.githubusercontent.com/masudibnbelat/ktdev-cli/main/install.sh | bash
```

### Windows — paste one line in PowerShell

```powershell
irm https://raw.githubusercontent.com/masudibnbelat/ktdev-cli/main/install.ps1 | iex
```

Close and reopen the terminal, then go to any Android project:

```bash
cd CalNyx
ktdev run
ktdev release v1.5.0 "Fixed percent bug"
```

> **First time on Windows?** Install Android Studio once (it brings the Android SDK + `adb`), then run `gh auth login` once. After that, `ktdev` works everywhere.

---

## 📥 Install

### Option 1 — One-line installer (recommended)

| OS | Command |
|---|---|
| Linux / macOS | `curl -fsSL https://raw.githubusercontent.com/masudibnbelat/ktdev-cli/main/install.sh \| bash` |
| Windows (PowerShell) | `irm https://raw.githubusercontent.com/masudibnbelat/ktdev-cli/main/install.ps1 \| iex` |

What the installer does:

1. Finds the latest GitHub Release.
2. Downloads the matching binary (`ktdev-linux-amd64` / `ktdev-windows-amd64.exe`).
3. Puts it on `PATH` (`~/.local/bin` on Linux, `%LOCALAPPDATA%\ktdev\bin` on Windows).
4. Creates two shortcuts so the familiar names keep working:

### Option 2 — Download the binary

Go to **[Releases](https://github.com/masudibnbelat/ktdev-cli/releases)**, download the file for your OS, and put it on `PATH`:

| OS | File |
|---|---|
| Linux (amd64) | `ktdev-linux-amd64` |
| Linux (arm64) | `ktdev-linux-arm64` |
| Windows (amd64) | `ktdev-windows-amd64.exe` |
| macOS (arm64) | `ktdev-darwin-arm64` |

### Option 3 — Build from source

Requires Go 1.23 or newer.

```bash
git clone https://github.com/masudibnbelat/ktdev-cli.git
cd ktdev-cli
go build -o ktdev .
./ktdev --help
```

Cross-compile from Linux:

```bash
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o ktdev-linux-amd64 .
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ktdev-windows-amd64.exe .
```

---

## ✅ Requirements

| Need | Why | Check |
|---|---|---|
| Android SDK (`ANDROID_HOME`) | `adb` + `aapt` live here | `ktdev doctor` |
| Java | Gradle needs it | `java -version` |
| `git` | Release tagging + pushing | `git --version` |
| `gh` (GitHub CLI) | Creating the GitHub Release | `gh auth status` |
| A phone with USB debugging on | `ktdev run` installs on it | `adb devices` |

`ktdev doctor` checks all of these at once:

```bash
ktdev doctor
```

Example output:

```text
ktdev 0.1.0 — environment check
ANDROID_HOME: /home/professor/Android/Sdk
adb: /home/professor/Android/Sdk/platform-tools/adb
java: /usr/bin/java
gh: /usr/bin/gh
gradle: /home/professor/.local/opt/gradle-9.3.1/bin/gradle
---
List of devices attached
bb0758c3        device

Project: /path/to/CalNyx
Wrapper Gradle version: 8.9
Gradle: /path/to/CalNyx/gradlew
```

> The `gradle:` line in `doctor` only shows what is on `PATH`.
> The real build never uses it when the wrapper wants another version —
> it uses `./gradlew` (or the exact wrapper version).

---

## 🎮 Usage

All commands work from **any folder inside the project** — `ktdev` walks up
until it finds `settings.gradle(.kts)` or `app/build.gradle(.kts)`.

### Run on the phone

```bash
ktdev run
ktdev run --clean                 # clean first, then build + install
ktdev run --release               # installRelease instead of installDebug
ktdev run --device bb0758c3       # pick a phone when several are connected
```

Short form (installed by the installer):

```bash
ktdev run
ktdev run --clean
```

### Build only (no phone needed)

```bash
ktdev build
ktdev build --release
ktdev build --clean
```

### Release on GitHub

```bash
ktdev release                     # auto-bump patch: v1.4.2 -> v1.4.3
ktdev release v1.5.0              # release exactly this version
ktdev release v1.5.0 "Fixed percent bug"
ktdev release --minor             # v1.4.2 -> v1.5.0
ktdev release --major             # v1.4.2 -> v2.0.0
ktdev release --dry-run           # show the tag, stop before commit/tag/build
ktdev release --yes               # allow a dirty tree (see below)
```

Short form:

```bash
ktdev release
ktdev release v1.5.0 "Fixed percent bug"
ktdev release --dry-run
```

What `ktdev release` does, in order:

1. Fetches tags and works out the next tag (`v0.0.0` → `v0.0.1` on a fresh repo).
2. Refuses the tag if it already exists locally, on GitHub, or as a Release.
3. Stops on a dirty tree and shows the files — pass `--yes` only when you mean it.
4. Commits and tags (`git tag v1.5.0`).
5. Verifies `git describe` prints the new tag — if not, it stops before building.
6. Runs `assembleRelease` with the project's own Gradle version.
7. Checks the APK stamp: `aapt dump badging` must report the same `versionName` as the tag. A mismatch stops the upload — this is the wrong-version update loop, never override it.
8. Copies the APK to `~/<repo>-<tag>.apk`, pushes the branch + tag, and runs `gh release create`.
9. If the build or the stamp check fails, it **deletes the local tag itself**, so the next run does not die on `tag already exists`.

### Other commands

```bash
ktdev devices                     # adb devices
ktdev logs                        # logcat for this app (Ctrl+C to stop)
ktdev uninstall                   # uninstall this app from the phone
ktdev clean                       # gradle clean
ktdev doctor                      # environment + project check
ktdev --help
ktdev --version
```

---

## 📋 All commands

| Command | What it does |
|---|---|
| `ktdev run [--clean] [--device SERIAL] [--release]` | Build + install + open on the phone |
| `ktdev build [--release] [--clean]` | Build only (`installDebug` / `assembleRelease`) |
| `ktdev release [vX.Y.Z] ["notes"] [--major\|--minor\|--dry-run\|--yes]` | Tag, build, and publish a GitHub Release |
| `ktdev devices` | List connected phones |
| `ktdev logs` | `logcat` for this app |
| `ktdev uninstall` | Uninstall this app from the phone |
| `ktdev clean` | `gradle clean` |
| `ktdev doctor` | Check SDK, tools, phone, and project |
| `ktdev --help` / `ktdev --version` | Help / version |

---

## ⚙️ How Gradle is picked (dynamic, never hardcoded)

```
1. ./gradlew (or gradlew.bat on Windows)
   → the wrapper itself downloads and uses the exact wrapper version.
2. A local install of that exact version:
   ~/.local/opt/gradle-<version>/bin/gradle
   ~/.gradle/wrapper/dists/gradle-<version>-bin/.../bin/gradle
   ~/.ktdev/gradle/gradle-<version>/bin/gradle
3. Auto-download that exact version once into ~/.ktdev/gradle/
   (Windows: %USERPROFILE%\.ktdev\gradle\)
```

The version comes from `gradle/wrapper/gradle-wrapper.properties`:

```properties
distributionUrl=https\://services.gradle.org/distributions/gradle-8.9-bin.zip
```

→ version `8.9`. A project with `8.5` there uses `8.5`, a project with `9.3` uses `9.3`.
The Gradle on `PATH` is **never** used when the wrapper wants another version —
that is the bug that broke AGP 8.x projects with Gradle 9.x.

Builds always use `--no-daemon --max-workers=1` (kind to 8 GB RAM machines).

---

## 🧩 Customize it

`ktdev` is a small Go project — add your own command in two places:

1. **`main.go`** — add a case in the `switch`:
   ```go
   case "screenshot":
       cmdScreenshot()
   ```
2. **`commands.go`** — add the function:
   ```go
   func cmdScreenshot() {
       // your code here — runLive("adb", "shell", "screencap", ...)
   }
   ```

Then rebuild:

```bash
go build -o ktdev .
./ktdev screenshot
```

---

## 🛠️ Troubleshooting

| Problem | What it means | Fix |
|---|---|---|
| `no Android project found above this folder` | You ran it outside an Android project | `cd` into the project (any subfolder works) |
| `no phone found` | No device in `adb devices` | Connect USB, allow USB debugging, run `adb devices` |
| `Multiple devices found` | Several phones connected | `ktdev run --device SERIAL` |
| `wrapper wants Gradle 8.9 but ... is missing` | That exact Gradle is not installed | Let it auto-download, or install Gradle 8.9 once |
| `tag v1.5.0 already exists` | You already tagged it | Use a fresh number, or delete the old tag if it was a failed run |
| `Working tree is not clean` | Uncommitted files would be released | Commit / stash first, or `ktdev release --yes` |
| `APK stamp mismatch` | APK `versionName` ≠ tag | Do not override it — check the tag and the version source, then retry |
| `gh is not logged in` | GitHub CLI has no login | `gh auth login` once |
| `adb not found in PATH` | SDK `platform-tools` not on `PATH` | Set `ANDROID_HOME` to your SDK folder and reopen the terminal |
| Windows: SmartScreen blocks `ktdev.exe` | Unsigned binary from GitHub | Click **More info → Run anyway** (first run only) |

---

## 🗂️ Project structure

```text
ktdev-cli/
├── main.go                  # command dispatch, --help, --version
├── project.go               # find project root, ANDROID_HOME setup
├── gradle.go                # dynamic Gradle detection + auto-download
├── android.go               # adb devices, package lookup, app launch
├── commands.go              # run / build / devices / logs / uninstall / clean / doctor
├── release.go               # git tag + assembleRelease + gh release create
├── go.mod                   # module github.com/masudibnbelat/ktdev-cli
├── install.sh               # one-line installer for Linux / macOS
├── install.ps1              # one-line installer for Windows PowerShell
├── .goreleaser.yaml         # builds Linux + Windows + macOS binaries per tag
├── .github/workflows/
│   └── release.yml          # runs GoReleaser on every v* tag
└── README.md                # this file
```

---

## 🚢 Releasing a new version of ktdev itself

```bash
git tag v0.2.0
git push origin v0.2.0
```

The GitHub Actions workflow runs GoReleaser, which builds
`ktdev-linux-amd64`, `ktdev-windows-amd64.exe`, `ktdev-darwin-arm64`, …
and uploads them as bare binaries (not `.tar.gz` / `.zip`) to the Release —
that is exactly what `install.sh` / `install.ps1` download.

Before telling anyone to install, check the Release assets show the bare
names (`ktdev-linux-amd64`, `ktdev-windows-amd64.exe`).

---

## ❓ FAQ

**Do I still need the old shell functions in `.zshrc`?**
No. Delete them — `ktdev run` and `ktdev release` replace them.

**Does it work on Windows?**
Yes — `ktdev-windows-amd64.exe` is a native Windows binary. It uses
`gradlew.bat` when the project has one. You still need the Android SDK,
`adb`, Java, and `gh` installed once on Windows.

**Why not Rust?**
Rust works too (`ripgrep`, `fd`, `bat` are Rust CLIs). For this tool Go was
chosen because cross-compiling to Windows from Linux is one command,
builds are seconds instead of minutes on modest hardware, and almost all
the time is spent inside Gradle / `adb` anyway — the CLI itself is just
the wrapper. See the full comparison in the project history.

**Why does `doctor` show Gradle 9.3.1 when my project uses 8.9?**
`doctor` lists what is on `PATH`. The build itself reads the wrapper
version and uses `./gradlew` (8.9). The `PATH` Gradle is never a fallback.

---

<div align="center">

**Made for Android developers who would rather type `ktdev run` than copy-paste shell functions.**

`ktdev run` · `ktdev release v1.5.0`

</div>
