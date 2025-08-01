/* eslint-disable */
// @generated by protobuf-ts 2.11.1 with parameter output_javascript,optimize_code_size,long_type_string,add_pb_suffix,ts_nocheck,eslint_disable
// @generated from protobuf file "runme/runner/v2/runner.proto" (package "runme.runner.v2", syntax proto3)
// tslint:disable
// @ts-nocheck
import type { BinaryWriteOptions } from "@protobuf-ts/runtime";
import type { IBinaryWriter } from "@protobuf-ts/runtime";
import type { BinaryReadOptions } from "@protobuf-ts/runtime";
import type { IBinaryReader } from "@protobuf-ts/runtime";
import type { PartialMessage } from "@protobuf-ts/runtime";
import { MessageType } from "@protobuf-ts/runtime";
import { UInt32Value } from "../../../google/protobuf/wrappers_pb";
import { ProgramConfig } from "./config_pb";
/**
 * @generated from protobuf message runme.runner.v2.Project
 */
export interface Project {
    /**
     * root is a root directory of the project.
     * The semantic is the same as for the "--project"
     * flag in "runme".
     *
     * @generated from protobuf field: string root = 1
     */
    root: string;
    /**
     * env_load_order is list of environment files
     * to try and load env from.
     *
     * @generated from protobuf field: repeated string env_load_order = 2
     */
    envLoadOrder: string[];
}
/**
 * @generated from protobuf message runme.runner.v2.Session
 */
export interface Session {
    /**
     * @generated from protobuf field: string id = 1
     */
    id: string;
    /**
     * env keeps track of session environment variables.
     * They can be modified by executing programs which
     * alter them through "export" and "unset" commands.
     *
     * @generated from protobuf field: repeated string env = 2
     */
    env: string[];
    /**
     * metadata is a map of client specific metadata.
     *
     * @generated from protobuf field: map<string, string> metadata = 3
     */
    metadata: {
        [key: string]: string;
    };
}
/**
 * @generated from protobuf message runme.runner.v2.CreateSessionRequest
 */
export interface CreateSessionRequest {
    /**
     * metadata is a map of client specific metadata.
     *
     * @generated from protobuf field: map<string, string> metadata = 1
     */
    metadata: {
        [key: string]: string;
    };
    /**
     * env field provides an initial set of environment variables
     * for a newly created session.
     *
     * @generated from protobuf field: repeated string env = 2
     */
    env: string[];
    /**
     * project from which to load environment variables.
     * They will be appended to the list from the env field.
     * The env field has a higher priority.
     *
     * @generated from protobuf field: optional runme.runner.v2.Project project = 3
     */
    project?: Project;
    /**
     * Deprecated use config instead. optional selection
     * of which env store implementation to use.
     *
     * @generated from protobuf field: optional runme.runner.v2.SessionEnvStoreType env_store_type = 4
     */
    envStoreType?: SessionEnvStoreType;
    /**
     * @generated from protobuf field: runme.runner.v2.CreateSessionRequest.Config config = 5
     */
    config?: CreateSessionRequest_Config;
}
/**
 * @generated from protobuf message runme.runner.v2.CreateSessionRequest.Config
 */
export interface CreateSessionRequest_Config {
    /**
     * optional selection of which env store implementation to use.
     *
     * @generated from protobuf field: optional runme.runner.v2.SessionEnvStoreType env_store_type = 1
     */
    envStoreType?: SessionEnvStoreType;
    /**
     * how to seed initial ENV
     *
     * @generated from protobuf field: optional runme.runner.v2.CreateSessionRequest.Config.SessionEnvStoreSeeding env_store_seeding = 2
     */
    envStoreSeeding?: CreateSessionRequest_Config_SessionEnvStoreSeeding;
}
/**
 * @generated from protobuf enum runme.runner.v2.CreateSessionRequest.Config.SessionEnvStoreSeeding
 */
export declare enum CreateSessionRequest_Config_SessionEnvStoreSeeding {
    /**
     * default seeding; ignore system
     *
     * @generated from protobuf enum value: SESSION_ENV_STORE_SEEDING_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,
    /**
     * enable seeding from system
     *
     * @generated from protobuf enum value: SESSION_ENV_STORE_SEEDING_SYSTEM = 1;
     */
    SYSTEM = 1
}
/**
 * @generated from protobuf message runme.runner.v2.CreateSessionResponse
 */
