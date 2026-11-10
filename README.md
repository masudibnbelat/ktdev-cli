<div align="center">

# ktdev

**One small CLI for any Android project.**
Build it. Run it on your phone. Release it on GitHub. One command each.

[![Release](https://img.shields.io/github/v/release/masudibnbelat/ktdev-cli)](https://github.com/masudibnbelat/ktdev-cli/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/masudibnbelat/ktdev-cli)](https://go.dev/)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-blue)](https://github.com/masudibnbelat/ktdev-cli/releases)
[![Downloads](https://img.shields.io/github/downloads/masudibnbelat/ktdev-cli/total)](https://github.com/masudibnbelat/ktdev-cli/releases)

</div>

---

## Contents

- [Why ktdev?](#why-ktdev)
- [Quick start](#quick-start)
- [The 3 main uses](#the-3-main-uses)
- [All commands](#all-commands)
- [How it works](#how-it-works)
- [What ktdev does NOT do](#what-ktdev-does-not-do)
- [Troubleshooting](#troubleshooting)
- [Developer guide](#developer-guide)

---

## Why ktdev?

| | |
|---|---|
| **Any Android project** | Kotlin or Java — it makes no difference. `ktdev` never reads your source code. It only runs Gradle and talks to `adb`. |
| **No setup fights** | Java, Gradle, and `adb` install themselves when they are missing. No Android Studio needed for them. |
| **Right Gradle, every time** | The version comes from your own project. Gradle 8.9 in one project, 8.5 in another — `ktdev` uses the right one. It never uses the wrong Gradle from `PATH`. |
| **APK where you can find it** | Every build also copies the APK to `MyApp/apk/MyApp-debug.apk` (or `-release.apk`). No digging through `app/build/outputs/apk/`. |
| **Safe releases** | A release stops on a dirty tree, checks the APK version matches the tag, and never uploads the wrong version. |
| **Small and fast** | One binary, about 6 MB. Written in Go, standard library only — no dependencies. |

---

## Quick start

**Linux / macOS** — paste this in a terminal:

```bash
curl -fsSL https://raw.githubusercontent.com/masudibnbelat/ktdev-cli/main/install.sh | bash
```

**Windows** — paste this in PowerShell:

```powershell
irm https://raw.githubusercontent.com/masudibnbelat/ktdev-cli/main/install.ps1 | iex
```

Then close the terminal and open it again.

Check everything at once:

```bash
ktdev doctor
```

Example output:

```
ktdev 0.2.0 — environment check
ANDROID_HOME: /home/professor/Android/Sdk
java: /usr/bin/java (system)
adb: /home/professor/Android/Sdk/platform-tools/adb (system)
gh: /usr/bin/gh
gradle: /home/professor/.local/opt/gradle-9.3.1/bin/gradle
---
List of devices attached
bb0758c3	device
Project: /home/professor/MyApp
Wrapper Gradle version: 8.9
Gradle: /home/professor/MyApp/gradlew
```

### What you need first

| Need | For |
|---|---|
| Android SDK (Android Studio installs it) | Building |
| `git` | `release` |
| `gh` (GitHub CLI, `gh auth login` once) | `release` only |
| A phone with USB debugging on | `run`, `devices`, `logs`, `uninstall` |

Java and `adb` are **not** in this list — `ktdev` installs them itself
when they are missing. See [How it works](#how-it-works).

> **Runs on your computer only — Windows, Linux, or macOS.**
> It does **not** run on your phone. Your phone is only used to install
> and open the app. `build` and `release` do not need a phone at all.

---

## The 3 main uses

### 1. Build only — no phone needed

```bash
ktdev build
```

Builds the app and makes an APK file. No phone, no cable, no `adb`.

The APK is also copied to a folder in your project
(the original stays where Gradle put it):

```
MyApp/apk/MyApp-debug.apk
```

For a release build:

```bash
ktdev build --release
```

That makes:

```
MyApp/apk/MyApp-release.apk
```

### 2. Build + install on your phone — phone needed

```bash
ktdev run
```

Builds the app, installs it on your connected phone, and opens it.
The APK is also copied to `MyApp/apk/MyApp-debug.apk`
(or `MyApp-release.apk` with `ktdev run --release`).

Your phone must be connected by USB, with USB debugging on.
If you have two phones connected, pick one:

```bash
ktdev run --device bb0758c3
```

### 3. Build + GitHub release — no phone needed

```bash
ktdev release v1.5.0 "Fixed percent bug"
```

This does everything for you:

1. Makes the tag `v1.5.0`
2. Builds the release APK
3. Checks the APK version matches the tag
4. Copies the APK to `MyApp/apk/MyApp-release.apk`
5. Pushes the code and the tag to GitHub
6. Creates the GitHub Release with the APK

No phone needed. You need internet and `gh auth login` done once.

To see what will happen, without doing it:

```bash
ktdev release --dry-run
```

To release the next patch version without typing a number:

```bash
ktdev release
```

`v1.4.2` becomes `v1.4.3`.

---

## All commands

| Command | What it does | Phone needed? |
|---|---|---|
| `ktdev run [--clean] [--device SERIAL] [--release]` | Build, install on your phone, open it | **Yes** |
| `ktdev build [--release] [--clean]` | Build only, copy APK to `apk/` | No |
| `ktdev release [v1.5.0] ["notes"] [--major\|--minor\|--dry-run\|--yes]` | Tag, build, check version, push, GitHub Release | No |
| `ktdev devices` | List connected phones | Yes |
| `ktdev logs` | Live logcat for your app | Yes |
| `ktdev uninstall` | Uninstall your app from the phone | Yes |
| `ktdev clean` | Gradle clean | No |
| `ktdev doctor` | Check SDK, Java, adb, Gradle, project | No |
| `ktdev --help` | Show help | No |
| `ktdev --version` | Show version | No |

---

## How it works

<details>
<summary><b>Gradle — picked from your project, never hardcoded</b></summary>

`ktdev` reads the version from your own project:

```
gradle/wrapper/gradle-wrapper.properties
```

Order:

1. `./gradlew` in your project (the wrapper downloads its own Gradle).
2. A local install of that exact version
   (`~/.local/opt/gradle-<version>`, the wrapper cache, or `~/.ktdev/gradle/`).
3. If it is nowhere, `ktdev` downloads that exact version once
   to `~/.ktdev/gradle/gradle-<version>/`.

The Gradle on `PATH` is never used as a fallback when the wrapper
wants a different version — that is the bug that broke AGP 8.x
projects with Gradle 9.x.

</details>

<details>
<summary><b>Java — Temurin 17, only when you need it</b></summary>

- You already have Java 17–21? `ktdev` uses yours. Nothing downloads.
- No Java, or the wrong version? On the first build, `ktdev` downloads
  Eclipse Temurin 17 once (about 190 MB) to `~/.ktdev/jdk/`
  (`%USERPROFILE%\.ktdev\jdk\` on Windows) and sets `JAVA_HOME`
  for Gradle. Next builds use the saved copy.

</details>

<details>
<summary><b>adb — Google platform-tools, only when you need it</b></summary>

- `adb` already on your computer? `ktdev` uses it. Nothing downloads.
- No `adb`? On the first `run`, `devices`, `logs`, or `uninstall`,
  `ktdev` downloads Google's platform-tools once (about 10–15 MB)
  to `~/.ktdev/platform-tools/`
  (`%USERPROFILE%\.ktdev\platform-tools\` on Windows).

</details>

---

## What ktdev does NOT do

- **It does not install the full Android SDK.** Java and `adb` install
  themselves, but building still needs the SDK (platforms +
  build-tools). Install Android Studio once for that.
- **It does not work with Flutter or React Native directly.** Their
  project layout is different. Running inside their `android/`
  folder may build, but release naming will be wrong. `ktdev` is
  for native Android (Kotlin/Java) projects.
- **It does not run on your phone.** It runs on Windows, Linux,
  or macOS only.

---

## Troubleshooting

| Problem | Fix |
|---|---|
| `no Android project found` | Go into your project folder first (`cd MyApp`) |
| `no phone found` | Connect the phone, turn on USB debugging, run `adb devices` |
| First build downloads Java | Normal, only once (about 190 MB). Next builds use the saved copy. |
| First `run` downloads adb | Normal, only once (about 10–15 MB). |
| First build downloads Gradle | Normal, only once. It downloads your project's exact Gradle version. |
| `tag v1.5.0 already exists` | Use a new version number |
| `Working tree is not clean` | Commit your changes first, or add `--yes` |
| `gh is not logged in` | Run `gh auth login` once |
| `APK stamp mismatch` | Do not upload it. Check the tag and version, then try again. |

---

## Developer guide

You want to change `ktdev` itself? You only need Go 1.23 or newer.
Everything below is copy-paste. No extra setup.

### 1. Get the code

```bash
git clone https://github.com/masudibnbelat/ktdev-cli.git
cd ktdev-cli
```

### 2. Change the code

Edit any `.go` file. When you get changed files from someone,
put them in this folder and replace the old ones.

### 3. Build and install — one command

**Linux / macOS:**

```bash
go build -o ktdev . && cp ktdev ~/.local/bin/ktdev && ktdev --version
```

**Windows (PowerShell):**

```powershell
go build -o ktdev.exe .; Copy-Item ktdev.exe "$env:LOCALAPPDATA\ktdev\bin\ktdev.exe" -Force; ktdev --version
```

That one command builds it, installs it, and shows the new version.
If it shows the old version, close the terminal and open it again.

### 4. Test it

```bash
ktdev doctor
```

Then test the command you changed, for example `ktdev build`
in a test Android project.

### 5. Release a new version

Change the version number in `main.go` first
(`const version = "0.2.0"`), then run this:

```bash
git add -A
git commit -m "v0.2.0: what changed"
git push
git tag v0.2.0
git push origin v0.2.0
```

Tag only after you pushed the latest code — the tag must be on
the newest commit, or the Release will build old code.

After the push, GitHub Actions builds the Release by itself.
Before you tell anyone to install, check the Release page has
files named `ktdev-linux-amd64` and `ktdev-windows-amd64.exe`
(no `.tar.gz` / `.zip`).

---

<div align="center">

Made with Go — standard library only, no dependencies.

</div>
