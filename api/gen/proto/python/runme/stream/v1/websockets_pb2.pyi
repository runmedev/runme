from google.rpc import code_pb2 as _code_pb2
from runme.runner.v2 import runner_pb2 as _runner_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

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

class WebsocketRequest(_message.Message):
    __slots__ = ("execute_request", "ping", "authorization", "known_id", "run_id")
    EXECUTE_REQUEST_FIELD_NUMBER: _ClassVar[int]
    PING_FIELD_NUMBER: _ClassVar[int]
    AUTHORIZATION_FIELD_NUMBER: _ClassVar[int]
    KNOWN_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    execute_request: _runner_pb2.ExecuteRequest
    ping: Ping
    authorization: str
    known_id: str
    run_id: str
    def __init__(self, execute_request: _Optional[_Union[_runner_pb2.ExecuteRequest, _Mapping]] = ..., ping: _Optional[_Union[Ping, _Mapping]] = ..., authorization: _Optional[str] = ..., known_id: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...

class WebsocketResponse(_message.Message):
    __slots__ = ("execute_response", "pong", "status", "known_id", "run_id")
    EXECUTE_RESPONSE_FIELD_NUMBER: _ClassVar[int]
    PONG_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    KNOWN_ID_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    execute_response: _runner_pb2.ExecuteResponse
    pong: Pong
    status: WebsocketStatus
    known_id: str
    run_id: str
    def __init__(self, execute_response: _Optional[_Union[_runner_pb2.ExecuteResponse, _Mapping]] = ..., pong: _Optional[_Union[Pong, _Mapping]] = ..., status: _Optional[_Union[WebsocketStatus, _Mapping]] = ..., known_id: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...
