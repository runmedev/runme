from runme.parser.v1 import parser_pb2 as _parser_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class LoadEventType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    LOAD_EVENT_TYPE_UNSPECIFIED: _ClassVar[LoadEventType]
    LOAD_EVENT_TYPE_STARTED_WALK: _ClassVar[LoadEventType]
    LOAD_EVENT_TYPE_FOUND_DIR: _ClassVar[LoadEventType]
    LOAD_EVENT_TYPE_FOUND_FILE: _ClassVar[LoadEventType]
    LOAD_EVENT_TYPE_FINISHED_WALK: _ClassVar[LoadEventType]
    LOAD_EVENT_TYPE_STARTED_PARSING_DOC: _ClassVar[LoadEventType]
    LOAD_EVENT_TYPE_FINISHED_PARSING_DOC: _ClassVar[LoadEventType]
    LOAD_EVENT_TYPE_FOUND_TASK: _ClassVar[LoadEventType]
    LOAD_EVENT_TYPE_ERROR: _ClassVar[LoadEventType]
LOAD_EVENT_TYPE_UNSPECIFIED: LoadEventType
LOAD_EVENT_TYPE_STARTED_WALK: LoadEventType
LOAD_EVENT_TYPE_FOUND_DIR: LoadEventType
LOAD_EVENT_TYPE_FOUND_FILE: LoadEventType
LOAD_EVENT_TYPE_FINISHED_WALK: LoadEventType
LOAD_EVENT_TYPE_STARTED_PARSING_DOC: LoadEventType
LOAD_EVENT_TYPE_FINISHED_PARSING_DOC: LoadEventType
LOAD_EVENT_TYPE_FOUND_TASK: LoadEventType
LOAD_EVENT_TYPE_ERROR: LoadEventType

class DirectoryProjectOptions(_message.Message):
    __slots__ = ("path", "skip_gitignore", "ignore_file_patterns", "skip_repo_lookup_upward")
    PATH_FIELD_NUMBER: _ClassVar[int]
    SKIP_GITIGNORE_FIELD_NUMBER: _ClassVar[int]
    IGNORE_FILE_PATTERNS_FIELD_NUMBER: _ClassVar[int]
    SKIP_REPO_LOOKUP_UPWARD_FIELD_NUMBER: _ClassVar[int]
    path: str
    skip_gitignore: bool
    ignore_file_patterns: _containers.RepeatedScalarFieldContainer[str]
    skip_repo_lookup_upward: bool
    def __init__(self, path: _Optional[str] = ..., skip_gitignore: _Optional[bool] = ..., ignore_file_patterns: _Optional[_Iterable[str]] = ..., skip_repo_lookup_upward: _Optional[bool] = ...) -> None: ...

class FileProjectOptions(_message.Message):
    __slots__ = ("path",)
    PATH_FIELD_NUMBER: _ClassVar[int]
    path: str
    def __init__(self, path: _Optional[str] = ...) -> None: ...

class LoadRequest(_message.Message):
    __slots__ = ("directory", "file", "identity")
    DIRECTORY_FIELD_NUMBER: _ClassVar[int]
    FILE_FIELD_NUMBER: _ClassVar[int]
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    directory: DirectoryProjectOptions
    file: FileProjectOptions
    identity: _parser_pb2.RunmeIdentity
    def __init__(self, directory: _Optional[_Union[DirectoryProjectOptions, _Mapping]] = ..., file: _Optional[_Union[FileProjectOptions, _Mapping]] = ..., identity: _Optional[_Union[_parser_pb2.RunmeIdentity, str]] = ...) -> None: ...

class LoadEventStartedWalk(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class LoadEventFoundDir(_message.Message):
    __slots__ = ("path",)
    PATH_FIELD_NUMBER: _ClassVar[int]
    path: str
    def __init__(self, path: _Optional[str] = ...) -> None: ...

class LoadEventFoundFile(_message.Message):
    __slots__ = ("path",)
    PATH_FIELD_NUMBER: _ClassVar[int]
    path: str
    def __init__(self, path: _Optional[str] = ...) -> None: ...

class LoadEventFinishedWalk(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class LoadEventStartedParsingDoc(_message.Message):
    __slots__ = ("path",)
    PATH_FIELD_NUMBER: _ClassVar[int]
    path: str
    def __init__(self, path: _Optional[str] = ...) -> None: ...

class LoadEventFinishedParsingDoc(_message.Message):
    __slots__ = ("path",)
    PATH_FIELD_NUMBER: _ClassVar[int]
    path: str
    def __init__(self, path: _Optional[str] = ...) -> None: ...

class LoadEventFoundTask(_message.Message):
    __slots__ = ("document_path", "id", "name", "is_name_generated")
    DOCUMENT_PATH_FIELD_NUMBER: _ClassVar[int]
    ID_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    IS_NAME_GENERATED_FIELD_NUMBER: _ClassVar[int]
    document_path: str
    id: str
    name: str
    is_name_generated: bool
    def __init__(self, document_path: _Optional[str] = ..., id: _Optional[str] = ..., name: _Optional[str] = ..., is_name_generated: _Optional[bool] = ...) -> None: ...

class LoadEventError(_message.Message):
    __slots__ = ("error_message",)
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    error_message: str
    def __init__(self, error_message: _Optional[str] = ...) -> None: ...

class LoadResponse(_message.Message):
    __slots__ = ("type", "started_walk", "found_dir", "found_file", "finished_walk", "started_parsing_doc", "finished_parsing_doc", "found_task", "error")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    STARTED_WALK_FIELD_NUMBER: _ClassVar[int]
    FOUND_DIR_FIELD_NUMBER: _ClassVar[int]
    FOUND_FILE_FIELD_NUMBER: _ClassVar[int]
    FINISHED_WALK_FIELD_NUMBER: _ClassVar[int]
    STARTED_PARSING_DOC_FIELD_NUMBER: _ClassVar[int]
    FINISHED_PARSING_DOC_FIELD_NUMBER: _ClassVar[int]
    FOUND_TASK_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    type: LoadEventType
    started_walk: LoadEventStartedWalk
    found_dir: LoadEventFoundDir
    found_file: LoadEventFoundFile
    finished_walk: LoadEventFinishedWalk
    started_parsing_doc: LoadEventStartedParsingDoc
    finished_parsing_doc: LoadEventFinishedParsingDoc
    found_task: LoadEventFoundTask
    error: LoadEventError
    def __init__(self, type: _Optional[_Union[LoadEventType, str]] = ..., started_walk: _Optional[_Union[LoadEventStartedWalk, _Mapping]] = ..., found_dir: _Optional[_Union[LoadEventFoundDir, _Mapping]] = ..., found_file: _Optional[_Union[LoadEventFoundFile, _Mapping]] = ..., finished_walk: _Optional[_Union[LoadEventFinishedWalk, _Mapping]] = ..., started_parsing_doc: _Optional[_Union[LoadEventStartedParsingDoc, _Mapping]] = ..., finished_parsing_doc: _Optional[_Union[LoadEventFinishedParsingDoc, _Mapping]] = ..., found_task: _Optional[_Union[LoadEventFoundTask, _Mapping]] = ..., error: _Optional[_Union[LoadEventError, _Mapping]] = ...) -> None: ...
