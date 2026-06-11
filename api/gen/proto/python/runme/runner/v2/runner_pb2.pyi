from google.protobuf import wrappers_pb2 as _wrappers_pb2
from runme.runner.v2 import config_pb2 as _config_pb2
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
SESSION_STRATEGY_UNSPECIFIED: SessionStrategy
SESSION_STRATEGY_MOST_RECENT: SessionStrategy
MONITOR_ENV_STORE_TYPE_UNSPECIFIED: MonitorEnvStoreType
MONITOR_ENV_STORE_TYPE_SNAPSHOT: MonitorEnvStoreType

class Project(_message.Message):
    __slots__ = ("root", "env_load_order")
    ROOT_FIELD_NUMBER: _ClassVar[int]
    ENV_LOAD_ORDER_FIELD_NUMBER: _ClassVar[int]
    root: str
    env_load_order: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, root: _Optional[str] = ..., env_load_order: _Optional[_Iterable[str]] = ...) -> None: ...

class Session(_message.Message):
    __slots__ = ("id", "env", "metadata")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    ENV_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    id: str
    env: _containers.RepeatedScalarFieldContainer[str]
    metadata: _containers.ScalarMap[str, str]
    def __init__(self, id: _Optional[str] = ..., env: _Optional[_Iterable[str]] = ..., metadata: _Optional[_Mapping[str, str]] = ...) -> None: ...

