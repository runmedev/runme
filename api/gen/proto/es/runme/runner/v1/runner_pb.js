// @generated by protoc-gen-es v2.2.3 with parameter "import_extension=js"
// @generated from file runme/runner/v1/runner.proto (package runme.runner.v1, syntax proto3)
/* eslint-disable */

import { enumDesc, fileDesc, messageDesc, serviceDesc, tsEnum } from "@bufbuild/protobuf/codegenv1";
import { file_google_protobuf_wrappers } from "@bufbuild/protobuf/wkt";

/**
 * Describes the file runme/runner/v1/runner.proto.
 */
export const file_runme_runner_v1_runner = /*@__PURE__*/
  fileDesc("ChxydW5tZS9ydW5uZXIvdjEvcnVubmVyLnByb3RvEg9ydW5tZS5ydW5uZXIudjEijgEKB1Nlc3Npb24SCgoCaWQYASABKAkSDAoEZW52cxgCIAMoCRI4CghtZXRhZGF0YRgDIAMoCzImLnJ1bm1lLnJ1bm5lci52MS5TZXNzaW9uLk1ldGFkYXRhRW50cnkaLwoNTWV0YWRhdGFFbnRyeRILCgNrZXkYASABKAkSDQoFdmFsdWUYAiABKAk6AjgBIpYCChRDcmVhdGVTZXNzaW9uUmVxdWVzdBJFCghtZXRhZGF0YRgBIAMoCzIzLnJ1bm1lLnJ1bm5lci52MS5DcmVhdGVTZXNzaW9uUmVxdWVzdC5NZXRhZGF0YUVudHJ5EgwKBGVudnMYAiADKAkSLgoHcHJvamVjdBgDIAEoCzIYLnJ1bm1lLnJ1bm5lci52MS5Qcm9qZWN0SACIAQESPAoOZW52X3N0b3JlX3R5cGUYBCABKA4yJC5ydW5tZS5ydW5uZXIudjEuU2Vzc2lvbkVudlN0b3JlVHlwZRovCg1NZXRhZGF0YUVudHJ5EgsKA2tleRgBIAEoCRINCgV2YWx1ZRgCIAEoCToCOAFCCgoIX3Byb2plY3QiQgoVQ3JlYXRlU2Vzc2lvblJlc3BvbnNlEikKB3Nlc3Npb24YASABKAsyGC5ydW5tZS5ydW5uZXIudjEuU2Vzc2lvbiIfChFHZXRTZXNzaW9uUmVxdWVzdBIKCgJpZBgBIAEoCSI/ChJHZXRTZXNzaW9uUmVzcG9uc2USKQoHc2Vzc2lvbhgBIAEoCzIYLnJ1bm1lLnJ1bm5lci52MS5TZXNzaW9uIhUKE0xpc3RTZXNzaW9uc1JlcXVlc3QiQgoUTGlzdFNlc3Npb25zUmVzcG9uc2USKgoIc2Vzc2lvbnMYASADKAsyGC5ydW5tZS5ydW5uZXIudjEuU2Vzc2lvbiIiChREZWxldGVTZXNzaW9uUmVxdWVzdBIKCgJpZBgBIAEoCSIXChVEZWxldGVTZXNzaW9uUmVzcG9uc2Ui0gEKB1Byb2plY3QSDAoEcm9vdBgBIAEoCRIWCg5lbnZfbG9hZF9vcmRlchgCIAMoCRIzCgplbnZfZGlyZW52GAMgASgOMh8ucnVubWUucnVubmVyLnYxLlByb2plY3QuRGlyRW52ImwKBkRpckVudhIXChNESVJfRU5WX1VOU1BFQ0lGSUVEEAASGAoURElSX0VOVl9FTkFCTEVEX1dBUk4QARIZChVESVJfRU5WX0VOQUJMRURfRVJST1IQAhIUChBESVJfRU5WX0RJU0FCTEVEEAMiOwoHV2luc2l6ZRIMCgRyb3dzGAEgASgNEgwKBGNvbHMYAiABKA0SCQoBeBgDIAEoDRIJCgF5GAQgASgNIscECg5FeGVjdXRlUmVxdWVzdBIUCgxwcm9ncmFtX25hbWUYASABKAkSEQoJYXJndW1lbnRzGAIgAygJEhEKCWRpcmVjdG9yeRgDIAEoCRIMCgRlbnZzGAQgAygJEhAKCGNvbW1hbmRzGAUgAygJEg4KBnNjcmlwdBgGIAEoCRILCgN0dHkYByABKAgSEgoKaW5wdXRfZGF0YRgIIAEoDBIqCgRzdG9wGAkgASgOMhwucnVubWUucnVubmVyLnYxLkV4ZWN1dGVTdG9wEi4KB3dpbnNpemUYCiABKAsyGC5ydW5tZS5ydW5uZXIudjEuV2luc2l6ZUgAiAEBEhIKCmJhY2tncm91bmQYCyABKAgSEgoKc2Vzc2lvbl9pZBgUIAEoCRI6ChBzZXNzaW9uX3N0cmF0ZWd5GBUgASgOMiAucnVubWUucnVubmVyLnYxLlNlc3Npb25TdHJhdGVneRIuCgdwcm9qZWN0GBYgASgLMhgucnVubWUucnVubmVyLnYxLlByb2plY3RIAYgBARIZChFzdG9yZV9sYXN0X291dHB1dBgXIAEoCBIyCgxjb21tYW5kX21vZGUYGCABKA4yHC5ydW5tZS5ydW5uZXIudjEuQ29tbWFuZE1vZGUSEwoLbGFuZ3VhZ2VfaWQYGSABKAkSFgoOZmlsZV9leHRlbnNpb24YGiABKAkSEAoIa25vd25faWQYGyABKAkSEgoKa25vd25fbmFtZRgcIAEoCUIKCghfd2luc2l6ZUIKCghfcHJvamVjdCIZCgpQcm9jZXNzUElEEgsKA3BpZBgBIAEoAyKpAQoPRXhlY3V0ZVJlc3BvbnNlEi8KCWV4aXRfY29kZRgBIAEoCzIcLmdvb2dsZS5wcm90b2J1Zi5VSW50MzJWYWx1ZRITCgtzdGRvdXRfZGF0YRgCIAEoDBITCgtzdGRlcnJfZGF0YRgDIAEoDBIoCgNwaWQYBCABKAsyGy5ydW5tZS5ydW5uZXIudjEuUHJvY2Vzc1BJRBIRCgltaW1lX3R5cGUYBSABKAkiKgoZUmVzb2x2ZVByb2dyYW1Db21tYW5kTGlzdBINCgVsaW5lcxgBIAMoCSLABAoVUmVzb2x2ZVByb2dyYW1SZXF1ZXN0Ej4KCGNvbW1hbmRzGAEgASgLMioucnVubWUucnVubmVyLnYxLlJlc29sdmVQcm9ncmFtQ29tbWFuZExpc3RIABIQCgZzY3JpcHQYAiABKAlIABI5CgRtb2RlGAMgASgOMisucnVubWUucnVubmVyLnYxLlJlc29sdmVQcm9ncmFtUmVxdWVzdC5Nb2RlEgsKA2VudhgEIAMoCRISCgpzZXNzaW9uX2lkGAUgASgJEjoKEHNlc3Npb25fc3RyYXRlZ3kYBiABKA4yIC5ydW5tZS5ydW5uZXIudjEuU2Vzc2lvblN0cmF0ZWd5Ei4KB3Byb2plY3QYByABKAsyGC5ydW5tZS5ydW5uZXIudjEuUHJvamVjdEgBiAEBEhMKC2xhbmd1YWdlX2lkGAggASgJEkMKCXJldGVudGlvbhgJIAEoDjIwLnJ1bm1lLnJ1bm5lci52MS5SZXNvbHZlUHJvZ3JhbVJlcXVlc3QuUmV0ZW50aW9uIkQKBE1vZGUSFAoQTU9ERV9VTlNQRUNJRklFRBAAEhMKD01PREVfUFJPTVBUX0FMTBABEhEKDU1PREVfU0tJUF9BTEwQAiJXCglSZXRlbnRpb24SGQoVUkVURU5USU9OX1VOU1BFQ0lGSUVEEAASFwoTUkVURU5USU9OX0ZJUlNUX1JVThABEhYKElJFVEVOVElPTl9MQVNUX1JVThACQggKBnNvdXJjZUIKCghfcHJvamVjdCLaAwoWUmVzb2x2ZVByb2dyYW1SZXNwb25zZRIOCgZzY3JpcHQYASABKAkSPAoIY29tbWFuZHMYAiABKAsyKi5ydW5tZS5ydW5uZXIudjEuUmVzb2x2ZVByb2dyYW1Db21tYW5kTGlzdBI/CgR2YXJzGAMgAygLMjEucnVubWUucnVubmVyLnYxLlJlc29sdmVQcm9ncmFtUmVzcG9uc2UuVmFyUmVzdWx0GokBCglWYXJSZXN1bHQSPgoGc3RhdHVzGAEgASgOMi4ucnVubWUucnVubmVyLnYxLlJlc29sdmVQcm9ncmFtUmVzcG9uc2UuU3RhdHVzEgwKBG5hbWUYAiABKAkSFgoOb3JpZ2luYWxfdmFsdWUYAyABKAkSFgoOcmVzb2x2ZWRfdmFsdWUYBCABKAkipAEKBlN0YXR1cxIWChJTVEFUVVNfVU5TUEVDSUZJRUQQABIiCh5TVEFUVVNfVU5SRVNPTFZFRF9XSVRIX01FU1NBR0UQARImCiJTVEFUVVNfVU5SRVNPTFZFRF9XSVRIX1BMQUNFSE9MREVSEAISEwoPU1RBVFVTX1JFU09MVkVEEAMSIQodU1RBVFVTX1VOUkVTT0xWRURfV0lUSF9TRUNSRVQQBCJDChZNb25pdG9yRW52U3RvcmVSZXF1ZXN0EikKB3Nlc3Npb24YASABKAsyGC5ydW5tZS5ydW5uZXIudjEuU2Vzc2lvbiLCBAofTW9uaXRvckVudlN0b3JlUmVzcG9uc2VTbmFwc2hvdBJKCgRlbnZzGAEgAygLMjwucnVubWUucnVubmVyLnYxLk1vbml0b3JFbnZTdG9yZVJlc3BvbnNlU25hcHNob3QuU25hcHNob3RFbnYazgIKC1NuYXBzaG90RW52EkcKBnN0YXR1cxgBIAEoDjI3LnJ1bm1lLnJ1bm5lci52MS5Nb25pdG9yRW52U3RvcmVSZXNwb25zZVNuYXBzaG90LlN0YXR1cxIMCgRuYW1lGAIgASgJEgwKBHNwZWMYAyABKAkSDgoGb3JpZ2luGAQgASgJEhYKDm9yaWdpbmFsX3ZhbHVlGAUgASgJEhYKDnJlc29sdmVkX3ZhbHVlGAYgASgJEhMKC2NyZWF0ZV90aW1lGAcgASgJEhMKC3VwZGF0ZV90aW1lGAggASgJEkYKBmVycm9ycxgJIAMoCzI2LnJ1bm1lLnJ1bm5lci52MS5Nb25pdG9yRW52U3RvcmVSZXNwb25zZVNuYXBzaG90LkVycm9yEhMKC2lzX3JlcXVpcmVkGAogASgIEhMKC2Rlc2NyaXB0aW9uGAsgASgJGiYKBUVycm9yEgwKBGNvZGUYASABKA0SDwoHbWVzc2FnZRgCIAEoCSJaCgZTdGF0dXMSFgoSU1RBVFVTX1VOU1BFQ0lGSUVEEAASEgoOU1RBVFVTX0xJVEVSQUwQARIRCg1TVEFUVVNfSElEREVOEAISEQoNU1RBVFVTX01BU0tFRBADIpsBChdNb25pdG9yRW52U3RvcmVSZXNwb25zZRIyCgR0eXBlGAEgASgOMiQucnVubWUucnVubmVyLnYxLk1vbml0b3JFbnZTdG9yZVR5cGUSRAoIc25hcHNob3QYAiABKAsyMC5ydW5tZS5ydW5uZXIudjEuTW9uaXRvckVudlN0b3JlUmVzcG9uc2VTbmFwc2hvdEgAQgYKBGRhdGEqXQoTU2Vzc2lvbkVudlN0b3JlVHlwZRImCiJTRVNTSU9OX0VOVl9TVE9SRV9UWVBFX1VOU1BFQ0lGSUVEEAASHgoaU0VTU0lPTl9FTlZfU1RPUkVfVFlQRV9PV0wQASpeCgtFeGVjdXRlU3RvcBIcChhFWEVDVVRFX1NUT1BfVU5TUEVDSUZJRUQQABIaChZFWEVDVVRFX1NUT1BfSU5URVJSVVBUEAESFQoRRVhFQ1VURV9TVE9QX0tJTEwQAiqaAQoLQ29tbWFuZE1vZGUSHAoYQ09NTUFORF9NT0RFX1VOU1BFQ0lGSUVEEAASHQoZQ09NTUFORF9NT0RFX0lOTElORV9TSEVMTBABEhoKFkNPTU1BTkRfTU9ERV9URU1QX0ZJTEUQAhIZChVDT01NQU5EX01PREVfVEVSTUlOQUwQAxIXChNDT01NQU5EX01PREVfREFHR0VSEAQqVQoPU2Vzc2lvblN0cmF0ZWd5EiAKHFNFU1NJT05fU1RSQVRFR1lfVU5TUEVDSUZJRUQQABIgChxTRVNTSU9OX1NUUkFURUdZX01PU1RfUkVDRU5UEAEqYgoTTW9uaXRvckVudlN0b3JlVHlwZRImCiJNT05JVE9SX0VOVl9TVE9SRV9UWVBFX1VOU1BFQ0lGSUVEEAASIwofTU9OSVRPUl9FTlZfU1RPUkVfVFlQRV9TTkFQU0hPVBABMq4FCg1SdW5uZXJTZXJ2aWNlEmAKDUNyZWF0ZVNlc3Npb24SJS5ydW5tZS5ydW5uZXIudjEuQ3JlYXRlU2Vzc2lvblJlcXVlc3QaJi5ydW5tZS5ydW5uZXIudjEuQ3JlYXRlU2Vzc2lvblJlc3BvbnNlIgASVwoKR2V0U2Vzc2lvbhIiLnJ1bm1lLnJ1bm5lci52MS5HZXRTZXNzaW9uUmVxdWVzdBojLnJ1bm1lLnJ1bm5lci52MS5HZXRTZXNzaW9uUmVzcG9uc2UiABJdCgxMaXN0U2Vzc2lvbnMSJC5ydW5tZS5ydW5uZXIudjEuTGlzdFNlc3Npb25zUmVxdWVzdBolLnJ1bm1lLnJ1bm5lci52MS5MaXN0U2Vzc2lvbnNSZXNwb25zZSIAEmAKDURlbGV0ZVNlc3Npb24SJS5ydW5tZS5ydW5uZXIudjEuRGVsZXRlU2Vzc2lvblJlcXVlc3QaJi5ydW5tZS5ydW5uZXIudjEuRGVsZXRlU2Vzc2lvblJlc3BvbnNlIgASaAoPTW9uaXRvckVudlN0b3JlEicucnVubWUucnVubmVyLnYxLk1vbml0b3JFbnZTdG9yZVJlcXVlc3QaKC5ydW5tZS5ydW5uZXIudjEuTW9uaXRvckVudlN0b3JlUmVzcG9uc2UiADABElIKB0V4ZWN1dGUSHy5ydW5tZS5ydW5uZXIudjEuRXhlY3V0ZVJlcXVlc3QaIC5ydW5tZS5ydW5uZXIudjEuRXhlY3V0ZVJlc3BvbnNlIgAoATABEmMKDlJlc29sdmVQcm9ncmFtEiYucnVubWUucnVubmVyLnYxLlJlc29sdmVQcm9ncmFtUmVxdWVzdBonLnJ1bm1lLnJ1bm5lci52MS5SZXNvbHZlUHJvZ3JhbVJlc3BvbnNlIgBCSFpGZ2l0aHViLmNvbS9ydW5tZWRldi9ydW5tZS92My9hcGkvZ2VuL3Byb3RvL2dvL3J1bm1lL3J1bm5lci92MTtydW5uZXJ2MWIGcHJvdG8z", [file_google_protobuf_wrappers]);

