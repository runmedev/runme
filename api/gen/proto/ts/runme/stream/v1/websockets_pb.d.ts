/* eslint-disable */
// @generated by protobuf-ts 2.11.1 with parameter output_javascript,optimize_code_size,long_type_string,add_pb_suffix,ts_nocheck,eslint_disable
// @generated from protobuf file "runme/stream/v1/websockets.proto" (package "runme.stream.v1", syntax proto3)
// tslint:disable
// @ts-nocheck
import type { BinaryWriteOptions } from "@protobuf-ts/runtime";
import type { IBinaryWriter } from "@protobuf-ts/runtime";
import type { BinaryReadOptions } from "@protobuf-ts/runtime";
import type { IBinaryReader } from "@protobuf-ts/runtime";
import type { PartialMessage } from "@protobuf-ts/runtime";
import { MessageType } from "@protobuf-ts/runtime";
import { ExecuteResponse } from "../../runner/v2/runner_pb";
import { ExecuteRequest } from "../../runner/v2/runner_pb";
import { Code } from "../../../google/rpc/code_pb";
/**
 * Represents websocket-level status (e.g., for auth, protocol, or other errors).
 *
 * @generated from protobuf message runme.stream.v1.WebsocketStatus
 */
export interface WebsocketStatus {
    /**
     * @generated from protobuf field: google.rpc.Code code = 1
     */
    code: Code;
    /**
     * @generated from protobuf field: string message = 2
     */
    message: string;
}
/**
 * Ping message for protocol-level keep-alive
 *
 * @generated from protobuf message runme.stream.v1.Ping
 */
export interface Ping {
    /**
     * @generated from protobuf field: int64 timestamp = 1
     */
    timestamp: string;
}
/**
 * Pong message for protocol-level keep-alive response
 *
 * @generated from protobuf message runme.stream.v1.Pong
 */
export interface Pong {
    /**
     * @generated from protobuf field: int64 timestamp = 1
     */
    timestamp: string;
}
/**
 * WebsocketRequest defines the message sent by the client over a websocket.
 * The request is a union of types that indicate the type of message.
 *
 * @generated from protobuf message runme.stream.v1.WebsocketRequest
 */
export interface WebsocketRequest {
    /**
     * @generated from protobuf oneof: payload
     */
    payload: {
        oneofKind: "executeRequest";
        /**
         * @generated from protobuf field: runme.runner.v2.ExecuteRequest execute_request = 1
         */
        executeRequest: ExecuteRequest;
    } | {
        oneofKind: undefined;
    };
    /**
     * Protocol-level ping for frontend heartbeat. Unlike websocket servers which
     * have a spec-integral heartbeat (https://developer.mozilla.org/en-US/docs/Web/API/WebWebsockets_API/Writing_WebWebsocket_servers#pings_and_pongs_the_heartbeat_of_websockets),
     * we need to specify our own to cover client->server. The integral heartbeat
     * only works server->client and the browser sandbox is not privy to it.
     * Once the server receives a ping, it will send a pong response with the
     * exact same timestamp.
     *
     * @generated from protobuf field: runme.stream.v1.Ping ping = 100
     */
    ping?: Ping;
    /**
     * Optional authorization header, similar to the HTTP Authorization header.
     *
     * @generated from protobuf field: string authorization = 200
     */
    authorization: string;
    /**
     * Optional Known ID to track the origin cell/block of the request.
     *
     * @generated from protobuf field: string known_id = 210
     */
    knownId: string;
    /**
     * Optional Run ID to track and resume execution.
     *
     * @generated from protobuf field: string run_id = 220
     */
    runId: string;
}
/**
 * WebsocketResponse defines the message sent by the server over a websocket.
 * The response is a union of types that indicate the type of message.
 *
 * @generated from protobuf message runme.stream.v1.WebsocketResponse
 */
export interface WebsocketResponse {
    /**
     * @generated from protobuf oneof: payload
     */
    payload: {
        oneofKind: "executeResponse";
        /**
         * @generated from protobuf field: runme.runner.v2.ExecuteResponse execute_response = 1
         */
        executeResponse: ExecuteResponse;
    } | {
        oneofKind: undefined;
    };
    /**
     * Protocol-level pong for frontend heartbeat. Once the server receives
     * a ping, it will send a pong response with the exact same timestamp.
     * This allows the frontend (client) to detect if the connection is
     * still alive or stale/inactive. See WebsocketRequest's ping for more details.
     *
     * @generated from protobuf field: runme.stream.v1.Pong pong = 100
     */
    pong?: Pong;
    /**
     * Optional websocket-level status.
     *
     * @generated from protobuf field: runme.stream.v1.WebsocketStatus status = 200
     */
    status?: WebsocketStatus;
    /**
     * Optional Known ID to track the origin cell/block of the request.
     *
     * @generated from protobuf field: string known_id = 210
     */
    knownId: string;
    /**
     * Optional Run ID to track and resume execution.
     *
     * @generated from protobuf field: string run_id = 220
     */
    runId: string;
}
declare class WebsocketStatus$Type extends MessageType<WebsocketStatus> {
    constructor();
    create(value?: PartialMessage<WebsocketStatus>): WebsocketStatus;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: WebsocketStatus): WebsocketStatus;
    internalBinaryWrite(message: WebsocketStatus, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.stream.v1.WebsocketStatus
 */
export declare const WebsocketStatus: WebsocketStatus$Type;
declare class Ping$Type extends MessageType<Ping> {
    constructor();
    create(value?: PartialMessage<Ping>): Ping;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: Ping): Ping;
    internalBinaryWrite(message: Ping, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.stream.v1.Ping
 */
export declare const Ping: Ping$Type;
declare class Pong$Type extends MessageType<Pong> {
    constructor();
    create(value?: PartialMessage<Pong>): Pong;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: Pong): Pong;
    internalBinaryWrite(message: Pong, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.stream.v1.Pong
 */
export declare const Pong: Pong$Type;
declare class WebsocketRequest$Type extends MessageType<WebsocketRequest> {
    constructor();
    create(value?: PartialMessage<WebsocketRequest>): WebsocketRequest;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: WebsocketRequest): WebsocketRequest;
    internalBinaryWrite(message: WebsocketRequest, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.stream.v1.WebsocketRequest
 */
export declare const WebsocketRequest: WebsocketRequest$Type;
declare class WebsocketResponse$Type extends MessageType<WebsocketResponse> {
    constructor();
    create(value?: PartialMessage<WebsocketResponse>): WebsocketResponse;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: WebsocketResponse): WebsocketResponse;
    internalBinaryWrite(message: WebsocketResponse, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message runme.stream.v1.WebsocketResponse
 */
export declare const WebsocketResponse: WebsocketResponse$Type;
export {};
