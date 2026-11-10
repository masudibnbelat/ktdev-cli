package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

var gradleVerRe = regexp.MustCompile(`gradle-([0-9][0-9.]*)-(bin|all)\.zip`)

// wrapperGradleVersion reads gradle/wrapper/gradle-wrapper.properties
// in root and returns the version from distributionUrl, e.g. "8.9".
// It returns "" when the file or the URL is missing.
func wrapperGradleVersion(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "gradle", "wrapper", "gradle-wrapper.properties"))
	if err != nil {
		return ""
	}
	m := gradleVerRe.FindSubmatch(data)
	if m == nil {
		return ""
	}
	return string(m[1])
}

func gradleExeName() string {
	if runtime.GOOS == "windows" {
		return "gradle.bat"
	}
	return "gradle"
}

func gradlewName() string {
	if runtime.GOOS == "windows" {
		return "gradlew.bat"
	}
	return "gradlew"
}

// pickGradle returns the Gradle command to run for the project in root.
// Order (dynamic, never hardcoded):
//  1. ./gradlew (or gradlew.bat on Windows) — the wrapper itself
//     downloads and uses the exact wrapper version.
//  2. A local install of the exact wrapper version:
//     ~/.local/opt/gradle-<ver>/bin/gradle, ~/.gradle/wrapper/dists/...,
//     or the ktdev cache ~/.ktdev/gradle/gradle-<ver>/bin/gradle.
//  3. Auto-download that exact version once into the ktdev cache.
//
// The Gradle on PATH is never used when the wrapper wants another
// version — that is the bug that broke AGP 8.x projects with 9.x.
func pickGradle(root string) (string, error) {
	ver := wrapperGradleVersion(root)

	// 1. Wrapper script in the project.
	wrapper := filepath.Join(root, gradlewName())
	if st, err := os.Stat(wrapper); err == nil && !st.IsDir() {
		if runtime.GOOS != "windows" {
			// Make sure it is executable; harmless if it already is.
			_ = os.Chmod(wrapper, 0755)
		}
		return wrapper, nil
	}

	if ver == "" {
		return "", fmt.Errorf("no usable Gradle found for this project: no gradlew and no wrapper version in gradle/wrapper/gradle-wrapper.properties")
	}

	// 2. Local install of the exact version.
	if found := findLocalGradle(ver); found != "" {
		return found, nil
	}

	// 3. Auto-download the exact version once.
	say("Gradle " + ver + " not installed locally — downloading it once...")
	downloaded, err := downloadGradle(ver)
	if err != nil {
		return "", fmt.Errorf("wrapper wants Gradle %s but it is not installed and the download failed (%v). Install that exact version, or run: gradle wrapper --gradle-version %s", ver, err, ver)
	}
	return downloaded, nil
}

func findLocalGradle(ver string) string {
	home, _ := os.UserHomeDir()
	exe := gradleExeName()
	var candidates []string
	if home != "" {
		candidates = append(candidates,
			filepath.Join(home, ".local", "opt", "gradle-"+ver, "bin", exe),
			filepath.Join(home, ".ktdev", "gradle", "gradle-"+ver, "bin", exe),
		)
		// Gradle's own wrapper cache: ~/.gradle/wrapper/dists/gradle-<ver>-bin/<hash>/gradle-<ver>/bin/gradle
		glob := filepath.Join(home, ".gradle", "wrapper", "dists", "gradle-"+ver+"-bin", "*", "gradle-"+ver, "bin", exe)
		if matches, _ := filepath.Glob(glob); len(matches) > 0 {
			sort.Strings(matches)
			candidates = append(candidates, matches...)
		}
	}
	if runtime.GOOS == "windows" {
		if home != "" {
			candidates = append(candidates,
				filepath.Join(home, ".local", "opt", "gradle-"+ver, "bin", exe),
			)
		}
		for _, base := range []string{os.Getenv("ProgramFiles"), os.Getenv("LOCALAPPDATA")} {
			if base != "" {
				candidates = append(candidates, filepath.Join(base, "Gradle", "gradle-"+ver, "bin", exe))
			}
		}
		candidates = append(candidates, filepath.Join("C:\\", "Gradle", "gradle-"+ver, "bin", exe))
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

func gradleCacheDir(ver string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ktdev", "gradle", "gradle-"+ver)
}

// downloadGradle downloads gradle-<ver>-bin.zip from services.gradle.org
// and extracts it to ~/.ktdev/gradle/gradle-<ver>/, then returns the
// path to bin/gradle (or bin/gradle.bat on Windows).
func downloadGradle(ver string) (string, error) {
	url := "https://services.gradle.org/distributions/gradle-" + ver + "-bin.zip"
	home, _ := os.UserHomeDir()
	dlDir := filepath.Join(home, ".ktdev", "gradle", "downloads")
	if err := os.MkdirAll(dlDir, 0755); err != nil {
		return "", err
	}
	zipPath := filepath.Join(dlDir, "gradle-"+ver+"-bin.zip")
	if _, err := os.Stat(zipPath); err != nil {
		client := &http.Client{Timeout: 10 * time.Minute}
		resp, err := client.Get(url)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return "", fmt.Errorf("HTTP %s", resp.Status)
		}
		f, err := os.Create(zipPath)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(f, resp.Body); err != nil {
			f.Close()
			return "", err
		}
		f.Close()
	}
	dest := gradleCacheDir(ver)
	if err := unzip(zipPath, filepath.Join(home, ".ktdev", "gradle")); err != nil {
		return "", err
	}
	exe := filepath.Join(dest, "bin", gradleExeName())
	if _, err := os.Stat(exe); err != nil {
		return "", fmt.Errorf("downloaded Gradle %s but %s was not found after unzip", ver, exe)
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(exe, 0755)
	}
	return exe, nil
}

func unzip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		// Guard against zip-slip: the entry must stay inside destDir.
		target := filepath.Join(destDir, f.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("unsafe zip entry: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
