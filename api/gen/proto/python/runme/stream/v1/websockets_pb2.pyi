from google.rpc import code_pb2 as _code_pb2
from runme.runner.v2 import runner_pb2 as _runner_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class RunIntent(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RUN_INTENT_UNSPECIFIED: _ClassVar[RunIntent]
    RUN_INTENT_START: _ClassVar[RunIntent]
    RUN_INTENT_RESUME: _ClassVar[RunIntent]

class RunState(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    RUN_STATE_UNSPECIFIED: _ClassVar[RunState]
    RUN_STATE_CREATED: _ClassVar[RunState]
    RUN_STATE_RUNNING: _ClassVar[RunState]
RUN_INTENT_UNSPECIFIED: RunIntent
RUN_INTENT_START: RunIntent
RUN_INTENT_RESUME: RunIntent
RUN_STATE_UNSPECIFIED: RunState
RUN_STATE_CREATED: RunState
RUN_STATE_RUNNING: RunState

class WebsocketStatus(_message.Message):
    __slots__ = ("code", "message")
    CODE_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    code: _code_pb2.Code
    message: str
    def __init__(self, code: _Optional[_Union[_code_pb2.Code, str]] = ..., message: _Optional[str] = ...) -> None: ...

class Ping(_message.Message):
    __slots__ = ("timestamp",)
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    timestamp: int
    def __init__(self, timestamp: _Optional[int] = ...) -> None: ...

class Pong(_message.Message):
    __slots__ = ("timestamp",)
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    timestamp: int
    def __init__(self, timestamp: _Optional[int] = ...) -> None: ...

class OpenRunRequest(_message.Message):
    __slots__ = ("intent",)
    INTENT_FIELD_NUMBER: _ClassVar[int]
    intent: RunIntent
    def __init__(self, intent: _Optional[_Union[RunIntent, str]] = ...) -> None: ...

class OpenRunResponse(_message.Message):
    __slots__ = ("state",)
    STATE_FIELD_NUMBER: _ClassVar[int]
    state: RunState
    def __init__(self, state: _Optional[_Union[RunState, str]] = ...) -> None: ...

class WebsocketRequest(_message.Message):
    __slots__ = ("execute_request", "open_run_request", "ping", "authorization", "known_id", "run_id")
    EXECUTE_REQUEST_FIELD_NUMBER: _ClassVar[int]
    OPEN_RUN_REQUEST_FIELD_NUMBER: _ClassVar[int]
    PING_FIELD_NUMBER: _ClassVar[int]
    AUTHORIZATION_FIELD_NUMBER: _ClassVar[int]
    KNOWN_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    execute_request: _runner_pb2.ExecuteRequest
    open_run_request: OpenRunRequest
    ping: Ping
    authorization: str
    known_id: str
    run_id: str
    def __init__(self, execute_request: _Optional[_Union[_runner_pb2.ExecuteRequest, _Mapping]] = ..., open_run_request: _Optional[_Union[OpenRunRequest, _Mapping]] = ..., ping: _Optional[_Union[Ping, _Mapping]] = ..., authorization: _Optional[str] = ..., known_id: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class WebsocketResponse(_message.Message):
    __slots__ = ("execute_response", "open_run_response", "pong", "status", "known_id", "run_id")
    EXECUTE_RESPONSE_FIELD_NUMBER: _ClassVar[int]
    OPEN_RUN_RESPONSE_FIELD_NUMBER: _ClassVar[int]
    PONG_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    KNOWN_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    execute_response: _runner_pb2.ExecuteResponse
    open_run_response: OpenRunResponse
    pong: Pong
    status: WebsocketStatus
    known_id: str
    run_id: str
    def __init__(self, execute_response: _Optional[_Union[_runner_pb2.ExecuteResponse, _Mapping]] = ..., open_run_response: _Optional[_Union[OpenRunResponse, _Mapping]] = ..., pong: _Optional[_Union[Pong, _Mapping]] = ..., status: _Optional[_Union[WebsocketStatus, _Mapping]] = ..., known_id: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...
