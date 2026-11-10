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
	for _, tool := range []string{"adb", "java", "gh", "gradle"} {
		if p, err := lookPath(tool); err == nil {
			fmt.Println(tool + ": " + p)
		} else {
			fmt.Println(tool + ": MISSING")
		}
	}
	fmt.Println("---")
	if out, err := runCapture("adb", "devices"); err == nil {
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
	if _, err := lookPath("adb"); err != nil {
		die("adb not found in PATH.")
	}
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
	task := "installDebug"
	if release {
		task = "assembleRelease"
	}
	root := mustRoot()
	if err := os.Chdir(root); err != nil {
		die(err.Error())
	}
	say("Project: " + root)
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
	gradle := mustGradle(root)
	say("Gradle: " + gradle)
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
	gradle := mustGradle(root)
	if err := runLive(gradle, append(gradleArgs(), "clean")...); err != nil {
		die("clean failed: " + err.Error())
	}
}

func cmdLogs(args []string) {
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