/**
 * Describes the message runme.runner.v1.Session.
 * Use `create(SessionSchema)` to create a new message.
 */
export const SessionSchema = /*@__PURE__*/
  messageDesc(file_runme_runner_v1_runner, 0);

/**
 * Describes the message runme.runner.v1.CreateSessionRequest.
 * Use `create(CreateSessionRequestSchema)` to create a new message.
 */
export const CreateSessionRequestSchema = /*@__PURE__*/
  messageDesc(file_runme_runner_v1_runner, 1);

/**
 * Describes the message runme.runner.v1.CreateSessionResponse.
 * Use `create(CreateSessionResponseSchema)` to create a new message.
 */
export const CreateSessionResponseSchema = /*@__PURE__*/
  messageDesc(file_runme_runner_v1_runner, 2);

/**
 * Describes the message runme.runner.v1.GetSessionRequest.
 * Use `create(GetSessionRequestSchema)` to create a new message.
 */
export const GetSessionRequestSchema = /*@__PURE__*/
  messageDesc(file_runme_runner_v1_runner, 3);

/**
 * Describes the message runme.runner.v1.GetSessionResponse.
 * Use `create(GetSessionResponseSchema)` to create a new message.
 */
export const GetSessionResponseSchema = /*@__PURE__*/
  messageDesc(file_runme_runner_v1_runner, 4);

