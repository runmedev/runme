package contents

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	contentsv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/contents/v1"
)

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func newHandler(t *testing.T) *Handler {
	t.Helper()
	h, err := NewHandler(t.TempDir())
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	return h
}

func writeFile(t *testing.T, root, relPath string, data []byte) {
	t.Helper()
	abs := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolvePath_TraversalRejected(t *testing.T) {
	h := newHandler(t)
	ctx := context.Background()

	paths := []string{"../etc/passwd", "foo/../../etc/passwd"}
	for _, p := range paths {
		_, err := h.Stat(ctx, connect.NewRequest(&contentsv1.StatRequest{Path: p}))
		if err == nil {
			t.Errorf("expected error for path %q, got nil", p)
			continue
		}
		if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
			t.Errorf("path %q: got code %v, want InvalidArgument", p, code)
		}
	}
}

func TestResolvePath_SymlinkEscape(t *testing.T) {
	h := newHandler(t)
	ctx := context.Background()

	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	linkPath := filepath.Join(h.rootDir, "escape")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Fatal(err)
	}

	_, err := h.Read(ctx, connect.NewRequest(&contentsv1.ReadRequest{Path: "escape/secret.txt"}))
	if err == nil {
		t.Fatal("expected error for symlink escape, got nil")
	}
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Errorf("got code %v, want InvalidArgument", code)
	}
}

func TestList_ShallowChildren(t *testing.T) {
	h := newHandler(t)
	ctx := context.Background()

	writeFile(t, h.rootDir, "a.txt", []byte("a"))
	writeFile(t, h.rootDir, "b.txt", []byte("b"))
	writeFile(t, h.rootDir, ".hidden", []byte("h"))
	writeFile(t, h.rootDir, "sub/nested.txt", []byte("n"))

	resp, err := h.List(ctx, connect.NewRequest(&contentsv1.ListRequest{Path: ""}))
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	names := make(map[string]bool)
	for _, item := range resp.Msg.Items {
		names[item.Name] = true
	}

	if !names["a.txt"] || !names["b.txt"] || !names["sub"] {
		t.Errorf("missing expected entries; got %v", names)
	}
	if names[".hidden"] {
		t.Error("dotfile should be skipped")
	}
	if names["nested.txt"] {
		t.Error("should not recurse into subdirectories")
	}
}

func TestReadWriteRoundtrip(t *testing.T) {
	h := newHandler(t)
	ctx := context.Background()

	content := []byte("hello, world")

	wResp, err := h.Write(ctx, connect.NewRequest(&contentsv1.WriteRequest{
		Path:    "test.txt",
		Content: content,
	}))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if wResp.Msg.Info.Name != "test.txt" {
		t.Errorf("got name %q, want test.txt", wResp.Msg.Info.Name)
	}

	rResp, err := h.Read(ctx, connect.NewRequest(&contentsv1.ReadRequest{
		Path:        "test.txt",
		IncludeHash: true,
	}))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !bytes.Equal(rResp.Msg.Content, content) {
		t.Errorf("content mismatch: got %q, want %q", rResp.Msg.Content, content)
	}

	wantHash := sha256Hex(content)
	if rResp.Msg.Info.Sha256Hex != wantHash {
		t.Errorf("hash mismatch: got %q, want %q", rResp.Msg.Info.Sha256Hex, wantHash)
	}
	if rResp.Msg.Info.SizeBytes != int64(len(content)) {
		t.Errorf("size mismatch: got %d, want %d", rResp.Msg.Info.SizeBytes, len(content))
	}
}

