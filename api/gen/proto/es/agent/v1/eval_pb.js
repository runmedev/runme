// @generated by protoc-gen-es v2.6.0 with parameter "target=js+dts,import_extension=none,json_types=true"
// @generated from file agent/v1/eval.proto (package agent.v1, syntax proto3)
/* eslint-disable */

import { enumDesc, fileDesc, messageDesc, tsEnum } from "@bufbuild/protobuf/codegenv2";
import { file_buf_validate_validate } from "../../buf/validate/validate_pb";

/**
 * Describes the file agent/v1/eval.proto.
 */
export const file_agent_v1_eval = /*@__PURE__*/
  fileDesc("ChNhZ2VudC92MS9ldmFsLnByb3RvEghhZ2VudC52MSLoBwoJQXNzZXJ0aW9uEhgKBG5hbWUYASABKAlCCrpIB8gBAXICEAESLgoEdHlwZRgCIAEoDjIYLmFnZW50LnYxLkFzc2VydGlvbi5UeXBlQga6SAPIAQESKgoGcmVzdWx0GAMgASgOMhouYWdlbnQudjEuQXNzZXJ0aW9uLlJlc3VsdBJEChNzaGVsbF9yZXF1aXJlZF9mbGFnGAQgASgLMiUuYWdlbnQudjEuQXNzZXJ0aW9uLlNoZWxsUmVxdWlyZWRGbGFnSAASPQoPdG9vbF9pbnZvY2F0aW9uGAUgASgLMiIuYWdlbnQudjEuQXNzZXJ0aW9uLlRvb2xJbnZvY2F0aW9uSAASOwoOZmlsZV9yZXRyaWV2YWwYBiABKAsyIS5hZ2VudC52MS5Bc3NlcnRpb24uRmlsZVJldHJpZXZhbEgAEjEKCWxsbV9qdWRnZRgHIAEoCzIcLmFnZW50LnYxLkFzc2VydGlvbi5MTE1KdWRnZUgAEj0KD2NvZGVibG9ja19yZWdleBgIIAEoCzIiLmFnZW50LnYxLkFzc2VydGlvbi5Db2RlYmxvY2tSZWdleEgAEhYKDmZhaWx1cmVfcmVhc29uGAkgASgJGkwKEVNoZWxsUmVxdWlyZWRGbGFnEhsKB2NvbW1hbmQYASABKAlCCrpIB8gBAXICEAESGgoFZmxhZ3MYAiADKAlCC7pICMgBAZIBAggBGi8KDlRvb2xJbnZvY2F0aW9uEh0KCXRvb2xfbmFtZRgBIAEoCUIKukgHyAEBcgIQARo/Cg1GaWxlUmV0cmlldmFsEhsKB2ZpbGVfaWQYASABKAlCCrpIB8gBAXICEAESEQoJZmlsZV9uYW1lGAIgASgJGiYKCExMTUp1ZGdlEhoKBnByb21wdBgBIAEoCUIKukgHyAEBcgIQARorCg5Db2RlYmxvY2tSZWdleBIZCgVyZWdleBgBIAEoCUIKukgHyAEBcgIQASKYAQoEVHlwZRIUChBUWVBFX1VOU1BFQ0lGSUVEEAASHAoYVFlQRV9TSEVMTF9SRVFVSVJFRF9GTEFHEAESFQoRVFlQRV9UT09MX0lOVk9LRUQQAhIXChNUWVBFX0ZJTEVfUkVUUklFVkVEEAMSEgoOVFlQRV9MTE1fSlVER0UQBBIYChRUWVBFX0NPREVCTE9DS19SRUdFWBAFIlcKBlJlc3VsdBIWChJSRVNVTFRfVU5TUEVDSUZJRUQQABIPCgtSRVNVTFRfVFJVRRABEhAKDFJFU1VMVF9GQUxTRRACEhIKDlJFU1VMVF9TS0lQUEVEEANCEAoHcGF5bG9hZBIFukgCCAEirAEKCkV2YWxTYW1wbGUSGAoEa2luZBgBIAEoCUIKukgHyAEBcgIQARIuCghtZXRhZGF0YRgCIAEoCzIULmFnZW50LnYxLk9iamVjdE1ldGFCBrpIA8gBARIeCgppbnB1dF90ZXh0GAMgASgJQgq6SAfIAQFyAhABEjQKCmFzc2VydGlvbnMYBCADKAsyEy5hZ2VudC52MS5Bc3NlcnRpb25CC7pICMgBAZIBAggBIjQKC0V2YWxEYXRhc2V0EiUKB3NhbXBsZXMYASADKAsyFC5hZ2VudC52MS5FdmFsU2FtcGxlIiYKCk9iamVjdE1ldGESGAoEbmFtZRgBIAEoCUIKukgHyAEBcgIQASJ6Cg5FeHBlcmltZW50U3BlYxIgCgxkYXRhc2V0X3BhdGgYASABKAlCCrpIB8gBAXICEAESHgoKb3V0cHV0X2RpchgCIAEoCUIKukgHyAEBcgIQARImChJpbmZlcmVuY2VfZW5kcG9pbnQYAyABKAlCCrpIB8gBAXICEAEipwEKCkV4cGVyaW1lbnQSHwoLYXBpX3ZlcnNpb24YASABKAlCCrpIB8gBAXICEAESGAoEa2luZBgCIAEoCUIKukgHyAEBcgIQARIuCghtZXRhZGF0YRgDIAEoCzIULmFnZW50LnYxLk9iamVjdE1ldGFCBrpIA8gBARIuCgRzcGVjGAQgASgLMhguYWdlbnQudjEuRXhwZXJpbWVudFNwZWNCBrpIA8gBAUJAWj5naXRodWIuY29tL3J1bm1lZGV2L3J1bm1lL3YzL2FwaS9nZW4vcHJvdG8vZ28vYWdlbnQvdjE7YWdlbnR2MWIGcHJvdG8z", [file_buf_validate_validate]);

