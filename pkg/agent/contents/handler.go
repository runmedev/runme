// Package contents implements the ContentsService, which provides sandboxed
// local filesystem access for the notebook web application.
package contents

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	contentsv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/contents/v1"
	"github.com/runmedev/runme/v3/api/gen/proto/go/runme/contents/v1/contentsv1connect"
)

// Handler implements contentsv1connect.ContentsServiceHandler.
// Every filesystem operation is sandboxed under rootDir. Symlinks are
// disallowed anywhere in resolved paths to prevent sandbox escapes.
type Handler struct {
	contentsv1connect.UnimplementedContentsServiceHandler

	// rootDir is the absolute, symlink-resolved workspace root directory.
	rootDir string
}

// NewHandler creates a new ContentsService handler rooted at the given
// directory. If rootDir is empty the current working directory is used.
func NewHandler(rootDir string) (*Handler, error) {
	if rootDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("contents: failed to determine cwd: %w", err)
		}
		rootDir = cwd
	}

	resolved, err := filepath.EvalSymlinks(rootDir)
	if err != nil {
		return nil, fmt.Errorf("contents: failed to resolve root dir %q: %w", rootDir, err)
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return nil, fmt.Errorf("contents: failed to make root dir absolute: %w", err)
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return nil, fmt.Errorf("contents: root dir %q does not exist: %w", resolved, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("contents: root dir %q is not a directory", resolved)
	}

	return &Handler{rootDir: resolved}, nil
}

// resolvePath validates and resolves a workspace-relative path to an absolute
// path under rootDir. It rejects directory traversal and symlinks anywhere in
// the path to prevent sandbox escapes — including for paths that don't fully
// exist yet (Write, Mkdir, Rename destinations).
func (h *Handler) resolvePath(reqPath string) (string, error) {
	if reqPath == "" || reqPath == "." || reqPath == "/" {
		return h.rootDir, nil
	}

	// Reject backslash separators (Windows hardening for web-originated paths).
	if strings.ContainsRune(reqPath, '\\') {
		return "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("backslash not allowed in path: %q", reqPath))
	}

	cleaned := filepath.Clean(reqPath)
	if strings.HasPrefix(cleaned, "..") || strings.Contains(cleaned, string(filepath.Separator)+"..") {
		return "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("path traversal not allowed: %q", reqPath))
	}

	cleaned = strings.TrimPrefix(cleaned, string(filepath.Separator))
	abs := filepath.Join(h.rootDir, cleaned)

	// Verify the joined path is still under rootDir.
	if !strings.HasPrefix(abs, h.rootDir+string(filepath.Separator)) && abs != h.rootDir {
		return "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("path escapes workspace root: %q", reqPath))
	}

	// Walk each component from rootDir to abs. Reject symlinks in any existing
	// component. Components that don't exist yet are allowed (for creation ops).
	cur := h.rootDir
	components := strings.Split(cleaned, string(filepath.Separator))
	for _, comp := range components {
		if comp == "" {
			continue
		}
		next := filepath.Join(cur, comp)
		fi, err := os.Lstat(next)
		if err != nil {
			if os.IsNotExist(err) {
				cur = next
				continue
			}
			return "", connect.NewError(connect.CodeInternal, fmt.Errorf("lstat %q: %w", next, err))
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("symlinks not allowed in path: %q", reqPath))
		}
		cur = next
	}

	return abs, nil
}

// List returns the direct children of a directory. Symlinks and dotfiles are
// skipped to avoid sandbox escapes and noise.
func (h *Handler) List(ctx context.Context, req *connect.Request[contentsv1.ListRequest]) (*connect.Response[contentsv1.ListResponse], error) {
	log := zapr.NewLogger(zap.L())

	dirPath, err := h.resolvePath(req.Msg.Path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fsError("failed to list directory", err)
	}

	items := make([]*contentsv1.FileInfo, 0, len(entries))
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		entryPath := filepath.Join(dirPath, entry.Name())
		relPath, _ := filepath.Rel(h.rootDir, entryPath)

		info, err := os.Lstat(entryPath)
		if err != nil {
			log.Error(err, "skipping entry due to stat failure", "path", relPath)
			continue
		}

		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}

		fi := &contentsv1.FileInfo{
			Path:               relPath,
			Name:               entry.Name(),
			Type:               fileType(entry.IsDir()),
			SizeBytes:          info.Size(),
			LastModifiedUnixMs: info.ModTime().UnixMilli(),
		}

		if req.Msg.IncludeHashes && !entry.IsDir() {
			hash, hashErr := hashFile(entryPath)
			if hashErr != nil {
				log.Error(hashErr, "failed to hash file", "path", relPath)
			} else {
				fi.Sha256Hex = hash
			}
		}

		items = append(items, fi)
	}

	return connect.NewResponse(&contentsv1.ListResponse{Items: items}), nil
}

