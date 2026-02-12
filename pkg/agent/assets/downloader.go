package assets

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

const (
	defaultArchiveName = "app-assets.tgz"
)

// DownloadFromImage pulls assets from an OCI image and unpacks them into outputDir.
func DownloadFromImage(ctx context.Context, imageRef, outputDir string) error {
	if imageRef == "" {
		return errors.New("image reference is required")
	}
	if outputDir == "" {
		return errors.New("assets output directory is required")
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return errors.Wrapf(err, "failed to create assets output directory %s", outputDir)
	}

	tempDir, err := os.MkdirTemp("", "runme-agent-assets-*")
	if err != nil {
		return errors.Wrap(err, "failed to create temporary assets directory")
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	if err := pullImage(ctx, imageRef, tempDir); err != nil {
		return err
	}

	archivePath := filepath.Join(tempDir, defaultArchiveName)
	if _, err := os.Stat(archivePath); err != nil {
		return errors.Wrapf(err, "expected assets archive not found: %s", archivePath)
	}

	if err := removeIndexFiles(outputDir); err != nil {
		return err
	}
	if err := extractTarGz(archivePath, outputDir); err != nil {
		return errors.Wrapf(err, "failed to extract assets archive %s", archivePath)
	}

	return nil
}

func removeIndexFiles(outputDir string) error {
	matches, err := filepath.Glob(filepath.Join(outputDir, "index.*"))
	if err != nil {
		return errors.Wrapf(err, "failed to glob index files in %s", outputDir)
	}
	for _, match := range matches {
		if err := os.RemoveAll(match); err != nil {
			return errors.Wrapf(err, "failed to remove %s", match)
		}
	}
	return nil
}

func pullImage(ctx context.Context, imageRef, outputDir string) error {
	ref, err := registry.ParseReference(imageRef)
	if err != nil {
		return errors.Wrapf(err, "invalid image reference %q", imageRef)
	}

	repo, err := remote.NewRepository(ref.Registry + "/" + ref.Repository)
	if err != nil {
		return errors.Wrapf(err, "failed to create repository for %q", imageRef)
	}

	repo.Client = &auth.Client{
		Client: http.DefaultClient,
		Cache:  auth.NewCache(),
	}

	if ref.Reference == "" {
		ref.Reference = "latest"
	}

	store, err := file.New(outputDir)
	if err != nil {
		return errors.Wrapf(err, "failed to create output file store %s", outputDir)
	}
	defer store.Close()

	if _, err := oras.Copy(ctx, repo, ref.Reference, store, "", oras.DefaultCopyOptions); err != nil {
		return errors.Wrapf(err, "failed to pull image %s", imageRef)
	}

	return nil
}

func extractTarGz(archivePath, destDir string) error {
	fileHandle, err := os.Open(archivePath)
	if err != nil {
		return errors.Wrapf(err, "failed to open archive %s", archivePath)
	}
	defer fileHandle.Close()

	gzipReader, err := gzip.NewReader(fileHandle)
	if err != nil {
		return errors.Wrap(err, "failed to create gzip reader")
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	destDirClean := filepath.Clean(destDir)
	if !strings.HasSuffix(destDirClean, string(os.PathSeparator)) {
		destDirClean += string(os.PathSeparator)
	}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "failed to read tar entry")
		}

		if header == nil {
			continue
		}

		targetPath := filepath.Join(destDir, header.Name)
		cleanTarget := filepath.Clean(targetPath)
		if !strings.HasPrefix(cleanTarget, destDirClean) {
			return errors.Errorf("invalid tar entry path: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(cleanTarget, os.FileMode(header.Mode)); err != nil {
				return errors.Wrapf(err, "failed to create directory %s", cleanTarget)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(cleanTarget), 0o755); err != nil {
				return errors.Wrapf(err, "failed to create parent directory for %s", cleanTarget)
			}
			outFile, err := os.OpenFile(cleanTarget, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return errors.Wrapf(err, "failed to create file %s", cleanTarget)
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				_ = outFile.Close()
				return errors.Wrapf(err, "failed to write file %s", cleanTarget)
			}
			if err := outFile.Close(); err != nil {
				return errors.Wrapf(err, "failed to close file %s", cleanTarget)
			}
		default:
			return errors.Errorf("unsupported tar entry type %v for %s", header.Typeflag, header.Name)
		}
	}

	return nil
}
