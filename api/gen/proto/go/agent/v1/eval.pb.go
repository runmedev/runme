// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        (unknown)
// source: agent/v1/eval.proto

package agentv1

import (
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"

	_ "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// What we are checking for.
type Assertion_Type int32

const (
	Assertion_TYPE_UNSPECIFIED         Assertion_Type = 0
	Assertion_TYPE_SHELL_REQUIRED_FLAG Assertion_Type = 1 // Were all required CLI flags present?
	Assertion_TYPE_TOOL_INVOKED        Assertion_Type = 2 // Was a tool invoked (or not)?
	Assertion_TYPE_FILE_RETRIEVED      Assertion_Type = 3 // Was a file retrieved (or not)?
	Assertion_TYPE_LLM_JUDGE           Assertion_Type = 4 // Ask an LLM to grade the final answer.
	Assertion_TYPE_CODEBLOCK_REGEX     Assertion_Type = 5 // Does at least one code block match the regex?
)

// Enum value maps for Assertion_Type.
var (
	Assertion_Type_name = map[int32]string{
		0: "TYPE_UNSPECIFIED",
		1: "TYPE_SHELL_REQUIRED_FLAG",
		2: "TYPE_TOOL_INVOKED",
		3: "TYPE_FILE_RETRIEVED",
		4: "TYPE_LLM_JUDGE",
		5: "TYPE_CODEBLOCK_REGEX",
	}
	Assertion_Type_value = map[string]int32{
		"TYPE_UNSPECIFIED":         0,
		"TYPE_SHELL_REQUIRED_FLAG": 1,
		"TYPE_TOOL_INVOKED":        2,
		"TYPE_FILE_RETRIEVED":      3,
		"TYPE_LLM_JUDGE":           4,
		"TYPE_CODEBLOCK_REGEX":     5,
	}
)

func (x Assertion_Type) Enum() *Assertion_Type {
	p := new(Assertion_Type)
	*p = x
	return p
}

func (x Assertion_Type) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Assertion_Type) Descriptor() protoreflect.EnumDescriptor {
	return file_agent_v1_eval_proto_enumTypes[0].Descriptor()
}

func (Assertion_Type) Type() protoreflect.EnumType {
	return &file_agent_v1_eval_proto_enumTypes[0]
}

func (x Assertion_Type) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Assertion_Type.Descriptor instead.
func (Assertion_Type) EnumDescriptor() ([]byte, []int) {
	return file_agent_v1_eval_proto_rawDescGZIP(), []int{0, 0}
}

// Outcome of an assertion after a test run.
type Assertion_Result int32

const (
	Assertion_RESULT_UNSPECIFIED Assertion_Result = 0
	Assertion_RESULT_TRUE        Assertion_Result = 1
	Assertion_RESULT_FALSE       Assertion_Result = 2
	Assertion_RESULT_SKIPPED     Assertion_Result = 3
)

// Enum value maps for Assertion_Result.
var (
	Assertion_Result_name = map[int32]string{
		0: "RESULT_UNSPECIFIED",
		1: "RESULT_TRUE",
		2: "RESULT_FALSE",
		3: "RESULT_SKIPPED",
	}
	Assertion_Result_value = map[string]int32{
		"RESULT_UNSPECIFIED": 0,
		"RESULT_TRUE":        1,
		"RESULT_FALSE":       2,
		"RESULT_SKIPPED":     3,
	}
)

func (x Assertion_Result) Enum() *Assertion_Result {
	p := new(Assertion_Result)
	*p = x
	return p
}

func (x Assertion_Result) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Assertion_Result) Descriptor() protoreflect.EnumDescriptor {
	return file_agent_v1_eval_proto_enumTypes[1].Descriptor()
}

func (Assertion_Result) Type() protoreflect.EnumType {
	return &file_agent_v1_eval_proto_enumTypes[1]
}

func (x Assertion_Result) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Assertion_Result.Descriptor instead.
func (Assertion_Result) EnumDescriptor() ([]byte, []int) {
	return file_agent_v1_eval_proto_rawDescGZIP(), []int{0, 1}
}

