from google.protobuf import wrappers_pb2 as _wrappers_pb2
from runme.parser.v1 import parser_pb2 as _parser_pb2
from runme.runner.v1 import runner_pb2 as _runner_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ResolveNotebookRequest(_message.Message):
    __slots__ = ("notebook", "command_mode", "cell_index")
    NOTEBOOK_FIELD_NUMBER: _ClassVar[int]
    COMMAND_MODE_FIELD_NUMBER: _ClassVar[int]
    CELL_INDEX_FIELD_NUMBER: _ClassVar[int]
    notebook: _parser_pb2.Notebook
    command_mode: _runner_pb2.CommandMode
    cell_index: _wrappers_pb2.UInt32Value
    def __init__(self, notebook: _Optional[_Union[_parser_pb2.Notebook, _Mapping]] = ..., command_mode: _Optional[_Union[_runner_pb2.CommandMode, str]] = ..., cell_index: _Optional[_Union[_wrappers_pb2.UInt32Value, _Mapping]] = ...) -> None: ...

class ResolveNotebookResponse(_message.Message):
    __slots__ = ("script",)
    SCRIPT_FIELD_NUMBER: _ClassVar[int]
    script: str
    def __init__(self, script: _Optional[str] = ...) -> None: ...