/**
 * Describes the message runme.runner.v1.ListSessionsRequest.
 * Use `create(ListSessionsRequestSchema)` to create a new message.
 */
export const ListSessionsRequestSchema = /*@__PURE__*/
  messageDesc(file_runme_runner_v1_runner, 5);

/**
 * Describes the message runme.runner.v1.ListSessionsResponse.
 * Use `create(ListSessionsResponseSchema)` to create a new message.
 */
export const ListSessionsResponseSchema = /*@__PURE__*/
  messageDesc(file_runme_runner_v1_runner, 6);

/**
 * Describes the message runme.runner.v1.DeleteSessionRequest.
 * Use `create(DeleteSessionRequestSchema)` to create a new message.
 */
export const DeleteSessionRequestSchema = /*@__PURE__*/
  messageDesc(file_runme_runner_v1_runner, 7);

/**
 * Describes the message runme.runner.v1.DeleteSessionResponse.
 * Use `create(DeleteSessionResponseSchema)` to create a new message.
 */
export const DeleteSessionResponseSchema = /*@__PURE__*/
  messageDesc(file_runme_runner_v1_runner, 8);

/**
 * Describes the message runme.runner.v1.Project.
 * Use `create(ProjectSchema)` to create a new message.
 */
export const ProjectSchema = /*@__PURE__*/
  messageDesc(file_runme_runner_v1_runner, 9);

