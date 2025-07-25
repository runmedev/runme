// @generated by protoc-gen-es v2.6.0 with parameter "target=js+dts,import_extension=none,json_types=true"
// @generated from file agent/v1/webapp.proto (package agent.v1, syntax proto3)
/* eslint-disable */

import type { GenFile, GenMessage } from "@bufbuild/protobuf/codegenv2";
import type { Message } from "@bufbuild/protobuf";

/**
 * Describes the file agent/v1/webapp.proto.
 */
export declare const file_agent_v1_webapp: GenFile;

/**
 * WebAppConfig is the application configuration.
 *
 * @generated from message agent.v1.WebAppConfig
 */
export declare type WebAppConfig = Message<"agent.v1.WebAppConfig"> & {
  /**
   * runner is the address of the runner that the application should use.
   *
   * @generated from field: string runner = 1;
   */
  runner: string;

  /**
   * Reconnect is a flag to enable automatic reconnecting to the runner.
   *
   * @generated from field: optional bool reconnect = 2;
   */
  reconnect?: boolean;

  /**
   * InvertedOrder is a flag to invert the order of the blocks.
   *
   * @generated from field: optional bool inverted_order = 3;
   */
  invertedOrder?: boolean;
};

/**
 * WebAppConfig is the application configuration.
 *
 * @generated from message agent.v1.WebAppConfig
 */
export declare type WebAppConfigJson = {
  /**
   * runner is the address of the runner that the application should use.
   *
   * @generated from field: string runner = 1;
   */
  runner?: string;

  /**
   * Reconnect is a flag to enable automatic reconnecting to the runner.
   *
   * @generated from field: optional bool reconnect = 2;
   */
  reconnect?: boolean;

  /**
   * InvertedOrder is a flag to invert the order of the blocks.
   *
   * @generated from field: optional bool inverted_order = 3;
   */
  invertedOrder?: boolean;
};

/**
 * Describes the message agent.v1.WebAppConfig.
 * Use `create(WebAppConfigSchema)` to create a new message.
 */
export declare const WebAppConfigSchema: GenMessage<WebAppConfig, {jsonType: WebAppConfigJson}>;
