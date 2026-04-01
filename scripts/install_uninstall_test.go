package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestUninstallScriptRemovesInstalledFiles(t *testing.T) {
	scriptsDir := scriptDir(t)

	tempHome := t.TempDir()
	tempPrefix := filepath.Join(t.TempDir(), "prefix")

	runScript(t, scriptsDir, "install.sh", tempHome, tempPrefix, "--autostart")

	assertExists(t, filepath.Join(tempPrefix, "bin", "codex-switch"))
	assertExists(t, filepath.Join(tempPrefix, "share", "applications", "codex-switch.desktop"))
	assertExists(t, filepath.Join(tempPrefix, "share", "pixmaps", "codex-switch.png"))
	assertExists(t, filepath.Join(tempPrefix, "share", "icons", "hicolor", "256x256", "apps", "codex-switch.png"))
	assertExists(t, filepath.Join(tempHome, ".config", "autostart", "codex-switch.desktop"))

	runScript(t, scriptsDir, "uninstall.sh", tempHome, tempPrefix)

	assertMissing(t, filepath.Join(tempPrefix, "bin", "codex-switch"))
	assertMissing(t, filepath.Join(tempPrefix, "share", "applications", "codex-switch.desktop"))
	assertMissing(t, filepath.Join(tempPrefix, "share", "pixmaps", "codex-switch.png"))
	assertMissing(t, filepath.Join(tempPrefix, "share", "icons", "hicolor", "64x64", "apps", "codex-switch.png"))
	assertMissing(t, filepath.Join(tempPrefix, "share", "icons", "hicolor", "256x256", "apps", "codex-switch.png"))
	assertMissing(t, filepath.Join(tempHome, ".config", "autostart", "codex-switch.desktop"))
}

func scriptDir(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	return dir
}

func runScript(t *testing.T, scriptsDir string, scriptName string, home string, prefix string, args ...string) {
	t.Helper()

	scriptPath := filepath.Join(scriptsDir, scriptName)
	cmd := exec.Command(scriptPath, args...)
	cmd.Dir = filepath.Dir(scriptsDir)
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"PREFIX="+prefix,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run %s: %v\n%s", scriptName, err, output)
	}
}

func assertExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func assertMissing(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be removed, got err=%v", path, err)
	}
}
