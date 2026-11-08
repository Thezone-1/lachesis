package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type check struct {
	name string
	ok   bool
	hint string
}

// doctorCmd checks that Lachesis can do its job here and says how to fix what it cannot.
func doctorCmd(args []string) error {
	o := termOpts(loadConfig())
	p := o.p()

	var checks []check
	dirErr := os.MkdirAll(baseDir(), 0o700)
	if dirErr == nil {
		probe := filepath.Join(baseDir(), ".probe")
		if dirErr = os.WriteFile(probe, nil, 0o600); dirErr == nil {
			os.Remove(probe)
		}
	}
	checks = append(checks, check{"data folder is writable (" + baseDir() + ")", dirErr == nil, fmt.Sprint(dirErr)})
	_, gitErr := exec.LookPath("git")
	checks = append(checks, check{"git is installed (needed for commit yield)", gitErr == nil, "install git, or ignore this if you do not list any repos"})
	checks = append(checks, platformChecks()...)

	bad := 0
	fmt.Println()
	for _, c := range checks {
		if c.ok {
			fmt.Println("  " + paint(o, p.ok, "✓") + " " + c.name)
		} else {
			bad++
			fmt.Println("  " + paint(o, p.bad, "✗") + " " + c.name)
			fmt.Println("    " + paint(o, p.hint, c.hint))
		}
	}
	fmt.Println()
	if bad == 0 {
		fmt.Println("  Everything is ready. Run `lachesis start` when you begin to work.")
	} else {
		fmt.Printf("  %d thing%s to fix, then run `lachesis doctor` again.\n", bad, map[bool]string{true: "", false: "s"}[bad == 1])
		if flagPresent(args, "--open") {
			openPrivacySettings()
		}
	}
	fmt.Println()
	return nil
}

func flagPresent(args []string, name string) bool {
	for _, a := range args {
		if a == name {
			return true
		}
	}
	return false
}
