package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

// capturedRun captures everything a command handler writes to os.Stdout and
// os.Stderr while it runs, and reports whether the command "succeeded".
//
// The Rust integration tests shell out to the `sm` binary and assert on
// output.status.success(), output.stdout and output.stderr. Calling the ported
// handlers directly, we reproduce the same observable behavior:
//
//   - stdout is the handler's success/preview text (fmt.Println / OutputSuccess).
//   - stderr is the handler's text-mode error output (cli.OutputError writes
//     "error: <msg>\n" + optional "hint: <hint>\n" to os.Stderr). Handlers that
//     emit via OutputError return errors.New(""), so the message text is already
//     on stderr.
//   - For handlers that instead return a non-empty error, the `sm` binary's
//     dispatchError formats it via output_error in text mode, writing
//     "error: <msg>\n" to stderr. We mirror that here so error-text assertions
//     match the binary's stderr.
//   - success == (the binary would exit 0) == (the returned error is nil). Any
//     non-nil error (including the empty-message errors.New("")) causes exit(1)
//     in the binary, i.e. !status.success().
type capturedRun struct {
	stdout  string
	stderr  string
	success bool
}

// runCaptured runs fn (a command handler invocation) with os.Stdout and
// os.Stderr redirected to pipes, returning the captured output.
func runCaptured(t *testing.T, fn func() error) capturedRun {
	t.Helper()

	origStdout := os.Stdout
	origStderr := os.Stderr

	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}

	os.Stdout = outW
	os.Stderr = errW

	var outBuf, errBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(&outBuf, outR)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(&errBuf, errR)
	}()

	runErr := fn()

	os.Stdout = origStdout
	os.Stderr = origStderr
	_ = outW.Close()
	_ = errW.Close()
	wg.Wait()
	_ = outR.Close()
	_ = errR.Close()

	stderr := errBuf.String()
	// Mirror the binary's dispatchError: a returned non-empty error message is
	// emitted via output_error (text mode -> "error: <msg>\n" on stderr).
	if runErr != nil && runErr.Error() != "" {
		stderr += "error: " + runErr.Error() + "\n"
	}

	return capturedRun{
		stdout:  outBuf.String(),
		stderr:  stderr,
		success: runErr == nil,
	}
}

// repoRoot returns the absolute path to the repository root (the directory that
// contains the test-data/ directory). It mirrors the Rust CARGO_MANIFEST_DIR
// helper used by the integration tests.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to determine caller path")
	}
	// thisFile = <repo>/internal/cli/cmd/integration_helpers_test.go
	dir := filepath.Dir(thisFile)
	root := filepath.Clean(filepath.Join(dir, "..", "..", ".."))
	return root
}

// testDataPath builds an absolute path to a fixture under test-data/, mirroring
// the Rust test_data_path helper.
func testDataPath(t *testing.T, filename string) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "test-data", filename)
}
