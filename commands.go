package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func gradleArgs() []string {
	// Kept on purpose: --no-daemon --max-workers=1 (8 GB RAM).
	return []string{"--no-daemon", "--max-workers=1"}
}

func mustRoot() string {
	root, err := findProjectRoot()
	if err != nil {
		die("no Android project found above this folder.")
	}
	return root
}

func mustGradle(root string) string {
	g, err := pickGradle(root)
	if err != nil {
		die(err.Error())
	}
	return g
}

func cmdDoctor() {
	fmt.Println("ktdev " + version + " — environment check")
	fmt.Println("ANDROID_HOME: " + os.Getenv("ANDROID_HOME"))
	// Java and adb are reported from where ktdev would actually use
	// them (system first, ktdev cache second). Doctor never downloads
	// anything — the install happens on the first build / run.
	if bin, _, src := findJava(); bin != "" {
		fmt.Println("java: " + bin + " (" + src + ")")
	} else {
		fmt.Println("java: MISSING (auto-installs Temurin 17 on build)")
	}
	if p, src := findAdb(); p != "" {
		fmt.Println("adb: " + p + " (" + src + ")")
	} else {
		fmt.Println("adb: MISSING (auto-installs platform-tools on run)")
	}
	for _, tool := range []string{"gh", "gradle"} {
		if p, err := lookPath(tool); err == nil {
			fmt.Println(tool + ": " + p)
		} else {
			fmt.Println(tool + ": MISSING")
		}
	}
	fmt.Println("---")
	adbBin, _ := findAdb()
	if adbBin == "" {
		adbBin = "adb"
	}
	if out, err := runCapture(adbBin, "devices"); err == nil {
		fmt.Print(out)
	} else {
		fmt.Println("adb: no devices / adb not running")
	}
	root, err := findProjectRoot()
	if err != nil {
		fmt.Println("Project: none found above " + mustCwd())
		return
	}
	fmt.Println("Project: " + root)
	if v := wrapperGradleVersion(root); v != "" {
		fmt.Println("Wrapper Gradle version: " + v)
	} else {
		fmt.Println("Wrapper Gradle version: not found in gradle-wrapper.properties")
	}
	if g, err := pickGradle(root); err == nil {
		fmt.Println("Gradle: " + g)
	} else {
		fmt.Println("Gradle: none usable (" + err.Error() + ")")
	}
}

func mustCwd() string {
	d, _ := os.Getwd()
	return d
}

func cmdDevices() {
	ensureAdb()
	if err := runLive("adb", "devices"); err != nil {
		die("adb devices failed: " + err.Error())
	}
}

func cmdBuild(args []string) {
	release, clean := false, false
	for _, a := range args {
		switch a {
		case "--release":
			release = true
		case "--clean":
			clean = true
		default:
			die("unknown flag for build: " + a + " (try: ktdev build [--release] [--clean])")
		}
	}
	// Build only — never install. assembleDebug needs no phone and no adb.
	// (installDebug would fail with "No connected devices!" when no phone
	// is plugged in, so it belongs to `ktdev run` only, not to `ktdev build`.)
	task := "assembleDebug"
	if release {
		task = "assembleRelease"
	}
	root := mustRoot()
	if err := os.Chdir(root); err != nil {
		die(err.Error())
	}
	say("Project: " + root)
	ensureJava()
	gradle := mustGradle(root)
	say("Gradle: " + gradle)
	all := append(gradleArgs(), task)
	if clean {
		if err := runLive(gradle, append(gradleArgs(), "clean")...); err != nil {
			die("clean failed: " + err.Error())
		}
	}
	if err := runLive(gradle, all...); err != nil {
		die("build failed: " + err.Error())
	}
	// Copy the built APK to <project>/apk/<project>-debug.apk (or
	// -release.apk), so it is easy to find and share — no digging
	// through app/build/outputs/apk/. Copy, not move: the original
	// stays where Gradle put it.
	variant := "debug"
	if release {
		variant = "release"
	}
	copyBuiltApk(root, variant)
}