// -------------------------------------------------------------------------
// Assertions
// -------------------------------------------------------------------------
type Assertion struct {
	state  protoimpl.MessageState `protogen:"open.v1"`
	Name   string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Type   Assertion_Type         `protobuf:"varint,2,opt,name=type,proto3,enum=agent.v1.Assertion_Type" json:"type,omitempty"`
	Result Assertion_Result       `protobuf:"varint,3,opt,name=result,proto3,enum=agent.v1.Assertion_Result" json:"result,omitempty"`
	// Exactly one concrete assertion payload must be present.
	//
	// Types that are valid to be assigned to Payload:
	//
	//	*Assertion_ShellRequiredFlag_
	//	*Assertion_ToolInvocation_
	//	*Assertion_FileRetrieval_
	//	*Assertion_LlmJudge
	//	*Assertion_CodeblockRegex_
	Payload       isAssertion_Payload `protobuf_oneof:"payload"`
	FailureReason string              `protobuf:"bytes,9,opt,name=failure_reason,json=failureReason,proto3" json:"failure_reason,omitempty"` // If the assertion failed, this will contain the reason.
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Assertion) Reset() {
	*x = Assertion{}
	mi := &file_agent_v1_eval_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Assertion) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Assertion) ProtoMessage() {}

func (x *Assertion) ProtoReflect() protoreflect.Message {
	mi := &file_agent_v1_eval_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Assertion.ProtoReflect.Descriptor instead.
func (*Assertion) Descriptor() ([]byte, []int) {
	return file_agent_v1_eval_proto_rawDescGZIP(), []int{0}
}

func (x *Assertion) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Assertion) GetType() Assertion_Type {
	if x != nil {
		return x.Type
	}
	return Assertion_TYPE_UNSPECIFIED
}

func (x *Assertion) GetResult() Assertion_Result {
	if x != nil {
		return x.Result
	}
	return Assertion_RESULT_UNSPECIFIED
}

func (x *Assertion) GetPayload() isAssertion_Payload {
	if x != nil {
		return x.Payload
	}
	return nil
}

func (x *Assertion) GetShellRequiredFlag() *Assertion_ShellRequiredFlag {
	if x != nil {
		if x, ok := x.Payload.(*Assertion_ShellRequiredFlag_); ok {
			return x.ShellRequiredFlag
		}
	}
	return nil
}

func (x *Assertion) GetToolInvocation() *Assertion_ToolInvocation {
	if x != nil {
		if x, ok := x.Payload.(*Assertion_ToolInvocation_); ok {
			return x.ToolInvocation
		}
	}
	return nil
}

func (x *Assertion) GetFileRetrieval() *Assertion_FileRetrieval {
	if x != nil {
		if x, ok := x.Payload.(*Assertion_FileRetrieval_); ok {
			return x.FileRetrieval
		}
	}
	return nil
}

func (x *Assertion) GetLlmJudge() *Assertion_LLMJudge {
	if x != nil {
		if x, ok := x.Payload.(*Assertion_LlmJudge); ok {
			return x.LlmJudge
		}
	}
	return nil
}

func (x *Assertion) GetCodeblockRegex() *Assertion_CodeblockRegex {
	if x != nil {
		if x, ok := x.Payload.(*Assertion_CodeblockRegex_); ok {
			return x.CodeblockRegex
		}
	}
	return nil
}

func (x *Assertion) GetFailureReason() string {
	if x != nil {
		return x.FailureReason
	}
	return ""
}

type isAssertion_Payload interface {
	isAssertion_Payload()
}

type Assertion_ShellRequiredFlag_ struct {
	ShellRequiredFlag *Assertion_ShellRequiredFlag `protobuf:"bytes,4,opt,name=shell_required_flag,json=shellRequiredFlag,proto3,oneof"`
}

type Assertion_ToolInvocation_ struct {
	ToolInvocation *Assertion_ToolInvocation `protobuf:"bytes,5,opt,name=tool_invocation,json=toolInvocation,proto3,oneof"`
}

type Assertion_FileRetrieval_ struct {
	FileRetrieval *Assertion_FileRetrieval `protobuf:"bytes,6,opt,name=file_retrieval,json=fileRetrieval,proto3,oneof"`
}