/**
 * Describes the enum runme.runner.v1.Project.DirEnv.
 */
export const Project_DirEnvSchema = /*@__PURE__*/
  enumDesc(file_runme_runner_v1_runner, 9, 0);

/**
 * @generated from enum runme.runner.v1.Project.DirEnv
 */
export const Project_DirEnv = /*@__PURE__*/
  tsEnum(Project_DirEnvSchema);

/**
 * Describes the message runme.runner.v1.Winsize.
 * Use `create(WinsizeSchema)` to create a new message.
 */
export const WinsizeSchema = /*@__PURE__*/
  messageDesc(file_runme_runner_v1_runner, 10);

/**
 * Describes the message runme.runner.v1.ExecuteRequest.
 * Use `create(ExecuteRequestSchema)` to create a new message.
 */
export const ExecuteRequestSchema = /*@__PURE__*/
  messageDesc(file_runme_runner_v1_runner, 11);

/**
 * Describes the message runme.runner.v1.ProcessPID.
 * Use `create(ProcessPIDSchema)` to create a new message.
 */
export const ProcessPIDSchema = /*@__PURE__*/
  messageDesc(file_runme_runner_v1_runner, 12);

/**
 * Describes the message runme.runner.v1.ExecuteResponse.
 * Use `create(ExecuteResponseSchema)` to create a new message.
 */
