from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class CommandMode(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    COMMAND_MODE_UNSPECIFIED: _ClassVar[CommandMode]
    COMMAND_MODE_INLINE: _ClassVar[CommandMode]
    COMMAND_MODE_FILE: _ClassVar[CommandMode]
    COMMAND_MODE_TERMINAL: _ClassVar[CommandMode]
    COMMAND_MODE_CLI: _ClassVar[CommandMode]
COMMAND_MODE_UNSPECIFIED: CommandMode
COMMAND_MODE_INLINE: CommandMode
COMMAND_MODE_FILE: CommandMode
COMMAND_MODE_TERMINAL: CommandMode
COMMAND_MODE_CLI: CommandMode

class ProgramConfig(_message.Message):
    __slots__ = ("program_name", "arguments", "directory", "language_id", "background", "file_extension", "env", "commands", "script", "interactive", "mode", "known_id", "known_name", "run_id")
    class CommandList(_message.Message):
        __slots__ = ("items",)
        ITEMS_FIELD_NUMBER: _ClassVar[int]
        items: _containers.RepeatedScalarFieldContainer[str]
        def __init__(self, items: _Optional[_Iterable[str]] = ...) -> None: ...
    PROGRAM_NAME_FIELD_NUMBER: _ClassVar[int]
    ARGUMENTS_FIELD_NUMBER: _ClassVar[int]
    DIRECTORY_FIELD_NUMBER: _ClassVar[int]
    LANGUAGE_ID_FIELD_NUMBER: _ClassVar[int]
    BACKGROUND_FIELD_NUMBER: _ClassVar[int]
    FILE_EXTENSION_FIELD_NUMBER: _ClassVar[int]
    ENV_FIELD_NUMBER: _ClassVar[int]
    COMMANDS_FIELD_NUMBER: _ClassVar[int]
    SCRIPT_FIELD_NUMBER: _ClassVar[int]
    INTERACTIVE_FIELD_NUMBER: _ClassVar[int]
    MODE_FIELD_NUMBER: _ClassVar[int]
    KNOWN_ID_FIELD_NUMBER: _ClassVar[int]
    KNOWN_NAME_FIELD_NUMBER: _ClassVar[int]
    RUN_ID_FIELD_NUMBER: _ClassVar[int]
    program_name: str
    arguments: _containers.RepeatedScalarFieldContainer[str]
    directory: str
    language_id: str
    background: bool
    file_extension: str
    env: _containers.RepeatedScalarFieldContainer[str]
    commands: ProgramConfig.CommandList
    script: str
    interactive: bool
    mode: CommandMode
    known_id: str
    known_name: str
    run_id: str
    def __init__(self, program_name: _Optional[str] = ..., arguments: _Optional[_Iterable[str]] = ..., directory: _Optional[str] = ..., language_id: _Optional[str] = ..., background: _Optional[bool] = ..., file_extension: _Optional[str] = ..., env: _Optional[_Iterable[str]] = ..., commands: _Optional[_Union[ProgramConfig.CommandList, _Mapping]] = ..., script: _Optional[str] = ..., interactive: _Optional[bool] = ..., mode: _Optional[_Union[CommandMode, str]] = ..., known_id: _Optional[str] = ..., known_name: _Optional[str] = ..., run_id: _Optional[str] = ...) -> None: ...