/**
 * Describes the message agent.v1.Assertion.
 * Use `create(AssertionSchema)` to create a new message.
 */
export const AssertionSchema = /*@__PURE__*/
  messageDesc(file_agent_v1_eval, 0);

/**
 * Describes the message agent.v1.Assertion.ShellRequiredFlag.
 * Use `create(Assertion_ShellRequiredFlagSchema)` to create a new message.
 */
export const Assertion_ShellRequiredFlagSchema = /*@__PURE__*/
  messageDesc(file_agent_v1_eval, 0, 0);

/**
 * Describes the message agent.v1.Assertion.ToolInvocation.
 * Use `create(Assertion_ToolInvocationSchema)` to create a new message.
 */
export const Assertion_ToolInvocationSchema = /*@__PURE__*/
  messageDesc(file_agent_v1_eval, 0, 1);

/**
 * Describes the message agent.v1.Assertion.FileRetrieval.
 * Use `create(Assertion_FileRetrievalSchema)` to create a new message.
 */
export const Assertion_FileRetrievalSchema = /*@__PURE__*/
  messageDesc(file_agent_v1_eval, 0, 2);

/**
 * Describes the message agent.v1.Assertion.LLMJudge.
 * Use `create(Assertion_LLMJudgeSchema)` to create a new message.
 */
export const Assertion_LLMJudgeSchema = /*@__PURE__*/
  messageDesc(file_agent_v1_eval, 0, 3);

/**
 * Describes the message agent.v1.Assertion.CodeblockRegex.
 * Use `create(Assertion_CodeblockRegexSchema)` to create a new message.
 */
export const Assertion_CodeblockRegexSchema = /*@__PURE__*/
  messageDesc(file_agent_v1_eval, 0, 4);

/**
 * Describes the enum agent.v1.Assertion.Type.
 */
export const Assertion_TypeSchema = /*@__PURE__*/
  enumDesc(file_agent_v1_eval, 0, 0);

/**
 * What we are checking for.
 *
 * @generated from enum agent.v1.Assertion.Type
 */
export const Assertion_Type = /*@__PURE__*/
  tsEnum(Assertion_TypeSchema);

/**
 * Describes the enum agent.v1.Assertion.Result.
 */
export const Assertion_ResultSchema = /*@__PURE__*/
  enumDesc(file_agent_v1_eval, 0, 1);

/**
 * Outcome of an assertion after a test run.
 *
 * @generated from enum agent.v1.Assertion.Result
 */
export const Assertion_Result = /*@__PURE__*/
  tsEnum(Assertion_ResultSchema);

/**
 * Describes the message agent.v1.EvalSample.
 * Use `create(EvalSampleSchema)` to create a new message.
 */
export const EvalSampleSchema = /*@__PURE__*/
  messageDesc(file_agent_v1_eval, 1);

/**
 * Describes the message agent.v1.EvalDataset.
 * Use `create(EvalDatasetSchema)` to create a new message.
 */
export const EvalDatasetSchema = /*@__PURE__*/
  messageDesc(file_agent_v1_eval, 2);

/**
 * Describes the message agent.v1.ObjectMeta.
 * Use `create(ObjectMetaSchema)` to create a new message.
 */
export const ObjectMetaSchema = /*@__PURE__*/
  messageDesc(file_agent_v1_eval, 3);

/**
 * Describes the message agent.v1.ExperimentSpec.
 * Use `create(ExperimentSpecSchema)` to create a new message.
 */
export const ExperimentSpecSchema = /*@__PURE__*/
  messageDesc(file_agent_v1_eval, 4);

/**
 * Describes the message agent.v1.Experiment.
 * Use `create(ExperimentSchema)` to create a new message.
 */
export const ExperimentSchema = /*@__PURE__*/
  messageDesc(file_agent_v1_eval, 5);
