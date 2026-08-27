package main

import "testing"

func TestObjectPathEscapesSegments(t *testing.T) {
	got := objectPath(objectPresignPathTemplate, "saas snapshots", "snapshots/2026-07-31/saas.json")
	want := "/v1/storage/object/presign/saas%20snapshots/snapshots%2F2026-07-31%2Fsaas.json"
	if got != want {
		t.Fatalf("objectPath() = %q, want %q", got, want)
	}
}
