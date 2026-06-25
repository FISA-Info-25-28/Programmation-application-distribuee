package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// customErrReader is a reader that always returns an error.
type customErrReader struct{ err error }

func (e customErrReader) Read(p []byte) (int, error) { return 0, e.err }

// zeroReader is a reader that always returns (0, nil) — no data, no error.
type zeroReader struct{}

func (z zeroReader) Read(p []byte) (int, error) { return 0, nil }

// ── maxFileReader extra branches ──────────────────────────────────────────────

// TestMaxFileReaderProbeError covers the branch where the probe read (r.read >= r.max)
// returns a non-EOF error, which is wrapped and returned.
func TestMaxFileReaderProbeError(t *testing.T) {
	sentinel := errors.New("probe-disk-error")
	r := &maxFileReader{reader: customErrReader{err: sentinel}, read: 10, max: 10}
	var p [1]byte
	_, err := r.Read(p[:])
	if err == nil || !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want wrapped sentinel", err)
	}
}

// TestMaxFileReaderProbeZero covers the (0, nil) return from probe when the
// underlying reader has no data yet but also no error.
func TestMaxFileReaderProbeZero(t *testing.T) {
	r := &maxFileReader{reader: zeroReader{}, read: 10, max: 10}
	var p [1]byte
	n, err := r.Read(p[:])
	if n != 0 || err != nil {
		t.Fatalf("n=%d err=%v, want (0, nil)", n, err)
	}
}

// TestMaxFileReaderNormalReadError covers the branch where the underlying reader
// returns a non-EOF error during a regular (below-limit) read.
func TestMaxFileReaderNormalReadError(t *testing.T) {
	sentinel := errors.New("read-chunk-error")
	r := &maxFileReader{reader: customErrReader{err: sentinel}, read: 0, max: 100}
	var p [10]byte
	_, err := r.Read(p[:])
	if err == nil || !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want wrapped sentinel", err)
	}
}

// ── detectMediaType extra branches ────────────────────────────────────────────

// TestDetectMediaTypeImageVideo covers the image and video cases; audio is
// already tested in TestDetectMediaTypeAcceptsAudio.
func TestDetectMediaTypeImageVideo(t *testing.T) {
	t.Run("image/jpeg returns image", func(t *testing.T) {
		mt, ok := detectMediaType("image/jpeg")
		if !ok || mt != mediaTypeImage {
			t.Fatalf("got %q, %v; want %q, true", mt, ok, mediaTypeImage)
		}
	})
	t.Run("video/mp4 returns video", func(t *testing.T) {
		mt, ok := detectMediaType("video/mp4")
		if !ok || mt != mediaTypeVideo {
			t.Fatalf("got %q, %v; want %q, true", mt, ok, mediaTypeVideo)
		}
	})
	t.Run("application/pdf returns false", func(t *testing.T) {
		_, ok := detectMediaType("application/pdf")
		if ok {
			t.Fatal("application/pdf should not be accepted")
		}
	})
}

// ── detectMediaByExtension extra branches ─────────────────────────────────────

func TestDetectMediaByExtension(t *testing.T) {
	t.Run("known .jpg returns image", func(t *testing.T) {
		mt, ct, ok := detectMediaByExtension(".jpg")
		if !ok || mt != mediaTypeImage || ct != mimeJPEG {
			t.Fatalf("got %q %q %v, want image %q true", mt, ct, ok, mimeJPEG)
		}
	})
	t.Run("case-insensitive .JPG returns image", func(t *testing.T) {
		mt, ct, ok := detectMediaByExtension(".JPG")
		if !ok || mt != mediaTypeImage || ct != mimeJPEG {
			t.Fatalf("got %q %q %v, want image %q true", mt, ct, ok, mimeJPEG)
		}
	})
	t.Run("unknown .xyz returns false", func(t *testing.T) {
		_, _, ok := detectMediaByExtension(".xyz")
		if ok {
			t.Fatal("unknown extension should return false")
		}
	})
}

// ── buildMediaUploadPart extra branches ───────────────────────────────────────

// TestBuildMediaUploadPartNoFilename covers the errMediaFileRequired path when
// the multipart part has no filename.
func TestBuildMediaUploadPartNoFilename(t *testing.T) {
	part := mediaPartWithContentType(t, "", "image/jpeg")
	_, err := buildMediaUploadPart(part)
	if !errors.Is(err, errMediaFileRequired) {
		t.Fatalf("err = %v, want errMediaFileRequired", err)
	}
}

