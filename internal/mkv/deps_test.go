package mkv

import (
	"errors"
	"testing"
)

func TestVerifyDependencies(t *testing.T) {
	original := execLookPath
	t.Cleanup(func() { execLookPath = original })

	t.Run("all present", func(t *testing.T) {
		execLookPath = func(file string) (string, error) {
			return "C:/tools/" + file, nil
		}
		if err := VerifyDependencies(); err != nil {
			t.Fatalf("VerifyDependencies() unexpected error: %v", err)
		}
	})

	t.Run("missing one", func(t *testing.T) {
		execLookPath = func(file string) (string, error) {
			if file == "ffprobe" {
				return "", errors.New("not found")
			}
			return "C:/tools/" + file, nil
		}
		err := VerifyDependencies()
		if err == nil {
			t.Fatal("VerifyDependencies() expected error, got nil")
		}
		if got := err.Error(); got == "" {
			t.Fatal("VerifyDependencies() expected non-empty error")
		}
	})
}
