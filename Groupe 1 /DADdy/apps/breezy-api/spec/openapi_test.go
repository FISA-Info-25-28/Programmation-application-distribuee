package spec_test

import (
	"os"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestPostAndCommentIDsUsePrefixedSchemas(t *testing.T) {
	content, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatalf("read OpenAPI spec: %v", err)
	}

	var document map[string]any
	if err := yaml.Unmarshal(content, &document); err != nil {
		t.Fatalf("parse OpenAPI spec: %v", err)
	}

	assertValue(t, document, "#/components/schemas/PostId", "components", "parameters", "PostId", "schema", "$ref")
	assertValue(t, document, "^pst_[a-f0-9]{16}$", "components", "schemas", "PostId", "pattern")
	assertValue(t, document, "^cmt_[a-f0-9]{16}$", "components", "schemas", "CommentId", "pattern")
	assertValue(t, document, "#/components/schemas/PostId", "components", "schemas", "Post", "properties", "id", "$ref")
	assertValue(t, document, "#/components/schemas/CommentId", "components", "schemas", "Comment", "properties", "id", "$ref")
	assertValue(t, document, "#/components/schemas/PostId", "components", "schemas", "Comment", "properties", "postId", "$ref")
	assertValue(t, document, "#/components/schemas/CommentId", "components", "schemas", "Comment", "properties", "parentCommentId", "allOf", 0, "$ref")
	assertValue(t, document, "#/components/schemas/CommentId", "components", "schemas", "CreateCommentRequest", "properties", "parentCommentId", "allOf", 0, "$ref")
}

func assertValue(t *testing.T, document any, expected any, path ...any) {
	t.Helper()

	current := document
	for _, segment := range path {
		switch key := segment.(type) {
		case string:
			object, ok := current.(map[string]any)
			if !ok {
				t.Fatalf("%v: expected object before key %q, got %T", path, key, current)
			}
			current, ok = object[key]
			if !ok {
				t.Fatalf("%v: missing key %q", path, key)
			}
		case int:
			items, ok := current.([]any)
			if !ok || key < 0 || key >= len(items) {
				t.Fatalf("%v: invalid index %d in %T", path, key, current)
			}
			current = items[key]
		default:
			t.Fatalf("%v: unsupported path segment %T", path, segment)
		}
	}

	if current != expected {
		t.Fatalf("%v: got %v, want %v", path, current, expected)
	}
}
