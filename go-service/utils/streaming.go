package utils

import (
	"fmt"
	"net/http"
)

// StreamSSEResponse streams data from the out and errs channels to the ResponseWriter.
// The processChunk callback lets you customize per-chunk behavior (e.g. accumulating text).
// Returns an error if the stream fails.
func StreamSSEResponse(
	w http.ResponseWriter,
	out <-chan string,
	errs <-chan error,
	processChunk func(chunk string),
) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported")
	}
	// Set up SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for {
		select {
		case chunk, ok := <-out:
			if !ok {
				return nil // normal termination
			}
			// Write the chunk to the client.
			fmt.Fprintf(w, "%s\n", chunk)
			flusher.Flush()
			// Allow caller to process the chunk (e.g. accumulate it).
			processChunk(chunk)
		case err, ok := <-errs:
			if ok {
				return err
			}
		}
	}
}