func TestWrite_ConditionalSuccess(t *testing.T) {
	h := newHandler(t)
	ctx := context.Background()

	original := []byte("v1")
	writeFile(t, h.rootDir, "cond.txt", original)

	expectedVersion := sha256Hex(original)
	_, err := h.Write(ctx, connect.NewRequest(&contentsv1.WriteRequest{
		Path:            "cond.txt",
		Content:         []byte("v2"),
		ExpectedVersion: &expectedVersion,
	}))
	if err != nil {
		t.Fatalf("conditional write should succeed: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(h.rootDir, "cond.txt"))
	if string(data) != "v2" {
		t.Errorf("file content should be v2, got %q", data)
	}
}

func TestWrite_ConditionalConflict(t *testing.T) {
	h := newHandler(t)
	ctx := context.Background()

	writeFile(t, h.rootDir, "cond.txt", []byte("v1"))

	wrongVersion := sha256Hex([]byte("wrong"))
	_, err := h.Write(ctx, connect.NewRequest(&contentsv1.WriteRequest{
		Path:            "cond.txt",
		Content:         []byte("v2"),
		ExpectedVersion: &wrongVersion,
	}))
	if err == nil {
		t.Fatal("expected error for version mismatch, got nil")
	}
	if code := connect.CodeOf(err); code != connect.CodeAborted {
		t.Errorf("got code %v, want Aborted", code)
	}
}

func TestWrite_FailIfExists(t *testing.T) {
	h := newHandler(t)
	ctx := context.Background()

	writeFile(t, h.rootDir, "exists.txt", []byte("data"))

	_, err := h.Write(ctx, connect.NewRequest(&contentsv1.WriteRequest{
		Path:    "exists.txt",
		Content: []byte("new"),
		Mode:    contentsv1.WriteMode_WRITE_MODE_FAIL_IF_EXISTS,
	}))
	if err == nil {
		t.Fatal("expected error for FAIL_IF_EXISTS, got nil")
	}
	if code := connect.CodeOf(err); code != connect.CodeAlreadyExists {
		t.Errorf("got code %v, want AlreadyExists", code)
	}
}

func TestWrite_DirtyCheck(t *testing.T) {
	h := newHandler(t)
	ctx := context.Background()

	content := []byte("same content")
	writeFile(t, h.rootDir, "dirty.txt", content)

	absPath := filepath.Join(h.rootDir, "dirty.txt")
	infoBefore, _ := os.Stat(absPath)
	mtimeBefore := infoBefore.ModTime()

	resp, err := h.Write(ctx, connect.NewRequest(&contentsv1.WriteRequest{
		Path:    "dirty.txt",
		Content: content,
	}))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	infoAfter, _ := os.Stat(absPath)
	mtimeAfter := infoAfter.ModTime()

	if !mtimeAfter.Equal(mtimeBefore) {
		t.Error("mtime should not change when content is identical (dirty check)")
	}

	if resp.Msg.Info.Sha256Hex != sha256Hex(content) {
		t.Errorf("hash mismatch in dirty-check response")
	}
}

func TestStat_WithHash(t *testing.T) {
	h := newHandler(t)
	ctx := context.Background()

	content := []byte("hash me")
	writeFile(t, h.rootDir, "hashme.txt", content)

	resp, err := h.Stat(ctx, connect.NewRequest(&contentsv1.StatRequest{
		Path:        "hashme.txt",
		IncludeHash: true,
	}))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	wantHash := sha256Hex(content)
	if resp.Msg.Info.Sha256Hex != wantHash {
		t.Errorf("got hash %q, want %q", resp.Msg.Info.Sha256Hex, wantHash)
	}
	if resp.Msg.Info.Type != contentsv1.FileType_FILE_TYPE_FILE {
		t.Errorf("got type %v, want FILE", resp.Msg.Info.Type)
	}
}

func TestRename_WithVersionGuard(t *testing.T) {
	h := newHandler(t)
	ctx := context.Background()

	writeFile(t, h.rootDir, "old.txt", []byte("data"))

	wrongVersion := sha256Hex([]byte("wrong"))
	_, err := h.Rename(ctx, connect.NewRequest(&contentsv1.RenameRequest{
		OldPath:         "old.txt",
		NewPath:         "new.txt",
		ExpectedVersion: &wrongVersion,
	}))
	if err == nil {
		t.Fatal("expected error for version guard, got nil")
	}
	if code := connect.CodeOf(err); code != connect.CodeAborted {
		t.Errorf("got code %v, want Aborted", code)
	}
}

func TestRename_Success(t *testing.T) {
	h := newHandler(t)
	ctx := context.Background()

	content := []byte("moveme")
	writeFile(t, h.rootDir, "src.txt", content)

	resp, err := h.Rename(ctx, connect.NewRequest(&contentsv1.RenameRequest{
		OldPath: "src.txt",
		NewPath: "dst.txt",
	}))
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if resp.Msg.Info.Name != "dst.txt" {
		t.Errorf("got name %q, want dst.txt", resp.Msg.Info.Name)
	}

	if _, err := os.Stat(filepath.Join(h.rootDir, "src.txt")); !os.IsNotExist(err) {
		t.Error("old path should no longer exist")
	}
	data, err := os.ReadFile(filepath.Join(h.rootDir, "dst.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, content) {
		t.Errorf("content mismatch after rename")
	}
}

func TestWrite_SymlinkParentEscape(t *testing.T) {
	h := newHandler(t)
	ctx := context.Background()

	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(h.rootDir, "escape")); err != nil {
		t.Fatal(err)
	}

	_, err := h.Write(ctx, connect.NewRequest(&contentsv1.WriteRequest{
		Path:    "escape/new.txt",
		Content: []byte("pwned"),
	}))
	if err == nil {
		t.Fatal("expected error for write via symlinked parent, got nil")
	}
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Errorf("got code %v, want InvalidArgument", code)
	}

	if _, statErr := os.Stat(filepath.Join(outside, "new.txt")); !os.IsNotExist(statErr) {
		t.Error("file should NOT have been created outside workspace")
	}
}

func TestMkdir_SymlinkParentEscape(t *testing.T) {
	h := newHandler(t)
	ctx := context.Background()

	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(h.rootDir, "escape")); err != nil {
		t.Fatal(err)
	}

	_, err := h.Mkdir(ctx, connect.NewRequest(&contentsv1.MkdirRequest{
		Path: "escape/subdir",
	}))
	if err == nil {
		t.Fatal("expected error for mkdir via symlinked parent, got nil")
	}
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Errorf("got code %v, want InvalidArgument", code)
	}
}