type Assertion_LlmJudge struct {
	LlmJudge *Assertion_LLMJudge `protobuf:"bytes,7,opt,name=llm_judge,json=llmJudge,proto3,oneof"`
}

type Assertion_CodeblockRegex_ struct {
	CodeblockRegex *Assertion_CodeblockRegex `protobuf:"bytes,8,opt,name=codeblock_regex,json=codeblockRegex,proto3,oneof"`
}

func (*Assertion_ShellRequiredFlag_) isAssertion_Payload() {}

func (*Assertion_ToolInvocation_) isAssertion_Payload() {}

func (*Assertion_FileRetrieval_) isAssertion_Payload() {}

func (*Assertion_LlmJudge) isAssertion_Payload() {}

func (*Assertion_CodeblockRegex_) isAssertion_Payload() {}

// -------------------------------------------------------------------------
// EvalSample – Represents a single evaluation input and its expected assertions
// -------------------------------------------------------------------------
type EvalSample struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Kind          string                 `protobuf:"bytes,1,opt,name=kind,proto3" json:"kind,omitempty"`                            // Resource kind, always "EvalSample"
	Metadata      *ObjectMeta            `protobuf:"bytes,2,opt,name=metadata,proto3" json:"metadata,omitempty"`                    // Standard metadata (name, labels, etc.)
	InputText     string                 `protobuf:"bytes,3,opt,name=input_text,json=inputText,proto3" json:"input_text,omitempty"` // The input text to be evaluated
	Assertions    []*Assertion           `protobuf:"bytes,4,rep,name=assertions,proto3" json:"assertions,omitempty"`                // List of assertions to check for this input
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *EvalSample) Reset() {
	*x = EvalSample{}
	mi := &file_agent_v1_eval_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EvalSample) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EvalSample) ProtoMessage() {}