export const ExecuteResponseSchema = /*@__PURE__*/
  messageDesc(file_runme_runner_v1_runner, 13);

/**
 * Describes the message runme.runner.v1.ResolveProgramCommandList.
 * Use `create(ResolveProgramCommandListSchema)` to create a new message.
 */
export const ResolveProgramCommandListSchema = /*@__PURE__*/
  messageDesc(file_runme_runner_v1_runner, 14);

/**
 * Describes the message runme.runner.v1.ResolveProgramRequest.
 * Use `create(ResolveProgramRequestSchema)` to create a new message.
 */
export const ResolveProgramRequestSchema = /*@__PURE__*/
  messageDesc(file_runme_runner_v1_runner, 15);

/**
 * Describes the enum runme.runner.v1.ResolveProgramRequest.Mode.
 */
export const ResolveProgramRequest_ModeSchema = /*@__PURE__*/
  enumDesc(file_runme_runner_v1_runner, 15, 0);

/**
 * @generated from enum runme.runner.v1.ResolveProgramRequest.Mode
 */
export const ResolveProgramRequest_Mode = /*@__PURE__*/
  tsEnum(ResolveProgramRequest_ModeSchema);

/**
 * Describes the enum runme.runner.v1.ResolveProgramRequest.Retention.
 */
