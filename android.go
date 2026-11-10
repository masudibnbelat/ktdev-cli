package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)






// runLive runs a command with stdout/stderr connected to the terminal,
// so the user sees Gradle / adb output live. It returns the exit error.
func runLive(name string, args ...string) error {
	return runLiveEnv(name, nil, args...)
}

func runLiveEnv(name string, extraEnv []string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}
	return cmd.Run()
}

// runCapture runs a command and returns its combined stdout.
func runCapture(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	return string(out), err
}

func lookPath(name string) (string, error) {
	return exec.LookPath(name)
}

// pickDevice returns the adb serial to use.
// --device wins; one device is used as-is; several devices are listed
// and the user picks one (or the command stops when stdin is not a tty).
func pickDevice(want string) (string, error) {
	if _, err := lookPath("adb"); err != nil {
		return "", fmt.Errorf("adb not found in PATH")
	}
	out, err := runCapture("adb", "devices")
	if err != nil {
		return "", fmt.Errorf("adb devices failed: %v", err)
	}
	var devs []string
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) >= 2 && f[1] == "device" {
			devs = append(devs, f[0])
		}
	}
	if len(devs) == 0 {
		return "", fmt.Errorf("no phone found. Connect USB, allow USB debugging, then check: adb devices")
	}
	if want != "" {
		for _, d := range devs {
			if d == want {
				return d, nil
			}
		}
		return "", fmt.Errorf("device %s not found (connected: %s)", want, strings.Join(devs, ", "))
	}
	if len(devs) == 1 {
		return devs[0], nil
	}
	fmt.Fprintln(os.Stderr, "Multiple devices found:")
	for i, d := range devs {
		fmt.Fprintf(os.Stderr, "  %d) %s\n", i+1, d)
	}
	// Only prompt when stdin is a terminal.
	if fi, _ := os.Stdin.Stat(); fi == nil || (fi.Mode()&os.ModeCharDevice) == 0 {
		return "", fmt.Errorf("pass one with: ktdev run --device SERIAL")
	}
	fmt.Fprintf(os.Stderr, "Pick one [1-%d]: ", len(devs))
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	var n int
	if _, err := fmt.Sscanf(line, "%d", &n); err != nil || n < 1 || n > len(devs) {
		return "", fmt.Errorf("invalid choice")
	}
	return devs[n-1], nil
}

// findPackage finds the app's applicationId: build output-metadata.json
// first, then `aapt dump badging`, then the build.gradle file.
func findPackage(root string) string {
	foundPackage = ""
	// 1. output-metadata.json under app/build/outputs/apk
	apkDir := filepath.Join(root, "app", "build", "outputs", "apk")
	_ = filepath.Walk(apkDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || info.Name() != "output-metadata.json" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var meta struct {
			ApplicationID string `json:"applicationId"`
		}
		if json.Unmarshal(data, &meta) == nil && meta.ApplicationID != "" {
			// Store via a package-level trick: return value through closure.
			// We use the first hit by writing it to the outer variable.
			// (Walk continues, but the first hit wins because we only set once.)
			if foundPackage == "" {
				foundPackage = meta.ApplicationID
			}
		}
		return nil
	})
	if foundPackage != "" {
		return foundPackage
	}

	// 2. aapt dump badging on the first APK found.
	var apk string
	_ = filepath.Walk(apkDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(info.Name(), ".apk") && apk == "" {
			apk = path
		}
		return nil
	})
	if apk != "" {
		if aapt := findAapt(); aapt != "" {
			if out, err := runCapture(aapt, "dump", "badging", apk); err == nil {
				if m := regexp.MustCompile(`package: name='([^']+)'`).FindStringSubmatch(out); m != nil {
					return m[1]
				}
			}
		}
	}

	// 3. applicationId in app/build.gradle[.kts]
	for _, name := range []string{"app/build.gradle.kts", "app/build.gradle"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			continue
		}
		if m := regexp.MustCompile(`applicationId\s*=?\s*"([^"]+)"`).FindSubmatch(data); m != nil {
			return string(m[1])
		}
		// Groovy style: applicationId "com.example.app"
		if m := regexp.MustCompile(`applicationId\s+"([^"]+)"`).FindSubmatch(data); m != nil {
			return string(m[1])
		}
	}
	return ""
}

var foundPackage string

func findAapt() string {
	sdk := os.Getenv("ANDROID_HOME")
	if sdk == "" {
		return ""
	}
	// Newest build-tools wins.
	pattern := filepath.Join(sdk, "build-tools", "*", "aapt")
	if runtimeIsWindows() {
		pattern += ".exe"
	}
	matches, _ := filepath.Glob(pattern)
	if len(matches) == 0 {
		return ""
	}
	sort.Strings(matches)
	return matches[len(matches)-1]
}

func runtimeIsWindows() bool {
	return filepath.Separator == '\\'
}

// launchApp opens the app on the phone: resolve the real launcher
// activity first, fall back to `adb shell monkey`.
func launchApp(serial, pkg string) {
	out, err := runCapture("adb", "-s", serial, "shell", "cmd", "package", "resolve-activity", "--brief", "-c", "android.intent.category.LAUNCHER", pkg)
	activity := ""
	if err == nil {
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) > 0 {
			activity = strings.TrimSpace(lines[len(lines)-1])
			activity = strings.Trim(activity, "\r")
		}
	}
	if activity != "" && activity != "No activity found" {
		say("Opening: " + pkg + " (" + activity + ")")
		if err := runLive("adb", "-s", serial, "shell", "am", "start", "-W", "-n", activity); err != nil {
			_ = runLive("adb", "-s", serial, "shell", "monkey", "-p", pkg, "-c", "android.intent.category.LAUNCHER", "1")
		}
		return
	}
	say("Opening: " + pkg)
	_ = runLive("adb", "-s", serial, "shell", "monkey", "-p", pkg, "-c", "android.intent.category.LAUNCHER", "1")
}
