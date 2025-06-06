// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        (unknown)
// source: runme/project/v1/project.proto

package projectv1

import (
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"

	v1 "github.com/runmedev/runme/v3/api/gen/proto/go/runme/parser/v1"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type LoadEventType int32

const (
	LoadEventType_LOAD_EVENT_TYPE_UNSPECIFIED          LoadEventType = 0
	LoadEventType_LOAD_EVENT_TYPE_STARTED_WALK         LoadEventType = 1
	LoadEventType_LOAD_EVENT_TYPE_FOUND_DIR            LoadEventType = 2
	LoadEventType_LOAD_EVENT_TYPE_FOUND_FILE           LoadEventType = 3
	LoadEventType_LOAD_EVENT_TYPE_FINISHED_WALK        LoadEventType = 4
	LoadEventType_LOAD_EVENT_TYPE_STARTED_PARSING_DOC  LoadEventType = 5
	LoadEventType_LOAD_EVENT_TYPE_FINISHED_PARSING_DOC LoadEventType = 6
	LoadEventType_LOAD_EVENT_TYPE_FOUND_TASK           LoadEventType = 7
	LoadEventType_LOAD_EVENT_TYPE_ERROR                LoadEventType = 8
)

// Enum value maps for LoadEventType.
var (
	LoadEventType_name = map[int32]string{
		0: "LOAD_EVENT_TYPE_UNSPECIFIED",
		1: "LOAD_EVENT_TYPE_STARTED_WALK",
		2: "LOAD_EVENT_TYPE_FOUND_DIR",
		3: "LOAD_EVENT_TYPE_FOUND_FILE",
		4: "LOAD_EVENT_TYPE_FINISHED_WALK",
		5: "LOAD_EVENT_TYPE_STARTED_PARSING_DOC",
		6: "LOAD_EVENT_TYPE_FINISHED_PARSING_DOC",
		7: "LOAD_EVENT_TYPE_FOUND_TASK",
		8: "LOAD_EVENT_TYPE_ERROR",
	}
	LoadEventType_value = map[string]int32{
		"LOAD_EVENT_TYPE_UNSPECIFIED":          0,
		"LOAD_EVENT_TYPE_STARTED_WALK":         1,
		"LOAD_EVENT_TYPE_FOUND_DIR":            2,
		"LOAD_EVENT_TYPE_FOUND_FILE":           3,
		"LOAD_EVENT_TYPE_FINISHED_WALK":        4,
		"LOAD_EVENT_TYPE_STARTED_PARSING_DOC":  5,
		"LOAD_EVENT_TYPE_FINISHED_PARSING_DOC": 6,
		"LOAD_EVENT_TYPE_FOUND_TASK":           7,
		"LOAD_EVENT_TYPE_ERROR":                8,
	}
)

func (x LoadEventType) Enum() *LoadEventType {
	p := new(LoadEventType)
	*p = x
	return p
}

func (x LoadEventType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (LoadEventType) Descriptor() protoreflect.EnumDescriptor {
	return file_runme_project_v1_project_proto_enumTypes[0].Descriptor()
}

func (LoadEventType) Type() protoreflect.EnumType {
	return &file_runme_project_v1_project_proto_enumTypes[0]
}

func (x LoadEventType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use LoadEventType.Descriptor instead.
func (LoadEventType) EnumDescriptor() ([]byte, []int) {
	return file_runme_project_v1_project_proto_rawDescGZIP(), []int{0}
}

type DirectoryProjectOptions struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Path to a directory containing the project.
	Path string `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	// If true, .gitignore file is ignored, as well as .git/info/exclude.
	SkipGitignore bool `protobuf:"varint,2,opt,name=skip_gitignore,json=skipGitignore,proto3" json:"skip_gitignore,omitempty"`
	// A list of file patterns, compatible with .gitignore syntax,
	// to ignore.
	IgnoreFilePatterns []string `protobuf:"bytes,3,rep,name=ignore_file_patterns,json=ignoreFilePatterns,proto3" json:"ignore_file_patterns,omitempty"`
	// If true, it disables lookuping up for .git folder
	// in the parent directories.
	SkipRepoLookupUpward bool `protobuf:"varint,4,opt,name=skip_repo_lookup_upward,json=skipRepoLookupUpward,proto3" json:"skip_repo_lookup_upward,omitempty"`
	unknownFields        protoimpl.UnknownFields
	sizeCache            protoimpl.SizeCache
}

func (x *DirectoryProjectOptions) Reset() {
	*x = DirectoryProjectOptions{}
	mi := &file_runme_project_v1_project_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DirectoryProjectOptions) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DirectoryProjectOptions) ProtoMessage() {}

func (x *DirectoryProjectOptions) ProtoReflect() protoreflect.Message {
	mi := &file_runme_project_v1_project_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DirectoryProjectOptions.ProtoReflect.Descriptor instead.
func (*DirectoryProjectOptions) Descriptor() ([]byte, []int) {
	return file_runme_project_v1_project_proto_rawDescGZIP(), []int{0}
}

func (x *DirectoryProjectOptions) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}

func (x *DirectoryProjectOptions) GetSkipGitignore() bool {
	if x != nil {
		return x.SkipGitignore
	}
	return false
}

func (x *DirectoryProjectOptions) GetIgnoreFilePatterns() []string {
	if x != nil {
		return x.IgnoreFilePatterns
	}
	return nil
}

func (x *DirectoryProjectOptions) GetSkipRepoLookupUpward() bool {
	if x != nil {
		return x.SkipRepoLookupUpward
	}
	return false
}

type FileProjectOptions struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Path          string                 `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *FileProjectOptions) Reset() {
	*x = FileProjectOptions{}
	mi := &file_runme_project_v1_project_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *FileProjectOptions) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FileProjectOptions) ProtoMessage() {}

func (x *FileProjectOptions) ProtoReflect() protoreflect.Message {
	mi := &file_runme_project_v1_project_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FileProjectOptions.ProtoReflect.Descriptor instead.
func (*FileProjectOptions) Descriptor() ([]byte, []int) {
	return file_runme_project_v1_project_proto_rawDescGZIP(), []int{1}
}

func (x *FileProjectOptions) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}

type LoadRequest struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Types that are valid to be assigned to Kind:
	//
	//	*LoadRequest_Directory
	//	*LoadRequest_File
	Kind          isLoadRequest_Kind `protobuf_oneof:"kind"`
	Identity      v1.RunmeIdentity   `protobuf:"varint,3,opt,name=identity,proto3,enum=runme.parser.v1.RunmeIdentity" json:"identity,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *LoadRequest) Reset() {
	*x = LoadRequest{}
	mi := &file_runme_project_v1_project_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LoadRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LoadRequest) ProtoMessage() {}

func (x *LoadRequest) ProtoReflect() protoreflect.Message {
	mi := &file_runme_project_v1_project_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LoadRequest.ProtoReflect.Descriptor instead.
func (*LoadRequest) Descriptor() ([]byte, []int) {
	return file_runme_project_v1_project_proto_rawDescGZIP(), []int{2}
}

func (x *LoadRequest) GetKind() isLoadRequest_Kind {
	if x != nil {
		return x.Kind
	}
	return nil
}

func (x *LoadRequest) GetDirectory() *DirectoryProjectOptions {
	if x != nil {
		if x, ok := x.Kind.(*LoadRequest_Directory); ok {
			return x.Directory
		}
	}
	return nil
}

func (x *LoadRequest) GetFile() *FileProjectOptions {
	if x != nil {
		if x, ok := x.Kind.(*LoadRequest_File); ok {
			return x.File
		}
	}
	return nil
}

func (x *LoadRequest) GetIdentity() v1.RunmeIdentity {
	if x != nil {
		return x.Identity
	}
	return v1.RunmeIdentity(0)
}

type isLoadRequest_Kind interface {
	isLoadRequest_Kind()
}

type LoadRequest_Directory struct {
	Directory *DirectoryProjectOptions `protobuf:"bytes,1,opt,name=directory,proto3,oneof"`
}

type LoadRequest_File struct {
	File *FileProjectOptions `protobuf:"bytes,2,opt,name=file,proto3,oneof"`
}

func (*LoadRequest_Directory) isLoadRequest_Kind() {}

func (*LoadRequest_File) isLoadRequest_Kind() {}

type LoadEventStartedWalk struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *LoadEventStartedWalk) Reset() {
	*x = LoadEventStartedWalk{}
	mi := &file_runme_project_v1_project_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LoadEventStartedWalk) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LoadEventStartedWalk) ProtoMessage() {}

func (x *LoadEventStartedWalk) ProtoReflect() protoreflect.Message {
	mi := &file_runme_project_v1_project_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LoadEventStartedWalk.ProtoReflect.Descriptor instead.
func (*LoadEventStartedWalk) Descriptor() ([]byte, []int) {
	return file_runme_project_v1_project_proto_rawDescGZIP(), []int{3}
}

type LoadEventFoundDir struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Path          string                 `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *LoadEventFoundDir) Reset() {
	*x = LoadEventFoundDir{}
	mi := &file_runme_project_v1_project_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LoadEventFoundDir) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LoadEventFoundDir) ProtoMessage() {}

