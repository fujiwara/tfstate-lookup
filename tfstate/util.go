package tfstate

// hideStderr temporarily hides stderr output while executing the given function.

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"syscall"
)

// hideStderr temporarily hides stderr output while executing the given function.
func hideStderr(fn func() error) error {
	origFd, err := syscall.Dup(int(os.Stderr.Fd()))
	if err != nil {
		return fmt.Errorf("failed to dup stderr: %w", err)
	}
	defer syscall.Close(origFd)

	r, w, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}

	if err := syscall.Dup2(int(w.Fd()), int(os.Stderr.Fd())); err != nil {
		r.Close()
		w.Close()
		return fmt.Errorf("failed to redirect stderr: %w", err)
	}

	var buf bytes.Buffer
	done := make(chan struct{})

	go func() {
		io.Copy(&buf, r)
		close(done)
	}()

	fnErr := fn()

	syscall.Dup2(origFd, int(os.Stderr.Fd()))
	w.Close()
	<-done
	r.Close()

	if fnErr != nil && buf.Len() > 0 {
		syscall.Write(origFd, buf.Bytes())
	}

	return fnErr
}
