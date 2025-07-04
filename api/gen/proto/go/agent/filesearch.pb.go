// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        (unknown)
// source: agent/filesearch.proto

package agent

import (
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type FileSearchResult struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// The unique ID of the file.
	FileID string `protobuf:"bytes,1,opt,name=FileID,proto3" json:"FileID,omitempty"`
	// The name of the file.
	FileName string `protobuf:"bytes,2,opt,name=FileName,proto3" json:"FileName,omitempty"`
	// The relevance score of the file.
	Score float64 `protobuf:"fixed64,3,opt,name=Score,proto3" json:"Score,omitempty"`
	// Link to display for this file
	Link          string `protobuf:"bytes,4,opt,name=Link,proto3" json:"Link,omitempty"` // TOO(jlewi): Should we include the file contents?
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *FileSearchResult) Reset() {
	*x = FileSearchResult{}
	mi := &file_agent_filesearch_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *FileSearchResult) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FileSearchResult) ProtoMessage() {}

func (x *FileSearchResult) ProtoReflect() protoreflect.Message {
	mi := &file_agent_filesearch_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FileSearchResult.ProtoReflect.Descriptor instead.
func (*FileSearchResult) Descriptor() ([]byte, []int) {
	return file_agent_filesearch_proto_rawDescGZIP(), []int{0}
}

func (x *FileSearchResult) GetFileID() string {
	if x != nil {
		return x.FileID
	}
	return ""
}

func (x *FileSearchResult) GetFileName() string {
	if x != nil {
		return x.FileName
	}
	return ""
}

func (x *FileSearchResult) GetScore() float64 {
	if x != nil {
		return x.Score
	}
	return 0
}

func (x *FileSearchResult) GetLink() string {
	if x != nil {
		return x.Link
	}
	return ""
}

var File_agent_filesearch_proto protoreflect.FileDescriptor

const file_agent_filesearch_proto_rawDesc = "" +
	"\n" +
	"\x16agent/filesearch.proto\"p\n" +
	"\x10FileSearchResult\x12\x16\n" +
	"\x06FileID\x18\x01 \x01(\tR\x06FileID\x12\x1a\n" +
	"\bFileName\x18\x02 \x01(\tR\bFileName\x12\x14\n" +
	"\x05Score\x18\x03 \x01(\x01R\x05Score\x12\x12\n" +
	"\x04Link\x18\x04 \x01(\tR\x04LinkB5Z3github.com/runmedev/runme/v3/api/gen/proto/go/agentb\x06proto3"

var (
	file_agent_filesearch_proto_rawDescOnce sync.Once
	file_agent_filesearch_proto_rawDescData []byte
)

func file_agent_filesearch_proto_rawDescGZIP() []byte {
	file_agent_filesearch_proto_rawDescOnce.Do(func() {
		file_agent_filesearch_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_agent_filesearch_proto_rawDesc), len(file_agent_filesearch_proto_rawDesc)))
	})
	return file_agent_filesearch_proto_rawDescData
}

var file_agent_filesearch_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_agent_filesearch_proto_goTypes = []any{
	(*FileSearchResult)(nil), // 0: FileSearchResult
}
var file_agent_filesearch_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_agent_filesearch_proto_init() }
func file_agent_filesearch_proto_init() {
	if File_agent_filesearch_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_agent_filesearch_proto_rawDesc), len(file_agent_filesearch_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_agent_filesearch_proto_goTypes,
		DependencyIndexes: file_agent_filesearch_proto_depIdxs,
		MessageInfos:      file_agent_filesearch_proto_msgTypes,
	}.Build()
	File_agent_filesearch_proto = out.File
	file_agent_filesearch_proto_goTypes = nil
	file_agent_filesearch_proto_depIdxs = nil
}