func (x *EvalSample) ProtoReflect() protoreflect.Message {
	mi := &file_agent_v1_eval_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EvalSample.ProtoReflect.Descriptor instead.
func (*EvalSample) Descriptor() ([]byte, []int) {
	return file_agent_v1_eval_proto_rawDescGZIP(), []int{1}
}

func (x *EvalSample) GetKind() string {
	if x != nil {
		return x.Kind
	}
	return ""
}

func (x *EvalSample) GetMetadata() *ObjectMeta {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *EvalSample) GetInputText() string {
	if x != nil {
		return x.InputText
	}
	return ""
}

func (x *EvalSample) GetAssertions() []*Assertion {
	if x != nil {
		return x.Assertions
	}
	return nil
}

type EvalDataset struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Samples       []*EvalSample          `protobuf:"bytes,1,rep,name=samples,proto3" json:"samples,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *EvalDataset) Reset() {
	*x = EvalDataset{}
	mi := &file_agent_v1_eval_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EvalDataset) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EvalDataset) ProtoMessage() {}

func (x *EvalDataset) ProtoReflect() protoreflect.Message {
	mi := &file_agent_v1_eval_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EvalDataset.ProtoReflect.Descriptor instead.
func (*EvalDataset) Descriptor() ([]byte, []int) {
	return file_agent_v1_eval_proto_rawDescGZIP(), []int{2}
}

func (x *EvalDataset) GetSamples() []*EvalSample {
	if x != nil {
		return x.Samples
	}
	return nil
}

type ObjectMeta struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Name of the resource, e.g. "experiment-test".
	Name          string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ObjectMeta) Reset() {
	*x = ObjectMeta{}
	mi := &file_agent_v1_eval_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ObjectMeta) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ObjectMeta) ProtoMessage() {}

func (x *ObjectMeta) ProtoReflect() protoreflect.Message {
	mi := &file_agent_v1_eval_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ObjectMeta.ProtoReflect.Descriptor instead.
func (*ObjectMeta) Descriptor() ([]byte, []int) {
	return file_agent_v1_eval_proto_rawDescGZIP(), []int{3}
}

func (x *ObjectMeta) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type ExperimentSpec struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Path to the folder containing the dataset to evaluate.
	DatasetPath string `protobuf:"bytes,1,opt,name=dataset_path,json=datasetPath,proto3" json:"dataset_path,omitempty"`
	// Directory where experiment reports will be written.
	OutputDir string `protobuf:"bytes,2,opt,name=output_dir,json=outputDir,proto3" json:"output_dir,omitempty"`
	// URL of the backend inference service to call during evaluation.
	InferenceEndpoint string `protobuf:"bytes,3,opt,name=inference_endpoint,json=inferenceEndpoint,proto3" json:"inference_endpoint,omitempty"`
	unknownFields     protoimpl.UnknownFields
	sizeCache         protoimpl.SizeCache
}

func (x *ExperimentSpec) Reset() {
	*x = ExperimentSpec{}
	mi := &file_agent_v1_eval_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ExperimentSpec) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ExperimentSpec) ProtoMessage() {}

func (x *ExperimentSpec) ProtoReflect() protoreflect.Message {
	mi := &file_agent_v1_eval_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ExperimentSpec.ProtoReflect.Descriptor instead.
func (*ExperimentSpec) Descriptor() ([]byte, []int) {
	return file_agent_v1_eval_proto_rawDescGZIP(), []int{4}
}

func (x *ExperimentSpec) GetDatasetPath() string {
	if x != nil {
		return x.DatasetPath
	}
	return ""
}

func (x *ExperimentSpec) GetOutputDir() string {
	if x != nil {
		return x.OutputDir
	}
	return ""
}

func (x *ExperimentSpec) GetInferenceEndpoint() string {
	if x != nil {
		return x.InferenceEndpoint
	}
	return ""
}

type Experiment struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// API version of the resource, e.g. "cloudassistant.io/v1alpha1".
	ApiVersion string `protobuf:"bytes,1,opt,name=api_version,json=apiVersion,proto3" json:"api_version,omitempty"`
	// Kind of the resource. Always "Experiment" for this CRD.
	Kind string `protobuf:"bytes,2,opt,name=kind,proto3" json:"kind,omitempty"`
	// Standard Kubernetes object metadata (name, labels, annotations, etc.).
	Metadata *ObjectMeta `protobuf:"bytes,3,opt,name=metadata,proto3" json:"metadata,omitempty"`
	// User-defined configuration for the experiment.
	Spec          *ExperimentSpec `protobuf:"bytes,4,opt,name=spec,proto3" json:"spec,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Experiment) Reset() {
	*x = Experiment{}
	mi := &file_agent_v1_eval_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Experiment) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Experiment) ProtoMessage() {}

func (x *Experiment) ProtoReflect() protoreflect.Message {
	mi := &file_agent_v1_eval_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Experiment.ProtoReflect.Descriptor instead.
func (*Experiment) Descriptor() ([]byte, []int) {
	return file_agent_v1_eval_proto_rawDescGZIP(), []int{5}
}

func (x *Experiment) GetApiVersion() string {
	if x != nil {
		return x.ApiVersion
	}
	return ""
}

func (x *Experiment) GetKind() string {
	if x != nil {
		return x.Kind
	}
	return ""
}

func (x *Experiment) GetMetadata() *ObjectMeta {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *Experiment) GetSpec() *ExperimentSpec {
	if x != nil {
		return x.Spec
	}
	return nil
}

// Verifies that a shell command includes specific flags.
type Assertion_ShellRequiredFlag struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Command       string                 `protobuf:"bytes,1,opt,name=command,proto3" json:"command,omitempty"` // e.g. "kubectl"
	Flags         []string               `protobuf:"bytes,2,rep,name=flags,proto3" json:"flags,omitempty"`     // e.g. ["--context"]
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Assertion_ShellRequiredFlag) Reset() {
	*x = Assertion_ShellRequiredFlag{}
	mi := &file_agent_v1_eval_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Assertion_ShellRequiredFlag) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Assertion_ShellRequiredFlag) ProtoMessage() {}