// copyBuiltApk finds the APK Gradle just built (app/build/outputs/apk/
// <variant>/*.apk), creates <root>/apk/ if needed, and copies the APK
// there renamed to <project>-<variant>.apk, e.g. MyApp-debug.apk.
// The original file stays in place. An existing file with the same
// name is replaced.
func copyBuiltApk(root, variant string) {
	apkDir := filepath.Join(root, "app", "build", "outputs", "apk", variant)
	var src string
	// Prefer the standard app-<variant>.apk name, else any .apk here.
	std := filepath.Join(apkDir, "app-"+variant+".apk")
	if _, err := os.Stat(std); err == nil {
		src = std
	} else {
		entries, _ := os.ReadDir(apkDir)
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".apk") {
				src = filepath.Join(apkDir, e.Name())
				break
			}
		}
	}
	if src == "" {
		die("build passed, but no APK was found in " + apkDir)
	}
	outDir := filepath.Join(root, "apk")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		die("could not create apk folder: " + err.Error())
	}
	dest := filepath.Join(outDir, filepath.Base(root)+"-"+variant+".apk")
	if err := copyFile(src, dest); err != nil {
		die("could not copy APK to " + dest + ": " + err.Error())
	}
	say("APK: " + dest)
}

func cmdRun(args []string) {
	clean, release := false, false
	device := ""
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--clean":
			clean = true
		case args[i] == "--release":
			release = true
		case args[i] == "--device" && i+1 < len(args):
			device = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--device="):
			device = strings.TrimPrefix(args[i], "--device=")
		default:
			die("unknown flag for run: " + args[i] + " (try: ktdev run [--clean] [--device SERIAL] [--release])")
		}
	}
	task := "installDebug"
	if release {
		task = "installRelease"
	}
	root := mustRoot()
	if err := os.Chdir(root); err != nil {
		die(err.Error())
	}
	say("Project: " + root)
	ensureJava()
	gradle := mustGradle(root)
	say("Gradle: " + gradle)
	ensureAdb()
	serial, err := pickDevice(device)
	if err != nil {
		die(err.Error())
	}
	say("Device: " + serial)
	if clean {
		if err := runLive(gradle, append(gradleArgs(), "clean")...); err != nil {
			die("clean failed: " + err.Error())
		}
	}
	env := []string{"ANDROID_SERIAL=" + serial}
	if err := runLiveEnv(gradle, env, append(gradleArgs(), task)...); err != nil {
		die("build/install failed.")
	}
	// Same as `ktdev build`: copy the APK to <project>/apk/ too.
	variant := "debug"
	if release {
		variant = "release"
	}
	copyBuiltApk(root, variant)
	pkg := findPackage(root)
	if pkg == "" {
		die("built and installed, but package name not found — open the app manually.")
	}
	launchApp(serial, pkg)
}

func cmdUninstall() {
	root := mustRoot()
	if err := os.Chdir(root); err != nil {
		die(err.Error())
	}
	pkg := findPackage(root)
	if pkg == "" {
		die("package name not found — build once first (ktdev build).")
	}
	ensureAdb()
	serial, err := pickDevice("")
	if err != nil {
		die(err.Error())
	}
	say("Uninstalling " + pkg + " from " + serial)
	if err := runLive("adb", "-s", serial, "uninstall", pkg); err != nil {
		die("uninstall failed: " + err.Error())
	}
}

func cmdClean() {
	root := mustRoot()
	if err := os.Chdir(root); err != nil {
		die(err.Error())
	}
	ensureJava()
	gradle := mustGradle(root)
	if err := runLive(gradle, append(gradleArgs(), "clean")...); err != nil {
		die("clean failed: " + err.Error())
	}
}

func cmdLogs(args []string) {
	ensureAdb()
	serial, err := pickDevice("")
	if err != nil {
		die(err.Error())
	}
	pkg := ""
	if root, err := findProjectRoot(); err == nil {
		pkg = findPackage(root)
	}
	if pkg != "" {
		say("Logcat for " + pkg + " on " + serial + " (Ctrl+C to stop)")
		// pidof may be empty when the app is not running; logcat with
		// --pid= then shows nothing, so fall back to the full logcat.
		pid, _ := runCapture("adb", "-s", serial, "shell", "pidof", pkg)
		pid = strings.TrimSpace(pid)
		if pid != "" {
			_ = runLive("adb", "-s", serial, "logcat", "--pid="+pid)
			return
		}
	}
	say("Logcat on " + serial + " (Ctrl+C to stop)")
	_ = runLive("adb", "-s", serial, "logcat")
}

var _ = filepath.Join
