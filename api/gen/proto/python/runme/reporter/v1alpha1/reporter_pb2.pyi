from runme.parser.v1 import parser_pb2 as _parser_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class TransformRequest(_message.Message):
    __slots__ = ("notebook", "extension")
    NOTEBOOK_FIELD_NUMBER: _ClassVar[int]
    EXTENSION_FIELD_NUMBER: _ClassVar[int]
    notebook: _parser_pb2.Notebook
    extension: TransformRequestExtension
    def __init__(self, notebook: _Optional[_Union[_parser_pb2.Notebook, _Mapping]] = ..., extension: _Optional[_Union[TransformRequestExtension, _Mapping]] = ...) -> None: ...

class TransformRequestExtension(_message.Message):
    __slots__ = ("auto_save", "repository", "branch", "commit", "file_path", "file_content", "plain_output", "masked_output", "mac_address", "hostname", "platform", "release", "arch", "vendor", "shell", "vs_app_host", "vs_app_name", "vs_app_session_id", "vs_machine_id", "vs_metadata")
    class VsMetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    AUTO_SAVE_FIELD_NUMBER: _ClassVar[int]
    REPOSITORY_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    COMMIT_FIELD_NUMBER: _ClassVar[int]
    FILE_PATH_FIELD_NUMBER: _ClassVar[int]
    FILE_CONTENT_FIELD_NUMBER: _ClassVar[int]
    PLAIN_OUTPUT_FIELD_NUMBER: _ClassVar[int]
    MASKED_OUTPUT_FIELD_NUMBER: _ClassVar[int]
    MAC_ADDRESS_FIELD_NUMBER: _ClassVar[int]
    HOSTNAME_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    RELEASE_FIELD_NUMBER: _ClassVar[int]
    ARCH_FIELD_NUMBER: _ClassVar[int]
    VENDOR_FIELD_NUMBER: _ClassVar[int]
    SHELL_FIELD_NUMBER: _ClassVar[int]
    VS_APP_HOST_FIELD_NUMBER: _ClassVar[int]
    VS_APP_NAME_FIELD_NUMBER: _ClassVar[int]
    VS_APP_SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    VS_MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    VS_METADATA_FIELD_NUMBER: _ClassVar[int]
    auto_save: bool
    repository: str
    branch: str
    commit: str
    file_path: str
    file_content: bytes
    plain_output: bytes
    masked_output: bytes
    mac_address: str
    hostname: str
    platform: str
    release: str
    arch: str
    vendor: str
    shell: str
    vs_app_host: str
    vs_app_name: str
    vs_app_session_id: str
    vs_machine_id: str
    vs_metadata: _containers.ScalarMap[str, str]
    def __init__(self, auto_save: _Optional[bool] = ..., repository: _Optional[str] = ..., branch: _Optional[str] = ..., commit: _Optional[str] = ..., file_path: _Optional[str] = ..., file_content: _Optional[bytes] = ..., plain_output: _Optional[bytes] = ..., masked_output: _Optional[bytes] = ..., mac_address: _Optional[str] = ..., hostname: _Optional[str] = ..., platform: _Optional[str] = ..., release: _Optional[str] = ..., arch: _Optional[str] = ..., vendor: _Optional[str] = ..., shell: _Optional[str] = ..., vs_app_host: _Optional[str] = ..., vs_app_name: _Optional[str] = ..., vs_app_session_id: _Optional[str] = ..., vs_machine_id: _Optional[str] = ..., vs_metadata: _Optional[_Mapping[str, str]] = ...) -> None: ...

class TransformResponse(_message.Message):
    __slots__ = ("notebook", "extension")
    NOTEBOOK_FIELD_NUMBER: _ClassVar[int]
    EXTENSION_FIELD_NUMBER: _ClassVar[int]
    notebook: _parser_pb2.Notebook
    extension: ReporterExtension
    def __init__(self, notebook: _Optional[_Union[_parser_pb2.Notebook, _Mapping]] = ..., extension: _Optional[_Union[ReporterExtension, _Mapping]] = ...) -> None: ...