func (x *LoadEventFoundDir) ProtoReflect() protoreflect.Message {
	mi := &file_runme_project_v1_project_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LoadEventFoundDir.ProtoReflect.Descriptor instead.
func (*LoadEventFoundDir) Descriptor() ([]byte, []int) {
	return file_runme_project_v1_project_proto_rawDescGZIP(), []int{4}
}

func (x *LoadEventFoundDir) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}

type LoadEventFoundFile struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Path          string                 `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *LoadEventFoundFile) Reset() {
	*x = LoadEventFoundFile{}
	mi := &file_runme_project_v1_project_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LoadEventFoundFile) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LoadEventFoundFile) ProtoMessage() {}

func (x *LoadEventFoundFile) ProtoReflect() protoreflect.Message {
	mi := &file_runme_project_v1_project_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LoadEventFoundFile.ProtoReflect.Descriptor instead.
func (*LoadEventFoundFile) Descriptor() ([]byte, []int) {
	return file_runme_project_v1_project_proto_rawDescGZIP(), []int{5}
}

func (x *LoadEventFoundFile) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}

type LoadEventFinishedWalk struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *LoadEventFinishedWalk) Reset() {
	*x = LoadEventFinishedWalk{}
	mi := &file_runme_project_v1_project_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LoadEventFinishedWalk) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LoadEventFinishedWalk) ProtoMessage() {}

func (x *LoadEventFinishedWalk) ProtoReflect() protoreflect.Message {
	mi := &file_runme_project_v1_project_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LoadEventFinishedWalk.ProtoReflect.Descriptor instead.
func (*LoadEventFinishedWalk) Descriptor() ([]byte, []int) {
	return file_runme_project_v1_project_proto_rawDescGZIP(), []int{6}
}

type LoadEventStartedParsingDoc struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Path          string                 `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *LoadEventStartedParsingDoc) Reset() {
	*x = LoadEventStartedParsingDoc{}
	mi := &file_runme_project_v1_project_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LoadEventStartedParsingDoc) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LoadEventStartedParsingDoc) ProtoMessage() {}

