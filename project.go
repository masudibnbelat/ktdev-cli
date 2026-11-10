package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ensureAndroidEnv sets ANDROID_HOME / ANDROID_SDK_ROOT and adds
// platform-tools + cmdline-tools to PATH if they are not already there.
// It does not overwrite a value the user already set.
func ensureAndroidEnv() {
	home, _ := os.UserHomeDir()
	sdk := os.Getenv("ANDROID_HOME")
	if sdk == "" {
		sdk = os.Getenv("ANDROID_SDK_ROOT")
	}
	if sdk == "" {
		if runtime.GOOS == "windows" {
			// %LOCALAPPDATA%\Android\Sdk is the Android Studio default.
			if la := os.Getenv("LOCALAPPDATA"); la != "" {
				sdk = filepath.Join(la, "Android", "Sdk")
			} else if home != "" {
				sdk = filepath.Join(home, "AppData", "Local", "Android", "Sdk")
			}
		} else if home != "" {
			sdk = filepath.Join(home, "Android", "Sdk")
		}
	}
	if sdk != "" {
		os.Setenv("ANDROID_HOME", sdk)
		os.Setenv("ANDROID_SDK_ROOT", sdk)
		// Prepend platform-tools so `adb` is found even without a shell export.
		addToPath(filepath.Join(sdk, "platform-tools"))
		addToPath(filepath.Join(sdk, "cmdline-tools", "latest", "bin"))
	}
}

func addToPath(dir string) {
	if dir == "" {
		return
	}
	sep := string(os.PathListSeparator)
	for _, p := range strings.Split(os.Getenv("PATH"), sep) {
		if p == dir {
			return
		}
	}
	os.Setenv("PATH", dir+sep+os.Getenv("PATH"))
}

// findProjectRoot walks up from the current directory until it finds
// an Android project (settings.gradle[.kts] or app/build.gradle[.kts]).
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		for _, name := range []string{
			"settings.gradle.kts", "settings.gradle",
			filepath.Join("app", "build.gradle.kts"), filepath.Join("app", "build.gradle"),
		} {
			if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