func (x *Assertion_ShellRequiredFlag) ProtoReflect() protoreflect.Message {
	mi := &file_agent_v1_eval_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Assertion_ShellRequiredFlag.ProtoReflect.Descriptor instead.
func (*Assertion_ShellRequiredFlag) Descriptor() ([]byte, []int) {
	return file_agent_v1_eval_proto_rawDescGZIP(), []int{0, 0}
}

func (x *Assertion_ShellRequiredFlag) GetCommand() string {
	if x != nil {
		return x.Command
	}
	return ""
}

func (x *Assertion_ShellRequiredFlag) GetFlags() []string {
	if x != nil {
		return x.Flags
	}
	return nil
}

// Verifies that a tool **is** or **is not** invoked.
type Assertion_ToolInvocation struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	ToolName      string                 `protobuf:"bytes,1,opt,name=tool_name,json=toolName,proto3" json:"tool_name,omitempty"` // e.g. "file_search"
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Assertion_ToolInvocation) Reset() {
	*x = Assertion_ToolInvocation{}
	mi := &file_agent_v1_eval_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Assertion_ToolInvocation) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Assertion_ToolInvocation) ProtoMessage() {}

func (x *Assertion_ToolInvocation) ProtoReflect() protoreflect.Message {
	mi := &file_agent_v1_eval_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Assertion_ToolInvocation.ProtoReflect.Descriptor instead.
func (*Assertion_ToolInvocation) Descriptor() ([]byte, []int) {
	return file_agent_v1_eval_proto_rawDescGZIP(), []int{0, 1}
}

func (x *Assertion_ToolInvocation) GetToolName() string {
	if x != nil {
		return x.ToolName
	}
	return ""
}

// Verifies that a file **is** or **is not** retrieved.
type Assertion_FileRetrieval struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	FileId        string                 `protobuf:"bytes,1,opt,name=file_id,json=fileId,proto3" json:"file_id,omitempty"`
	FileName      string                 `protobuf:"bytes,2,opt,name=file_name,json=fileName,proto3" json:"file_name,omitempty"` // Optional human-readable name
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Assertion_FileRetrieval) Reset() {
	*x = Assertion_FileRetrieval{}
	mi := &file_agent_v1_eval_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Assertion_FileRetrieval) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Assertion_FileRetrieval) ProtoMessage() {}