class CreateSessionRequest(_message.Message):
    __slots__ = ("metadata", "env", "project", "env_store_type", "config")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class Config(_message.Message):
        __slots__ = ("env_store_type", "env_store_seeding")
        class SessionEnvStoreSeeding(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
            __slots__ = ()
            SESSION_ENV_STORE_SEEDING_UNSPECIFIED: _ClassVar[CreateSessionRequest.Config.SessionEnvStoreSeeding]
            SESSION_ENV_STORE_SEEDING_SYSTEM: _ClassVar[CreateSessionRequest.Config.SessionEnvStoreSeeding]
        SESSION_ENV_STORE_SEEDING_UNSPECIFIED: CreateSessionRequest.Config.SessionEnvStoreSeeding
        SESSION_ENV_STORE_SEEDING_SYSTEM: CreateSessionRequest.Config.SessionEnvStoreSeeding
        ENV_STORE_TYPE_FIELD_NUMBER: _ClassVar[int]
        ENV_STORE_SEEDING_FIELD_NUMBER: _ClassVar[int]
        env_store_type: SessionEnvStoreType
        env_store_seeding: CreateSessionRequest.Config.SessionEnvStoreSeeding
        def __init__(self, env_store_type: _Optional[_Union[SessionEnvStoreType, str]] = ..., env_store_seeding: _Optional[_Union[CreateSessionRequest.Config.SessionEnvStoreSeeding, str]] = ...) -> None: ...
    METADATA_FIELD_NUMBER: _ClassVar[int]
    ENV_FIELD_NUMBER: _ClassVar[int]
    PROJECT_FIELD_NUMBER: _ClassVar[int]
    ENV_STORE_TYPE_FIELD_NUMBER: _ClassVar[int]
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    metadata: _containers.ScalarMap[str, str]
    env: _containers.RepeatedScalarFieldContainer[str]
    project: Project
    env_store_type: SessionEnvStoreType
    config: CreateSessionRequest.Config
    def __init__(self, metadata: _Optional[_Mapping[str, str]] = ..., env: _Optional[_Iterable[str]] = ..., project: _Optional[_Union[Project, _Mapping]] = ..., env_store_type: _Optional[_Union[SessionEnvStoreType, str]] = ..., config: _Optional[_Union[CreateSessionRequest.Config, _Mapping]] = ...) -> None: ...

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

class UpdateSessionRequest(_message.Message):
    __slots__ = ("id", "metadata", "env", "project")
    class MetadataEntry(_message.Message):
        __slots__ = ("key", "value")
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    ENV_FIELD_NUMBER: _ClassVar[int]
    PROJECT_FIELD_NUMBER: _ClassVar[int]
    id: str
    metadata: _containers.ScalarMap[str, str]
    env: _containers.RepeatedScalarFieldContainer[str]
    project: Project
    def __init__(self, id: _Optional[str] = ..., metadata: _Optional[_Mapping[str, str]] = ..., env: _Optional[_Iterable[str]] = ..., project: _Optional[_Union[Project, _Mapping]] = ...) -> None: ...

class UpdateSessionResponse(_message.Message):
    __slots__ = ("session",)
    SESSION_FIELD_NUMBER: _ClassVar[int]
    session: Session
    def __init__(self, session: _Optional[_Union[Session, _Mapping]] = ...) -> None: ...

class DeleteSessionRequest(_message.Message):
    __slots__ = ("id",)
    ID_FIELD_NUMBER: _ClassVar[int]
    id: str
    def __init__(self, id: _Optional[str] = ...) -> None: ...

class DeleteSessionResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

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
    __slots__ = ("config", "input_data", "stop", "winsize", "session_id", "session_strategy", "project", "store_stdout_in_env")
    CONFIG_FIELD_NUMBER: _ClassVar[int]
    INPUT_DATA_FIELD_NUMBER: _ClassVar[int]
    STOP_FIELD_NUMBER: _ClassVar[int]
    WINSIZE_FIELD_NUMBER: _ClassVar[int]
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_STRATEGY_FIELD_NUMBER: _ClassVar[int]
    PROJECT_FIELD_NUMBER: _ClassVar[int]
    STORE_STDOUT_IN_ENV_FIELD_NUMBER: _ClassVar[int]
    config: _config_pb2.ProgramConfig
    input_data: bytes
    stop: ExecuteStop
    winsize: Winsize
    session_id: str
    session_strategy: SessionStrategy
    project: Project
    store_stdout_in_env: bool
    def __init__(self, config: _Optional[_Union[_config_pb2.ProgramConfig, _Mapping]] = ..., input_data: _Optional[bytes] = ..., stop: _Optional[_Union[ExecuteStop, str]] = ..., winsize: _Optional[_Union[Winsize, _Mapping]] = ..., session_id: _Optional[str] = ..., session_strategy: _Optional[_Union[SessionStrategy, str]] = ..., project: _Optional[_Union[Project, _Mapping]] = ..., store_stdout_in_env: _Optional[bool] = ...) -> None: ...

class ExecuteResponse(_message.Message):
    __slots__ = ("exit_code", "stdout_data", "stderr_data", "pid", "mime_type", "pwd")
    class Pwd(_message.Message):
        __slots__ = ("current", "previous")
        CURRENT_FIELD_NUMBER: _ClassVar[int]
        PREVIOUS_FIELD_NUMBER: _ClassVar[int]
        current: str
        previous: str
        def __init__(self, current: _Optional[str] = ..., previous: _Optional[str] = ...) -> None: ...
    EXIT_CODE_FIELD_NUMBER: _ClassVar[int]
    STDOUT_DATA_FIELD_NUMBER: _ClassVar[int]
    STDERR_DATA_FIELD_NUMBER: _ClassVar[int]
    PID_FIELD_NUMBER: _ClassVar[int]
    MIME_TYPE_FIELD_NUMBER: _ClassVar[int]
    PWD_FIELD_NUMBER: _ClassVar[int]
    exit_code: _wrappers_pb2.UInt32Value
    stdout_data: bytes
    stderr_data: bytes
    pid: _wrappers_pb2.UInt32Value
    mime_type: str
    pwd: ExecuteResponse.Pwd
    def __init__(self, exit_code: _Optional[_Union[_wrappers_pb2.UInt32Value, _Mapping]] = ..., stdout_data: _Optional[bytes] = ..., stderr_data: _Optional[bytes] = ..., pid: _Optional[_Union[_wrappers_pb2.UInt32Value, _Mapping]] = ..., mime_type: _Optional[str] = ..., pwd: _Optional[_Union[ExecuteResponse.Pwd, _Mapping]] = ...) -> None: ...

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
        STATUS_RESOLVED: _ClassVar[ResolveProgramResponse.Status]
        STATUS_UNRESOLVED_WITH_MESSAGE: _ClassVar[ResolveProgramResponse.Status]
        STATUS_UNRESOLVED_WITH_PLACEHOLDER: _ClassVar[ResolveProgramResponse.Status]
        STATUS_UNRESOLVED_WITH_SECRET: _ClassVar[ResolveProgramResponse.Status]
    STATUS_UNSPECIFIED: ResolveProgramResponse.Status
    STATUS_RESOLVED: ResolveProgramResponse.Status
    STATUS_UNRESOLVED_WITH_MESSAGE: ResolveProgramResponse.Status
    STATUS_UNRESOLVED_WITH_PLACEHOLDER: ResolveProgramResponse.Status
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
        __slots__ = ("status", "name", "description", "spec", "is_required", "origin", "original_value", "resolved_value", "create_time", "update_time", "errors")
        STATUS_FIELD_NUMBER: _ClassVar[int]
        NAME_FIELD_NUMBER: _ClassVar[int]
        DESCRIPTION_FIELD_NUMBER: _ClassVar[int]
        SPEC_FIELD_NUMBER: _ClassVar[int]
        IS_REQUIRED_FIELD_NUMBER: _ClassVar[int]
        ORIGIN_FIELD_NUMBER: _ClassVar[int]
        ORIGINAL_VALUE_FIELD_NUMBER: _ClassVar[int]
        RESOLVED_VALUE_FIELD_NUMBER: _ClassVar[int]
        CREATE_TIME_FIELD_NUMBER: _ClassVar[int]
        UPDATE_TIME_FIELD_NUMBER: _ClassVar[int]
        ERRORS_FIELD_NUMBER: _ClassVar[int]
        status: MonitorEnvStoreResponseSnapshot.Status
        name: str
        description: str
        spec: str
        is_required: bool
        origin: str
        original_value: str
        resolved_value: str
        create_time: str
        update_time: str
        errors: _containers.RepeatedCompositeFieldContainer[MonitorEnvStoreResponseSnapshot.Error]
        def __init__(self, status: _Optional[_Union[MonitorEnvStoreResponseSnapshot.Status, str]] = ..., name: _Optional[str] = ..., description: _Optional[str] = ..., spec: _Optional[str] = ..., is_required: _Optional[bool] = ..., origin: _Optional[str] = ..., original_value: _Optional[str] = ..., resolved_value: _Optional[str] = ..., create_time: _Optional[str] = ..., update_time: _Optional[str] = ..., errors: _Optional[_Iterable[_Union[MonitorEnvStoreResponseSnapshot.Error, _Mapping]]] = ...) -> None: ...
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
