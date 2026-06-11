from google.protobuf import wrappers_pb2 as _wrappers_pb2
from runme.parser.v1 import docresult_pb2 as _docresult_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CellKind(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CELL_KIND_UNSPECIFIED: _ClassVar[CellKind]
    CELL_KIND_MARKUP: _ClassVar[CellKind]
    CELL_KIND_CODE: _ClassVar[CellKind]
    CELL_KIND_DOC_RESULTS: _ClassVar[CellKind]
    CELL_KIND_TOOL: _ClassVar[CellKind]

class CellRole(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    CELL_ROLE_UNSPECIFIED: _ClassVar[CellRole]
    CELL_ROLE_USER: _ClassVar[CellRole]
    CELL_ROLE_ASSISTANT: _ClassVar[CellRole]

class RunmeIdentity(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RUNME_IDENTITY_UNSPECIFIED: _ClassVar[RunmeIdentity]
    RUNME_IDENTITY_ALL: _ClassVar[RunmeIdentity]
    RUNME_IDENTITY_DOCUMENT: _ClassVar[RunmeIdentity]
    RUNME_IDENTITY_CELL: _ClassVar[RunmeIdentity]
CELL_KIND_UNSPECIFIED: CellKind
CELL_KIND_MARKUP: CellKind
CELL_KIND_CODE: CellKind
CELL_KIND_DOC_RESULTS: CellKind
CELL_KIND_TOOL: CellKind
CELL_ROLE_UNSPECIFIED: CellRole
CELL_ROLE_USER: CellRole
CELL_ROLE_ASSISTANT: CellRole
RUNME_IDENTITY_UNSPECIFIED: RunmeIdentity
RUNME_IDENTITY_ALL: RunmeIdentity
RUNME_IDENTITY_DOCUMENT: RunmeIdentity
RUNME_IDENTITY_CELL: RunmeIdentity

class Notebook(_message.Message):
    __slots__ = ("cells", "metadata", "frontmatter")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    CELLS_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    FRONTMATTER_FIELD_NUMBER: _ClassVar[int]
    cells: _containers.RepeatedCompositeFieldContainer[Cell]
    metadata: _containers.ScalarMap[str, str]
    frontmatter: Frontmatter
    def __init__(self, cells: _Optional[_Iterable[_Union[Cell, _Mapping]]] = ..., metadata: _Optional[_Mapping[str, str]] = ..., frontmatter: _Optional[_Union[Frontmatter, _Mapping]] = ...) -> None: ...

class ExecutionSummaryTiming(_message.Message):
    __slots__ = ("start_time", "end_time")
    START_TIME_FIELD_NUMBER: _ClassVar[int]
    END_TIME_FIELD_NUMBER: _ClassVar[int]
    start_time: _wrappers_pb2.Int64Value
    end_time: _wrappers_pb2.Int64Value
    def __init__(self, start_time: _Optional[_Union[_wrappers_pb2.Int64Value, _Mapping]] = ..., end_time: _Optional[_Union[_wrappers_pb2.Int64Value, _Mapping]] = ...) -> None: ...

class CellOutputItem(_message.Message):
    __slots__ = ("data", "type", "mime")
    DATA_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    MIME_FIELD_NUMBER: _ClassVar[int]
    data: bytes
    type: str
    mime: str
    def __init__(self, data: _Optional[bytes] = ..., type: _Optional[str] = ..., mime: _Optional[str] = ...) -> None: ...

class ProcessInfoExitReason(_message.Message):
    __slots__ = ("type", "code")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    CODE_FIELD_NUMBER: _ClassVar[int]
    type: str
    code: _wrappers_pb2.UInt32Value
    def __init__(self, type: _Optional[str] = ..., code: _Optional[_Union[_wrappers_pb2.UInt32Value, _Mapping]] = ...) -> None: ...

class CellOutputProcessInfo(_message.Message):
    __slots__ = ("exit_reason", "pid")
    EXIT_REASON_FIELD_NUMBER: _ClassVar[int]
    PID_FIELD_NUMBER: _ClassVar[int]
    exit_reason: ProcessInfoExitReason
    pid: _wrappers_pb2.Int64Value
    def __init__(self, exit_reason: _Optional[_Union[ProcessInfoExitReason, _Mapping]] = ..., pid: _Optional[_Union[_wrappers_pb2.Int64Value, _Mapping]] = ...) -> None: ...

class CellOutput(_message.Message):
    __slots__ = ("items", "metadata", "process_info")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ITEMS_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    PROCESS_INFO_FIELD_NUMBER: _ClassVar[int]
    items: _containers.RepeatedCompositeFieldContainer[CellOutputItem]
    metadata: _containers.ScalarMap[str, str]
    process_info: CellOutputProcessInfo
    def __init__(self, items: _Optional[_Iterable[_Union[CellOutputItem, _Mapping]]] = ..., metadata: _Optional[_Mapping[str, str]] = ..., process_info: _Optional[_Union[CellOutputProcessInfo, _Mapping]] = ...) -> None: ...

class CellExecutionSummary(_message.Message):
    __slots__ = ("execution_order", "success", "timing")
    EXECUTION_ORDER_FIELD_NUMBER: _ClassVar[int]
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    TIMING_FIELD_NUMBER: _ClassVar[int]
    execution_order: _wrappers_pb2.UInt32Value
    success: _wrappers_pb2.BoolValue
    timing: ExecutionSummaryTiming
    def __init__(self, execution_order: _Optional[_Union[_wrappers_pb2.UInt32Value, _Mapping]] = ..., success: _Optional[_Union[_wrappers_pb2.BoolValue, _Mapping]] = ..., timing: _Optional[_Union[ExecutionSummaryTiming, _Mapping]] = ...) -> None: ...

class TextRange(_message.Message):
    __slots__ = ("start", "end")
    START_FIELD_NUMBER: _ClassVar[int]
    END_FIELD_NUMBER: _ClassVar[int]
    start: int
    end: int
    def __init__(self, start: _Optional[int] = ..., end: _Optional[int] = ...) -> None: ...

class Cell(_message.Message):
    __slots__ = ("kind", "value", "language_id", "metadata", "text_range", "outputs", "execution_summary", "ref_id", "role", "call_id", "doc_results")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    KIND_FIELD_NUMBER: _ClassVar[int]
    VALUE_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_ID_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    TEXT_RANGE_FIELD_NUMBER: _ClassVar[int]
    OUTPUTS_FIELD_NUMBER: _ClassVar[int]
    EXECUTION_SUMMARY_FIELD_NUMBER: _ClassVar[int]
    REF_ID_FIELD_NUMBER: _ClassVar[int]
    ROLE_FIELD_NUMBER: _ClassVar[int]
    CALL_ID_FIELD_NUMBER: _ClassVar[int]
    DOC_RESULTS_FIELD_NUMBER: _ClassVar[int]
    kind: CellKind
    value: str
    language_id: str
    metadata: _containers.ScalarMap[str, str]
    text_range: TextRange
    outputs: _containers.RepeatedCompositeFieldContainer[CellOutput]
    execution_summary: CellExecutionSummary
    ref_id: str
    role: CellRole
    call_id: str
    doc_results: _containers.RepeatedCompositeFieldContainer[_docresult_pb2.DocResult]
    def __init__(self, kind: _Optional[_Union[CellKind, str]] = ..., value: _Optional[str] = ..., language_id: _Optional[str] = ..., metadata: _Optional[_Mapping[str, str]] = ..., text_range: _Optional[_Union[TextRange, _Mapping]] = ..., outputs: _Optional[_Iterable[_Union[CellOutput, _Mapping]]] = ..., execution_summary: _Optional[_Union[CellExecutionSummary, _Mapping]] = ..., ref_id: _Optional[str] = ..., role: _Optional[_Union[CellRole, str]] = ..., call_id: _Optional[str] = ..., doc_results: _Optional[_Iterable[_Union[_docresult_pb2.DocResult, _Mapping]]] = ...) -> None: ...

class RunmeSessionDocument(_message.Message):
    __slots__ = ("relative_path",)
    RELATIVE_PATH_FIELD_NUMBER: _ClassVar[int]
    relative_path: str
    def __init__(self, relative_path: _Optional[str] = ...) -> None: ...

class RunmeSession(_message.Message):
    __slots__ = ("id", "document")
    ID_FIELD_NUMBER: _ClassVar[int]
    DOCUMENT_FIELD_NUMBER: _ClassVar[int]
    id: str
    document: RunmeSessionDocument
    def __init__(self, id: _Optional[str] = ..., document: _Optional[_Union[RunmeSessionDocument, _Mapping]] = ...) -> None: ...

class FrontmatterRunme(_message.Message):
    __slots__ = ("id", "version", "session")
    ID_FIELD_NUMBER: _ClassVar[int]
    VERSION_FIELD_NUMBER: _ClassVar[int]
    SESSION_FIELD_NUMBER: _ClassVar[int]
    id: str
    version: str
    session: RunmeSession
    def __init__(self, id: _Optional[str] = ..., version: _Optional[str] = ..., session: _Optional[_Union[RunmeSession, _Mapping]] = ...) -> None: ...

class Frontmatter(_message.Message):
    __slots__ = ("shell", "cwd", "skip_prompts", "runme", "category", "terminal_rows", "tag")
    SHELL_FIELD_NUMBER: _ClassVar[int]
    CWD_FIELD_NUMBER: _ClassVar[int]
    SKIP_PROMPTS_FIELD_NUMBER: _ClassVar[int]
    RUNME_FIELD_NUMBER: _ClassVar[int]
    CATEGORY_FIELD_NUMBER: _ClassVar[int]
    TERMINAL_ROWS_FIELD_NUMBER: _ClassVar[int]
    TAG_FIELD_NUMBER: _ClassVar[int]
    shell: str
    cwd: str
    skip_prompts: bool
    runme: FrontmatterRunme
    category: str
    terminal_rows: str
    tag: str
    def __init__(self, shell: _Optional[str] = ..., cwd: _Optional[str] = ..., skip_prompts: _Optional[bool] = ..., runme: _Optional[_Union[FrontmatterRunme, _Mapping]] = ..., category: _Optional[str] = ..., terminal_rows: _Optional[str] = ..., tag: _Optional[str] = ...) -> None: ...

class DeserializeRequestOptions(_message.Message):
    __slots__ = ("identity",)
    IDENTITY_FIELD_NUMBER: _ClassVar[int]
    identity: RunmeIdentity
    def __init__(self, identity: _Optional[_Union[RunmeIdentity, str]] = ...) -> None: ...

class DeserializeRequest(_message.Message):
    __slots__ = ("source", "options")
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    source: bytes
    options: DeserializeRequestOptions
    def __init__(self, source: _Optional[bytes] = ..., options: _Optional[_Union[DeserializeRequestOptions, _Mapping]] = ...) -> None: ...

class DeserializeResponse(_message.Message):
    __slots__ = ("notebook",)
    NOTEBOOK_FIELD_NUMBER: _ClassVar[int]
    notebook: Notebook
    def __init__(self, notebook: _Optional[_Union[Notebook, _Mapping]] = ...) -> None: ...

class SerializeRequestOutputOptions(_message.Message):
    __slots__ = ("enabled", "summary", "profile")
    ENABLED_FIELD_NUMBER: _ClassVar[int]
    SUMMARY_FIELD_NUMBER: _ClassVar[int]
    PROFILE_FIELD_NUMBER: _ClassVar[int]
    enabled: bool
    summary: bool
    profile: str
    def __init__(self, enabled: _Optional[bool] = ..., summary: _Optional[bool] = ..., profile: _Optional[str] = ...) -> None: ...

class SerializeRequestOptions(_message.Message):
    __slots__ = ("outputs", "session")
    OUTPUTS_FIELD_NUMBER: _ClassVar[int]
    SESSION_FIELD_NUMBER: _ClassVar[int]
    outputs: SerializeRequestOutputOptions
    session: RunmeSession
    def __init__(self, outputs: _Optional[_Union[SerializeRequestOutputOptions, _Mapping]] = ..., session: _Optional[_Union[RunmeSession, _Mapping]] = ...) -> None: ...

class SerializeRequest(_message.Message):
    __slots__ = ("notebook", "options")
    NOTEBOOK_FIELD_NUMBER: _ClassVar[int]
    OPTIONS_FIELD_NUMBER: _ClassVar[int]
    notebook: Notebook
    options: SerializeRequestOptions
    def __init__(self, notebook: _Optional[_Union[Notebook, _Mapping]] = ..., options: _Optional[_Union[SerializeRequestOptions, _Mapping]] = ...) -> None: ...

class SerializeResponse(_message.Message):
    __slots__ = ("result",)
    RESULT_FIELD_NUMBER: _ClassVar[int]
    result: bytes
    def __init__(self, result: _Optional[bytes] = ...) -> None: ...