export interface CreateSessionResponse {
    /**
     * @generated from protobuf field: runme.runner.v2.Session session = 1
     */
    session?: Session;
}
/**
 * @generated from protobuf message runme.runner.v2.GetSessionRequest
 */
export interface GetSessionRequest {
    /**
     * @generated from protobuf field: string id = 1
     */
    id: string;
}
/**
 * @generated from protobuf message runme.runner.v2.GetSessionResponse
 */
export interface GetSessionResponse {
    /**
     * @generated from protobuf field: runme.runner.v2.Session session = 1
     */
    session?: Session;
}
/**
 * @generated from protobuf message runme.runner.v2.ListSessionsRequest
 */
export interface ListSessionsRequest {
}
/**
 * @generated from protobuf message runme.runner.v2.ListSessionsResponse
 */
export interface ListSessionsResponse {
    /**
     * @generated from protobuf field: repeated runme.runner.v2.Session sessions = 1
     */
    sessions: Session[];
}
/**
 * @generated from protobuf message runme.runner.v2.UpdateSessionRequest
 */
export interface UpdateSessionRequest {
    /**
     * @generated from protobuf field: string id = 1
     */
    id: string;
    /**
     * metadata is a map of client specific metadata.
     *
     * @generated from protobuf field: map<string, string> metadata = 2
     */
    metadata: {
        [key: string]: string;
    };
    /**
     * env field provides an initial set of environment variables
     * for a newly created session.
     *
     * @generated from protobuf field: repeated string env = 3
     */
    env: string[];
    /**
     * project from which to load environment variables.
     * They will be appended to the list from the env field.
     * The env field has a higher priority.
     *
     * @generated from protobuf field: optional runme.runner.v2.Project project = 4
     */
    project?: Project;
}
/**
 * @generated from protobuf message runme.runner.v2.UpdateSessionResponse
 */
export interface UpdateSessionResponse {
    /**
     * @generated from protobuf field: runme.runner.v2.Session session = 1
     */
    session?: Session;
}
/**
 * @generated from protobuf message runme.runner.v2.DeleteSessionRequest
 */
export interface DeleteSessionRequest {
    /**
     * @generated from protobuf field: string id = 1
     */
    id: string;
}
/**
 * @generated from protobuf message runme.runner.v2.DeleteSessionResponse
 */
export interface DeleteSessionResponse {
}
/**
 * @generated from protobuf message runme.runner.v2.Winsize
 */
export interface Winsize {
    /**
     * @generated from protobuf field: uint32 rows = 1
     */
    rows: number;
    /**
     * @generated from protobuf field: uint32 cols = 2
     */
    cols: number;
    /**
     * @generated from protobuf field: uint32 x = 3
     */
    x: number;
    /**
     * @generated from protobuf field: uint32 y = 4
     */
    y: number;
}
/**
 * @generated from protobuf message runme.runner.v2.ExecuteRequest
 */
export interface ExecuteRequest {
    /**
     * @generated from protobuf field: runme.runner.v2.ProgramConfig config = 1
     */
    config?: ProgramConfig;
    /**
     * input_data is a byte array that will be send as input
     * to the program.
     *
     * @generated from protobuf field: bytes input_data = 8
     */
    inputData: Uint8Array;
    /**
     * stop requests the running process to be stopped.
     * It is allowed only in the consecutive calls.
     *
     * @generated from protobuf field: runme.runner.v2.ExecuteStop stop = 9
     */
    stop: ExecuteStop;
    /**
     * sets pty winsize
     * has no effect in non-interactive mode
     *
     * @generated from protobuf field: optional runme.runner.v2.Winsize winsize = 10
     */
    winsize?: Winsize;
    /**
     * session_id indicates in which Session the program should execute.
     * Executing in a Session might provide additional context like
     * environment variables.
     *
     * @generated from protobuf field: string session_id = 20
     */
    sessionId: string;
    /**
     * session_strategy is a strategy for selecting the session.
     *
     * @generated from protobuf field: runme.runner.v2.SessionStrategy session_strategy = 21
     */
    sessionStrategy: SessionStrategy;
    /**
     * project used to load environment variables from .env files.
     *
     * @generated from protobuf field: optional runme.runner.v2.Project project = 22
     */
    project?: Project;
    /**
     * store_stdout_in_env, if true, will store the stdout under well known name
     * and the last ran block in the environment variable `__`.
     *
     * @generated from protobuf field: bool store_stdout_in_env = 23
     */
    storeStdoutInEnv: boolean;
}
/**
 * @generated from protobuf message runme.runner.v2.ExecuteResponse
 */