class ReporterExtension(_message.Message):
    __slots__ = ("auto_save", "git", "file", "session", "device")
    AUTO_SAVE_FIELD_NUMBER: _ClassVar[int]
    GIT_FIELD_NUMBER: _ClassVar[int]
    FILE_FIELD_NUMBER: _ClassVar[int]
    SESSION_FIELD_NUMBER: _ClassVar[int]
    DEVICE_FIELD_NUMBER: _ClassVar[int]
    auto_save: bool
    git: ReporterGit
    file: ReporterFile
    session: ReporterSession
    device: ReporterDevice
    def __init__(self, auto_save: _Optional[bool] = ..., git: _Optional[_Union[ReporterGit, _Mapping]] = ..., file: _Optional[_Union[ReporterFile, _Mapping]] = ..., session: _Optional[_Union[ReporterSession, _Mapping]] = ..., device: _Optional[_Union[ReporterDevice, _Mapping]] = ...) -> None: ...

class ReporterGit(_message.Message):
    __slots__ = ("repository", "branch", "commit")
    REPOSITORY_FIELD_NUMBER: _ClassVar[int]
    BRANCH_FIELD_NUMBER: _ClassVar[int]
    COMMIT_FIELD_NUMBER: _ClassVar[int]
    repository: str
    branch: str
    commit: str
    def __init__(self, repository: _Optional[str] = ..., branch: _Optional[str] = ..., commit: _Optional[str] = ...) -> None: ...

class ReporterSession(_message.Message):
    __slots__ = ("plain_output", "masked_output")
    PLAIN_OUTPUT_FIELD_NUMBER: _ClassVar[int]
    MASKED_OUTPUT_FIELD_NUMBER: _ClassVar[int]
    plain_output: bytes
    masked_output: bytes
    def __init__(self, plain_output: _Optional[bytes] = ..., masked_output: _Optional[bytes] = ...) -> None: ...

class ReporterFile(_message.Message):
    __slots__ = ("path", "content")
    PATH_FIELD_NUMBER: _ClassVar[int]
    CONTENT_FIELD_NUMBER: _ClassVar[int]
    path: str
    content: bytes
    def __init__(self, path: _Optional[str] = ..., content: _Optional[bytes] = ...) -> None: ...

class ReporterDevice(_message.Message):
    __slots__ = ("mac_address", "hostname", "platform", "release", "arch", "vendor", "shell", "vs_app_host", "vs_app_name", "vs_app_session_id", "vs_machine_id", "vs_metadata")
    class VsMetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    MAC_ADDRESS_FIELD_NUMBER: _ClassVar[int]
    HOSTNAME_FIELD_NUMBER: _ClassVar[int]
    PLATFORM_FIELD_NUMBER: _ClassVar[int]
    RELEASE_FIELD_NUMBER: _ClassVar[int]
    ARCH_FIELD_NUMBER: _ClassVar[int]
    VENDOR_FIELD_NUMBER: _ClassVar[int]
    SHELL_FIELD_NUMBER: _ClassVar[int]
    VS_APP_HOST_FIELD_NUMBER: _ClassVar[int]
    VS_APP_NAME_FIELD_NUMBER: _ClassVar[int]
    VS_APP_SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    VS_MACHINE_ID_FIELD_NUMBER: _ClassVar[int]
    VS_METADATA_FIELD_NUMBER: _ClassVar[int]
    mac_address: str
    hostname: str
    platform: str
    release: str
    arch: str
    vendor: str
    shell: str
    vs_app_host: str
    vs_app_name: str
    vs_app_session_id: str
    vs_machine_id: str
    vs_metadata: _containers.ScalarMap[str, str]
    def __init__(self, mac_address: _Optional[str] = ..., hostname: _Optional[str] = ..., platform: _Optional[str] = ..., release: _Optional[str] = ..., arch: _Optional[str] = ..., vendor: _Optional[str] = ..., shell: _Optional[str] = ..., vs_app_host: _Optional[str] = ..., vs_app_name: _Optional[str] = ..., vs_app_session_id: _Optional[str] = ..., vs_machine_id: _Optional[str] = ..., vs_metadata: _Optional[_Mapping[str, str]] = ...) -> None: ...