export const ResolveProgramRequest_RetentionSchema = /*@__PURE__*/
  enumDesc(file_runme_runner_v1_runner, 15, 1);

/**
 * @generated from enum runme.runner.v1.ResolveProgramRequest.Retention
 */
export const ResolveProgramRequest_Retention = /*@__PURE__*/
  tsEnum(ResolveProgramRequest_RetentionSchema);

/**
 * Describes the message runme.runner.v1.ResolveProgramResponse.
 * Use `create(ResolveProgramResponseSchema)` to create a new message.
 */
export const ResolveProgramResponseSchema = /*@__PURE__*/
  messageDesc(file_runme_runner_v1_runner, 16);

/**
 * Describes the message runme.runner.v1.ResolveProgramResponse.VarResult.
 * Use `create(ResolveProgramResponse_VarResultSchema)` to create a new message.
 */
export const ResolveProgramResponse_VarResultSchema = /*@__PURE__*/
  messageDesc(file_runme_runner_v1_runner, 16, 0);

/**
 * Describes the enum runme.runner.v1.ResolveProgramResponse.Status.
 */
export const ResolveProgramResponse_StatusSchema = /*@__PURE__*/
  enumDesc(file_runme_runner_v1_runner, 16, 0);

/**
 * @generated from enum runme.runner.v1.ResolveProgramResponse.Status
 */
export const ResolveProgramResponse_Status = /*@__PURE__*/
  tsEnum(ResolveProgramResponse_StatusSchema);

/**
 * Describes the message runme.runner.v1.MonitorEnvStoreRequest.
 * Use `create(MonitorEnvStoreRequestSchema)` to create a new message.
 */
export const MonitorEnvStoreRequestSchema = /*@__PURE__*/
  messageDesc(file_runme_runner_v1_runner, 17);

/**
 * Describes the message runme.runner.v1.MonitorEnvStoreResponseSnapshot.
 * Use `create(MonitorEnvStoreResponseSnapshotSchema)` to create a new message.
 */