export interface ExecuteResponse {
    /**
     * exit_code is sent only in the final message.
     *
     * @generated from protobuf field: google.protobuf.UInt32Value exit_code = 1
     */
    exitCode?: UInt32Value;
    /**
     * stdout_data contains bytes from stdout since the last response.
     *
     * @generated from protobuf field: bytes stdout_data = 2
     */
    stdoutData: Uint8Array;
    /**
     * stderr_data contains bytes from stderr since the last response.
     *
     * @generated from protobuf field: bytes stderr_data = 3
     */
    stderrData: Uint8Array;
    /**
     * pid contains the process' PID.
     *
     * This is only sent once in an initial response for background processes.
     *
     * @generated from protobuf field: google.protobuf.UInt32Value pid = 4
     */
    pid?: UInt32Value;
    /**
     * mime_type is a detected MIME type of the stdout_data.
     *
     * This is only sent once in the first response containing stdout_data.
     *
     * @generated from protobuf field: string mime_type = 5
     */
    mimeType: string;
}
/**
 * @generated from protobuf message runme.runner.v2.ResolveProgramCommandList
 */
export interface ResolveProgramCommandList {
    /**
     * commands are commands to be executed by the program.
     * The commands are joined and executed as a script.
     * For example: ["echo 'Hello, World'", "ls -l /etc"].
     *
     * @generated from protobuf field: repeated string lines = 1
     */
    lines: string[];
}
/**
 * @generated from protobuf message runme.runner.v2.ResolveProgramRequest
 */
export interface ResolveProgramRequest {
    /**
     * use script for unnormalized cell content
     * whereas commands is for normalized shell commands
     *
     * @generated from protobuf oneof: source
     */
    source: {
        oneofKind: "commands";
        /**
         * commands are commands to be executed by the program.
         * The commands are joined and executed as a script.
         *
         * @generated from protobuf field: runme.runner.v2.ResolveProgramCommandList commands = 1
         */
        commands: ResolveProgramCommandList;
    } | {
        oneofKind: "script";
        /**
         * script is code to be executed by the program.
         * Individual lines are joined with the new line character.
         *
         * @generated from protobuf field: string script = 2
         */
        script: string;
    } | {
        oneofKind: undefined;
    };
    /**
     * mode determines how variables resolution occurs.
     * It is usually based on document or cell annotation config.
     *
     * @generated from protobuf field: runme.runner.v2.ResolveProgramRequest.Mode mode = 3
     */
    mode: ResolveProgramRequest_Mode;
    /**
     * env is a list of explicit environment variables that will be used
     * to resolve the environment variables found in the source.
     *
     * @generated from protobuf field: repeated string env = 4
     */
    env: string[];
    /**
     * session_id indicates which session is the source of
     * environment variables. If not provided, the most recent
     * session can be used using session_strategy.
     *
     * @generated from protobuf field: string session_id = 5
     */
    sessionId: string;
    /**
     * session_strategy is a strategy for selecting the session.
     *
     * @generated from protobuf field: runme.runner.v2.SessionStrategy session_strategy = 6
     */
    sessionStrategy: SessionStrategy;
    /**
     * project used to load environment variables from .env files.
     *
     * @generated from protobuf field: optional runme.runner.v2.Project project = 7
     */
    project?: Project;
    /**
     * language id associated with script.
     *
     * @generated from protobuf field: string language_id = 8
     */
    languageId: string;
    /**
     * retention determines how variables are retained once resolved.
     *
     * @generated from protobuf field: runme.runner.v2.ResolveProgramRequest.Retention retention = 9
     */
    retention: ResolveProgramRequest_Retention;
}
/**
 * @generated from protobuf enum runme.runner.v2.ResolveProgramRequest.Mode
 */