func (x *LoadEventStartedParsingDoc) ProtoReflect() protoreflect.Message {
	mi := &file_runme_project_v1_project_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LoadEventStartedParsingDoc.ProtoReflect.Descriptor instead.
func (*LoadEventStartedParsingDoc) Descriptor() ([]byte, []int) {
	return file_runme_project_v1_project_proto_rawDescGZIP(), []int{7}
}

func (x *LoadEventStartedParsingDoc) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}

type LoadEventFinishedParsingDoc struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Path          string                 `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *LoadEventFinishedParsingDoc) Reset() {
	*x = LoadEventFinishedParsingDoc{}
	mi := &file_runme_project_v1_project_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LoadEventFinishedParsingDoc) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LoadEventFinishedParsingDoc) ProtoMessage() {}

func (x *LoadEventFinishedParsingDoc) ProtoReflect() protoreflect.Message {
	mi := &file_runme_project_v1_project_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LoadEventFinishedParsingDoc.ProtoReflect.Descriptor instead.
func (*LoadEventFinishedParsingDoc) Descriptor() ([]byte, []int) {
	return file_runme_project_v1_project_proto_rawDescGZIP(), []int{8}
}

func (x *LoadEventFinishedParsingDoc) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}

type LoadEventFoundTask struct {
	state           protoimpl.MessageState `protogen:"open.v1"`
	DocumentPath    string                 `protobuf:"bytes,1,opt,name=document_path,json=documentPath,proto3" json:"document_path,omitempty"`
	Id              string                 `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
	Name            string                 `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
	IsNameGenerated bool                   `protobuf:"varint,4,opt,name=is_name_generated,json=isNameGenerated,proto3" json:"is_name_generated,omitempty"`
	unknownFields   protoimpl.UnknownFields
	sizeCache       protoimpl.SizeCache
}

func (x *LoadEventFoundTask) Reset() {
	*x = LoadEventFoundTask{}
	mi := &file_runme_project_v1_project_proto_msgTypes[9]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LoadEventFoundTask) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LoadEventFoundTask) ProtoMessage() {}

func (x *LoadEventFoundTask) ProtoReflect() protoreflect.Message {
	mi := &file_runme_project_v1_project_proto_msgTypes[9]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LoadEventFoundTask.ProtoReflect.Descriptor instead.
func (*LoadEventFoundTask) Descriptor() ([]byte, []int) {
	return file_runme_project_v1_project_proto_rawDescGZIP(), []int{9}
}

func (x *LoadEventFoundTask) GetDocumentPath() string {
	if x != nil {
		return x.DocumentPath
	}
	return ""
}

func (x *LoadEventFoundTask) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *LoadEventFoundTask) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *LoadEventFoundTask) GetIsNameGenerated() bool {
	if x != nil {
		return x.IsNameGenerated
	}
	return false
}

type LoadEventError struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	ErrorMessage  string                 `protobuf:"bytes,1,opt,name=error_message,json=errorMessage,proto3" json:"error_message,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *LoadEventError) Reset() {
	*x = LoadEventError{}
	mi := &file_runme_project_v1_project_proto_msgTypes[10]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LoadEventError) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LoadEventError) ProtoMessage() {}

