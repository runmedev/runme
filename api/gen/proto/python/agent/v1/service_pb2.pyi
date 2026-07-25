from agent.tools.v1 import notebooks_pb2 as _notebooks_pb2
from runme.parser.v1 import parser_pb2 as _parser_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GenerateRequest(_message.Message):
    __slots__ = ("cells", "previous_response_id", "openai_access_token", "model", "context", "kernels", "container", "message", "tool_call_outputs", "notebook_path")
    class Context(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        CONTEXT_UNSPECIFIED: _ClassVar[GenerateRequest.Context]
        CONTEXT_WEBAPP: _ClassVar[GenerateRequest.Context]
        CONTEXT_SLACK: _ClassVar[GenerateRequest.Context]
        CONTEXT_ALERT_TRIGGERED: _ClassVar[GenerateRequest.Context]
    CONTEXT_UNSPECIFIED: GenerateRequest.Context
    CONTEXT_WEBAPP: GenerateRequest.Context
    CONTEXT_SLACK: GenerateRequest.Context
    CONTEXT_ALERT_TRIGGERED: GenerateRequest.Context
    CELLS_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_RESPONSE_ID_FIELD_NUMBER: _ClassVar[int]
    OPENAI_ACCESS_TOKEN_FIELD_NUMBER: _ClassVar[int]
    MODEL_FIELD_NUMBER: _ClassVar[int]
    CONTEXT_FIELD_NUMBER: _ClassVar[int]
    KERNELS_FIELD_NUMBER: _ClassVar[int]
    CONTAINER_FIELD_NUMBER: _ClassVar[int]
    MESSAGE_FIELD_NUMBER: _ClassVar[int]
    TOOL_CALL_OUTPUTS_FIELD_NUMBER: _ClassVar[int]
    NOTEBOOK_PATH_FIELD_NUMBER: _ClassVar[int]
    cells: _containers.RepeatedCompositeFieldContainer[_parser_pb2.Cell]
    previous_response_id: str
    openai_access_token: str
    model: str
    context: GenerateRequest.Context
    kernels: _containers.RepeatedScalarFieldContainer[str]
    container: str
    message: str
    tool_call_outputs: _containers.RepeatedCompositeFieldContainer[_notebooks_pb2.ToolCallOutput]
    notebook_path: str
    def __init__(self, cells: _Optional[_Iterable[_Union[_parser_pb2.Cell, _Mapping]]] = ..., previous_response_id: _Optional[str] = ..., openai_access_token: _Optional[str] = ..., model: _Optional[str] = ..., context: _Optional[_Union[GenerateRequest.Context, str]] = ..., kernels: _Optional[_Iterable[str]] = ..., container: _Optional[str] = ..., message: _Optional[str] = ..., tool_call_outputs: _Optional[_Iterable[_Union[_notebooks_pb2.ToolCallOutput, _Mapping]]] = ..., notebook_path: _Optional[str] = ...) -> None: ...

class GenerateResponse(_message.Message):
    __slots__ = ("cells", "response_id")
    CELLS_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_ID_FIELD_NUMBER: _ClassVar[int]
    cells: _containers.RepeatedCompositeFieldContainer[_parser_pb2.Cell]
    response_id: str
    def __init__(self, cells: _Optional[_Iterable[_Union[_parser_pb2.Cell, _Mapping]]] = ..., response_id: _Optional[str] = ...) -> None: ...

class LogRequest(_message.Message):
    __slots__ = ("notebook",)
    NOTEBOOK_FIELD_NUMBER: _ClassVar[int]
    notebook: _parser_pb2.Notebook
    def __init__(self, notebook: _Optional[_Union[_parser_pb2.Notebook, _Mapping]] = ...) -> None: ...

class LogResponse(_message.Message):
    __slots__ = ()
    def __init__(self) -> None: ...
