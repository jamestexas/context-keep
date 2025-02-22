package utils_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jamestexas/context-keep/go-service/utils"
)

// flushableResponseRecorder wraps httptest.ResponseRecorder to satisfy http.Flusher.
type flushableResponseRecorder struct {
	*httptest.ResponseRecorder
}

type nonFlushableResponseWriter struct {
	header http.Header
	body   strings.Builder
}

func (f *flushableResponseRecorder) Flush() {
	// No-op: httptest.ResponseRecorder doesn't flush, but this satisfies the interface.
}
func (n *nonFlushableResponseWriter) Header() http.Header {
	if n.header == nil {
		n.header = make(http.Header)
	}
	return n.header
}
func (n *nonFlushableResponseWriter) Write(b []byte) (int, error) {
	return n.body.Write(b)
}
func (n *nonFlushableResponseWriter) WriteHeader(statusCode int) {
	// no-op for testing
}
func TestStreamSSEResponse_NormalTermination(t *testing.T) {
	// Create output channel with some chunks.
	out := make(chan string, 3)
	errs := make(chan error, 1)
	out <- "chunk1"
	out <- "chunk2"
	close(out) // normal termination; no error.

	// Collect chunks processed by the callback.
	var processedChunks []string
	processFunc := func(chunk string) {
		processedChunks = append(processedChunks, chunk)
	}

	rr := httptest.NewRecorder()
	// Wrap rr so it implements http.Flusher.
	rec := &flushableResponseRecorder{rr}

	err := utils.StreamSSEResponse(rec, out, errs, processFunc)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check SSE headers.
	headers := rec.Header()
	if headers.Get("Content-Type") != "text/event-stream" {
		t.Errorf("Expected Content-Type 'text/event-stream', got: %s", headers.Get("Content-Type"))
	}
	if headers.Get("Cache-Control") != "no-cache" {
		t.Errorf("Expected Cache-Control 'no-cache', got: %s", headers.Get("Cache-Control"))
	}
	if headers.Get("Connection") != "keep-alive" {
		t.Errorf("Expected Connection 'keep-alive', got: %s", headers.Get("Connection"))
	}

	// Verify the response body contains the chunks.
	body := rec.Body.String()
	if !strings.Contains(body, "chunk1") || !strings.Contains(body, "chunk2") {
		t.Errorf("Expected body to contain 'chunk1' and 'chunk2', got: %s", body)
	}

	// Check that the callback was called for each chunk.
	if len(processedChunks) != 2 {
		t.Errorf("Expected processChunk to be called 2 times, got: %d", len(processedChunks))
	}
}

func TestStreamSSEResponse_ErrorCase(t *testing.T) {
	// Create channels: out remains open (no value sent), errs gets an error.
	out := make(chan string)
	errs := make(chan error, 1)
	testErr := errors.New("test error")
	errs <- testErr
	// Note: Do not close out or errs yet.

	var processedChunks []string
	processFunc := func(chunk string) {
		processedChunks = append(processedChunks, chunk)
	}

	rr := httptest.NewRecorder()
	rec := &flushableResponseRecorder{rr}

	// Since out has no value and errs has a value, the select should pick the error case.
	err := utils.StreamSSEResponse(rec, out, errs, processFunc)
	if err == nil {
		t.Fatal("Expected an error, but got nil")
	}
	if err.Error() != testErr.Error() {
		t.Errorf("Expected error %q, got: %q", testErr.Error(), err.Error())
	}

	// Close channels now to clean up.
	close(out)
	close(errs)
}

func TestStreamSSEResponse_Unsupported(t *testing.T) {
	// Use our non-flushable writer.
	nfw := &nonFlushableResponseWriter{}
	out := make(chan string)
	errs := make(chan error, 1)
	close(out)
	close(errs)

	err := utils.StreamSSEResponse(nfw, out, errs, func(chunk string) {})
	if err == nil {
		t.Fatal("Expected error due to unsupported streaming, got nil")
	}
	expected := "streaming unsupported"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("Expected error to contain %q, got: %v", expected, err)
	}
}