export const MonitorEnvStoreResponseSnapshotSchema = /*@__PURE__*/
  messageDesc(file_runme_runner_v1_runner, 18);

/**
 * Describes the message runme.runner.v1.MonitorEnvStoreResponseSnapshot.SnapshotEnv.
 * Use `create(MonitorEnvStoreResponseSnapshot_SnapshotEnvSchema)` to create a new message.
 */
export const MonitorEnvStoreResponseSnapshot_SnapshotEnvSchema = /*@__PURE__*/
  messageDesc(file_runme_runner_v1_runner, 18, 0);

/**
 * Describes the message runme.runner.v1.MonitorEnvStoreResponseSnapshot.Error.
 * Use `create(MonitorEnvStoreResponseSnapshot_ErrorSchema)` to create a new message.
 */
export const MonitorEnvStoreResponseSnapshot_ErrorSchema = /*@__PURE__*/
  messageDesc(file_runme_runner_v1_runner, 18, 1);

/**
 * Describes the enum runme.runner.v1.MonitorEnvStoreResponseSnapshot.Status.
 */
export const MonitorEnvStoreResponseSnapshot_StatusSchema = /*@__PURE__*/
  enumDesc(file_runme_runner_v1_runner, 18, 0);

/**
 * @generated from enum runme.runner.v1.MonitorEnvStoreResponseSnapshot.Status
 */
export const MonitorEnvStoreResponseSnapshot_Status = /*@__PURE__*/
  tsEnum(MonitorEnvStoreResponseSnapshot_StatusSchema);

/**
 * Describes the message runme.runner.v1.MonitorEnvStoreResponse.
 * Use `create(MonitorEnvStoreResponseSchema)` to create a new message.
 */
export const MonitorEnvStoreResponseSchema = /*@__PURE__*/
  messageDesc(file_runme_runner_v1_runner, 19);

/**
 * Describes the enum runme.runner.v1.SessionEnvStoreType.
 */
export const SessionEnvStoreTypeSchema = /*@__PURE__*/
  enumDesc(file_runme_runner_v1_runner, 0);

/**
 * env store implementation
 *
 * @generated from enum runme.runner.v1.SessionEnvStoreType
 */
export const SessionEnvStoreType = /*@__PURE__*/
  tsEnum(SessionEnvStoreTypeSchema);

/**
 * Describes the enum runme.runner.v1.ExecuteStop.
 */
export const ExecuteStopSchema = /*@__PURE__*/
  enumDesc(file_runme_runner_v1_runner, 1);

/**
 * @generated from enum runme.runner.v1.ExecuteStop
 */
export const ExecuteStop = /*@__PURE__*/
  tsEnum(ExecuteStopSchema);

/**
 * Describes the enum runme.runner.v1.CommandMode.
 */
export const CommandModeSchema = /*@__PURE__*/
  enumDesc(file_runme_runner_v1_runner, 2);

/**
 * @generated from enum runme.runner.v1.CommandMode
 */
export const CommandMode = /*@__PURE__*/
  tsEnum(CommandModeSchema);

/**
 * Describes the enum runme.runner.v1.SessionStrategy.
 */
export const SessionStrategySchema = /*@__PURE__*/
  enumDesc(file_runme_runner_v1_runner, 3);

/**
 * strategy for selecting a session in an initial execute request
 *
 * @generated from enum runme.runner.v1.SessionStrategy
 */
export const SessionStrategy = /*@__PURE__*/
  tsEnum(SessionStrategySchema);

/**
 * Describes the enum runme.runner.v1.MonitorEnvStoreType.
 */
export const MonitorEnvStoreTypeSchema = /*@__PURE__*/
  enumDesc(file_runme_runner_v1_runner, 4);

/**
 * @generated from enum runme.runner.v1.MonitorEnvStoreType
 */
export const MonitorEnvStoreType = /*@__PURE__*/
  tsEnum(MonitorEnvStoreTypeSchema);

/**
 * @generated from service runme.runner.v1.RunnerService
 */
export const RunnerService = /*@__PURE__*/
  serviceDesc(file_runme_runner_v1_runner, 0);