// TestBuildMediaUploadPartNoExtension covers the ext=="" fallback: when a
// file has no extension but a valid image MIME type, objectKey must end with
// the canonical extension (.jpg).
func TestBuildMediaUploadPartNoExtension(t *testing.T) {
	// Filename "photo" has no extension; detectMediaType("image/jpeg") succeeds,
	// so we skip detectMediaByExtension. Then ext == "" → fallback to ".jpg".
	part := mediaPartWithContentType(t, "photo", "image/jpeg")
	upload, err := buildMediaUploadPart(part)
	if err != nil {
		t.Fatalf("buildMediaUploadPart: %v", err)
	}
	if upload.mediaType != mediaTypeImage {
		t.Errorf("mediaType = %q, want %q", upload.mediaType, mediaTypeImage)
	}
	if !strings.HasSuffix(upload.objectKey, extJPG) {
		t.Errorf("objectKey = %q, want suffix %q", upload.objectKey, extJPG)
	}
}

// ── uniqueAuthorIDs ───────────────────────────────────────────────────────────

func TestUniqueAuthorIDs(t *testing.T) {
	t.Run("deduplicates repeated author ids", func(t *testing.T) {
		posts := []postModel{
			{AuthorID: "alice"},
			{AuthorID: "bob"},
			{AuthorID: "alice"},
		}
		got := uniqueAuthorIDs(posts)
		if len(got) != 2 || got[0] != "alice" || got[1] != "bob" {
			t.Errorf("uniqueAuthorIDs = %v, want [alice bob]", got)
		}
	})
	t.Run("empty slice returns empty", func(t *testing.T) {
		if got := uniqueAuthorIDs(nil); len(got) != 0 {
			t.Errorf("uniqueAuthorIDs(nil) = %v, want []", got)
		}
	})
}

// ── fetchAuthorAvatars ────────────────────────────────────────────────────────

func TestFetchAuthorAvatars(t *testing.T) {
	t.Run("empty authorIDs with set userURL returns nil", func(t *testing.T) {
		a := &app{userURL: "http://example.com", internalKey: "k"}
		if result := a.fetchAuthorAvatars(nil); result != nil {
			t.Fatalf("got %v, want nil (empty IDs)", result)
		}
	})

	t.Run("network error returns nil", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := srv.URL
		srv.Close() // port closed; connection-refused on first request
		a := &app{userURL: url, internalKey: "k"}
		if result := a.fetchAuthorAvatars([]string{"alice"}); result != nil {
			t.Fatalf("got %v, want nil (network error)", result)
		}
	})

	t.Run("non-200 response returns nil", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()
		a := &app{userURL: srv.URL, internalKey: "k"}
		if result := a.fetchAuthorAvatars([]string{"alice"}); result != nil {
			t.Fatalf("got %v, want nil (non-200)", result)
		}
	})

	t.Run("bad JSON returns nil", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not-json"))
		}))
		defer srv.Close()
		a := &app{userURL: srv.URL, internalKey: "k"}
		if result := a.fetchAuthorAvatars([]string{"alice"}); result != nil {
			t.Fatalf("got %v, want nil (bad JSON)", result)
		}
	})

	t.Run("success with null avatar URL", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "alice", "avatarUrl": nil},
				},
			})
		}))
		defer srv.Close()
		a := &app{userURL: srv.URL, internalKey: "k"}
		result := a.fetchAuthorAvatars([]string{"alice"})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if v, ok := result["alice"]; !ok || v != nil {
			t.Errorf("alice avatar = %v, want nil", v)
		}
	})

	t.Run("success with avatar URL", func(t *testing.T) {
		avatarURL := "https://cdn.example.com/alice.jpg"
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "alice", "avatarUrl": avatarURL},
				},
			})
		}))
		defer srv.Close()
		a := &app{userURL: srv.URL, internalKey: "k"}
		result := a.fetchAuthorAvatars([]string{"alice"})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		got := result["alice"]
		if got == nil || *got != avatarURL {
			t.Errorf("alice avatar = %v, want %q", got, avatarURL)
		}
	})
}

// ── abortIfPrivate ────────────────────────────────────────────────────────────

// TestAbortIfPrivateDenied covers the 403 branch when canViewProfile returns
// false. The allowed branch is always exercised in integration tests (userURL="").
func TestAbortIfPrivateDenied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"canView":false}`))
	}))
	defer srv.Close()

	a := &app{userURL: srv.URL, internalKey: "k"}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	if !a.abortIfPrivate(c, "private-user") {
		t.Fatal("abortIfPrivate should return true (access denied)")
	}
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}
