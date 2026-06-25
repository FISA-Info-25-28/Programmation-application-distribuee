package main

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestMaxFileReader(t *testing.T) {
	t.Run("reads data under the limit", func(t *testing.T) {
		r := &maxFileReader{reader: bytes.NewReader([]byte("hello")), max: 100}
		got, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		if string(got) != "hello" {
			t.Fatalf("got %q, want hello", got)
		}
	})

	t.Run("data exactly at the limit succeeds", func(t *testing.T) {
		r := &maxFileReader{reader: bytes.NewReader([]byte("abcde")), max: 5}
		got, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		if string(got) != "abcde" {
			t.Fatalf("got %q, want abcde", got)
		}
	})

	t.Run("data over the limit returns errMediaTooLarge", func(t *testing.T) {
		r := &maxFileReader{reader: bytes.NewReader([]byte("abcdef")), max: 5}
		_, err := io.ReadAll(r)
		if !errors.Is(err, errMediaTooLarge) {
			t.Fatalf("err = %v, want errMediaTooLarge", err)
		}
	})
}

func mediaPartWithContentType(t *testing.T, filename, contentType string) *multipart.Part {
	t.Helper()
	var body strings.Builder
	mw := multipart.NewWriter(&body)
	header := make(map[string][]string)
	header["Content-Disposition"] = []string{`form-data; name="file"; filename="` + filename + `"`}
	if contentType != "" {
		header["Content-Type"] = []string{contentType}
	}
	part, err := mw.CreatePart(header)
	if err != nil {
		t.Fatalf("create part: %v", err)
	}
	if _, err := part.Write([]byte("data")); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	reader := multipart.NewReader(strings.NewReader(body.String()), mw.Boundary())
	read, err := reader.NextPart()
	if err != nil {
		t.Fatalf("next part: %v", err)
	}
	return read
}

func TestBuildMediaUploadPartFallsBackToExtension(t *testing.T) {
	cases := []struct {
		name        string
		filename    string
		contentType string
		wantType    string
		wantCT      string
	}{
		{"mov with generic content type", "clip.MOV", mimeOctetStream, mediaTypeVideo, mimeQuicktime},
		{"mp3 with empty content type", "song.mp3", "", mediaTypeAudio, mimeMPEG},
		{"mp4 keeps browser content type", "clip.mp4", mimeMP4, mediaTypeVideo, mimeMP4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			part := mediaPartWithContentType(t, tc.filename, tc.contentType)
			upload, err := buildMediaUploadPart(part)
			if err != nil {
				t.Fatalf("buildMediaUploadPart returned error: %v", err)
			}
			if upload.mediaType != tc.wantType {
				t.Fatalf("mediaType = %q, want %q", upload.mediaType, tc.wantType)
			}
			if upload.contentType != tc.wantCT {
				t.Fatalf("contentType = %q, want %q", upload.contentType, tc.wantCT)
			}
		})
	}
}

func TestBuildMediaUploadPartRejectsUnknownExtension(t *testing.T) {
	part := mediaPartWithContentType(t, "archive.zip", mimeOctetStream)
	if _, err := buildMediaUploadPart(part); err != errUnsupportedMedia {
		t.Fatalf("error = %v, want errUnsupportedMedia", err)
	}
}

func TestDetectMediaTypeAcceptsAudio(t *testing.T) {
	mediaType, ok := detectMediaType("audio/webm")
	if !ok {
		t.Fatal("audio/webm should be accepted")
	}
	if mediaType != mediaTypeAudio {
		t.Fatalf("media type = %q, want %q", mediaType, mediaTypeAudio)
	}
}

func TestMediaModelConstraintIncludesAudio(t *testing.T) {
	field, ok := reflect.TypeOf(mediaModel{}).FieldByName("MediaType")
	if !ok {
		t.Fatal("MediaType field not found")
	}
	tag := string(field.Tag)
	if !strings.Contains(tag, "'audio'") {
		t.Fatalf("MediaType gorm tag should allow audio: %s", tag)
	}
}

func TestNormalizeMediaIDsDropsInvalidAndDuplicates(t *testing.T) {
	got := normalizeMediaIDs([]int64{4, 0, 4, -1, 9})
	want := []int64{4, 9}
	if !slices.Equal(got, want) {
		t.Fatalf("normalizeMediaIDs() = %v, want %v", got, want)
	}
}
