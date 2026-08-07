package etag

import (
	"bufio"
	"net"
	"net/http"

	errorfamily "github.com/larsartmann/go-error-family"
)

// responseWrapper provides common ResponseWriter wrapping behavior used by
// etagWriter. It buffers WriteHeader calls and delegates Hijack and Flush
// to the underlying writer when supported.
type responseWrapper struct {
	http.ResponseWriter

	status          int
	headerBuffered  bool // handler called WriteHeader; status captured but not yet committed
	headerCommitted bool // status written to the underlying ResponseWriter
}

func newResponseWrapper(resp http.ResponseWriter) responseWrapper {
	return responseWrapper{
		ResponseWriter:  resp,
		status:          0,
		headerBuffered:  false,
		headerCommitted: false,
	}
}

func (w *responseWrapper) WriteHeader(code int) {
	if !w.headerBuffered {
		w.status = code
		w.headerBuffered = true
	}
}

func (w *responseWrapper) writeHeaderToUnderlying() {
	if w.headerBuffered && !w.headerCommitted {
		w.ResponseWriter.WriteHeader(w.status)
		w.headerCommitted = true
	}
}

// writeDefaultOK commits a 200 OK status to the underlying ResponseWriter if
// no status has been written yet. Use this at the top of Write methods on
// wrapper types to honor Go's net/http contract: the first Write implicitly
// sends 200 if WriteHeader was not called.
func (w *responseWrapper) writeDefaultOK() {
	if !w.headerBuffered {
		w.WriteHeader(http.StatusOK)
	}
}

func hijackDelegate(w http.ResponseWriter) (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, nil, errorfamily.WrapInfrastructure(
			http.ErrNotSupported,
			ErrCodeHijackUnsupported,
			"response writer does not implement http.Hijacker",
		).WithContextf("writer_type", "%T", w)
	}

	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return conn, rw, errorfamily.WrapTransient(
			err,
			ErrCodeHijackFailed,
			"response writer hijack failed",
		).WithContextf("writer_type", "%T", w)
	}

	return conn, rw, nil
}

func flushDelegate(w http.ResponseWriter) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *responseWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return hijackDelegate(w.ResponseWriter)
}

func (w *responseWrapper) Flush() {
	flushDelegate(w.ResponseWriter)
}
