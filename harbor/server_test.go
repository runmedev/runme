package harbor

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	harborv1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/harbor/v1"
)

func TestProtoJSONRoundTripRequestAndResponseTypes(t *testing.T) {
	requests := []*harborv1.Request{
		{Id: "preflight", Payload: &harborv1.Request_Preflight{Preflight: &harborv1.PreflightRequest{}}},
		{Id: "start", Payload: &harborv1.Request_Start{Start: &harborv1.StartRequest{Root: t.TempDir(), Env: []string{"A=B"}}}},
		{Id: "stop", Payload: &harborv1.Request_Stop{Stop: &harborv1.StopRequest{}}},
		{Id: "exec", Payload: &harborv1.Request_Exec{Exec: &harborv1.ExecRequest{Command: "echo ok", Cwd: ".", Env: []string{"A=B"}}}},
		{Id: "upload-file", Payload: &harborv1.Request_UploadFile{UploadFile: &harborv1.UploadFileRequest{Path: "a.txt", Data: []byte("a"), Mode: 0o600}}},
		{Id: "download-file", Payload: &harborv1.Request_DownloadFile{DownloadFile: &harborv1.DownloadFileRequest{Path: "a.txt"}}},
		{Id: "upload-dir", Payload: &harborv1.Request_UploadDirectory{UploadDirectory: &harborv1.UploadDirectoryRequest{Path: "d", Files: []*harborv1.FileEntry{{Path: "a.txt", Data: []byte("a"), Mode: 0o600}}}}},
		{Id: "download-dir", Payload: &harborv1.Request_DownloadDirectory{DownloadDirectory: &harborv1.DownloadDirectoryRequest{Path: "d"}}},
	}
	for _, req := range requests {
		data, err := protojson.Marshal(req)
		if err != nil {
			t.Fatalf("Marshal(%s): %v", req.GetId(), err)
		}
		var got harborv1.Request
		if err := protojson.Unmarshal(data, &got); err != nil {
			t.Fatalf("Unmarshal(%s): %v", req.GetId(), err)
		}
		if got.GetId() != req.GetId() {
			t.Fatalf("round trip id = %q, want %q", got.GetId(), req.GetId())
		}
	}

	responses := []*harborv1.Response{
		{Id: "preflight", Payload: &harborv1.Response_Preflight{Preflight: &harborv1.PreflightResponse{Protocol: ProtocolName}}},
		{Id: "start", Payload: &harborv1.Response_Start{Start: &harborv1.StartResponse{Root: t.TempDir()}}},
		{Id: "stop", Payload: &harborv1.Response_Stop{Stop: &harborv1.StopResponse{}}},
		{Id: "exec", Payload: &harborv1.Response_Exec{Exec: &harborv1.ExecResponse{Stdout: []byte("ok\n"), Stderr: []byte("warn\n"), ExitCode: 3}}},
		{Id: "upload-file", Payload: &harborv1.Response_UploadFile{UploadFile: &harborv1.UploadFileResponse{BytesWritten: 1}}},
		{Id: "download-file", Payload: &harborv1.Response_DownloadFile{DownloadFile: &harborv1.DownloadFileResponse{Data: []byte("a"), Mode: 0o600}}},
		{Id: "upload-dir", Payload: &harborv1.Response_UploadDirectory{UploadDirectory: &harborv1.UploadDirectoryResponse{FilesWritten: 1}}},
		{Id: "download-dir", Payload: &harborv1.Response_DownloadDirectory{DownloadDirectory: &harborv1.DownloadDirectoryResponse{Files: []*harborv1.FileEntry{{Path: "a.txt", Data: []byte("a"), Mode: 0o600}}}}},
		{Id: "error", Error: &harborv1.Error{Code: "invalid_argument", Message: "bad request"}},
	}
	for _, resp := range responses {
		data, err := protojson.Marshal(resp)
		if err != nil {
			t.Fatalf("Marshal(%s): %v", resp.GetId(), err)
		}
		var got harborv1.Response
		if err := protojson.Unmarshal(data, &got); err != nil {
			t.Fatalf("Unmarshal(%s): %v", resp.GetId(), err)
		}
		if got.GetId() != resp.GetId() {
			t.Fatalf("round trip id = %q, want %q", got.GetId(), resp.GetId())
		}
	}
}

func TestServeStdioPreservesRequestIDsAndReportsProtocolErrors(t *testing.T) {
	server, err := NewServer(Options{})
	if err != nil {
		t.Fatal(err)
	}
	input := strings.Join([]string{
		`{"id":"one","preflight":{}}`,
		`{"preflight":{}}`,
		`not-json`,
		"",
	}, "\n")
	var output bytes.Buffer
	if err := server.ServeStdio(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("response count = %d, want 3: %q", len(lines), output.String())
	}
	var first harborv1.Response
	if err := protojson.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatal(err)
	}
	if first.GetId() != "one" || first.GetPreflight().GetProtocol() != ProtocolName {
		t.Fatalf("first response = %+v", &first)
	}
	var missingID harborv1.Response
	if err := protojson.Unmarshal([]byte(lines[1]), &missingID); err != nil {
		t.Fatal(err)
	}
	if missingID.GetError().GetCode() != "invalid_argument" {
		t.Fatalf("missing id error = %+v", missingID.GetError())
	}
	var invalidJSON harborv1.Response
	if err := protojson.Unmarshal([]byte(lines[2]), &invalidJSON); err != nil {
		t.Fatal(err)
	}
	if invalidJSON.GetError().GetCode() != "invalid_json" {
		t.Fatalf("invalid json error = %+v", invalidJSON.GetError())
	}
}