// Read returns the contents and metadata of a file.
func (h *Handler) Read(ctx context.Context, req *connect.Request[contentsv1.ReadRequest]) (*connect.Response[contentsv1.ReadResponse], error) {
	filePath, err := h.resolvePath(req.Msg.Path)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fsError("failed to read file", err)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fsError("failed to stat file", err)
	}

	relPath, _ := filepath.Rel(h.rootDir, filePath)
	fi := &contentsv1.FileInfo{
		Path:               relPath,
		Name:               info.Name(),
		Type:               fileType(info.IsDir()),
		SizeBytes:          info.Size(),
		LastModifiedUnixMs: info.ModTime().UnixMilli(),
	}

	if req.Msg.IncludeHash {
		fi.Sha256Hex = hashBytes(data)
	}

	return connect.NewResponse(&contentsv1.ReadResponse{
		Content: data,
		Info:    fi,
	}), nil
}

// Stat returns metadata about a file or directory.
func (h *Handler) Stat(ctx context.Context, req *connect.Request[contentsv1.StatRequest]) (*connect.Response[contentsv1.StatResponse], error) {
	targetPath, err := h.resolvePath(req.Msg.Path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		return nil, fsError("failed to stat", err)
	}

	relPath, _ := filepath.Rel(h.rootDir, targetPath)
	fi := &contentsv1.FileInfo{
		Path:               relPath,
		Name:               info.Name(),
		Type:               fileType(info.IsDir()),
		SizeBytes:          info.Size(),
		LastModifiedUnixMs: info.ModTime().UnixMilli(),
	}

	if req.Msg.IncludeHash && !info.IsDir() {
		hash, hashErr := hashFile(targetPath)
		if hashErr != nil {
			return nil, fsError("failed to hash file", hashErr)
		}
		fi.Sha256Hex = hash
	}

	return connect.NewResponse(&contentsv1.StatResponse{Info: fi}), nil
}

// Write writes content to a file using atomic write semantics. It supports
// conditional writes via expected_version (SHA256 conflict detection) and
// skips the write entirely if the content has not changed (dirty check).
func (h *Handler) Write(ctx context.Context, req *connect.Request[contentsv1.WriteRequest]) (*connect.Response[contentsv1.WriteResponse], error) {
	filePath, err := h.resolvePath(req.Msg.Path)
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fsError("failed to create parent directory", err)
	}

	newContentHash := hashBytes(req.Msg.Content)

	existingInfo, statErr := os.Stat(filePath)
	fileExists := statErr == nil

	if fileExists {
		if req.Msg.Mode == contentsv1.WriteMode_WRITE_MODE_FAIL_IF_EXISTS {
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("file already exists: %s", req.Msg.Path))
		}

		// Compute hash once for both conditional write and dirty check.
		currentHash, hashErr := hashFile(filePath)
		if hashErr != nil {
			return nil, fsError("failed to hash existing file", hashErr)
		}

		// Conditional write: verify expected_version matches current hash.
		if req.Msg.ExpectedVersion != nil && currentHash != *req.Msg.ExpectedVersion {
			return nil, connect.NewError(connect.CodeAborted,
				fmt.Errorf("version mismatch: expected %s, current %s", *req.Msg.ExpectedVersion, currentHash))
		}

		// Dirty check: skip write if content hasn't changed.
		if currentHash == newContentHash {
			relPath, _ := filepath.Rel(h.rootDir, filePath)
			fi := &contentsv1.FileInfo{
				Path:               relPath,
				Name:               existingInfo.Name(),
				Type:               contentsv1.FileType_FILE_TYPE_FILE,
				SizeBytes:          existingInfo.Size(),
				LastModifiedUnixMs: existingInfo.ModTime().UnixMilli(),
				Sha256Hex:          currentHash,
			}
			return connect.NewResponse(&contentsv1.WriteResponse{Info: fi}), nil
		}
	} else {
		// File does not exist. If expected_version is set, that's a mismatch
		// because there's nothing to compare against.
		if req.Msg.ExpectedVersion != nil {
			return nil, connect.NewError(connect.CodeAborted,
				fmt.Errorf("version mismatch: expected %s, but file does not exist", *req.Msg.ExpectedVersion))
		}
	}

	if err := atomicWrite(filePath, req.Msg.Content); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to write file: %w", err))
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fsError("failed to stat written file", err)
	}

	relPath, _ := filepath.Rel(h.rootDir, filePath)
	fi := &contentsv1.FileInfo{
		Path:               relPath,
		Name:               info.Name(),
		Type:               contentsv1.FileType_FILE_TYPE_FILE,
		SizeBytes:          info.Size(),
		LastModifiedUnixMs: info.ModTime().UnixMilli(),
		Sha256Hex:          newContentHash,
	}

	return connect.NewResponse(&contentsv1.WriteResponse{Info: fi}), nil
}

