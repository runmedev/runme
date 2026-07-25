from buf.validate import validate_pb2 as _validate_pb2
from google.protobuf.internal import containers as _containers
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class Assertion(_message.Message):
    __slots__ = ("name", "type", "result", "shell_required_flag", "tool_invocation", "file_retrieval", "llm_judge", "codeblock_regex", "failure_reason")
    class Type(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        TYPE_UNSPECIFIED: _ClassVar[Assertion.Type]
        TYPE_SHELL_REQUIRED_FLAG: _ClassVar[Assertion.Type]
        TYPE_TOOL_INVOKED: _ClassVar[Assertion.Type]
        TYPE_FILE_RETRIEVED: _ClassVar[Assertion.Type]
        TYPE_LLM_JUDGE: _ClassVar[Assertion.Type]
        TYPE_CODEBLOCK_REGEX: _ClassVar[Assertion.Type]
    TYPE_UNSPECIFIED: Assertion.Type
    TYPE_SHELL_REQUIRED_FLAG: Assertion.Type
    TYPE_TOOL_INVOKED: Assertion.Type
    TYPE_FILE_RETRIEVED: Assertion.Type
    TYPE_LLM_JUDGE: Assertion.Type
    TYPE_CODEBLOCK_REGEX: Assertion.Type
    class Result(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
        __slots__ = ()
        RESULT_UNSPECIFIED: _ClassVar[Assertion.Result]
        RESULT_TRUE: _ClassVar[Assertion.Result]
        RESULT_FALSE: _ClassVar[Assertion.Result]
        RESULT_SKIPPED: _ClassVar[Assertion.Result]
    RESULT_UNSPECIFIED: Assertion.Result
    RESULT_TRUE: Assertion.Result
    RESULT_FALSE: Assertion.Result
    RESULT_SKIPPED: Assertion.Result
    class ShellRequiredFlag(_message.Message):
        __slots__ = ("command", "flags")
        COMMAND_FIELD_NUMBER: _ClassVar[int]
        FLAGS_FIELD_NUMBER: _ClassVar[int]
        command: str
        flags: _containers.RepeatedScalarFieldContainer[str]
        def __init__(self, command: _Optional[str] = ..., flags: _Optional[_Iterable[str]] = ...) -> None: ...
    class ToolInvocation(_message.Message):
        __slots__ = ("tool_name",)
        TOOL_NAME_FIELD_NUMBER: _ClassVar[int]
        tool_name: str
        def __init__(self, tool_name: _Optional[str] = ...) -> None: ...
    class FileRetrieval(_message.Message):
        __slots__ = ("file_id", "file_name")
        FILE_ID_FIELD_NUMBER: _ClassVar[int]
        FILE_NAME_FIELD_NUMBER: _ClassVar[int]
        file_id: str
        file_name: str
        def __init__(self, file_id: _Optional[str] = ..., file_name: _Optional[str] = ...) -> None: ...
    class LLMJudge(_message.Message):
        __slots__ = ("prompt",)
        PROMPT_FIELD_NUMBER: _ClassVar[int]
        prompt: str
        def __init__(self, prompt: _Optional[str] = ...) -> None: ...
    class CodeblockRegex(_message.Message):
        __slots__ = ("regex",)
        REGEX_FIELD_NUMBER: _ClassVar[int]
        regex: str
        def __init__(self, regex: _Optional[str] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    TYPE_FIELD_NUMBER: _ClassVar[int]
    RESULT_FIELD_NUMBER: _ClassVar[int]
    SHELL_REQUIRED_FLAG_FIELD_NUMBER: _ClassVar[int]
    TOOL_INVOCATION_FIELD_NUMBER: _ClassVar[int]
    FILE_RETRIEVAL_FIELD_NUMBER: _ClassVar[int]
    LLM_JUDGE_FIELD_NUMBER: _ClassVar[int]
    CODEBLOCK_REGEX_FIELD_NUMBER: _ClassVar[int]
    FAILURE_REASON_FIELD_NUMBER: _ClassVar[int]
    name: str
    type: Assertion.Type
    result: Assertion.Result
    shell_required_flag: Assertion.ShellRequiredFlag
    tool_invocation: Assertion.ToolInvocation
    file_retrieval: Assertion.FileRetrieval
    llm_judge: Assertion.LLMJudge
    codeblock_regex: Assertion.CodeblockRegex
    failure_reason: str
    def __init__(self, name: _Optional[str] = ..., type: _Optional[_Union[Assertion.Type, str]] = ..., result: _Optional[_Union[Assertion.Result, str]] = ..., shell_required_flag: _Optional[_Union[Assertion.ShellRequiredFlag, _Mapping]] = ..., tool_invocation: _Optional[_Union[Assertion.ToolInvocation, _Mapping]] = ..., file_retrieval: _Optional[_Union[Assertion.FileRetrieval, _Mapping]] = ..., llm_judge: _Optional[_Union[Assertion.LLMJudge, _Mapping]] = ..., codeblock_regex: _Optional[_Union[Assertion.CodeblockRegex, _Mapping]] = ..., failure_reason: _Optional[str] = ...) -> None: ...

class EvalSample(_message.Message):
    __slots__ = ("kind", "metadata", "input_text", "assertions")
    KIND_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    INPUT_TEXT_FIELD_NUMBER: _ClassVar[int]
    ASSERTIONS_FIELD_NUMBER: _ClassVar[int]
    kind: str
    metadata: ObjectMeta
    input_text: str
    assertions: _containers.RepeatedCompositeFieldContainer[Assertion]
    def __init__(self, kind: _Optional[str] = ..., metadata: _Optional[_Union[ObjectMeta, _Mapping]] = ..., input_text: _Optional[str] = ..., assertions: _Optional[_Iterable[_Union[Assertion, _Mapping]]] = ...) -> None: ...

class EvalDataset(_message.Message):
    __slots__ = ("samples",)
    SAMPLES_FIELD_NUMBER: _ClassVar[int]
    samples: _containers.RepeatedCompositeFieldContainer[EvalSample]
    def __init__(self, samples: _Optional[_Iterable[_Union[EvalSample, _Mapping]]] = ...) -> None: ...

class ObjectMeta(_message.Message):
    __slots__ = ("name",)
    NAME_FIELD_NUMBER: _ClassVar[int]
    name: str
    def __init__(self, name: _Optional[str] = ...) -> None: ...

class ExperimentSpec(_message.Message):
    __slots__ = ("dataset_path", "output_dir", "inference_endpoint")
    DATASET_PATH_FIELD_NUMBER: _ClassVar[int]
    OUTPUT_DIR_FIELD_NUMBER: _ClassVar[int]
    INFERENCE_ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    dataset_path: str
    output_dir: str
    inference_endpoint: str
    def __init__(self, dataset_path: _Optional[str] = ..., output_dir: _Optional[str] = ..., inference_endpoint: _Optional[str] = ...) -> None: ...

class Experiment(_message.Message):
    __slots__ = ("api_version", "kind", "metadata", "spec")
    API_VERSION_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    METADATA_FIELD_NUMBER: _ClassVar[int]
    SPEC_FIELD_NUMBER: _ClassVar[int]
    api_version: str
    kind: str
    metadata: ObjectMeta
    spec: ExperimentSpec
    def __init__(self, api_version: _Optional[str] = ..., kind: _Optional[str] = ..., metadata: _Optional[_Union[ObjectMeta, _Mapping]] = ..., spec: _Optional[_Union[ExperimentSpec, _Mapping]] = ...) -> None: ...
