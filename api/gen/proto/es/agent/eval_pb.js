// @generated by protoc-gen-es v2.2.3 with parameter "import_extension=js"
// @generated from file agent/eval.proto (syntax proto3)
/* eslint-disable */

import { enumDesc, fileDesc, messageDesc, tsEnum } from "@bufbuild/protobuf/codegenv1";
import { file_buf_validate_validate } from "../buf/validate/validate_pb.js";

/**
 * Describes the file agent/eval.proto.
 */
export const file_agent_eval = /*@__PURE__*/
  fileDesc("ChBhZ2VudC9ldmFsLnByb3RvIqEHCglBc3NlcnRpb24SGAoEbmFtZRgBIAEoCUIKukgHyAEBcgIQARIlCgR0eXBlGAIgASgOMg8uQXNzZXJ0aW9uLlR5cGVCBrpIA8gBARIhCgZyZXN1bHQYAyABKA4yES5Bc3NlcnRpb24uUmVzdWx0EjsKE3NoZWxsX3JlcXVpcmVkX2ZsYWcYBCABKAsyHC5Bc3NlcnRpb24uU2hlbGxSZXF1aXJlZEZsYWdIABI0Cg90b29sX2ludm9jYXRpb24YBSABKAsyGS5Bc3NlcnRpb24uVG9vbEludm9jYXRpb25IABIyCg5maWxlX3JldHJpZXZhbBgGIAEoCzIYLkFzc2VydGlvbi5GaWxlUmV0cmlldmFsSAASKAoJbGxtX2p1ZGdlGAcgASgLMhMuQXNzZXJ0aW9uLkxMTUp1ZGdlSAASNAoPY29kZWJsb2NrX3JlZ2V4GAggASgLMhkuQXNzZXJ0aW9uLkNvZGVibG9ja1JlZ2V4SAASFgoOZmFpbHVyZV9yZWFzb24YCSABKAkaTAoRU2hlbGxSZXF1aXJlZEZsYWcSGwoHY29tbWFuZBgBIAEoCUIKukgHyAEBcgIQARIaCgVmbGFncxgCIAMoCUILukgIyAEBkgECCAEaLwoOVG9vbEludm9jYXRpb24SHQoJdG9vbF9uYW1lGAEgASgJQgq6SAfIAQFyAhABGj8KDUZpbGVSZXRyaWV2YWwSGwoHZmlsZV9pZBgBIAEoCUIKukgHyAEBcgIQARIRCglmaWxlX25hbWUYAiABKAkaJgoITExNSnVkZ2USGgoGcHJvbXB0GAEgASgJQgq6SAfIAQFyAhABGisKDkNvZGVibG9ja1JlZ2V4EhkKBXJlZ2V4GAEgASgJQgq6SAfIAQFyAhABIpQBCgRUeXBlEhAKDFRZUEVfVU5LTk9XThAAEhwKGFRZUEVfU0hFTExfUkVRVUlSRURfRkxBRxABEhUKEVRZUEVfVE9PTF9JTlZPS0VEEAISFwoTVFlQRV9GSUxFX1JFVFJJRVZFRBADEhIKDlRZUEVfTExNX0pVREdFEAQSGAoUVFlQRV9DT0RFQkxPQ0tfUkVHRVgQBSJTCgZSZXN1bHQSEgoOUkVTVUxUX1VOS05PV04QABIPCgtSRVNVTFRfVFJVRRABEhAKDFJFU1VMVF9GQUxTRRACEhIKDlJFU1VMVF9TS0lQUEVEEANCEAoHcGF5bG9hZBIFukgCCAEimgEKCkV2YWxTYW1wbGUSGAoEa2luZBgBIAEoCUIKukgHyAEBcgIQARIlCghtZXRhZGF0YRgCIAEoCzILLk9iamVjdE1ldGFCBrpIA8gBARIeCgppbnB1dF90ZXh0GAMgASgJQgq6SAfIAQFyAhABEisKCmFzc2VydGlvbnMYBCADKAsyCi5Bc3NlcnRpb25CC7pICMgBAZIBAggBIisKC0V2YWxEYXRhc2V0EhwKB3NhbXBsZXMYASADKAsyCy5FdmFsU2FtcGxlIiYKCk9iamVjdE1ldGESGAoEbmFtZRgBIAEoCUIKukgHyAEBcgIQASJ6Cg5FeHBlcmltZW50U3BlYxIgCgxkYXRhc2V0X3BhdGgYASABKAlCCrpIB8gBAXICEAESHgoKb3V0cHV0X2RpchgCIAEoCUIKukgHyAEBcgIQARImChJpbmZlcmVuY2VfZW5kcG9pbnQYAyABKAlCCrpIB8gBAXICEAEilQEKCkV4cGVyaW1lbnQSHwoLYXBpX3ZlcnNpb24YASABKAlCCrpIB8gBAXICEAESGAoEa2luZBgCIAEoCUIKukgHyAEBcgIQARIlCghtZXRhZGF0YRgDIAEoCzILLk9iamVjdE1ldGFCBrpIA8gBARIlCgRzcGVjGAQgASgLMg8uRXhwZXJpbWVudFNwZWNCBrpIA8gBAUI1WjNnaXRodWIuY29tL3J1bm1lZGV2L3J1bm1lL3YzL2FwaS9nZW4vcHJvdG8vZ28vYWdlbnRiBnByb3RvMw", [file_buf_validate_validate]);

