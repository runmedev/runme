from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ExecuteCodeRequest(_message.Message):
    __slots__ = ("code",)
    CODE_FIELD_NUMBER: _ClassVar[int]
    code: str
    def __init__(self, code: _Optional[str] = ...) -> None: ...

class ExecuteCodeResponse(_message.Message):
    __slots__ = ("output",)
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    output: str
    def __init__(self, output: _Optional[str] = ...) -> None: ...

class TerminateRunRequest(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class TerminateRunResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...

class ToolCallInput(_message.Message):
    __slots__ = ("call_id", "previous_response_id", "execute_code")
    CALL_ID_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_RESPONSE_ID_FIELD_NUMBER: _ClassVar[int]
    EXECUTE_CODE_FIELD_NUMBER: _ClassVar[int]
    call_id: str
    previous_response_id: str
    execute_code: ExecuteCodeRequest
    def __init__(self, call_id: _Optional[str] = ..., previous_response_id: _Optional[str] = ..., execute_code: _Optional[_Union[ExecuteCodeRequest, _Mapping]] = ...) -> None: ...

class ToolCallOutput(_message.Message):
    __slots__ = ("call_id", "previous_response_id", "execute_code", "status", "client_error")
    class Status(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        STATUS_UNSPECIFIED: _ClassVar[ToolCallOutput.Status]
        STATUS_SUCCESS: _ClassVar[ToolCallOutput.Status]
        STATUS_FAILED: _ClassVar[ToolCallOutput.Status]
    STATUS_UNSPECIFIED: ToolCallOutput.Status
    STATUS_SUCCESS: ToolCallOutput.Status
    STATUS_FAILED: ToolCallOutput.Status
    CALL_ID_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_RESPONSE_ID_FIELD_NUMBER: _ClassVar[int]
    EXECUTE_CODE_FIELD_NUMBER: _ClassVar[int]
    STATUS_FIELD_NUMBER: _ClassVar[int]
    CLIENT_ERROR_FIELD_NUMBER: _ClassVar[int]
    call_id: str
    previous_response_id: str
    execute_code: ExecuteCodeResponse
    status: ToolCallOutput.Status
    client_error: str
    def __init__(self, call_id: _Optional[str] = ..., previous_response_id: _Optional[str] = ..., execute_code: _Optional[_Union[ExecuteCodeResponse, _Mapping]] = ..., status: _Optional[_Union[ToolCallOutput.Status, str]] = ..., client_error: _Optional[str] = ...) -> None: ...

class ChatkitState(_message.Message):
    __slots__ = ("previous_response_id", "thread_id")
    PREVIOUS_RESPONSE_ID_FIELD_NUMBER: _ClassVar[int]
    THREAD_ID_FIELD_NUMBER: _ClassVar[int]
    previous_response_id: str
    thread_id: str
    def __init__(self, previous_response_id: _Optional[str] = ..., thread_id: _Optional[str] = ...) -> None: ...

class SendSlackMessageRequest(_message.Message):
    __slots__ = ("channel", "timestamp", "text", "file_ids")
    CHANNEL_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    TEXT_FIELD_NUMBER: _ClassVar[int]
    FILE_IDS_FIELD_NUMBER: _ClassVar[int]
    channel: str
    timestamp: str
    text: str
    file_ids: _containers.RepeatedScalarFieldContainer[str]
    def __init__(self, channel: _Optional[str] = ..., timestamp: _Optional[str] = ..., text: _Optional[str] = ..., file_ids: _Optional[_Iterable[str]] = ...) -> None: ...

class SendSlackMessageResponse(_message.Message):
    __slots__ = ("error",)
    ERROR_FIELD_NUMBER: _ClassVar[int]
    error: str
    def __init__(self, error: _Optional[str] = ...) -> None: ...
