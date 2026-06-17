from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Error(_message.Message):
    __slots__ = ("code", "message")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    code: str
    message: str
    def __init__(self, code: _Optional[str] = ..., message: _Optional[str] = ...) -> None: ...

class Request(_message.Message):
    __slots__ = ("id", "preflight", "start", "stop", "exec", "upload_file", "download_file", "upload_directory", "download_directory")
    ID_FIELD_NUMBER: _ClassVar[int]
    PREFLIGHT_FIELD_NUMBER: _ClassVar[int]
    START_FIELD_NUMBER: _ClassVar[int]
    STOP_FIELD_NUMBER: _ClassVar[int]
    EXEC_FIELD_NUMBER: _ClassVar[int]
    UPLOAD_FILE_FIELD_NUMBER: _ClassVar[int]
    DOWNLOAD_FILE_FIELD_NUMBER: _ClassVar[int]
    UPLOAD_DIRECTORY_FIELD_NUMBER: _ClassVar[int]
    DOWNLOAD_DIRECTORY_FIELD_NUMBER: _ClassVar[int]
    id: str
    preflight: PreflightRequest
    start: StartRequest
    stop: StopRequest
    exec: ExecRequest
    upload_file: UploadFileRequest
    download_file: DownloadFileRequest
    upload_directory: UploadDirectoryRequest
    download_directory: DownloadDirectoryRequest
    def __init__(self, id: _Optional[str] = ..., preflight: _Optional[_Union[PreflightRequest, _Mapping]] = ..., start: _Optional[_Union[StartRequest, _Mapping]] = ..., stop: _Optional[_Union[StopRequest, _Mapping]] = ..., exec: _Optional[_Union[ExecRequest, _Mapping]] = ..., upload_file: _Optional[_Union[UploadFileRequest, _Mapping]] = ..., download_file: _Optional[_Union[DownloadFileRequest, _Mapping]] = ..., upload_directory: _Optional[_Union[UploadDirectoryRequest, _Mapping]] = ..., download_directory: _Optional[_Union[DownloadDirectoryRequest, _Mapping]] = ...) -> None: ...

class Response(_message.Message):
    __slots__ = ("id", "error", "preflight", "start", "stop", "exec", "upload_file", "download_file", "upload_directory", "download_directory")
    ID_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    PREFLIGHT_FIELD_NUMBER: _ClassVar[int]
    START_FIELD_NUMBER: _ClassVar[int]
    STOP_FIELD_NUMBER: _ClassVar[int]
    EXEC_FIELD_NUMBER: _ClassVar[int]
    UPLOAD_FILE_FIELD_NUMBER: _ClassVar[int]
    DOWNLOAD_FILE_FIELD_NUMBER: _ClassVar[int]
    UPLOAD_DIRECTORY_FIELD_NUMBER: _ClassVar[int]
    DOWNLOAD_DIRECTORY_FIELD_NUMBER: _ClassVar[int]
    id: str
    error: Error
    preflight: PreflightResponse
    start: StartResponse
    stop: StopResponse
    exec: ExecResponse
    upload_file: UploadFileResponse
    download_file: DownloadFileResponse
    upload_directory: UploadDirectoryResponse
    download_directory: DownloadDirectoryResponse
    def __init__(self, id: _Optional[str] = ..., error: _Optional[_Union[Error, _Mapping]] = ..., preflight: _Optional[_Union[PreflightResponse, _Mapping]] = ..., start: _Optional[_Union[StartResponse, _Mapping]] = ..., stop: _Optional[_Union[StopResponse, _Mapping]] = ..., exec: _Optional[_Union[ExecResponse, _Mapping]] = ..., upload_file: _Optional[_Union[UploadFileResponse, _Mapping]] = ..., download_file: _Optional[_Union[DownloadFileResponse, _Mapping]] = ..., upload_directory: _Optional[_Union[UploadDirectoryResponse, _Mapping]] = ..., download_directory: _Optional[_Union[DownloadDirectoryResponse, _Mapping]] = ...) -> None: ...

class PreflightRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class PreflightResponse(_message.Message):
    __slots__ = ("protocol", "version", "capabilities")
    PROTOCOL_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    CAPABILITIES_FIELD_NUMBER: _ClassVar[int]
    protocol: str
    version: str
    capabilities: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, protocol: _Optional[str] = ..., version: _Optional[str] = ..., capabilities: _Optional[_Iterable[str]] = ...) -> None: ...

