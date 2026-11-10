package main

// tools.go — Java and adb, installed by ktdev itself when missing.
// Same pattern as Gradle (gradle.go): system copy first, ktdev cache
// second, download once third. No package manager, no sudo, no
// Android Studio needed — the downloads come straight from the
// official sources and are cached under ~/.ktdev/ (Linux/macOS) or
// %USERPROFILE%\.ktdev\ (Windows), so they are downloaded once.
//
//   - Java: Eclipse Temurin 17 (Adoptium). Gradle 8.x + AGP 8.x need
//     JDK 17 to run, and Gradle 8.9 runs on Java 17-21, so a system
//     Java outside 17-21 is ignored and Temurin 17 is used instead.
//   - adb: Google's platform-tools zip (adb + fastboot, ~10-15 MB).
//
// Honest limit: this installs Java and adb only. Building an Android
// app still needs the full Android SDK (platforms + build-tools) —
// ktdev does NOT download that. `ktdev build` still stops without it.

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func javaExeName() string {
	if runtime.GOOS == "windows" {
		return "java.exe"
	}
	return "java"
}

func adbExeName() string {
	if runtime.GOOS == "windows" {
		return "adb.exe"
	}
	return "adb"
}

// downloadFile downloads url to destPath, skipping the download when
// the file is already there (a previous run's cache).
func downloadFile(url, destPath string) error {
	if _, err := os.Stat(destPath); err == nil {
		return nil
	}
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %s", resp.Status)
	}
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// untarGz extracts a .tar.gz archive to destDir, with the same
// zip-slip guard as unzip in gradle.go.
func untarGz(tarGzPath, destDir string) error {
	f, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		target := filepath.Join(destDir, hdr.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("unsafe tar entry: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, hdr.FileInfo().Mode())
			if err != nil {
				return err
			}
			_, err = io.Copy(out, tr)
			out.Close()
			if err != nil {
				return err
			}
		case tar.TypeSymlink:
			// Temurin tarballs carry a few symlinks; best effort only.
			_ = os.MkdirAll(filepath.Dir(target), 0755)
			_ = os.Symlink(hdr.Linkname, target)
		}
	}
	return nil
}

var javaVerRe = regexp.MustCompile(`version "([0-9]+)`)

// javaMajor runs `java -version` and returns the major version
// (17, 21, ...). It returns 0 when the version cannot be read —
// callers treat 0 as "unknown, trust it".
func javaMajor(javaBin string) int {
	// java -version prints to stderr, so CombinedOutput is needed.
	out, err := exec.Command(javaBin, "-version").CombinedOutput()
	if err != nil {
		return 0
	}
	m := javaVerRe.FindSubmatch(out)
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(string(m[1]))
	return n
}

// javaUsable reports whether a java binary can run Gradle 8.x +
// AGP 8.x: major version 17-21, or unknown (trust it).
func javaUsable(javaBin string) bool {
	v := javaMajor(javaBin)
	return v == 0 || (v >= 17 && v <= 21)
}

func ktdevJDKHome() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ktdev", "jdk", "temurin-17")
}

// findJava returns (javaBin, javaHome, source). source is one of
// "JAVA_HOME", "system", "ktdev", or "" when no usable Java exists.
// It never downloads anything — use ensureJava for that.
func findJava() (string, string, string) {
	exe := javaExeName()
	if jh := os.Getenv("JAVA_HOME"); jh != "" {
		if bin := filepath.Join(jh, "bin", exe); fileExists(bin) && javaUsable(bin) {
			return bin, jh, "JAVA_HOME"
		}
	}
	if p, err := lookPath("java"); err == nil && javaUsable(p) {
		return p, "", "system"
	}
	home := ktdevJDKHome()
	if bin := filepath.Join(home, "bin", exe); fileExists(bin) {
		return bin, home, "ktdev"
	}
	return "", "", ""
}

// ensureJava makes sure a `java` is available for Gradle: system Java
// wins, else the ktdev cache, else Temurin 17 is downloaded once.
// It sets JAVA_HOME + PATH for the Gradle child processes.
func ensureJava() {
	if bin, home, _ := findJava(); bin != "" {
		if home != "" {
			os.Setenv("JAVA_HOME", home)
		}
		addToPath(filepath.Dir(bin))
		return
	}
	say("Java not found — downloading Eclipse Temurin 17 once...")
	bin, home, err := downloadTemurin17()
	if err != nil {
		die("could not install Java (" + err.Error() + "). Install JDK 17 once yourself and retry (e.g. sudo pacman -S jdk17-openjdk).")
	}
	say("Java: " + bin)
	os.Setenv("JAVA_HOME", home)
	addToPath(filepath.Dir(bin))
}