export declare enum ResolveProgramRequest_Mode {
    /**
     * unspecified is auto (default) which prompts for all
     * unresolved environment variables.
     * Subsequent runs will likely resolve via the session.
     *
     * @generated from protobuf enum value: MODE_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,
    /**
     * prompt always means to prompt for all environment variables.
     *
     * @generated from protobuf enum value: MODE_PROMPT_ALL = 1;
     */
    PROMPT_ALL = 1,
    /**
     * skip means to not prompt for any environment variables.
     * All variables will be marked as resolved.
     *
     * @generated from protobuf enum value: MODE_SKIP_ALL = 2;
     */
    SKIP_ALL = 2
}
/**
 * @generated from protobuf enum runme.runner.v2.ResolveProgramRequest.Retention
 */
export declare enum ResolveProgramRequest_Retention {
    /**
     * @generated from protobuf enum value: RETENTION_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,
    /**
     * first run means to always retain the first resolved value.
     *
     * @generated from protobuf enum value: RETENTION_FIRST_RUN = 1;
     */
    FIRST_RUN = 1,
    /**
     * last run means to always retain the last resolved value.
     *
     * @generated from protobuf enum value: RETENTION_LAST_RUN = 2;
     */
    LAST_RUN = 2
}
/**
 * @generated from protobuf message runme.runner.v2.ResolveProgramResponse
 */
export interface ResolveProgramResponse {
    /**
     * @generated from protobuf field: string script = 1
     */
    script: string;
    /**
     * use script until commands normalization is implemented
     *
     * @generated from protobuf field: runme.runner.v2.ResolveProgramCommandList commands = 2
     */
    commands?: ResolveProgramCommandList;
    /**
     * @generated from protobuf field: repeated runme.runner.v2.ResolveProgramResponse.VarResult vars = 3
     */
    vars: ResolveProgramResponse_VarResult[];
}
/**
 * @generated from protobuf message runme.runner.v2.ResolveProgramResponse.VarResult
 */
export interface ResolveProgramResponse_VarResult {
    /**
     * prompt indicates the resolution status of the env variable.
     *
     * @generated from protobuf field: runme.runner.v2.ResolveProgramResponse.Status status = 1
     */
    status: ResolveProgramResponse_Status;
    /**
     * name is the name of the environment variable.
     *
     * @generated from protobuf field: string name = 2
     */
    name: string;
    /**
     * original_value is a default value of the environment variable.
     * It might be a value that is assigned to the variable in the script,
     * like FOO=bar or FOO=${FOO:-bar}.
     * If the variable is not assigned, it is an empty string.
     *
     * @generated from protobuf field: string original_value = 3
     */
    originalValue: string;
    /**
     * resolved_value is a value of the environment variable resolved from a source.
     * If it is an empty string, it means that the environment variable is not resolved.
     *
     * @generated from protobuf field: string resolved_value = 4
     */
    resolvedValue: string;
}
/**
 * @generated from protobuf enum runme.runner.v2.ResolveProgramResponse.Status
 */
export declare enum ResolveProgramResponse_Status {
    /**
     * unspecified is the default value and it means unresolved.
     *
     * @generated from protobuf enum value: STATUS_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,
    /**
     * resolved means that the variable is resolved.
     *
     * @generated from protobuf enum value: STATUS_RESOLVED = 1;
     */
    RESOLVED = 1,
    /**
     * unresolved with message means that the variable is unresolved
     * but it contains a message. E.g. FOO=this is message.
     *
     * @generated from protobuf enum value: STATUS_UNRESOLVED_WITH_MESSAGE = 2;
     */
    UNRESOLVED_WITH_MESSAGE = 2,
    /**
     * unresolved with placeholder means that the variable is unresolved
     * but it contains a placeholder. E.g. FOO="this is placeholder".
     *
     * @generated from protobuf enum value: STATUS_UNRESOLVED_WITH_PLACEHOLDER = 3;
     */
    UNRESOLVED_WITH_PLACEHOLDER = 3,
    /**
     * unresolved with secret means that the variable is unresolved
     * and it requires treatment as a secret.
     *
     * @generated from protobuf enum value: STATUS_UNRESOLVED_WITH_SECRET = 4;
     */
    UNRESOLVED_WITH_SECRET = 4
}
/**
 * @generated from protobuf message runme.runner.v2.MonitorEnvStoreRequest
 */
