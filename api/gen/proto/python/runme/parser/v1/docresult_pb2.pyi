from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from typing import ClassVar as _ClassVar, Optional as _Optional

DESCRIPTOR: _descriptor.FileDescriptor

class DocResult(_message.Message):
    __slots__ = ("file_id", "file_name", "score", "link")
    FILE_ID_FIELD_NUMBER: _ClassVar[int]
    FILE_NAME_FIELD_NUMBER: _ClassVar[int]
    SCORE_FIELD_NUMBER: _ClassVar[int]
    LINK_FIELD_NUMBER: _ClassVar[int]
    file_id: str
    file_name: str
    score: float
    link: str
    def __init__(self, file_id: _Optional[str] = ..., file_name: _Optional[str] = ..., score: _Optional[float] = ..., link: _Optional[str] = ...) -> None: ...
