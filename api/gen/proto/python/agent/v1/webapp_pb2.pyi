from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class InitialConfigState(_message.Message):
    __slots__ = ("web_app", "agent_endpoint", "require_auth", "system_shell")
    WEB_APP_FIELD_NUMBER: _ClassVar[int]
    AGENT_ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    REQUIRE_AUTH_FIELD_NUMBER: _ClassVar[int]
    SYSTEM_SHELL_FIELD_NUMBER: _ClassVar[int]
    web_app: WebAppConfig
    agent_endpoint: str
    require_auth: bool
    system_shell: str
    def __init__(self, web_app: _Optional[_Union[WebAppConfig, _Mapping]] = ..., agent_endpoint: _Optional[str] = ..., require_auth: _Optional[bool] = ..., system_shell: _Optional[str] = ...) -> None: ...

class WebAppConfig(_message.Message):
    __slots__ = ("runner", "reconnect", "inverted_order")
    RUNNER_FIELD_NUMBER: _ClassVar[int]
    RECONNECT_FIELD_NUMBER: _ClassVar[int]
    INVERTED_ORDER_FIELD_NUMBER: _ClassVar[int]
    runner: str
    reconnect: bool
    inverted_order: bool
    def __init__(self, runner: _Optional[str] = ..., reconnect: _Optional[bool] = ..., inverted_order: _Optional[bool] = ...) -> None: ...