func (x *LoadEventError) ProtoReflect() protoreflect.Message {
	mi := &file_runme_project_v1_project_proto_msgTypes[10]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LoadEventError.ProtoReflect.Descriptor instead.
func (*LoadEventError) Descriptor() ([]byte, []int) {
	return file_runme_project_v1_project_proto_rawDescGZIP(), []int{10}
}

func (x *LoadEventError) GetErrorMessage() string {
	if x != nil {
		return x.ErrorMessage
	}
	return ""
}

type LoadResponse struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	Type  LoadEventType          `protobuf:"varint,1,opt,name=type,proto3,enum=runme.project.v1.LoadEventType" json:"type,omitempty"`
	// Types that are valid to be assigned to Data:
	//
	//	*LoadResponse_StartedWalk
	//	*LoadResponse_FoundDir
	//	*LoadResponse_FoundFile
	//	*LoadResponse_FinishedWalk
	//	*LoadResponse_StartedParsingDoc
	//	*LoadResponse_FinishedParsingDoc
	//	*LoadResponse_FoundTask
	//	*LoadResponse_Error
	Data          isLoadResponse_Data `protobuf_oneof:"data"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *LoadResponse) Reset() {
	*x = LoadResponse{}
	mi := &file_runme_project_v1_project_proto_msgTypes[11]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LoadResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LoadResponse) ProtoMessage() {}

func (x *LoadResponse) ProtoReflect() protoreflect.Message {
	mi := &file_runme_project_v1_project_proto_msgTypes[11]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LoadResponse.ProtoReflect.Descriptor instead.
func (*LoadResponse) Descriptor() ([]byte, []int) {
	return file_runme_project_v1_project_proto_rawDescGZIP(), []int{11}
}

func (x *LoadResponse) GetType() LoadEventType {
	if x != nil {
		return x.Type
	}
	return LoadEventType_LOAD_EVENT_TYPE_UNSPECIFIED
}

func (x *LoadResponse) GetData() isLoadResponse_Data {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *LoadResponse) GetStartedWalk() *LoadEventStartedWalk {
	if x != nil {
		if x, ok := x.Data.(*LoadResponse_StartedWalk); ok {
			return x.StartedWalk
		}
	}
	return nil
}

func (x *LoadResponse) GetFoundDir() *LoadEventFoundDir {
	if x != nil {
		if x, ok := x.Data.(*LoadResponse_FoundDir); ok {
			return x.FoundDir
		}
	}
	return nil
}

func (x *LoadResponse) GetFoundFile() *LoadEventFoundFile {
	if x != nil {
		if x, ok := x.Data.(*LoadResponse_FoundFile); ok {
			return x.FoundFile
		}
	}
	return nil
}

func (x *LoadResponse) GetFinishedWalk() *LoadEventFinishedWalk {
	if x != nil {
		if x, ok := x.Data.(*LoadResponse_FinishedWalk); ok {
			return x.FinishedWalk
		}
	}
	return nil
}

func (x *LoadResponse) GetStartedParsingDoc() *LoadEventStartedParsingDoc {
	if x != nil {
		if x, ok := x.Data.(*LoadResponse_StartedParsingDoc); ok {
			return x.StartedParsingDoc
		}
	}
	return nil
}

func (x *LoadResponse) GetFinishedParsingDoc() *LoadEventFinishedParsingDoc {
	if x != nil {
		if x, ok := x.Data.(*LoadResponse_FinishedParsingDoc); ok {
			return x.FinishedParsingDoc
		}
	}
	return nil
}

func (x *LoadResponse) GetFoundTask() *LoadEventFoundTask {
	if x != nil {
		if x, ok := x.Data.(*LoadResponse_FoundTask); ok {
			return x.FoundTask
		}
	}
	return nil
}

func (x *LoadResponse) GetError() *LoadEventError {
	if x != nil {
		if x, ok := x.Data.(*LoadResponse_Error); ok {
			return x.Error
		}
	}
	return nil
}

type isLoadResponse_Data interface {
	isLoadResponse_Data()
}

type LoadResponse_StartedWalk struct {
	StartedWalk *LoadEventStartedWalk `protobuf:"bytes,2,opt,name=started_walk,json=startedWalk,proto3,oneof"`
}

type LoadResponse_FoundDir struct {
	FoundDir *LoadEventFoundDir `protobuf:"bytes,3,opt,name=found_dir,json=foundDir,proto3,oneof"`
}

type LoadResponse_FoundFile struct {
	FoundFile *LoadEventFoundFile `protobuf:"bytes,4,opt,name=found_file,json=foundFile,proto3,oneof"`
}

type LoadResponse_FinishedWalk struct {
	FinishedWalk *LoadEventFinishedWalk `protobuf:"bytes,5,opt,name=finished_walk,json=finishedWalk,proto3,oneof"`
}

type LoadResponse_StartedParsingDoc struct {
	StartedParsingDoc *LoadEventStartedParsingDoc `protobuf:"bytes,6,opt,name=started_parsing_doc,json=startedParsingDoc,proto3,oneof"`
}

type LoadResponse_FinishedParsingDoc struct {
	FinishedParsingDoc *LoadEventFinishedParsingDoc `protobuf:"bytes,7,opt,name=finished_parsing_doc,json=finishedParsingDoc,proto3,oneof"`
}

type LoadResponse_FoundTask struct {
	FoundTask *LoadEventFoundTask `protobuf:"bytes,8,opt,name=found_task,json=foundTask,proto3,oneof"`
}

type LoadResponse_Error struct {
	Error *LoadEventError `protobuf:"bytes,9,opt,name=error,proto3,oneof"`
}

func (*LoadResponse_StartedWalk) isLoadResponse_Data() {}

func (*LoadResponse_FoundDir) isLoadResponse_Data() {}

func (*LoadResponse_FoundFile) isLoadResponse_Data() {}

func (*LoadResponse_FinishedWalk) isLoadResponse_Data() {}

func (*LoadResponse_StartedParsingDoc) isLoadResponse_Data() {}

func (*LoadResponse_FinishedParsingDoc) isLoadResponse_Data() {}

func (*LoadResponse_FoundTask) isLoadResponse_Data() {}

func (*LoadResponse_Error) isLoadResponse_Data() {}

var File_runme_project_v1_project_proto protoreflect.FileDescriptor

const file_runme_project_v1_project_proto_rawDesc = "" +
	"\n" +
	"\x1erunme/project/v1/project.proto\x12\x10runme.project.v1\x1a\x1crunme/parser/v1/parser.proto\"\xbd\x01\n" +
	"\x17DirectoryProjectOptions\x12\x12\n" +
	"\x04path\x18\x01 \x01(\tR\x04path\x12%\n" +
	"\x0eskip_gitignore\x18\x02 \x01(\bR\rskipGitignore\x120\n" +
	"\x14ignore_file_patterns\x18\x03 \x03(\tR\x12ignoreFilePatterns\x125\n" +
	"\x17skip_repo_lookup_upward\x18\x04 \x01(\bR\x14skipRepoLookupUpward\"(\n" +
	"\x12FileProjectOptions\x12\x12\n" +
	"\x04path\x18\x01 \x01(\tR\x04path\"\xd8\x01\n" +
	"\vLoadRequest\x12I\n" +
	"\tdirectory\x18\x01 \x01(\v2).runme.project.v1.DirectoryProjectOptionsH\x00R\tdirectory\x12:\n" +
	"\x04file\x18\x02 \x01(\v2$.runme.project.v1.FileProjectOptionsH\x00R\x04file\x12:\n" +
	"\bidentity\x18\x03 \x01(\x0e2\x1e.runme.parser.v1.RunmeIdentityR\bidentityB\x06\n" +
	"\x04kind\"\x16\n" +
	"\x14LoadEventStartedWalk\"'\n" +
	"\x11LoadEventFoundDir\x12\x12\n" +
	"\x04path\x18\x01 \x01(\tR\x04path\"(\n" +
	"\x12LoadEventFoundFile\x12\x12\n" +
	"\x04path\x18\x01 \x01(\tR\x04path\"\x17\n" +
	"\x15LoadEventFinishedWalk\"0\n" +
	"\x1aLoadEventStartedParsingDoc\x12\x12\n" +
	"\x04path\x18\x01 \x01(\tR\x04path\"1\n" +
	"\x1bLoadEventFinishedParsingDoc\x12\x12\n" +
	"\x04path\x18\x01 \x01(\tR\x04path\"\x89\x01\n" +
	"\x12LoadEventFoundTask\x12#\n" +
	"\rdocument_path\x18\x01 \x01(\tR\fdocumentPath\x12\x0e\n" +
	"\x02id\x18\x02 \x01(\tR\x02id\x12\x12\n" +
	"\x04name\x18\x03 \x01(\tR\x04name\x12*\n" +
	"\x11is_name_generated\x18\x04 \x01(\bR\x0fisNameGenerated\"5\n" +
	"\x0eLoadEventError\x12#\n" +
	"\rerror_message\x18\x01 \x01(\tR\ferrorMessage\"\xb7\x05\n" +
	"\fLoadResponse\x123\n" +
	"\x04type\x18\x01 \x01(\x0e2\x1f.runme.project.v1.LoadEventTypeR\x04type\x12K\n" +
	"\fstarted_walk\x18\x02 \x01(\v2&.runme.project.v1.LoadEventStartedWalkH\x00R\vstartedWalk\x12B\n" +
	"\tfound_dir\x18\x03 \x01(\v2#.runme.project.v1.LoadEventFoundDirH\x00R\bfoundDir\x12E\n" +
	"\n" +
	"found_file\x18\x04 \x01(\v2$.runme.project.v1.LoadEventFoundFileH\x00R\tfoundFile\x12N\n" +
	"\rfinished_walk\x18\x05 \x01(\v2'.runme.project.v1.LoadEventFinishedWalkH\x00R\ffinishedWalk\x12^\n" +
	"\x13started_parsing_doc\x18\x06 \x01(\v2,.runme.project.v1.LoadEventStartedParsingDocH\x00R\x11startedParsingDoc\x12a\n" +
	"\x14finished_parsing_doc\x18\a \x01(\v2-.runme.project.v1.LoadEventFinishedParsingDocH\x00R\x12finishedParsingDoc\x12E\n" +
	"\n" +
	"found_task\x18\b \x01(\v2$.runme.project.v1.LoadEventFoundTaskH\x00R\tfoundTask\x128\n" +
	"\x05error\x18\t \x01(\v2 .runme.project.v1.LoadEventErrorH\x00R\x05errorB\x06\n" +
	"\x04data*\xc2\x02\n" +
	"\rLoadEventType\x12\x1f\n" +
	"\x1bLOAD_EVENT_TYPE_UNSPECIFIED\x10\x00\x12 \n" +
	"\x1cLOAD_EVENT_TYPE_STARTED_WALK\x10\x01\x12\x1d\n" +
	"\x19LOAD_EVENT_TYPE_FOUND_DIR\x10\x02\x12\x1e\n" +
	"\x1aLOAD_EVENT_TYPE_FOUND_FILE\x10\x03\x12!\n" +
	"\x1dLOAD_EVENT_TYPE_FINISHED_WALK\x10\x04\x12'\n" +
	"#LOAD_EVENT_TYPE_STARTED_PARSING_DOC\x10\x05\x12(\n" +
	"$LOAD_EVENT_TYPE_FINISHED_PARSING_DOC\x10\x06\x12\x1e\n" +
	"\x1aLOAD_EVENT_TYPE_FOUND_TASK\x10\a\x12\x19\n" +
	"\x15LOAD_EVENT_TYPE_ERROR\x10\b2[\n" +
	"\x0eProjectService\x12I\n" +
	"\x04Load\x12\x1d.runme.project.v1.LoadRequest\x1a\x1e.runme.project.v1.LoadResponse\"\x000\x01BJZHgithub.com/runmedev/runme/v3/api/gen/proto/go/runme/project/v1;projectv1b\x06proto3"

var (
	file_runme_project_v1_project_proto_rawDescOnce sync.Once
	file_runme_project_v1_project_proto_rawDescData []byte
)

func file_runme_project_v1_project_proto_rawDescGZIP() []byte {
	file_runme_project_v1_project_proto_rawDescOnce.Do(func() {
		file_runme_project_v1_project_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_runme_project_v1_project_proto_rawDesc), len(file_runme_project_v1_project_proto_rawDesc)))
	})
	return file_runme_project_v1_project_proto_rawDescData
}

var file_runme_project_v1_project_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_runme_project_v1_project_proto_msgTypes = make([]protoimpl.MessageInfo, 12)
var file_runme_project_v1_project_proto_goTypes = []any{
	(LoadEventType)(0),                  // 0: runme.project.v1.LoadEventType
	(*DirectoryProjectOptions)(nil),     // 1: runme.project.v1.DirectoryProjectOptions
	(*FileProjectOptions)(nil),          // 2: runme.project.v1.FileProjectOptions
	(*LoadRequest)(nil),                 // 3: runme.project.v1.LoadRequest
	(*LoadEventStartedWalk)(nil),        // 4: runme.project.v1.LoadEventStartedWalk
	(*LoadEventFoundDir)(nil),           // 5: runme.project.v1.LoadEventFoundDir
	(*LoadEventFoundFile)(nil),          // 6: runme.project.v1.LoadEventFoundFile
	(*LoadEventFinishedWalk)(nil),       // 7: runme.project.v1.LoadEventFinishedWalk
	(*LoadEventStartedParsingDoc)(nil),  // 8: runme.project.v1.LoadEventStartedParsingDoc
	(*LoadEventFinishedParsingDoc)(nil), // 9: runme.project.v1.LoadEventFinishedParsingDoc
	(*LoadEventFoundTask)(nil),          // 10: runme.project.v1.LoadEventFoundTask
	(*LoadEventError)(nil),              // 11: runme.project.v1.LoadEventError
	(*LoadResponse)(nil),                // 12: runme.project.v1.LoadResponse
	(v1.RunmeIdentity)(0),               // 13: runme.parser.v1.RunmeIdentity
}
var file_runme_project_v1_project_proto_depIdxs = []int32{
	1,  // 0: runme.project.v1.LoadRequest.directory:type_name -> runme.project.v1.DirectoryProjectOptions
	2,  // 1: runme.project.v1.LoadRequest.file:type_name -> runme.project.v1.FileProjectOptions
	13, // 2: runme.project.v1.LoadRequest.identity:type_name -> runme.parser.v1.RunmeIdentity
	0,  // 3: runme.project.v1.LoadResponse.type:type_name -> runme.project.v1.LoadEventType
	4,  // 4: runme.project.v1.LoadResponse.started_walk:type_name -> runme.project.v1.LoadEventStartedWalk
	5,  // 5: runme.project.v1.LoadResponse.found_dir:type_name -> runme.project.v1.LoadEventFoundDir
	6,  // 6: runme.project.v1.LoadResponse.found_file:type_name -> runme.project.v1.LoadEventFoundFile
	7,  // 7: runme.project.v1.LoadResponse.finished_walk:type_name -> runme.project.v1.LoadEventFinishedWalk
	8,  // 8: runme.project.v1.LoadResponse.started_parsing_doc:type_name -> runme.project.v1.LoadEventStartedParsingDoc
	9,  // 9: runme.project.v1.LoadResponse.finished_parsing_doc:type_name -> runme.project.v1.LoadEventFinishedParsingDoc
	10, // 10: runme.project.v1.LoadResponse.found_task:type_name -> runme.project.v1.LoadEventFoundTask
	11, // 11: runme.project.v1.LoadResponse.error:type_name -> runme.project.v1.LoadEventError
	3,  // 12: runme.project.v1.ProjectService.Load:input_type -> runme.project.v1.LoadRequest
	12, // 13: runme.project.v1.ProjectService.Load:output_type -> runme.project.v1.LoadResponse
	13, // [13:14] is the sub-list for method output_type
	12, // [12:13] is the sub-list for method input_type
	12, // [12:12] is the sub-list for extension type_name
	12, // [12:12] is the sub-list for extension extendee
	0,  // [0:12] is the sub-list for field type_name
}

func init() { file_runme_project_v1_project_proto_init() }
func file_runme_project_v1_project_proto_init() {
	if File_runme_project_v1_project_proto != nil {
		return
	}
	file_runme_project_v1_project_proto_msgTypes[2].OneofWrappers = []any{
		(*LoadRequest_Directory)(nil),
		(*LoadRequest_File)(nil),
	}
	file_runme_project_v1_project_proto_msgTypes[11].OneofWrappers = []any{
		(*LoadResponse_StartedWalk)(nil),
		(*LoadResponse_FoundDir)(nil),
		(*LoadResponse_FoundFile)(nil),
		(*LoadResponse_FinishedWalk)(nil),
		(*LoadResponse_StartedParsingDoc)(nil),
		(*LoadResponse_FinishedParsingDoc)(nil),
		(*LoadResponse_FoundTask)(nil),
		(*LoadResponse_Error)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_runme_project_v1_project_proto_rawDesc), len(file_runme_project_v1_project_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   12,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_runme_project_v1_project_proto_goTypes,
		DependencyIndexes: file_runme_project_v1_project_proto_depIdxs,
		EnumInfos:         file_runme_project_v1_project_proto_enumTypes,
		MessageInfos:      file_runme_project_v1_project_proto_msgTypes,
	}.Build()
	File_runme_project_v1_project_proto = out.File
	file_runme_project_v1_project_proto_goTypes = nil
	file_runme_project_v1_project_proto_depIdxs = nil
}
