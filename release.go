package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// cmdRelease mirrors the Bash release helper, with the same three fixes:
//   - no `git add -A` auto-commit on a dirty tree unless --yes,
//   - the tag is removed again if the build / stamp check fails,
//   - the APK versionName stamp must match the tag before upload.
func cmdRelease(args []string) {
	var raw, notes, bump string
	dryRun, yes := false, false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--major":
			bump = "major"
		case a == "--minor":
			bump = "minor"
		case a == "--dry-run":
			dryRun = true
		case a == "--yes":
			yes = true
		case strings.HasPrefix(a, "v") || (len(a) > 0 && a[0] >= '0' && a[0] <= '9'):
			if raw == "" {
				raw = a
			} else if notes == "" {
				notes = a
			} else {
				die("unexpected argument: " + a)
			}
		default:
			if notes == "" {
				notes = a
			} else {
				die("unexpected argument: " + a)
			}
		}
	}

	if _, err := runCapture("git", "rev-parse", "--is-inside-work-tree"); err != nil {
		die("not inside a git project.")
	}
	root := mustRoot()
	if err := os.Chdir(root); err != nil {
		die(err.Error())
	}
	say("Project: " + root)

	if err := runLive("git", "fetch", "--tags", "origin"); err != nil {
		if dryRun {
			fmt.Println("WARNING: git fetch failed (no origin?) — continuing dry-run only.")
		} else {
			die("git fetch failed.")
		}
	}
	latest, _ := runCapture("git", "describe", "--tags", "--abbrev=0")
	latest = strings.TrimSpace(latest)
	if latest == "" {
		latest = "v0.0.0"
	}

	tag := ""
	if raw == "" {
		ver := strings.TrimPrefix(latest, "v")
		parts := strings.Split(ver, ".")
		major, minor, patch := 0, 0, 0
		if len(parts) > 0 {
			major, _ = strconv.Atoi(parts[0])
		}
		if len(parts) > 1 {
			minor, _ = strconv.Atoi(parts[1])
		}
		if len(parts) > 2 {
			patch, _ = strconv.Atoi(parts[2])
		}
		switch bump {
		case "major":
			tag = fmt.Sprintf("v%d.0.0", major+1)
		case "minor":
			tag = fmt.Sprintf("v%d.%d.0", major, minor+1)
		default:
			tag = fmt.Sprintf("v%d.%d.%d", major, minor, patch+1)
		}
	} else if strings.HasPrefix(raw, "v") {
		tag = raw
	} else {
		tag = "v" + raw
	}
	if !regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(tag) {
		die("bad version '" + tag + "' — use a form like v1.4.3")
	}
	if _, err := runCapture("git", "rev-parse", tag); err == nil {
		die("tag " + tag + " already exists locally. Use a fresh number.")
	}
	if out, _ := runCapture("git", "ls-remote", "--tags", "origin", "refs/tags/"+tag); strings.TrimSpace(out) != "" {
		die("tag " + tag + " already exists on GitHub. Use a fresh number.")
	}
	if !isNewer(tag, latest) {
		die(tag + " is not newer than the latest tag " + latest + ".")
	}

	// Dirty tree stops unless --yes, and the status is shown.
	if out, _ := runCapture("git", "status", "--porcelain"); strings.TrimSpace(out) != "" {
		fmt.Println("Working tree is not clean:")
		if s, _ := runCapture("git", "status", "--short"); s != "" {
			fmt.Print(s)
		}
		if !yes {
			die("commit or stash first, or re-run with --yes to release with these changes.")
		}
	}
	branch, _ := runCapture("git", "branch", "--show-current")
	branch = strings.TrimSpace(branch)
	if branch == "" {
		branch = "main"
	}
	say(latest + " -> " + tag + " (branch " + branch + ")")
	if dryRun {
		fmt.Println("DRY RUN: stopping before commit / tag / build / push.")
		return
	}

	// gh is only needed from here on, so --dry-run works without a login.
	if _, err := lookPath("gh"); err != nil {
		die("GitHub CLI missing. Install once: sudo pacman -S github-cli   then: gh auth login")
	}
	if err := runLive("gh", "auth", "status"); err != nil {
		die("gh is not logged in. Run once: gh auth login")
	}
	if _, err := runCapture("gh", "release", "view", tag); err == nil {
		die("a GitHub release " + tag + " already exists.")
	}

	if err := runLive("git", "add", "-A"); err != nil {
		die("git add failed: " + err.Error())
	}
	// Empty commit when nothing was staged, otherwise a normal commit.
	if out, _ := runCapture("git", "diff", "--cached", "--quiet"); out == "" {
		// `git diff --cached --quiet` exits 0 when there is nothing staged,
		// but runCapture only returns stdout, so check the exit code directly.
		if exec.Command("git", "diff", "--cached", "--quiet").Run() == nil {
			if err := runLive("git", "commit", "--allow-empty", "-m", tag); err != nil {
				die("git commit failed: " + err.Error())
			}
		} else if err := runLive("git", "commit", "-m", tag); err != nil {
			die("git commit failed: " + err.Error())
		}
	}
	if err := runLive("git", "tag", tag); err != nil {
		die("git tag failed: " + err.Error())
	}

	// From here, a failed build / stamp check must remove the tag again.
	failed := true
	defer func() {
		if failed {
			fmt.Println("Build failed — removing local tag " + tag + " so you can retry.")
			_ = exec.Command("git", "tag", "-d", tag).Run()
		}
	}()

	describe, _ := runCapture("git", "describe", "--tags", "--abbrev=0")
	if strings.TrimSpace(describe) != tag {
		die("git describe does not print " + tag + " — NOT building, the APK stamp would be wrong.")
	}
	say("Tag check OK: " + strings.TrimSpace(describe))

	gradle := mustGradle(root)
	say("Gradle: " + gradle)
	if err := runLive(gradle, append(gradleArgs(), "assembleRelease")...); err != nil {
		die("build failed — nothing was pushed or released.")
	}

	apk := filepath.Join("app", "build", "outputs", "apk", "release", "app-release.apk")
	if _, err := os.Stat(apk); err != nil {
		// Fall back to any release APK.
		var found string
		_ = filepath.Walk(filepath.Join("app", "build", "outputs", "apk"), func(p string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && strings.HasSuffix(p, ".apk") && strings.Contains(p, "release") && found == "" {
				found = p
			}
			return nil
		})
		apk = found
	}
	if apk == "" {
		die("release APK was not produced.")
	}
	if _, err := os.Stat(apk); err != nil {
		die(apk + " was not produced.")
	}

	if aapt := findAapt(); aapt != "" {
		out, err := runCapture(aapt, "dump", "badging", apk)
		if err == nil {
			if m := regexp.MustCompile(`versionName='([^']+)'`).FindStringSubmatch(out); m != nil {
				stamp := m[1]
				fmt.Printf("APK stamp: versionName=%s  (tag wants %s)\n", stamp, strings.TrimPrefix(tag, "v"))
				if stamp != strings.TrimPrefix(tag, "v") {
					die("APK stamp mismatch — NOT uploading. This is exactly the wrong-version update loop; do not override it.")
				}
			}
		}
	} else {
		fmt.Println("WARNING: aapt not found, stamp check skipped. Verify once yourself:")
		fmt.Println("  adb install -r " + apk)
	}

	// Build + stamp passed; the tag is now safe to keep.
	failed = false

	// Same as `ktdev build`: copy the release APK to <project>/apk/
	// as <project>-release.apk too (copy, not move — the original
	// stays where Gradle put it).
	copyBuiltApk(root, "release")

	repo := filepath.Base(root)
	home, _ := os.UserHomeDir()
	out := filepath.Join(home, repo+"-"+tag+".apk")
	if err := copyFile(apk, out); err != nil {
		die("copy APK failed: " + err.Error())
	}
	if err := runLive("git", "push", "origin", branch); err != nil {
		die("git push branch failed: " + err.Error())
	}
	if err := runLive("git", "push", "origin", tag); err != nil {
		die("git push tag failed: " + err.Error())
	}
	note := notes
	if note == "" {
		note = "Release " + tag
	}
	if err := runLive("gh", "release", "create", tag, out, "--title", repo+" "+tag, "--notes", note); err != nil {
		die("gh release create failed — code and tag are pushed, the APK is at " + out)
	}
	latestAPI, _ := runCapture("gh", "api", "repos/{owner}/{repo}/releases/latest", "--jq", ".tag_name")
	fmt.Println("GitHub releases/latest = " + strings.TrimSpace(latestAPI))
	if strings.TrimSpace(latestAPI) != tag {
		fmt.Println("WARNING: releases/latest is not " + tag + " yet — check the release is Published, not Draft/Pre-release.")
	}
	fmt.Println("DONE: " + repo + " " + tag + " is LIVE -> " + out)
	fmt.Println("Phones on an older version will be offered this update once; after installing, the app reports v" + strings.TrimPrefix(tag, "v") + " and the updater will say 'You're up to date'.")
}

func isNewer(a, b string) bool {
	parse := func(s string) []int {
		s = strings.TrimPrefix(s, "v")
		parts := strings.Split(s, ".")
		out := []int{0, 0, 0}
		for i := 0; i < len(parts) && i < 3; i++ {
			n, _ := strconv.Atoi(parts[i])
			out[i] = n
		}
		return out
	}
	av, bv := parse(a), parse(b)
	for i := 0; i < 3; i++ {
		if av[i] != bv[i] {
			return av[i] > bv[i]
		}
	}
	return false
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