export interface MonitorEnvStoreRequest {
    /**
     * @generated from protobuf field: runme.runner.v2.Session session = 1
     */
    session?: Session;
}
/**
 * @generated from protobuf message runme.runner.v2.MonitorEnvStoreResponseSnapshot
 */
export interface MonitorEnvStoreResponseSnapshot {
    /**
     * @generated from protobuf field: repeated runme.runner.v2.MonitorEnvStoreResponseSnapshot.SnapshotEnv envs = 1
     */
    envs: MonitorEnvStoreResponseSnapshot_SnapshotEnv[];
}
/**
 * @generated from protobuf message runme.runner.v2.MonitorEnvStoreResponseSnapshot.SnapshotEnv
 */
export interface MonitorEnvStoreResponseSnapshot_SnapshotEnv {
    /**
     * @generated from protobuf field: runme.runner.v2.MonitorEnvStoreResponseSnapshot.Status status = 1
     */
    status: MonitorEnvStoreResponseSnapshot_Status;
    /**
     * @generated from protobuf field: string name = 2
     */
    name: string;
    /**
     * @generated from protobuf field: string description = 3
     */
    description: string;
    /**
     * @generated from protobuf field: string spec = 4
     */
    spec: string;
    /**
     * @generated from protobuf field: bool is_required = 5
     */
    isRequired: boolean;
    /**
     * @generated from protobuf field: string origin = 6
     */
    origin: string;
    /**
     * @generated from protobuf field: string original_value = 7
     */
    originalValue: string;
    /**
     * @generated from protobuf field: string resolved_value = 8
     */
    resolvedValue: string;
    /**
     * @generated from protobuf field: string create_time = 9
     */
    createTime: string;
    /**
     * @generated from protobuf field: string update_time = 10
     */
    updateTime: string;
    /**
     * @generated from protobuf field: repeated runme.runner.v2.MonitorEnvStoreResponseSnapshot.Error errors = 11
     */
    errors: MonitorEnvStoreResponseSnapshot_Error[];
}
/**
 * @generated from protobuf message runme.runner.v2.MonitorEnvStoreResponseSnapshot.Error
 */
export interface MonitorEnvStoreResponseSnapshot_Error {
    /**
     * @generated from protobuf field: uint32 code = 1
     */
    code: number;
    /**
     * @generated from protobuf field: string message = 2
     */
    message: string;
}
/**
 * @generated from protobuf enum runme.runner.v2.MonitorEnvStoreResponseSnapshot.Status
 */
export declare enum MonitorEnvStoreResponseSnapshot_Status {
    /**
     * @generated from protobuf enum value: STATUS_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,
    /**
     * @generated from protobuf enum value: STATUS_LITERAL = 1;
     */
    LITERAL = 1,
    /**
     * @generated from protobuf enum value: STATUS_HIDDEN = 2;
     */
    HIDDEN = 2,
    /**
     * @generated from protobuf enum value: STATUS_MASKED = 3;
     */
    MASKED = 3
}
/**
 * @generated from protobuf message runme.runner.v2.MonitorEnvStoreResponse
 */
export interface MonitorEnvStoreResponse {
    /**
     * @generated from protobuf field: runme.runner.v2.MonitorEnvStoreType type = 1
     */
    type: MonitorEnvStoreType;
    /**
     * @generated from protobuf oneof: data
     */
    data: {
        oneofKind: "snapshot";
        /**
         * @generated from protobuf field: runme.runner.v2.MonitorEnvStoreResponseSnapshot snapshot = 2
         */
        snapshot: MonitorEnvStoreResponseSnapshot;
    } | {
        oneofKind: undefined;
    };
}
/**
 * env store implementation
 *
 * @generated from protobuf enum runme.runner.v2.SessionEnvStoreType
 */
export declare enum SessionEnvStoreType {
    /**
     * uses default env store
     *
     * @generated from protobuf enum value: SESSION_ENV_STORE_TYPE_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,
    /**
     * uses owl store
     *
     * @generated from protobuf enum value: SESSION_ENV_STORE_TYPE_OWL = 1;
     */
    OWL = 1
}
/**
 * @generated from protobuf enum runme.runner.v2.ExecuteStop
 */
