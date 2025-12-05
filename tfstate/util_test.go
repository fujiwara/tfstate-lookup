package tfstate

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"testing"
)

func TestHideStderr(t *testing.T) {
	tests := []struct {
		name           string
		fn             func() error
		expectError    bool
		expectOutput   bool
		expectedStderr string
	}{
		{
			name: "no error, no stderr output",
			fn: func() error {
				return nil
			},
			expectError:    false,
			expectOutput:   false,
			expectedStderr: "",
		},
		{
			name: "no error, with stderr output (should be hidden)",
			fn: func() error {
				fmt.Fprintln(os.Stderr, "this should be hidden")
				return nil
			},
			expectError:    false,
			expectOutput:   false,
			expectedStderr: "",
		},
		{
			name: "with error, with stderr output (should be shown)",
			fn: func() error {
				fmt.Fprintln(os.Stderr, "error message")
				return errors.New("test error")
			},
			expectError:    true,
			expectOutput:   true,
			expectedStderr: "error message\n",
		},
		{
			name: "with error, no stderr output",
			fn: func() error {
				return errors.New("test error")
			},
			expectError:    true,
			expectOutput:   false,
			expectedStderr: "",
		},
		{
			name: "with error, log.Println to stderr",
			fn: func() error {
				logger := log.New(os.Stderr, "", 0)
				logger.Println("log error message")
				return errors.New("test error")
			},
			expectError:    true,
			expectOutput:   true,
			expectedStderr: "log error message\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture the actual stderr output
			origStderr := os.Stderr
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("failed to create pipe: %v", err)
			}
			os.Stderr = w

			var captured bytes.Buffer
			done := make(chan struct{})
			go func() {
				io.Copy(&captured, r)
				close(done)
			}()

			// Run the function under test
			fnErr := hideStderr(tt.fn)

			// Restore stderr and close pipe
			os.Stderr = origStderr
			w.Close()
			<-done
			r.Close()

			// Check error
			if tt.expectError && fnErr == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && fnErr != nil {
				t.Errorf("expected no error, got %v", fnErr)
			}

			// Check stderr output
			if tt.expectOutput {
				if !strings.Contains(captured.String(), tt.expectedStderr) {
					t.Errorf("expected stderr %q, got %q", tt.expectedStderr, captured.String())
				}
			} else {
				if captured.Len() > 0 {
					t.Errorf("expected no stderr output, got %q", captured.String())
				}
			}
		})
	}
}
