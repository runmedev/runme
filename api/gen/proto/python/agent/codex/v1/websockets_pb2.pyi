from agent.tools.v1 import notebooks_pb2 as _notebooks_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class NotebookToolCallRequest(_message.Message):
    __slots__ = ("bridge_call_id", "input")
    BRIDGE_CALL_ID_FIELD_NUMBER: _ClassVar[int]
    INPUT_FIELD_NUMBER: _ClassVar[int]
    bridge_call_id: str
    input: _notebooks_pb2.ToolCallInput
    def __init__(self, bridge_call_id: _Optional[str] = ..., input: _Optional[_Union[_notebooks_pb2.ToolCallInput, _Mapping]] = ...) -> None: ...

class NotebookToolCallResponse(_message.Message):
    __slots__ = ("bridge_call_id", "output", "error")
    BRIDGE_CALL_ID_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_FIELD_NUMBER: _ClassVar[int]
    ERROR_FIELD_NUMBER: _ClassVar[int]
    bridge_call_id: str
    output: _notebooks_pb2.ToolCallOutput
    error: str
    def __init__(self, bridge_call_id: _Optional[str] = ..., output: _Optional[_Union[_notebooks_pb2.ToolCallOutput, _Mapping]] = ..., error: _Optional[str] = ...) -> None: ...

class WebsocketRequest(_message.Message):
    __slots__ = ("notebook_tool_call_response",)
    NOTEBOOK_TOOL_CALL_RESPONSE_FIELD_NUMBER: _ClassVar[int]
    notebook_tool_call_response: NotebookToolCallResponse
    def __init__(self, notebook_tool_call_response: _Optional[_Union[NotebookToolCallResponse, _Mapping]] = ...) -> None: ...

class WebsocketResponse(_message.Message):
    __slots__ = ("notebook_tool_call_request",)
    NOTEBOOK_TOOL_CALL_REQUEST_FIELD_NUMBER: _ClassVar[int]
    notebook_tool_call_request: NotebookToolCallRequest
    def __init__(self, notebook_tool_call_request: _Optional[_Union[NotebookToolCallRequest, _Mapping]] = ...) -> None: ...
