/* eslint-disable */
// @generated by protobuf-ts 2.11.1 with parameter output_javascript,optimize_code_size,long_type_string,add_pb_suffix,ts_nocheck,eslint_disable
// @generated from protobuf file "runme/reporter/v1alpha1/reporter.proto" (package "runme.reporter.v1alpha1", syntax proto3)
// tslint:disable
// @ts-nocheck
import type { RpcTransport } from "@protobuf-ts/runtime-rpc";
import type { ServiceInfo } from "@protobuf-ts/runtime-rpc";
import type { TransformResponse } from "./reporter_pb";
import type { TransformRequest } from "./reporter_pb";
import type { UnaryCall } from "@protobuf-ts/runtime-rpc";
import type { RpcOptions } from "@protobuf-ts/runtime-rpc";
/**
 * @generated from protobuf service runme.reporter.v1alpha1.ReporterService
 */
export interface IReporterServiceClient {
    /**
     * @generated from protobuf rpc: Transform
     */
    transform(input: TransformRequest, options?: RpcOptions): UnaryCall<TransformRequest, TransformResponse>;
}
/**
 * @generated from protobuf service runme.reporter.v1alpha1.ReporterService
 */
export declare class ReporterServiceClient implements IReporterServiceClient, ServiceInfo {
    private readonly _transport;
    typeName: any;
    methods: any;
    options: any;
    constructor(_transport: RpcTransport);
    /**
     * @generated from protobuf rpc: Transform
     */
    transform(input: TransformRequest, options?: RpcOptions): UnaryCall<TransformRequest, TransformResponse>;
}