// Rename moves a file or directory. Supports conditional renames via
// expected_version.
func (h *Handler) Rename(ctx context.Context, req *connect.Request[contentsv1.RenameRequest]) (*connect.Response[contentsv1.RenameResponse], error) {
	oldPath, err := h.resolvePath(req.Msg.OldPath)
	if err != nil {
		return nil, err
	}
	newPath, err := h.resolvePath(req.Msg.NewPath)
	if err != nil {
		return nil, err
	}

	if req.Msg.ExpectedVersion != nil {
		info, statErr := os.Stat(oldPath)
		if statErr != nil {
			return nil, fsError("source not found", statErr)
		}
		if !info.IsDir() {
			currentHash, hashErr := hashFile(oldPath)
			if hashErr != nil {
				return nil, fsError("failed to hash source file", hashErr)
			}
			if currentHash != *req.Msg.ExpectedVersion {
				return nil, connect.NewError(connect.CodeAborted,
					fmt.Errorf("version mismatch: expected %s, current %s", *req.Msg.ExpectedVersion, currentHash))
			}
		}
	}

	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return nil, fsError("failed to create parent directory", err)
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		return nil, fsError("failed to rename", err)
	}

	info, err := os.Stat(newPath)
	if err != nil {
		return nil, fsError("failed to stat renamed entry", err)
	}

	relPath, _ := filepath.Rel(h.rootDir, newPath)
	fi := &contentsv1.FileInfo{
		Path:               relPath,
		Name:               info.Name(),
		Type:               fileType(info.IsDir()),
		SizeBytes:          info.Size(),
		LastModifiedUnixMs: info.ModTime().UnixMilli(),
	}

	return connect.NewResponse(&contentsv1.RenameResponse{Info: fi}), nil
}

// Mkdir creates a directory and any necessary parents. It is idempotent.
func (h *Handler) Mkdir(ctx context.Context, req *connect.Request[contentsv1.MkdirRequest]) (*connect.Response[contentsv1.MkdirResponse], error) {
	dirPath, err := h.resolvePath(req.Msg.Path)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return nil, fsError("failed to create directory", err)
	}

	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, fsError("failed to stat directory", err)
	}

	relPath, _ := filepath.Rel(h.rootDir, dirPath)
	fi := &contentsv1.FileInfo{
		Path:               relPath,
		Name:               info.Name(),
		Type:               contentsv1.FileType_FILE_TYPE_DIRECTORY,
		SizeBytes:          info.Size(),
		LastModifiedUnixMs: info.ModTime().UnixMilli(),
	}

	return connect.NewResponse(&contentsv1.MkdirResponse{Info: fi}), nil
}

// atomicWrite writes data to a temporary file in the same directory as dest,
// flushes it to disk, then atomically renames it to dest. The temp file is
// created with O_EXCL to avoid overwriting existing files.
func atomicWrite(dest string, data []byte) error {
	dir := filepath.Dir(dest)

	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		return fmt.Errorf("failed to generate random bytes: %w", err)
	}
	tmpName := filepath.Join(dir, fmt.Sprintf(".tmp.%s", hex.EncodeToString(randomBytes)))

	f, err := os.OpenFile(tmpName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	success := false
	defer func() {
		if !success {
			_ = f.Close()
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpName, dest); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	success = true
	return nil
}

// hashFile computes the hex-encoded SHA-256 hash of a file's contents.
func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return hashBytes(data), nil
}

// hashBytes computes the hex-encoded SHA-256 hash of a byte slice.
func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// fileType converts a boolean isDir into the protobuf FileType enum.
func fileType(isDir bool) contentsv1.FileType {
	if isDir {
		return contentsv1.FileType_FILE_TYPE_DIRECTORY
	}
	return contentsv1.FileType_FILE_TYPE_FILE
}

// fsError maps common os errors to appropriate ConnectRPC status codes.
func fsError(msg string, err error) *connect.Error {
	code := connect.CodeInternal
	if os.IsNotExist(err) {
		code = connect.CodeNotFound
	} else if os.IsPermission(err) {
		code = connect.CodePermissionDenied
	}
	return connect.NewError(code, fmt.Errorf("%s: %w", msg, err))
}

// Compile-time check that Handler implements the interface.
var _ contentsv1connect.ContentsServiceHandler = (*Handler)(nil)

// RootDir returns the resolved root directory for this handler.
func (h *Handler) RootDir() string {
	return h.rootDir
}

// fileInfoFromStat builds a FileInfo from an os.FileInfo and absolute path.
func (h *Handler) fileInfoFromStat(absPath string, info fs.FileInfo) *contentsv1.FileInfo {
	relPath, _ := filepath.Rel(h.rootDir, absPath)
	return &contentsv1.FileInfo{
		Path:               relPath,
		Name:               info.Name(),
		Type:               fileType(info.IsDir()),
		SizeBytes:          info.Size(),
		LastModifiedUnixMs: info.ModTime().UnixMilli(),
	}
}
