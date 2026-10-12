package updater

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestRestartDoesNotSpawnAfterShutdownFailure(t *testing.T) {
	const childEnv = "LOCALFINANCE_TEST_RESTART_FAILURE"
	if os.Getenv(childEnv) == "1" {
		// If the hook error is ignored, cmd.Start fails and log.Fatal exits this
		// subprocess instead of reporting a completed, safely aborted attempt.
		lastTargetExe = "localfinance-nonexistent-restart-test-executable"
		RegisterServerContext(0, "", func(context.Context) error {
			return errors.New("ledger could not be released")
		})
		RestartServer()
		select {
		case <-relaunchDone:
		case <-time.After(5 * time.Second):
			t.Fatal("aborted restart did not wake main")
		}
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestRestartDoesNotSpawnAfterShutdownFailure$")
	cmd.Env = append(os.Environ(), childEnv+"=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("restart spawned a child or failed to finish: %v\n%s", err, output)
	}
}