func TestRename_DestSymlinkParentEscape(t *testing.T) {
	h := newHandler(t)
	ctx := context.Background()

	writeFile(t, h.rootDir, "legit.txt", []byte("data"))

	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(h.rootDir, "escape")); err != nil {
		t.Fatal(err)
	}

	_, err := h.Rename(ctx, connect.NewRequest(&contentsv1.RenameRequest{
		OldPath: "legit.txt",
		NewPath: "escape/stolen.txt",
	}))
	if err == nil {
		t.Fatal("expected error for rename to symlinked parent, got nil")
	}
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Errorf("got code %v, want InvalidArgument", code)
	}
}

func TestList_SkipsSymlinks(t *testing.T) {
	h := newHandler(t)
	ctx := context.Background()

	writeFile(t, h.rootDir, "real.txt", []byte("real"))

	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(h.rootDir, "link_to_outside")); err != nil {
		t.Fatal(err)
	}

	resp, err := h.List(ctx, connect.NewRequest(&contentsv1.ListRequest{Path: "", IncludeHashes: true}))
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	for _, item := range resp.Msg.Items {
		if item.Name == "link_to_outside" {
			t.Error("symlink should be skipped in listing")
		}
	}
}

func TestWrite_ExpectedVersionOnMissingFile(t *testing.T) {
	h := newHandler(t)
	ctx := context.Background()

	version := sha256Hex([]byte("anything"))
	_, err := h.Write(ctx, connect.NewRequest(&contentsv1.WriteRequest{
		Path:            "nonexistent.txt",
		Content:         []byte("new content"),
		ExpectedVersion: &version,
	}))
	if err == nil {
		t.Fatal("expected error for expected_version on missing file, got nil")
	}
	if code := connect.CodeOf(err); code != connect.CodeAborted {
		t.Errorf("got code %v, want Aborted", code)
	}
}

func TestMkdir_Idempotent(t *testing.T) {
	h := newHandler(t)
	ctx := context.Background()

	req := connect.NewRequest(&contentsv1.MkdirRequest{Path: "a/b/c"})

	resp1, err := h.Mkdir(ctx, req)
	if err != nil {
		t.Fatalf("first Mkdir: %v", err)
	}
	if resp1.Msg.Info.Type != contentsv1.FileType_FILE_TYPE_DIRECTORY {
		t.Errorf("got type %v, want DIRECTORY", resp1.Msg.Info.Type)
	}

	_, err = h.Mkdir(ctx, connect.NewRequest(&contentsv1.MkdirRequest{Path: "a/b/c"}))
	if err != nil {
		t.Fatalf("second Mkdir should be idempotent: %v", err)
	}
}
