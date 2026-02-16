package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponseWriter_DefaultStatus(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	rw := newResponseWriter(rec)

	if rw.statusCode != http.StatusOK {
		t.Errorf("default statusCode = %d, want %d", rw.statusCode, http.StatusOK)
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	rw := newResponseWriter(rec)

	rw.WriteHeader(http.StatusNotFound)

	if rw.statusCode != http.StatusNotFound {
		t.Errorf("statusCode = %d, want %d", rw.statusCode, http.StatusNotFound)
	}
	if !rw.headerWritten {
		t.Error("headerWritten = false, want true")
	}
	if rec.Code != http.StatusNotFound {
		t.Errorf("recorder Code = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestResponseWriter_WriteHeaderOnlyFirstCallTakesEffect(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	rw := newResponseWriter(rec)

	rw.WriteHeader(http.StatusCreated)
	rw.WriteHeader(http.StatusNotFound) // should be ignored

	if rw.statusCode != http.StatusCreated {
		t.Errorf("statusCode = %d, want %d (first call)", rw.statusCode, http.StatusCreated)
	}
}

func TestResponseWriter_Write(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	rw := newResponseWriter(rec)

	n, err := rw.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != 5 {
		t.Errorf("Write() = %d, want 5", n)
	}
	if rw.written != 5 {
		t.Errorf("written = %d, want 5", rw.written)
	}
	if !rw.headerWritten {
		t.Error("headerWritten = false after Write, want true")
	}
}

func TestResponseWriter_WriteAccumulatesBytes(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	rw := newResponseWriter(rec)

	_, _ = rw.Write([]byte("abc"))
	_, _ = rw.Write([]byte("de"))

	if rw.written != 5 {
		t.Errorf("written = %d, want 5", rw.written)
	}
}

func TestResponseWriter_Unwrap(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	rw := newResponseWriter(rec)

	inner := rw.Unwrap()
	if inner != rec {
		t.Error("Unwrap() did not return the underlying writer")
	}
}