/**
 * Describes the message Assertion.
 * Use `create(AssertionSchema)` to create a new message.
 */
export const AssertionSchema = /*@__PURE__*/
  messageDesc(file_agent_eval, 0);

/**
 * Describes the message Assertion.ShellRequiredFlag.
 * Use `create(Assertion_ShellRequiredFlagSchema)` to create a new message.
 */
export const Assertion_ShellRequiredFlagSchema = /*@__PURE__*/
  messageDesc(file_agent_eval, 0, 0);

/**
 * Describes the message Assertion.ToolInvocation.
 * Use `create(Assertion_ToolInvocationSchema)` to create a new message.
 */
export const Assertion_ToolInvocationSchema = /*@__PURE__*/
  messageDesc(file_agent_eval, 0, 1);

/**
 * Describes the message Assertion.FileRetrieval.
 * Use `create(Assertion_FileRetrievalSchema)` to create a new message.
 */
export const Assertion_FileRetrievalSchema = /*@__PURE__*/
  messageDesc(file_agent_eval, 0, 2);

/**
 * Describes the message Assertion.LLMJudge.
 * Use `create(Assertion_LLMJudgeSchema)` to create a new message.
 */
export const Assertion_LLMJudgeSchema = /*@__PURE__*/
  messageDesc(file_agent_eval, 0, 3);

/**
 * Describes the message Assertion.CodeblockRegex.
 * Use `create(Assertion_CodeblockRegexSchema)` to create a new message.
 */
export const Assertion_CodeblockRegexSchema = /*@__PURE__*/
  messageDesc(file_agent_eval, 0, 4);

/**
 * Describes the enum Assertion.Type.
 */
export const Assertion_TypeSchema = /*@__PURE__*/
  enumDesc(file_agent_eval, 0, 0);

/**
 * What we are checking for.
 *
 * @generated from enum Assertion.Type
 */
export const Assertion_Type = /*@__PURE__*/
  tsEnum(Assertion_TypeSchema);

/**
 * Describes the enum Assertion.Result.
 */
export const Assertion_ResultSchema = /*@__PURE__*/
  enumDesc(file_agent_eval, 0, 1);

/**
 * Outcome of an assertion after a test run.
 *
 * @generated from enum Assertion.Result
 */
export const Assertion_Result = /*@__PURE__*/
  tsEnum(Assertion_ResultSchema);

/**
 * Describes the message EvalSample.
 * Use `create(EvalSampleSchema)` to create a new message.
 */
export const EvalSampleSchema = /*@__PURE__*/
  messageDesc(file_agent_eval, 1);

/**
 * Describes the message EvalDataset.
 * Use `create(EvalDatasetSchema)` to create a new message.
 */
export const EvalDatasetSchema = /*@__PURE__*/
  messageDesc(file_agent_eval, 2);

/**
 * Describes the message ObjectMeta.
 * Use `create(ObjectMetaSchema)` to create a new message.
 */
export const ObjectMetaSchema = /*@__PURE__*/
  messageDesc(file_agent_eval, 3);

/**
 * Describes the message ExperimentSpec.
 * Use `create(ExperimentSpecSchema)` to create a new message.
 */
export const ExperimentSpecSchema = /*@__PURE__*/
  messageDesc(file_agent_eval, 4);

/**
 * Describes the message Experiment.
 * Use `create(ExperimentSchema)` to create a new message.
 */
export const ExperimentSchema = /*@__PURE__*/
  messageDesc(file_agent_eval, 5);