func (x *Assertion_FileRetrieval) ProtoReflect() protoreflect.Message {
	mi := &file_agent_v1_eval_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Assertion_FileRetrieval.ProtoReflect.Descriptor instead.
func (*Assertion_FileRetrieval) Descriptor() ([]byte, []int) {
	return file_agent_v1_eval_proto_rawDescGZIP(), []int{0, 2}
}

func (x *Assertion_FileRetrieval) GetFileId() string {
	if x != nil {
		return x.FileId
	}
	return ""
}

func (x *Assertion_FileRetrieval) GetFileName() string {
	if x != nil {
		return x.FileName
	}
	return ""
}

// Asks an LLM to grade the assistant's answer.
type Assertion_LLMJudge struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Prompt        string                 `protobuf:"bytes,1,opt,name=prompt,proto3" json:"prompt,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Assertion_LLMJudge) Reset() {
	*x = Assertion_LLMJudge{}
	mi := &file_agent_v1_eval_proto_msgTypes[9]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Assertion_LLMJudge) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Assertion_LLMJudge) ProtoMessage() {}

func (x *Assertion_LLMJudge) ProtoReflect() protoreflect.Message {
	mi := &file_agent_v1_eval_proto_msgTypes[9]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Assertion_LLMJudge.ProtoReflect.Descriptor instead.
func (*Assertion_LLMJudge) Descriptor() ([]byte, []int) {
	return file_agent_v1_eval_proto_rawDescGZIP(), []int{0, 3}
}

func (x *Assertion_LLMJudge) GetPrompt() string {
	if x != nil {
		return x.Prompt
	}
	return ""
}

// Checks if at least one code block matches the regex.
type Assertion_CodeblockRegex struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Regex         string                 `protobuf:"bytes,1,opt,name=regex,proto3" json:"regex,omitempty"` // The regex pattern to match against code blocks
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Assertion_CodeblockRegex) Reset() {
	*x = Assertion_CodeblockRegex{}
	mi := &file_agent_v1_eval_proto_msgTypes[10]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Assertion_CodeblockRegex) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Assertion_CodeblockRegex) ProtoMessage() {}

func (x *Assertion_CodeblockRegex) ProtoReflect() protoreflect.Message {
	mi := &file_agent_v1_eval_proto_msgTypes[10]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Assertion_CodeblockRegex.ProtoReflect.Descriptor instead.
func (*Assertion_CodeblockRegex) Descriptor() ([]byte, []int) {
	return file_agent_v1_eval_proto_rawDescGZIP(), []int{0, 4}
}

func (x *Assertion_CodeblockRegex) GetRegex() string {
	if x != nil {
		return x.Regex
	}
	return ""
}

var File_agent_v1_eval_proto protoreflect.FileDescriptor

const file_agent_v1_eval_proto_rawDesc = "" +
	"\n" +
	"\x13agent/v1/eval.proto\x12\bagent.v1\x1a\x1bbuf/validate/validate.proto\"\x92\t\n" +
	"\tAssertion\x12\x1e\n" +
	"\x04name\x18\x01 \x01(\tB\n" +
	"\xbaH\a\xc8\x01\x01r\x02\x10\x01R\x04name\x124\n" +
	"\x04type\x18\x02 \x01(\x0e2\x18.agent.v1.Assertion.TypeB\x06\xbaH\x03\xc8\x01\x01R\x04type\x122\n" +
	"\x06result\x18\x03 \x01(\x0e2\x1a.agent.v1.Assertion.ResultR\x06result\x12W\n" +
	"\x13shell_required_flag\x18\x04 \x01(\v2%.agent.v1.Assertion.ShellRequiredFlagH\x00R\x11shellRequiredFlag\x12M\n" +
	"\x0ftool_invocation\x18\x05 \x01(\v2\".agent.v1.Assertion.ToolInvocationH\x00R\x0etoolInvocation\x12J\n" +
	"\x0efile_retrieval\x18\x06 \x01(\v2!.agent.v1.Assertion.FileRetrievalH\x00R\rfileRetrieval\x12;\n" +
	"\tllm_judge\x18\a \x01(\v2\x1c.agent.v1.Assertion.LLMJudgeH\x00R\bllmJudge\x12M\n" +
	"\x0fcodeblock_regex\x18\b \x01(\v2\".agent.v1.Assertion.CodeblockRegexH\x00R\x0ecodeblockRegex\x12%\n" +
	"\x0efailure_reason\x18\t \x01(\tR\rfailureReason\x1a\\\n" +
	"\x11ShellRequiredFlag\x12$\n" +
	"\acommand\x18\x01 \x01(\tB\n" +
	"\xbaH\a\xc8\x01\x01r\x02\x10\x01R\acommand\x12!\n" +
	"\x05flags\x18\x02 \x03(\tB\v\xbaH\b\xc8\x01\x01\x92\x01\x02\b\x01R\x05flags\x1a9\n" +
	"\x0eToolInvocation\x12'\n" +
	"\ttool_name\x18\x01 \x01(\tB\n" +
	"\xbaH\a\xc8\x01\x01r\x02\x10\x01R\btoolName\x1aQ\n" +
	"\rFileRetrieval\x12#\n" +
	"\afile_id\x18\x01 \x01(\tB\n" +
	"\xbaH\a\xc8\x01\x01r\x02\x10\x01R\x06fileId\x12\x1b\n" +
	"\tfile_name\x18\x02 \x01(\tR\bfileName\x1a.\n" +
	"\bLLMJudge\x12\"\n" +
	"\x06prompt\x18\x01 \x01(\tB\n" +
	"\xbaH\a\xc8\x01\x01r\x02\x10\x01R\x06prompt\x1a2\n" +
	"\x0eCodeblockRegex\x12 \n" +
	"\x05regex\x18\x01 \x01(\tB\n" +
	"\xbaH\a\xc8\x01\x01r\x02\x10\x01R\x05regex\"\x98\x01\n" +
	"\x04Type\x12\x14\n" +
	"\x10TYPE_UNSPECIFIED\x10\x00\x12\x1c\n" +
	"\x18TYPE_SHELL_REQUIRED_FLAG\x10\x01\x12\x15\n" +
	"\x11TYPE_TOOL_INVOKED\x10\x02\x12\x17\n" +
	"\x13TYPE_FILE_RETRIEVED\x10\x03\x12\x12\n" +
	"\x0eTYPE_LLM_JUDGE\x10\x04\x12\x18\n" +
	"\x14TYPE_CODEBLOCK_REGEX\x10\x05\"W\n" +
	"\x06Result\x12\x16\n" +
	"\x12RESULT_UNSPECIFIED\x10\x00\x12\x0f\n" +
	"\vRESULT_TRUE\x10\x01\x12\x10\n" +
	"\fRESULT_FALSE\x10\x02\x12\x12\n" +
	"\x0eRESULT_SKIPPED\x10\x03B\x10\n" +
	"\apayload\x12\x05\xbaH\x02\b\x01\"\xd3\x01\n" +
	"\n" +
	"EvalSample\x12\x1e\n" +
	"\x04kind\x18\x01 \x01(\tB\n" +
	"\xbaH\a\xc8\x01\x01r\x02\x10\x01R\x04kind\x128\n" +
	"\bmetadata\x18\x02 \x01(\v2\x14.agent.v1.ObjectMetaB\x06\xbaH\x03\xc8\x01\x01R\bmetadata\x12)\n" +
	"\n" +
	"input_text\x18\x03 \x01(\tB\n" +
	"\xbaH\a\xc8\x01\x01r\x02\x10\x01R\tinputText\x12@\n" +
	"\n" +
	"assertions\x18\x04 \x03(\v2\x13.agent.v1.AssertionB\v\xbaH\b\xc8\x01\x01\x92\x01\x02\b\x01R\n" +
	"assertions\"=\n" +
	"\vEvalDataset\x12.\n" +
	"\asamples\x18\x01 \x03(\v2\x14.agent.v1.EvalSampleR\asamples\",\n" +
	"\n" +
	"ObjectMeta\x12\x1e\n" +
	"\x04name\x18\x01 \x01(\tB\n" +
	"\xbaH\a\xc8\x01\x01r\x02\x10\x01R\x04name\"\xa5\x01\n" +
	"\x0eExperimentSpec\x12-\n" +
	"\fdataset_path\x18\x01 \x01(\tB\n" +
	"\xbaH\a\xc8\x01\x01r\x02\x10\x01R\vdatasetPath\x12)\n" +
	"\n" +
	"output_dir\x18\x02 \x01(\tB\n" +
	"\xbaH\a\xc8\x01\x01r\x02\x10\x01R\toutputDir\x129\n" +
	"\x12inference_endpoint\x18\x03 \x01(\tB\n" +
	"\xbaH\a\xc8\x01\x01r\x02\x10\x01R\x11inferenceEndpoint\"\xc9\x01\n" +
	"\n" +
	"Experiment\x12+\n" +
	"\vapi_version\x18\x01 \x01(\tB\n" +
	"\xbaH\a\xc8\x01\x01r\x02\x10\x01R\n" +
	"apiVersion\x12\x1e\n" +
	"\x04kind\x18\x02 \x01(\tB\n" +
	"\xbaH\a\xc8\x01\x01r\x02\x10\x01R\x04kind\x128\n" +
	"\bmetadata\x18\x03 \x01(\v2\x14.agent.v1.ObjectMetaB\x06\xbaH\x03\xc8\x01\x01R\bmetadata\x124\n" +
	"\x04spec\x18\x04 \x01(\v2\x18.agent.v1.ExperimentSpecB\x06\xbaH\x03\xc8\x01\x01R\x04specB@Z>github.com/runmedev/runme/v3/api/gen/proto/go/agent/v1;agentv1b\x06proto3"

var (
	file_agent_v1_eval_proto_rawDescOnce sync.Once
	file_agent_v1_eval_proto_rawDescData []byte
)

func file_agent_v1_eval_proto_rawDescGZIP() []byte {
	file_agent_v1_eval_proto_rawDescOnce.Do(func() {
		file_agent_v1_eval_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_agent_v1_eval_proto_rawDesc), len(file_agent_v1_eval_proto_rawDesc)))
	})
	return file_agent_v1_eval_proto_rawDescData
}

var file_agent_v1_eval_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_agent_v1_eval_proto_msgTypes = make([]protoimpl.MessageInfo, 11)
var file_agent_v1_eval_proto_goTypes = []any{
	(Assertion_Type)(0),                 // 0: agent.v1.Assertion.Type
	(Assertion_Result)(0),               // 1: agent.v1.Assertion.Result
	(*Assertion)(nil),                   // 2: agent.v1.Assertion
	(*EvalSample)(nil),                  // 3: agent.v1.EvalSample
	(*EvalDataset)(nil),                 // 4: agent.v1.EvalDataset
	(*ObjectMeta)(nil),                  // 5: agent.v1.ObjectMeta
	(*ExperimentSpec)(nil),              // 6: agent.v1.ExperimentSpec
	(*Experiment)(nil),                  // 7: agent.v1.Experiment
	(*Assertion_ShellRequiredFlag)(nil), // 8: agent.v1.Assertion.ShellRequiredFlag
	(*Assertion_ToolInvocation)(nil),    // 9: agent.v1.Assertion.ToolInvocation
	(*Assertion_FileRetrieval)(nil),     // 10: agent.v1.Assertion.FileRetrieval
	(*Assertion_LLMJudge)(nil),          // 11: agent.v1.Assertion.LLMJudge
	(*Assertion_CodeblockRegex)(nil),    // 12: agent.v1.Assertion.CodeblockRegex
}
var file_agent_v1_eval_proto_depIdxs = []int32{
	0,  // 0: agent.v1.Assertion.type:type_name -> agent.v1.Assertion.Type
	1,  // 1: agent.v1.Assertion.result:type_name -> agent.v1.Assertion.Result
	8,  // 2: agent.v1.Assertion.shell_required_flag:type_name -> agent.v1.Assertion.ShellRequiredFlag
	9,  // 3: agent.v1.Assertion.tool_invocation:type_name -> agent.v1.Assertion.ToolInvocation
	10, // 4: agent.v1.Assertion.file_retrieval:type_name -> agent.v1.Assertion.FileRetrieval
	11, // 5: agent.v1.Assertion.llm_judge:type_name -> agent.v1.Assertion.LLMJudge
	12, // 6: agent.v1.Assertion.codeblock_regex:type_name -> agent.v1.Assertion.CodeblockRegex
	5,  // 7: agent.v1.EvalSample.metadata:type_name -> agent.v1.ObjectMeta
	2,  // 8: agent.v1.EvalSample.assertions:type_name -> agent.v1.Assertion
	3,  // 9: agent.v1.EvalDataset.samples:type_name -> agent.v1.EvalSample
	5,  // 10: agent.v1.Experiment.metadata:type_name -> agent.v1.ObjectMeta
	6,  // 11: agent.v1.Experiment.spec:type_name -> agent.v1.ExperimentSpec
	12, // [12:12] is the sub-list for method output_type
	12, // [12:12] is the sub-list for method input_type
	12, // [12:12] is the sub-list for extension type_name
	12, // [12:12] is the sub-list for extension extendee
	0,  // [0:12] is the sub-list for field type_name
}

func init() { file_agent_v1_eval_proto_init() }
func file_agent_v1_eval_proto_init() {
	if File_agent_v1_eval_proto != nil {
		return
	}
	file_agent_v1_eval_proto_msgTypes[0].OneofWrappers = []any{
		(*Assertion_ShellRequiredFlag_)(nil),
		(*Assertion_ToolInvocation_)(nil),
		(*Assertion_FileRetrieval_)(nil),
		(*Assertion_LlmJudge)(nil),
		(*Assertion_CodeblockRegex_)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_agent_v1_eval_proto_rawDesc), len(file_agent_v1_eval_proto_rawDesc)),
			NumEnums:      2,
			NumMessages:   11,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_agent_v1_eval_proto_goTypes,
		DependencyIndexes: file_agent_v1_eval_proto_depIdxs,
		EnumInfos:         file_agent_v1_eval_proto_enumTypes,
		MessageInfos:      file_agent_v1_eval_proto_msgTypes,
	}.Build()
	File_agent_v1_eval_proto = out.File
	file_agent_v1_eval_proto_goTypes = nil
	file_agent_v1_eval_proto_depIdxs = nil
}