func TestExecCapturesStdoutStderrAndExitCode(t *testing.T) {
	server, err := NewServer(Options{})
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	server.Handle(context.Background(), &harborv1.Request{
		Id:      "start",
		Payload: &harborv1.Request_Start{Start: &harborv1.StartRequest{Root: root}},
	})

	resp := server.Handle(context.Background(), &harborv1.Request{
		Id: "exec",
		Payload: &harborv1.Request_Exec{Exec: &harborv1.ExecRequest{
			Command: `printf stdout; printf stderr >&2; exit 7`,
			Cwd:     ".",
		}},
	})
	if resp.GetError() != nil {
		t.Fatalf("exec error: %+v", resp.GetError())
	}
	exec := resp.GetExec()
	if string(exec.GetStdout()) != "stdout" {
		t.Fatalf("stdout = %q", exec.GetStdout())
	}
	if string(exec.GetStderr()) != "stderr" {
		t.Fatalf("stderr = %q", exec.GetStderr())
	}
	if exec.GetExitCode() != 7 {
		t.Fatalf("exit code = %d, want 7", exec.GetExitCode())
	}
}

func TestFileTransferAndPathMapping(t *testing.T) {
	server, err := NewServer(Options{})
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	server.Handle(context.Background(), &harborv1.Request{
		Id:      "start",
		Payload: &harborv1.Request_Start{Start: &harborv1.StartRequest{Root: root}},
	})

	upload := server.Handle(context.Background(), &harborv1.Request{
		Id: "upload",
		Payload: &harborv1.Request_UploadFile{UploadFile: &harborv1.UploadFileRequest{
			Path: "dir/quoted path.txt",
			Data: []byte("hello"),
			Mode: 0o600,
		}},
	})
	if upload.GetError() != nil {
		t.Fatalf("upload error: %+v", upload.GetError())
	}
	if upload.GetUploadFile().GetBytesWritten() != 5 {
		t.Fatalf("bytes written = %d", upload.GetUploadFile().GetBytesWritten())
	}

	download := server.Handle(context.Background(), &harborv1.Request{
		Id:      "download",
		Payload: &harborv1.Request_DownloadFile{DownloadFile: &harborv1.DownloadFileRequest{Path: filepath.Join("dir", "quoted path.txt")}},
	})
	if download.GetError() != nil {
		t.Fatalf("download error: %+v", download.GetError())
	}
	if string(download.GetDownloadFile().GetData()) != "hello" {
		t.Fatalf("download data = %q", download.GetDownloadFile().GetData())
	}

	escape := server.Handle(context.Background(), &harborv1.Request{
		Id:      "escape",
		Payload: &harborv1.Request_DownloadFile{DownloadFile: &harborv1.DownloadFileRequest{Path: "../outside.txt"}},
	})
	if escape.GetError().GetCode() != "invalid_argument" {
		t.Fatalf("escape error = %+v", escape.GetError())
	}
}

func TestDirectoryUploadDownload(t *testing.T) {
	server, err := NewServer(Options{})
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	server.Handle(context.Background(), &harborv1.Request{
		Id:      "start",
		Payload: &harborv1.Request_Start{Start: &harborv1.StartRequest{Root: root}},
	})

	upload := server.Handle(context.Background(), &harborv1.Request{
		Id: "upload-dir",
		Payload: &harborv1.Request_UploadDirectory{UploadDirectory: &harborv1.UploadDirectoryRequest{
			Path: "bundle",
			Files: []*harborv1.FileEntry{
				{Path: "a.txt", Data: []byte("a"), Mode: 0o644},
				{Path: "nested/b.txt", Data: []byte("b"), Mode: 0o600},
			},
		}},
	})
	if upload.GetError() != nil {
		t.Fatalf("upload directory error: %+v", upload.GetError())
	}
	if upload.GetUploadDirectory().GetFilesWritten() != 2 {
		t.Fatalf("files written = %d", upload.GetUploadDirectory().GetFilesWritten())
	}

	download := server.Handle(context.Background(), &harborv1.Request{
		Id:      "download-dir",
		Payload: &harborv1.Request_DownloadDirectory{DownloadDirectory: &harborv1.DownloadDirectoryRequest{Path: "bundle"}},
	})
	if download.GetError() != nil {
		t.Fatalf("download directory error: %+v", download.GetError())
	}
	if len(download.GetDownloadDirectory().GetFiles()) != 2 {
		t.Fatalf("downloaded file count = %d", len(download.GetDownloadDirectory().GetFiles()))
	}
}