export declare enum ExecuteStop {
    /**
     * @generated from protobuf enum value: EXECUTE_STOP_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,
    /**
     * @generated from protobuf enum value: EXECUTE_STOP_INTERRUPT = 1;
     */
    INTERRUPT = 1,
    /**
     * @generated from protobuf enum value: EXECUTE_STOP_KILL = 2;
     */
    KILL = 2
}
/**
 * SessionStrategy determines a session selection in
 * an initial execute request.
 *
 * @generated from protobuf enum runme.runner.v2.SessionStrategy
 */
export declare enum SessionStrategy {
    /**
     * Uses the session_id field to determine the session.
     * If none is present, a new session is created.
     *
     * @generated from protobuf enum value: SESSION_STRATEGY_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,
    /**
     * Uses the most recent session on the server.
     * If there is none, a new one is created.
     *
     * @generated from protobuf enum value: SESSION_STRATEGY_MOST_RECENT = 1;
     */
    MOST_RECENT = 1
}
/**
 * @generated from protobuf enum runme.runner.v2.MonitorEnvStoreType
 */
export declare enum MonitorEnvStoreType {
    /**
     * @generated from protobuf enum value: MONITOR_ENV_STORE_TYPE_UNSPECIFIED = 0;
     */
    UNSPECIFIED = 0,
    /**
     * possible expansion to have a "timeline" view
     * MONITOR_ENV_STORE_TYPE_TIMELINE = 2;
     *
     * @generated from protobuf enum value: MONITOR_ENV_STORE_TYPE_SNAPSHOT = 1;
     */
    SNAPSHOT = 1
}
declare class Project$Type extends MessageType<Project> {
    constructor();
    create(value?: PartialMessage<Project>): Project;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: Project): Project;
    internalBinaryWrite(message: Project, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.Project
 */
export declare const Project: Project$Type;
declare class Session$Type extends MessageType<Session> {
    constructor();
    create(value?: PartialMessage<Session>): Session;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: Session): Session;
    private binaryReadMap3;
    internalBinaryWrite(message: Session, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.Session
 */
export declare const Session: Session$Type;
declare class CreateSessionRequest$Type extends MessageType<CreateSessionRequest> {
    constructor();
    create(value?: PartialMessage<CreateSessionRequest>): CreateSessionRequest;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: CreateSessionRequest): CreateSessionRequest;
    private binaryReadMap1;
    internalBinaryWrite(message: CreateSessionRequest, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.CreateSessionRequest
 */
export declare const CreateSessionRequest: CreateSessionRequest$Type;
declare class CreateSessionRequest_Config$Type extends MessageType<CreateSessionRequest_Config> {
    constructor();
    create(value?: PartialMessage<CreateSessionRequest_Config>): CreateSessionRequest_Config;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: CreateSessionRequest_Config): CreateSessionRequest_Config;
    internalBinaryWrite(message: CreateSessionRequest_Config, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.CreateSessionRequest.Config
 */
export declare const CreateSessionRequest_Config: CreateSessionRequest_Config$Type;
declare class CreateSessionResponse$Type extends MessageType<CreateSessionResponse> {
    constructor();
    create(value?: PartialMessage<CreateSessionResponse>): CreateSessionResponse;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: CreateSessionResponse): CreateSessionResponse;
    internalBinaryWrite(message: CreateSessionResponse, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.CreateSessionResponse
 */
export declare const CreateSessionResponse: CreateSessionResponse$Type;
declare class GetSessionRequest$Type extends MessageType<GetSessionRequest> {
    constructor();
    create(value?: PartialMessage<GetSessionRequest>): GetSessionRequest;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: GetSessionRequest): GetSessionRequest;
    internalBinaryWrite(message: GetSessionRequest, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.GetSessionRequest
 */
export declare const GetSessionRequest: GetSessionRequest$Type;
declare class GetSessionResponse$Type extends MessageType<GetSessionResponse> {
    constructor();
    create(value?: PartialMessage<GetSessionResponse>): GetSessionResponse;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: GetSessionResponse): GetSessionResponse;
    internalBinaryWrite(message: GetSessionResponse, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.GetSessionResponse
 */
export declare const GetSessionResponse: GetSessionResponse$Type;
declare class ListSessionsRequest$Type extends MessageType<ListSessionsRequest> {
    constructor();
    create(value?: PartialMessage<ListSessionsRequest>): ListSessionsRequest;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: ListSessionsRequest): ListSessionsRequest;
    internalBinaryWrite(message: ListSessionsRequest, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.ListSessionsRequest
 */
export declare const ListSessionsRequest: ListSessionsRequest$Type;
declare class ListSessionsResponse$Type extends MessageType<ListSessionsResponse> {
    constructor();
    create(value?: PartialMessage<ListSessionsResponse>): ListSessionsResponse;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: ListSessionsResponse): ListSessionsResponse;
    internalBinaryWrite(message: ListSessionsResponse, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.ListSessionsResponse
 */
export declare const ListSessionsResponse: ListSessionsResponse$Type;
declare class UpdateSessionRequest$Type extends MessageType<UpdateSessionRequest> {
    constructor();
    create(value?: PartialMessage<UpdateSessionRequest>): UpdateSessionRequest;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: UpdateSessionRequest): UpdateSessionRequest;
    private binaryReadMap2;
    internalBinaryWrite(message: UpdateSessionRequest, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.UpdateSessionRequest
 */
export declare const UpdateSessionRequest: UpdateSessionRequest$Type;
declare class UpdateSessionResponse$Type extends MessageType<UpdateSessionResponse> {
    constructor();
    create(value?: PartialMessage<UpdateSessionResponse>): UpdateSessionResponse;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: UpdateSessionResponse): UpdateSessionResponse;
    internalBinaryWrite(message: UpdateSessionResponse, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.UpdateSessionResponse
 */
export declare const UpdateSessionResponse: UpdateSessionResponse$Type;
declare class DeleteSessionRequest$Type extends MessageType<DeleteSessionRequest> {
    constructor();
    create(value?: PartialMessage<DeleteSessionRequest>): DeleteSessionRequest;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: DeleteSessionRequest): DeleteSessionRequest;
    internalBinaryWrite(message: DeleteSessionRequest, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.DeleteSessionRequest
 */
export declare const DeleteSessionRequest: DeleteSessionRequest$Type;
declare class DeleteSessionResponse$Type extends MessageType<DeleteSessionResponse> {
    constructor();
    create(value?: PartialMessage<DeleteSessionResponse>): DeleteSessionResponse;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: DeleteSessionResponse): DeleteSessionResponse;
    internalBinaryWrite(message: DeleteSessionResponse, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.DeleteSessionResponse
 */
export declare const DeleteSessionResponse: DeleteSessionResponse$Type;
declare class Winsize$Type extends MessageType<Winsize> {
    constructor();
    create(value?: PartialMessage<Winsize>): Winsize;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: Winsize): Winsize;
    internalBinaryWrite(message: Winsize, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.Winsize
 */
export declare const Winsize: Winsize$Type;
declare class ExecuteRequest$Type extends MessageType<ExecuteRequest> {
    constructor();
    create(value?: PartialMessage<ExecuteRequest>): ExecuteRequest;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: ExecuteRequest): ExecuteRequest;
    internalBinaryWrite(message: ExecuteRequest, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.ExecuteRequest
 */
export declare const ExecuteRequest: ExecuteRequest$Type;
declare class ExecuteResponse$Type extends MessageType<ExecuteResponse> {
    constructor();
    create(value?: PartialMessage<ExecuteResponse>): ExecuteResponse;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: ExecuteResponse): ExecuteResponse;
    internalBinaryWrite(message: ExecuteResponse, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.ExecuteResponse
 */
export declare const ExecuteResponse: ExecuteResponse$Type;
declare class ResolveProgramCommandList$Type extends MessageType<ResolveProgramCommandList> {
    constructor();
    create(value?: PartialMessage<ResolveProgramCommandList>): ResolveProgramCommandList;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: ResolveProgramCommandList): ResolveProgramCommandList;
    internalBinaryWrite(message: ResolveProgramCommandList, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.ResolveProgramCommandList
 */
export declare const ResolveProgramCommandList: ResolveProgramCommandList$Type;
declare class ResolveProgramRequest$Type extends MessageType<ResolveProgramRequest> {
    constructor();
    create(value?: PartialMessage<ResolveProgramRequest>): ResolveProgramRequest;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: ResolveProgramRequest): ResolveProgramRequest;
    internalBinaryWrite(message: ResolveProgramRequest, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.ResolveProgramRequest
 */
export declare const ResolveProgramRequest: ResolveProgramRequest$Type;
declare class ResolveProgramResponse$Type extends MessageType<ResolveProgramResponse> {
    constructor();
    create(value?: PartialMessage<ResolveProgramResponse>): ResolveProgramResponse;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: ResolveProgramResponse): ResolveProgramResponse;
    internalBinaryWrite(message: ResolveProgramResponse, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.ResolveProgramResponse
 */
export declare const ResolveProgramResponse: ResolveProgramResponse$Type;
declare class ResolveProgramResponse_VarResult$Type extends MessageType<ResolveProgramResponse_VarResult> {
    constructor();
    create(value?: PartialMessage<ResolveProgramResponse_VarResult>): ResolveProgramResponse_VarResult;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: ResolveProgramResponse_VarResult): ResolveProgramResponse_VarResult;
    internalBinaryWrite(message: ResolveProgramResponse_VarResult, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.ResolveProgramResponse.VarResult
 */
export declare const ResolveProgramResponse_VarResult: ResolveProgramResponse_VarResult$Type;
declare class MonitorEnvStoreRequest$Type extends MessageType<MonitorEnvStoreRequest> {
    constructor();
    create(value?: PartialMessage<MonitorEnvStoreRequest>): MonitorEnvStoreRequest;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: MonitorEnvStoreRequest): MonitorEnvStoreRequest;
    internalBinaryWrite(message: MonitorEnvStoreRequest, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.MonitorEnvStoreRequest
 */
export declare const MonitorEnvStoreRequest: MonitorEnvStoreRequest$Type;
declare class MonitorEnvStoreResponseSnapshot$Type extends MessageType<MonitorEnvStoreResponseSnapshot> {
    constructor();
    create(value?: PartialMessage<MonitorEnvStoreResponseSnapshot>): MonitorEnvStoreResponseSnapshot;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: MonitorEnvStoreResponseSnapshot): MonitorEnvStoreResponseSnapshot;
    internalBinaryWrite(message: MonitorEnvStoreResponseSnapshot, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.MonitorEnvStoreResponseSnapshot
 */
export declare const MonitorEnvStoreResponseSnapshot: MonitorEnvStoreResponseSnapshot$Type;
declare class MonitorEnvStoreResponseSnapshot_SnapshotEnv$Type extends MessageType<MonitorEnvStoreResponseSnapshot_SnapshotEnv> {
    constructor();
    create(value?: PartialMessage<MonitorEnvStoreResponseSnapshot_SnapshotEnv>): MonitorEnvStoreResponseSnapshot_SnapshotEnv;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: MonitorEnvStoreResponseSnapshot_SnapshotEnv): MonitorEnvStoreResponseSnapshot_SnapshotEnv;
    internalBinaryWrite(message: MonitorEnvStoreResponseSnapshot_SnapshotEnv, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.MonitorEnvStoreResponseSnapshot.SnapshotEnv
 */
export declare const MonitorEnvStoreResponseSnapshot_SnapshotEnv: MonitorEnvStoreResponseSnapshot_SnapshotEnv$Type;
declare class MonitorEnvStoreResponseSnapshot_Error$Type extends MessageType<MonitorEnvStoreResponseSnapshot_Error> {
    constructor();
    create(value?: PartialMessage<MonitorEnvStoreResponseSnapshot_Error>): MonitorEnvStoreResponseSnapshot_Error;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: MonitorEnvStoreResponseSnapshot_Error): MonitorEnvStoreResponseSnapshot_Error;
    internalBinaryWrite(message: MonitorEnvStoreResponseSnapshot_Error, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.MonitorEnvStoreResponseSnapshot.Error
 */
export declare const MonitorEnvStoreResponseSnapshot_Error: MonitorEnvStoreResponseSnapshot_Error$Type;
declare class MonitorEnvStoreResponse$Type extends MessageType<MonitorEnvStoreResponse> {
    constructor();
    create(value?: PartialMessage<MonitorEnvStoreResponse>): MonitorEnvStoreResponse;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: MonitorEnvStoreResponse): MonitorEnvStoreResponse;
    internalBinaryWrite(message: MonitorEnvStoreResponse, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.runner.v2.MonitorEnvStoreResponse
 */
export declare const MonitorEnvStoreResponse: MonitorEnvStoreResponse$Type;
/**
 * @generated ServiceType for protobuf service runme.runner.v2.RunnerService
 */
export declare const RunnerService: any;
export {};
