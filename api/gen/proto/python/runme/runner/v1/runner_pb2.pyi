from google.protobuf import wrappers_pb2 as _wrappers_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class SessionEnvStoreType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SESSION_ENV_STORE_TYPE_UNSPECIFIED: _ClassVar[SessionEnvStoreType]
    SESSION_ENV_STORE_TYPE_OWL: _ClassVar[SessionEnvStoreType]

class ExecuteStop(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    EXECUTE_STOP_UNSPECIFIED: _ClassVar[ExecuteStop]
    EXECUTE_STOP_INTERRUPT: _ClassVar[ExecuteStop]
    EXECUTE_STOP_KILL: _ClassVar[ExecuteStop]

class CommandMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    COMMAND_MODE_UNSPECIFIED: _ClassVar[CommandMode]
    COMMAND_MODE_INLINE_SHELL: _ClassVar[CommandMode]
    COMMAND_MODE_TEMP_FILE: _ClassVar[CommandMode]
    COMMAND_MODE_TERMINAL: _ClassVar[CommandMode]
    COMMAND_MODE_DAGGER: _ClassVar[CommandMode]

class SessionStrategy(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    SESSION_STRATEGY_UNSPECIFIED: _ClassVar[SessionStrategy]
    SESSION_STRATEGY_MOST_RECENT: _ClassVar[SessionStrategy]

class MonitorEnvStoreType(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    MONITOR_ENV_STORE_TYPE_UNSPECIFIED: _ClassVar[MonitorEnvStoreType]
    MONITOR_ENV_STORE_TYPE_SNAPSHOT: _ClassVar[MonitorEnvStoreType]
SESSION_ENV_STORE_TYPE_UNSPECIFIED: SessionEnvStoreType
SESSION_ENV_STORE_TYPE_OWL: SessionEnvStoreType
EXECUTE_STOP_UNSPECIFIED: ExecuteStop
EXECUTE_STOP_INTERRUPT: ExecuteStop
EXECUTE_STOP_KILL: ExecuteStop
COMMAND_MODE_UNSPECIFIED: CommandMode
COMMAND_MODE_INLINE_SHELL: CommandMode
COMMAND_MODE_TEMP_FILE: CommandMode
COMMAND_MODE_TERMINAL: CommandMode
COMMAND_MODE_DAGGER: CommandMode
SESSION_STRATEGY_UNSPECIFIED: SessionStrategy
SESSION_STRATEGY_MOST_RECENT: SessionStrategy
MONITOR_ENV_STORE_TYPE_UNSPECIFIED: MonitorEnvStoreType
MONITOR_ENV_STORE_TYPE_SNAPSHOT: MonitorEnvStoreType

class Session(_message.Message):
    __slots__ = ("id", "envs", "metadata")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    ENVS_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    id: str
    envs: _containers.RepeatedScalarFieldContainer[str]
    metadata: _containers.ScalarMap[str, str]
    def __init__(self, id: _Optional[str] = ..., envs: _Optional[_Iterable[str]] = ..., metadata: _Optional[_Mapping[str, str]] = ...) -> None: ...

class CreateSessionRequest(_message.Message):
    __slots__ = ("metadata", "envs", "project", "env_store_type")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    METADATA_FIELD_NUMBER: _ClassVar[int]
    ENVS_FIELD_NUMBER: _ClassVar[int]
    PROJECT_FIELD_NUMBER: _ClassVar[int]
    ENV_STORE_TYPE_FIELD_NUMBER: _ClassVar[int]
    metadata: _containers.ScalarMap[str, str]
    envs: _containers.RepeatedScalarFieldContainer[str]
    project: Project
    env_store_type: SessionEnvStoreType
    def __init__(self, metadata: _Optional[_Mapping[str, str]] = ..., envs: _Optional[_Iterable[str]] = ..., project: _Optional[_Union[Project, _Mapping]] = ..., env_store_type: _Optional[_Union[SessionEnvStoreType, str]] = ...) -> None: ...

class CreateSessionResponse(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: Session
    def __init__(self, session: _Optional[_Union[Session, _Mapping]] = ...) -> None: ...

class GetSessionRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class GetSessionResponse(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: Session
    def __init__(self, session: _Optional[_Union[Session, _Mapping]] = ...) -> None: ...

class ListSessionsRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ListSessionsResponse(_message.Message):
    __slots__ = ("sessions",)
    SESSIONS_FIELD_NUMBER: _ClassVar[int]
    sessions: _containers.RepeatedCompositeFieldContainer[Session]
    def __init__(self, sessions: _Optional[_Iterable[_Union[Session, _Mapping]]] = ...) -> None: ...

class DeleteSessionRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteSessionResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class Project(_message.Message):
    __slots__ = ("root", "env_load_order", "env_direnv")
    class DirEnv(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        DIR_ENV_UNSPECIFIED: _ClassVar[Project.DirEnv]
        DIR_ENV_ENABLED_WARN: _ClassVar[Project.DirEnv]
        DIR_ENV_ENABLED_ERROR: _ClassVar[Project.DirEnv]
        DIR_ENV_DISABLED: _ClassVar[Project.DirEnv]
    DIR_ENV_UNSPECIFIED: Project.DirEnv
    DIR_ENV_ENABLED_WARN: Project.DirEnv
    DIR_ENV_ENABLED_ERROR: Project.DirEnv
    DIR_ENV_DISABLED: Project.DirEnv
    ROOT_FIELD_NUMBER: _ClassVar[int]
    ENV_LOAD_ORDER_FIELD_NUMBER: _ClassVar[int]
    ENV_DIRENV_FIELD_NUMBER: _ClassVar[int]
    root: str
    env_load_order: _containers.RepeatedScalarFieldContainer[str]
    env_direnv: Project.DirEnv
    def __init__(self, root: _Optional[str] = ..., env_load_order: _Optional[_Iterable[str]] = ..., env_direnv: _Optional[_Union[Project.DirEnv, str]] = ...) -> None: ...

class Winsize(_message.Message):
    __slots__ = ("rows", "cols", "x", "y")
    ROWS_FIELD_NUMBER: _ClassVar[int]
    COLS_FIELD_NUMBER: _ClassVar[int]
    X_FIELD_NUMBER: _ClassVar[int]
    Y_FIELD_NUMBER: _ClassVar[int]
    rows: int
    cols: int
    x: int
    y: int
    def __init__(self, rows: _Optional[int] = ..., cols: _Optional[int] = ..., x: _Optional[int] = ..., y: _Optional[int] = ...) -> None: ...

class ExecuteRequest(_message.Message):
    __slots__ = ("program_name", "arguments", "directory", "envs", "commands", "script", "tty", "input_data", "stop", "winsize", "background", "session_id", "session_strategy", "project", "store_last_output", "command_mode", "language_id", "file_extension", "known_id", "known_name", "run_id")
    PROGRAM_NAME_FIELD_NUMBER: _ClassVar[int]
    ARGUMENTS_FIELD_NUMBER: _ClassVar[int]
    DIRECTORY_FIELD_NUMBER: _ClassVar[int]
    ENVS_FIELD_NUMBER: _ClassVar[int]
    COMMANDS_FIELD_NUMBER: _ClassVar[int]
    SCRIPT_FIELD_NUMBER: _ClassVar[int]
    TTY_FIELD_NUMBER: _ClassVar[int]
    INPUT_DATA_FIELD_NUMBER: _ClassVar[int]
    STOP_FIELD_NUMBER: _ClassVar[int]
    WINSIZE_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_STRATEGY_FIELD_NUMBER: _ClassVar[int]
    PROJECT_FIELD_NUMBER: _ClassVar[int]
    STORE_LAST_OUTPUT_FIELD_NUMBER: _ClassVar[int]
    COMMAND_MODE_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_ID_FIELD_NUMBER: _ClassVar[int]
    FILE_EXTENSION_FIELD_NUMBER: _ClassVar[int]
    KNOWN_ID_FIELD_NUMBER: _ClassVar[int]
    KNOWN_NAME_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    program_name: str
    arguments: _containers.RepeatedScalarFieldContainer[str]
    directory: str
    envs: _containers.RepeatedScalarFieldContainer[str]
    commands: _containers.RepeatedScalarFieldContainer[str]
    script: str
    tty: bool
    input_data: bytes
    stop: ExecuteStop
    winsize: Winsize
    background: bool
    session_id: str
    session_strategy: SessionStrategy
    project: Project
    store_last_output: bool
    command_mode: CommandMode
    language_id: str
    file_extension: str
    known_id: str
    known_name: str
    run_id: str
    def __init__(self, program_name: _Optional[str] = ..., arguments: _Optional[_Iterable[str]] = ..., directory: _Optional[str] = ..., envs: _Optional[_Iterable[str]] = ..., commands: _Optional[_Iterable[str]] = ..., script: _Optional[str] = ..., tty: _Optional[bool] = ..., input_data: _Optional[bytes] = ..., stop: _Optional[_Union[ExecuteStop, str]] = ..., winsize: _Optional[_Union[Winsize, _Mapping]] = ..., background: _Optional[bool] = ..., session_id: _Optional[str] = ..., session_strategy: _Optional[_Union[SessionStrategy, str]] = ..., project: _Optional[_Union[Project, _Mapping]] = ..., store_last_output: _Optional[bool] = ..., command_mode: _Optional[_Union[CommandMode, str]] = ..., language_id: _Optional[str] = ..., file_extension: _Optional[str] = ..., known_id: _Optional[str] = ..., known_name: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class ProcessPID(_message.Message):
    __slots__ = ("pid",)
    PID_FIELD_NUMBER: _ClassVar[int]
    pid: int
    def __init__(self, pid: _Optional[int] = ...) -> None: ...

class ExecuteResponse(_message.Message):
    __slots__ = ("exit_code", "stdout_data", "stderr_data", "pid", "mime_type")
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    STDOUT_DATA_FIELD_NUMBER: _ClassVar[int]
    STDERR_DATA_FIELD_NUMBER: _ClassVar[int]
    PID_FIELD_NUMBER: _ClassVar[int]
    MIME_TYPE_FIELD_NUMBER: _ClassVar[int]
    exit_code: _wrappers_pb2.UInt32Value
    stdout_data: bytes
    stderr_data: bytes
    pid: ProcessPID
    mime_type: str
    def __init__(self, exit_code: _Optional[_Union[_wrappers_pb2.UInt32Value, _Mapping]] = ..., stdout_data: _Optional[bytes] = ..., stderr_data: _Optional[bytes] = ..., pid: _Optional[_Union[ProcessPID, _Mapping]] = ..., mime_type: _Optional[str] = ...) -> None: ...

class ResolveProgramCommandList(_message.Message):
    __slots__ = ("lines",)
    LINES_FIELD_NUMBER: _ClassVar[int]
    lines: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, lines: _Optional[_Iterable[str]] = ...) -> None: ...

class ResolveProgramRequest(_message.Message):
    __slots__ = ("commands", "script", "mode", "env", "session_id", "session_strategy", "project", "language_id", "retention")
    class Mode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        MODE_UNSPECIFIED: _ClassVar[ResolveProgramRequest.Mode]
        MODE_PROMPT_ALL: _ClassVar[ResolveProgramRequest.Mode]
        MODE_SKIP_ALL: _ClassVar[ResolveProgramRequest.Mode]
    MODE_UNSPECIFIED: ResolveProgramRequest.Mode
    MODE_PROMPT_ALL: ResolveProgramRequest.Mode
    MODE_SKIP_ALL: ResolveProgramRequest.Mode
    class Retention(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        RETENTION_UNSPECIFIED: _ClassVar[ResolveProgramRequest.Retention]
        RETENTION_FIRST_RUN: _ClassVar[ResolveProgramRequest.Retention]
        RETENTION_LAST_RUN: _ClassVar[ResolveProgramRequest.Retention]
    RETENTION_UNSPECIFIED: ResolveProgramRequest.Retention
    RETENTION_FIRST_RUN: ResolveProgramRequest.Retention
    RETENTION_LAST_RUN: ResolveProgramRequest.Retention
    COMMANDS_FIELD_NUMBER: _ClassVar[int]
    SCRIPT_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    ENV_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_STRATEGY_FIELD_NUMBER: _ClassVar[int]
    PROJECT_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_ID_FIELD_NUMBER: _ClassVar[int]
    RETENTION_FIELD_NUMBER: _ClassVar[int]
    commands: ResolveProgramCommandList
    script: str
    mode: ResolveProgramRequest.Mode
    env: _containers.RepeatedScalarFieldContainer[str]
    session_id: str
    session_strategy: SessionStrategy
    project: Project
    language_id: str
    retention: ResolveProgramRequest.Retention
    def __init__(self, commands: _Optional[_Union[ResolveProgramCommandList, _Mapping]] = ..., script: _Optional[str] = ..., mode: _Optional[_Union[ResolveProgramRequest.Mode, str]] = ..., env: _Optional[_Iterable[str]] = ..., session_id: _Optional[str] = ..., session_strategy: _Optional[_Union[SessionStrategy, str]] = ..., project: _Optional[_Union[Project, _Mapping]] = ..., language_id: _Optional[str] = ..., retention: _Optional[_Union[ResolveProgramRequest.Retention, str]] = ...) -> None: ...

class ResolveProgramResponse(_message.Message):
    __slots__ = ("script", "commands", "vars")
    class Status(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        STATUS_UNSPECIFIED: _ClassVar[ResolveProgramResponse.Status]
        STATUS_UNRESOLVED_WITH_MESSAGE: _ClassVar[ResolveProgramResponse.Status]
        STATUS_UNRESOLVED_WITH_PLACEHOLDER: _ClassVar[ResolveProgramResponse.Status]
        STATUS_RESOLVED: _ClassVar[ResolveProgramResponse.Status]
        STATUS_UNRESOLVED_WITH_SECRET: _ClassVar[ResolveProgramResponse.Status]
    STATUS_UNSPECIFIED: ResolveProgramResponse.Status
    STATUS_UNRESOLVED_WITH_MESSAGE: ResolveProgramResponse.Status
    STATUS_UNRESOLVED_WITH_PLACEHOLDER: ResolveProgramResponse.Status
    STATUS_RESOLVED: ResolveProgramResponse.Status
    STATUS_UNRESOLVED_WITH_SECRET: ResolveProgramResponse.Status
    class VarResult(_message.Message):
        __slots__ = ("status", "name", "original_value", "resolved_value")
        STATUS_FIELD_NUMBER: _ClassVar[int]
        NAME_FIELD_NUMBER: _ClassVar[int]
        ORIGINAL_VALUE_FIELD_NUMBER: _ClassVar[int]
        RESOLVED_VALUE_FIELD_NUMBER: _ClassVar[int]
        status: ResolveProgramResponse.Status
        name: str
        original_value: str
        resolved_value: str
        def __init__(self, status: _Optional[_Union[ResolveProgramResponse.Status, str]] = ..., name: _Optional[str] = ..., original_value: _Optional[str] = ..., resolved_value: _Optional[str] = ...) -> None: ...
    SCRIPT_FIELD_NUMBER: _ClassVar[int]
    COMMANDS_FIELD_NUMBER: _ClassVar[int]
    VARS_FIELD_NUMBER: _ClassVar[int]
    script: str
    commands: ResolveProgramCommandList
    vars: _containers.RepeatedCompositeFieldContainer[ResolveProgramResponse.VarResult]
    def __init__(self, script: _Optional[str] = ..., commands: _Optional[_Union[ResolveProgramCommandList, _Mapping]] = ..., vars: _Optional[_Iterable[_Union[ResolveProgramResponse.VarResult, _Mapping]]] = ...) -> None: ...

class MonitorEnvStoreRequest(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: Session
    def __init__(self, session: _Optional[_Union[Session, _Mapping]] = ...) -> None: ...

class MonitorEnvStoreResponseSnapshot(_message.Message):
    __slots__ = ("envs",)
    class Status(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        STATUS_UNSPECIFIED: _ClassVar[MonitorEnvStoreResponseSnapshot.Status]
        STATUS_LITERAL: _ClassVar[MonitorEnvStoreResponseSnapshot.Status]
        STATUS_HIDDEN: _ClassVar[MonitorEnvStoreResponseSnapshot.Status]
        STATUS_MASKED: _ClassVar[MonitorEnvStoreResponseSnapshot.Status]
    STATUS_UNSPECIFIED: MonitorEnvStoreResponseSnapshot.Status
    STATUS_LITERAL: MonitorEnvStoreResponseSnapshot.Status
    STATUS_HIDDEN: MonitorEnvStoreResponseSnapshot.Status
    STATUS_MASKED: MonitorEnvStoreResponseSnapshot.Status
    class SnapshotEnv(_message.Message):
        __slots__ = ("status", "name", "spec", "origin", "original_value", "resolved_value", "create_time", "update_time", "errors", "is_required", "description")
        STATUS_FIELD_NUMBER: _ClassVar[int]
        NAME_FIELD_NUMBER: _ClassVar[int]
        SPEC_FIELD_NUMBER: _ClassVar[int]
        ORIGIN_FIELD_NUMBER: _ClassVar[int]
        ORIGINAL_VALUE_FIELD_NUMBER: _ClassVar[int]
        RESOLVED_VALUE_FIELD_NUMBER: _ClassVar[int]
        CREATE_TIME_FIELD_NUMBER: _ClassVar[int]
        UPDATE_TIME_FIELD_NUMBER: _ClassVar[int]
        ERRORS_FIELD_NUMBER: _ClassVar[int]
        IS_REQUIRED_FIELD_NUMBER: _ClassVar[int]
        DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
        status: MonitorEnvStoreResponseSnapshot.Status
        name: str
        spec: str
        origin: str
        original_value: str
        resolved_value: str
        create_time: str
        update_time: str
        errors: _containers.RepeatedCompositeFieldContainer[MonitorEnvStoreResponseSnapshot.Error]
        is_required: bool
        description: str
        def __init__(self, status: _Optional[_Union[MonitorEnvStoreResponseSnapshot.Status, str]] = ..., name: _Optional[str] = ..., spec: _Optional[str] = ..., origin: _Optional[str] = ..., original_value: _Optional[str] = ..., resolved_value: _Optional[str] = ..., create_time: _Optional[str] = ..., update_time: _Optional[str] = ..., errors: _Optional[_Iterable[_Union[MonitorEnvStoreResponseSnapshot.Error, _Mapping]]] = ..., is_required: _Optional[bool] = ..., description: _Optional[str] = ...) -> None: ...
    class Error(_message.Message):
        __slots__ = ("code", "message")
        CODE_FIELD_NUMBER: _ClassVar[int]
        MESSAGE_FIELD_NUMBER: _ClassVar[int]
        code: int
        message: str
        def __init__(self, code: _Optional[int] = ..., message: _Optional[str] = ...) -> None: ...
    ENVS_FIELD_NUMBER: _ClassVar[int]
    envs: _containers.RepeatedCompositeFieldContainer[MonitorEnvStoreResponseSnapshot.SnapshotEnv]
    def __init__(self, envs: _Optional[_Iterable[_Union[MonitorEnvStoreResponseSnapshot.SnapshotEnv, _Mapping]]] = ...) -> None: ...

class MonitorEnvStoreResponse(_message.Message):
    __slots__ = ("type", "snapshot")
    TYPE_FIELD_NUMBER: _ClassVar[int]
    SNAPSHOT_FIELD_NUMBER: _ClassVar[int]
    type: MonitorEnvStoreType
    snapshot: MonitorEnvStoreResponseSnapshot
    def __init__(self, type: _Optional[_Union[MonitorEnvStoreType, str]] = ..., snapshot: _Optional[_Union[MonitorEnvStoreResponseSnapshot, _Mapping]] = ...) -> None: ...