class StartRequest(_message.Message):
    __slots__ = ("root", "env")
    ROOT_FIELD_NUMBER: _ClassVar[int]
    ENV_FIELD_NUMBER: _ClassVar[int]
    root: str
    env: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, root: _Optional[str] = ..., env: _Optional[_Iterable[str]] = ...) -> None: ...

class StartResponse(_message.Message):
    __slots__ = ("root",)
    ROOT_FIELD_NUMBER: _ClassVar[int]
    root: str
    def __init__(self, root: _Optional[str] = ...) -> None: ...

class StopRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class StopResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ExecRequest(_message.Message):
    __slots__ = ("command", "cwd", "env")
    COMMAND_FIELD_NUMBER: _ClassVar[int]
    CWD_FIELD_NUMBER: _ClassVar[int]
    ENV_FIELD_NUMBER: _ClassVar[int]
    command: str
    cwd: str
    env: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, command: _Optional[str] = ..., cwd: _Optional[str] = ..., env: _Optional[_Iterable[str]] = ...) -> None: ...

class ExecResponse(_message.Message):
    __slots__ = ("stdout", "stderr", "exit_code")
    STDOUT_FIELD_NUMBER: _ClassVar[int]
    STDERR_FIELD_NUMBER: _ClassVar[int]
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    stdout: bytes
    stderr: bytes
    exit_code: int
    def __init__(self, stdout: _Optional[bytes] = ..., stderr: _Optional[bytes] = ..., exit_code: _Optional[int] = ...) -> None: ...

class FileEntry(_message.Message):
    __slots__ = ("path", "data", "mode")
    PATH_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    path: str
    data: bytes
    mode: int
    def __init__(self, path: _Optional[str] = ..., data: _Optional[bytes] = ..., mode: _Optional[int] = ...) -> None: ...

class UploadFileRequest(_message.Message):
    __slots__ = ("path", "data", "mode")
    PATH_FIELD_NUMBER: _ClassVar[int]
    DATA_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    path: str
    data: bytes
    mode: int
    def __init__(self, path: _Optional[str] = ..., data: _Optional[bytes] = ..., mode: _Optional[int] = ...) -> None: ...

class UploadFileResponse(_message.Message):
    __slots__ = ("bytes_written",)
    BYTES_WRITTEN_FIELD_NUMBER: _ClassVar[int]
    bytes_written: int
    def __init__(self, bytes_written: _Optional[int] = ...) -> None: ...

class DownloadFileRequest(_message.Message):
    __slots__ = ("path",)
    PATH_FIELD_NUMBER: _ClassVar[int]
    path: str
    def __init__(self, path: _Optional[str] = ...) -> None: ...

class DownloadFileResponse(_message.Message):
    __slots__ = ("data", "mode")
    DATA_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    data: bytes
    mode: int
    def __init__(self, data: _Optional[bytes] = ..., mode: _Optional[int] = ...) -> None: ...

class UploadDirectoryRequest(_message.Message):
    __slots__ = ("path", "files")
    PATH_FIELD_NUMBER: _ClassVar[int]
    FILES_FIELD_NUMBER: _ClassVar[int]
    path: str
    files: _containers.RepeatedCompositeFieldContainer[FileEntry]
    def __init__(self, path: _Optional[str] = ..., files: _Optional[_Iterable[_Union[FileEntry, _Mapping]]] = ...) -> None: ...

class UploadDirectoryResponse(_message.Message):
    __slots__ = ("files_written",)
    FILES_WRITTEN_FIELD_NUMBER: _ClassVar[int]
    files_written: int
    def __init__(self, files_written: _Optional[int] = ...) -> None: ...

class DownloadDirectoryRequest(_message.Message):
    __slots__ = ("path",)
    PATH_FIELD_NUMBER: _ClassVar[int]
    path: str
    def __init__(self, path: _Optional[str] = ...) -> None: ...

class DownloadDirectoryResponse(_message.Message):
    __slots__ = ("files",)
    FILES_FIELD_NUMBER: _ClassVar[int]
    files: _containers.RepeatedCompositeFieldContainer[FileEntry]
    def __init__(self, files: _Optional[_Iterable[_Union[FileEntry, _Mapping]]] = ...) -> None: ...
