package etag

import (
	"bufio"
	"net"
	"net/http"
	"strconv"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

type etagWriter struct {
	responseWrapper

	body             []byte
	strength         Strength
	flushed          bool
	maxBufferSize    int
	hashFunc         func([]byte) string
	skipIfPresent    bool
	onError          func(*errorfamily.Error)
	onETagGenerated  func(ETag)
	on304            func(ETag)
	onBufferOverflow func(int)
}

func newETagWriter(resp http.ResponseWriter, cfg ETagConfig) *etagWriter {
	hashFunc := cfg.HashFunc
	if hashFunc == nil {
		hashFunc = defaultHashFunc
	}

	// A non-positive MaxBufferSize would disable the overflow guard in Write,
	// causing unbounded buffering. Clamp to the default so a zero-value
	// ETagConfig is safe even when the caller skips Validate.
	maxBufferSize := cfg.MaxBufferSize
	if maxBufferSize <= 0 {
		maxBufferSize = defaultMaxBufferSize
	}

	return &etagWriter{
		responseWrapper:  newResponseWrapper(resp),
		body:             nil,
		strength:         cfg.Strength,
		flushed:          false,
		maxBufferSize:    maxBufferSize,
		hashFunc:         hashFunc,
		skipIfPresent:    cfg.SkipIfPresent,
		onError:          cfg.OnError,
		onETagGenerated:  cfg.OnETagGenerated,
		on304:            cfg.On304,
		onBufferOverflow: cfg.OnBufferOverflow,
	}
}

// reportWriteErr wraps a post-commit write failure as a classified
// *errorfamily.Error and forwards it to the OnError callback if configured.
// The HTTP response is already in-flight, so the error cannot be surfaced
// to the client or returned from Write; this hook exists for observability
// (logging, metrics, tracing).
func (w *etagWriter) reportWriteErr(err error, message string) {
	classified := codeETagWriteFailed.WrapTransient(err, message)

	if w.onError != nil {
		w.onError(classified)
	}
}

func (w *etagWriter) Write(b []byte) (int, error) {
	w.writeDefaultOK()

	if w.flushed {
		n, err := w.ResponseWriter.Write(b)
		if err != nil {
			return n, codeETagWriteFailed.WrapTransient(
				err,
				"etag writer streaming write failed",
			)
		}

		return n, nil
	}

	if w.maxBufferSize > 0 && len(w.body)+len(b) > w.maxBufferSize {
		if w.onBufferOverflow != nil {
			w.onBufferOverflow(w.maxBufferSize)
		}

		w.Flush()

		n, err := w.ResponseWriter.Write(b)
		if err != nil {
			return n, codeETagWriteFailed.WrapTransient(
				err,
				"etag writer overflow write failed",
			)
		}

		return n, nil
	}

	w.body = append(w.body, b...)

	return len(b), nil
}

func (w *etagWriter) flush(req *http.Request) {
	if w.flushed {
		return
	}

	tag := w.resolveETag()

	if tag.IsValid() {
		if w.shouldReturnNotModified(req, tag) {
			// RFC 7232 §4.1: a 304 response carries no message body, so drop
			// any body-length metadata the handler may have attached.
			w.Header().Del(headerContentLength)
			w.ResponseWriter.WriteHeader(http.StatusNotModified)
			w.headerCommitted = true

			if w.on304 != nil {
				w.on304(tag)
			}

			return
		}
	}

	// RFC 7230 §3.3: HEAD responses MUST NOT include a message body. The
	// buffered bytes still let us advertise an accurate Content-Length so a
	// client can size the representation without transferring it.
	if req.Method == http.MethodHead {
		w.Header().Set(headerContentLength, strconv.Itoa(len(w.body)))
	}

	w.markFlushed()

	w.writeHeaderToUnderlying()

	if req.Method == http.MethodHead {
		return
	}

	// Post-header-commit body writes are fundamentally unreportable: the
	// handler has returned and the HTTP response is already in-flight.
	// The error is classified and forwarded to OnError for observability.
	if len(w.body) > 0 {
		_, err := w.ResponseWriter.Write(w.body)
		if err != nil {
			w.reportWriteErr(err, "etag flush body write failed")
		}
	}
}

// resolveETag determines which ETag to use for this response. If
// SkipIfPresent is configured and the handler already set a valid ETag
// header, that value is respected. Otherwise, a new ETag is computed from
// the buffered body and set on the response header.
func (w *etagWriter) resolveETag() ETag {
	if w.skipIfPresent {
		existing := w.Header().Get(headerETag)
		if existing != "" {
			if tag, ok := ParseETag(existing); ok {
				return tag
			}
		}
	}

	tag := w.computeETag()

	if tag.IsValid() {
		w.Header().Set(headerETag, tag.String())

		if w.onETagGenerated != nil {
			w.onETagGenerated(tag)
		}
	}

	return tag
}

func (w *etagWriter) shouldReturnNotModified(req *http.Request, tag ETag) bool {
	if !w.isCacheableStatus() {
		return false
	}

	inm := strings.Join(req.Header.Values(headerIfNoneMatch), ", ")
	if inm == "" {
		return false
	}

	return MatchesIfNoneMatch(tag, inm)
}

func (w *etagWriter) computeETag() ETag {
	if len(w.body) == 0 && !w.headerBuffered {
		return ETag{}
	}

	opaque := w.hashFunc(w.body)

	return NewETag(opaque, w.strength)
}

func (w *etagWriter) isCacheableStatus() bool {
	return w.status == 0 || (w.status >= http.StatusOK && w.status < http.StatusMultipleChoices)
}

func (w *etagWriter) Flush() {
	if w.flushed {
		w.responseWrapper.Flush()

		return
	}

	w.markFlushed()

	w.writeHeaderToUnderlying()

	// Post-header-commit body writes are fundamentally unreportable: the
	// handler has returned and the HTTP response is already in-flight.
	// The error is classified and forwarded to OnError for observability.
	if len(w.body) > 0 {
		_, err := w.ResponseWriter.Write(w.body)
		if err != nil {
			w.reportWriteErr(err, "etag flush body write failed")
		}

		w.body = w.body[:0]
	}

	w.responseWrapper.Flush()
}

func (w *etagWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.markFlushed()

	w.writeHeaderToUnderlying()

	return w.responseWrapper.Hijack()
}

// markFlushed transitions the writer into streaming mode by setting the
// flushed flag. The buffered status header is committed separately via
// writeHeaderToUnderlying.
func (w *etagWriter) markFlushed() {
	w.flushed = true
}
