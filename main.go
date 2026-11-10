package main

// ktdev — one small CLI for any Android project (CalNyx included).
// Go port of the Bash draft in workspace/android-cli/andro.
// Only the Go standard library is used, so there are no dependencies
// to download, and cross-compiling to Windows from Linux is one command.
//
//   CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o ktdev-linux-amd64 .
//   CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ktdev-windows-amd64.exe .
//
// No project name, package, or Gradle version is hardcoded here.
// Gradle is picked dynamically from the project's own wrapper version.

import (
	"fmt"
	"os"
)

const version = "0.1.0"

func main() {
	ensureAndroidEnv()
	args := os.Args[1:]
	if len(args) == 0 {
		printHelp()
		return
	}
	switch args[0] {
	case "--help", "-h", "help":
		printHelp()
	case "--version", "version":
		fmt.Println("ktdev " + version)
	case "run":
		cmdRun(args[1:])
	case "build":
		cmdBuild(args[1:])
	case "release":
		cmdRelease(args[1:])
	case "devices":
		cmdDevices()
	case "logs":
		cmdLogs(args[1:])
	case "uninstall":
		cmdUninstall()
	case "clean":
		cmdClean()
	case "doctor":
		cmdDoctor()
	default:
		die("unknown command: " + args[0] + " (try: ktdev --help)")
	}
}

func printHelp() {
	_, _ = os.Stdout.WriteString(`ktdev ` + version + ` — one small CLI for any Android project

Usage:
  ktdev run [--clean] [--device SERIAL] [--release]
  ktdev build [--release] [--clean]
  ktdev release [v1.5.0] ["notes"] [--major|--minor|--dry-run|--yes]
  ktdev devices | ktdev logs | ktdev uninstall | ktdev clean | ktdev doctor
  ktdev --help | ktdev --version

Shortcuts installed by install.sh / install.ps1:
  apprun            = ktdev run
  apprelease [tag]  = ktdev release [tag]

Gradle is never hardcoded: the version comes from the project's
gradle/wrapper/gradle-wrapper.properties (distributionUrl).
./gradlew is used first; if it is missing, the exact wrapper version
is looked up locally, and if it is not there it is downloaded once
to ~/.ktdev/gradle/gradle-<version>/ (Linux) or
%USERPROFILE%\.ktdev\gradle\gradle-<version>\ (Windows).
The Gradle on PATH is never used as a fallback when the wrapper
wants a different version.
`)
}

func die(msg string) {
	fmt.Fprintln(os.Stderr, "STOP: "+msg)
	os.Exit(1)
}

func say(msg string) {
	fmt.Println("== " + msg)
}