// downloadTemurin17 downloads the Temurin 17 JDK for this OS/arch
// from Adoptium and extracts it to ~/.ktdev/jdk/temurin-17/.
// It returns (javaBin, jdkHome).
func downloadTemurin17() (string, string, error) {
	osName := runtime.GOOS
	adoptOs := osName
	if osName == "darwin" {
		adoptOs = "mac"
	}
	arch := runtime.GOARCH
	adoptArch := arch
	if arch == "amd64" {
		adoptArch = "x64"
	} else if arch == "arm64" {
		adoptArch = "aarch64"
	}
	url := "https://api.adoptium.net/v3/binary/latest/17/ga/" + adoptOs + "/" + adoptArch + "/jdk/hotspot/normal/eclipse"

	home, _ := os.UserHomeDir()
	dlDir := filepath.Join(home, ".ktdev", "jdk", "downloads")
	if err := os.MkdirAll(dlDir, 0755); err != nil {
		return "", "", err
	}
	ext := ".tar.gz"
	if osName == "windows" {
		ext = ".zip"
	}
	archive := filepath.Join(dlDir, "temurin-17"+ext)
	if err := downloadFile(url, archive); err != nil {
		return "", "", err
	}
	tmpDir := filepath.Join(home, ".ktdev", "jdk", "tmp-extract")
	_ = os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", "", err
	}
	var err error
	if ext == ".zip" {
		err = unzip(archive, tmpDir)
	} else {
		err = untarGz(archive, tmpDir)
	}
	if err != nil {
		return "", "", err
	}
	// Find the extracted JDK root: the top-level dir with bin/java.
	exe := javaExeName()
	root := ""
	if fileExists(filepath.Join(tmpDir, "bin", exe)) {
		root = tmpDir
	} else if entries, _ := os.ReadDir(tmpDir); entries != nil {
		for _, e := range entries {
			if e.IsDir() && fileExists(filepath.Join(tmpDir, e.Name(), "bin", exe)) {
				root = filepath.Join(tmpDir, e.Name())
				break
			}
		}
	}
	if root == "" {
		return "", "", fmt.Errorf("downloaded Temurin 17 but bin/%s was not found after extract", exe)
	}
	dest := ktdevJDKHome()
	_ = os.RemoveAll(dest)
	if err := os.Rename(root, dest); err != nil {
		return "", "", err
	}
	_ = os.RemoveAll(tmpDir)
	bin := filepath.Join(dest, "bin", exe)
	if runtime.GOOS != "windows" {
		_ = os.Chmod(bin, 0755)
	}
	if !fileExists(bin) {
		return "", "", fmt.Errorf("Temurin 17 install failed: %s missing", bin)
	}
	return bin, dest, nil
}

func ktdevPlatformToolsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ktdev", "platform-tools")
}

// findAdb returns (adbPath, source). source is one of "system",
// "ktdev", or "" when adb is nowhere to be found. It never downloads
// anything — use ensureAdb for that.
func findAdb() (string, string) {
	if p, err := lookPath("adb"); err == nil {
		return p, "system"
	}
	if p := filepath.Join(ktdevPlatformToolsDir(), adbExeName()); fileExists(p) {
		return p, "ktdev"
	}
	return "", ""
}

// ensureAdb makes sure `adb` is available: system / SDK copy wins,
// else the ktdev cache, else Google's platform-tools zip is
// downloaded once. It adds the platform-tools dir to PATH.
func ensureAdb() {
	if p, _ := findAdb(); p != "" {
		addToPath(filepath.Dir(p))
		return
	}
	say("adb not found — downloading Google platform-tools once...")
	p, err := downloadPlatformTools()
	if err != nil {
		die("could not install adb (" + err.Error() + "). Install Android SDK platform-tools once yourself and retry.")
	}
	say("adb: " + p)
	addToPath(filepath.Dir(p))
}

// downloadPlatformTools downloads the platform-tools zip for this OS
// from Google and extracts it to ~/.ktdev/platform-tools/.
// It returns the path to adb.
func downloadPlatformTools() (string, error) {
	osName := runtime.GOOS // linux, windows, darwin — Google's names as-is.
	url := "https://dl.google.com/android/repository/platform-tools-latest-" + osName + ".zip"

	home, _ := os.UserHomeDir()
	dlDir := filepath.Join(home, ".ktdev", "downloads")
	if err := os.MkdirAll(dlDir, 0755); err != nil {
		return "", err
	}
	archive := filepath.Join(dlDir, "platform-tools-latest-"+osName+".zip")
	if err := downloadFile(url, archive); err != nil {
		return "", err
	}
	// The zip holds a top-level platform-tools/ dir; extract it
	// straight into ~/.ktdev/ so it lands at ~/.ktdev/platform-tools/.
	if err := unzip(archive, filepath.Join(home, ".ktdev")); err != nil {
		return "", err
	}
	adb := filepath.Join(ktdevPlatformToolsDir(), adbExeName())
	if runtime.GOOS != "windows" {
		_ = os.Chmod(adb, 0755)
	}
	if !fileExists(adb) {
		return "", fmt.Errorf("platform-tools downloaded but %s was not found after extract", adb)
	}
	return adb, nil
}
