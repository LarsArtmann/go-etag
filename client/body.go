package etagclient

import (
	"bytes"
	"io"
	"net/http"
)

// chainedBody chains an already-buffered prefix with the unread remainder of
// the original body, so a passthrough response still streams every byte and
// Close still reaches the original body.
type chainedBody struct {
	reader io.Reader
	body   io.ReadCloser
}

func newChainedBody(buffered []byte, body io.ReadCloser) *chainedBody {
	return &chainedBody{
		reader: io.MultiReader(bytes.NewReader(buffered), body),
		body:   body,
	}
}

// drainAndClose discards any residual 304 body and releases the connection.
// Both failures are deliberately ignored: the 304 response is discarded
// either way, and net/http closes a connection whose body was not fully
// drained, so the worst case is losing connection reuse.
func drainAndClose(resp *http.Response) {
	_, _ = io.Copy(io.Discard, resp.Body) //nolint:erraudit // best-effort drain of a body we are discarding
	_ = resp.Body.Close()                 //nolint:erraudit // a failed teardown of a discarded response needs no handling
}
