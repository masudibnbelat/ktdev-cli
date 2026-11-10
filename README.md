# ktdev-cli

A small tool for Android developers.
Build your app, run it on your phone, or release it on GitHub — with one command.

## Where does it run?

**On your computer only — Windows, Linux, or macOS.**
It does **not** run on your phone.

Your phone is only used to install and open the app.
`build` and `release` do not need a phone at all.

## Which projects work?

**Any Android project.** Kotlin or Java — it makes no difference.

`ktdev` never looks at your source code. It only runs Gradle,
finds your project folder, and talks to `adb`.
So a Kotlin app, a Java app, or a mixed app all work the same way.

## Install

**Linux / macOS** — paste this in a terminal:

```bash
curl -fsSL https://raw.githubusercontent.com/masudibnbelat/ktdev-cli/main/install.sh | bash
```

**Windows** — paste this in PowerShell:

```powershell
irm https://raw.githubusercontent.com/masudibnbelat/ktdev-cli/main/install.ps1 | iex
```

Then close the terminal and open it again.

## What you need first

- Android SDK (Android Studio installs it) — needed to **build**
- `git`
- `gh` (GitHub CLI) — only for `release`
- A phone with USB debugging on — only for `run`

Java and `adb` are **not** in this list: `ktdev` installs them itself
when they are missing (see below).

Check everything at once:

```bash
ktdev doctor
```

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

## Phone needed or not?

| Command | Phone needed? |
|---|---|
| `ktdev build` | No |
| `ktdev build --release` | No |
| `ktdev run` | **Yes** |
| `ktdev release` | No |
| `ktdev doctor` | No |
| `ktdev devices` | Yes (to list phones) |
| `ktdev logs` | Yes |
| `ktdev uninstall` | Yes |
| `ktdev clean` | No |

## All commands

```bash
ktdev run [--clean] [--device SERIAL] [--release]
ktdev build [--release] [--clean]
ktdev release [v1.5.0] ["notes"] [--major|--minor|--dry-run|--yes]
ktdev devices
ktdev logs
ktdev uninstall
ktdev clean
ktdev doctor
ktdev --help
ktdev --version
```

## How it picks the Gradle version

You do nothing. `ktdev` reads the version from your own project:

```
gradle/wrapper/gradle-wrapper.properties
```

If your project uses Gradle 8.9, it uses 8.9.
If another project uses 8.5, it uses 8.5.
It never uses the wrong Gradle from `PATH`.

## Java and adb install themselves

You do not need to install Java or `adb` first.

- **No Java?** On the first build, `ktdev` downloads Eclipse Temurin 17
  once (about 190 MB) and keeps it in `~/.ktdev/jdk/`.
  If you already have Java 17–21, it uses yours and downloads nothing.
- **No adb?** On the first `run`, `devices`, `logs`, or `uninstall`,
  `ktdev` downloads Google's platform-tools once (about 10–15 MB)
  and keeps it in `~/.ktdev/platform-tools/`.

On Windows the same folders are `%USERPROFILE%\.ktdev\jdk\` and
`%USERPROFILE%\.ktdev\platform-tools\`.

One honest limit: this installs Java and `adb` only.
**Building still needs the full Android SDK** (platforms + build-tools),
which `ktdev` does not download.

## If something goes wrong

| Problem | Fix |
|---|---|
| `no Android project found` | Go into your project folder first (`cd MyApp`) |
| `no phone found` | Connect the phone, turn on USB debugging, run `adb devices` |
| First build downloads Java | Normal, only once (about 190 MB). Next builds use the saved copy. |
| First `run` downloads adb | Normal, only once (about 10–15 MB). |
| `tag v1.5.0 already exists` | Use a new version number |
| `Working tree is not clean` | Commit your changes first, or add `--yes` |
| `gh is not logged in` | Run `gh auth login` once |
| `APK stamp mismatch` | Do not upload it. Check the tag and version, then try again. |

## Build it yourself

You need Go 1.23 or newer.

```bash
git clone https://github.com/masudibnbelat/ktdev-cli.git
cd ktdev-cli
go build -o ktdev .
./ktdev --help
```

## Releasing a new version of ktdev

```bash
git tag v0.2.0
git push origin v0.2.0
```

GitHub Actions builds the binaries and puts them in the Release.
Before you tell anyone to install, check the Release has files named
`ktdev-linux-amd64` and `ktdev-windows-amd64.exe` (no `.tar.gz` / `.zip`).
