/* eslint-disable */
// @generated by protobuf-ts 2.11.1 with parameter output_javascript,optimize_code_size,long_type_string,add_pb_suffix,ts_nocheck,eslint_disable
// @generated from protobuf file "agent/blocks.proto" (syntax proto3)
// tslint:disable
// @ts-nocheck
import type { BinaryWriteOptions } from "@protobuf-ts/runtime";
import type { IBinaryWriter } from "@protobuf-ts/runtime";
import type { BinaryReadOptions } from "@protobuf-ts/runtime";
import type { IBinaryReader } from "@protobuf-ts/runtime";
import type { PartialMessage } from "@protobuf-ts/runtime";
import { MessageType } from "@protobuf-ts/runtime";
import { FileSearchResult } from "./filesearch_pb";
/**
 * Block represents the data in an element in the UI.
 *
 * @generated from protobuf message Block
 */
export interface Block {
    /**
     * BlockKind is an enum indicating what type of block it is e.g text or output
     *
     * @generated from protobuf field: BlockKind kind = 1
     */
    kind: BlockKind;
    /**
     * language is a string identifying the language.
     *
     * @generated from protobuf field: string language = 2
     */
    language: string;
    /**
     * contents is the actual contents of the block.
     * Not the outputs of the block.
     *
     * @generated from protobuf field: string contents = 3
     */
    contents: string;
    /**
     * ID of the block.
     *
     * @generated from protobuf field: string id = 7
     */
    id: string;
    /**
     * Additional metadata
     *
     * @generated from protobuf field: map<string, string> metadata = 8
     */
    metadata: {
        [key: string]: string;
    };
    /**
     * @generated from protobuf field: BlockRole role = 9
     */
    role: BlockRole;
    /**
     * @generated from protobuf field: repeated FileSearchResult file_search_results = 10
     */
    fileSearchResults: FileSearchResult[];
    /**
     * @generated from protobuf field: repeated BlockOutput outputs = 11
     */
    outputs: BlockOutput[];
    /**
     * Call ID is the id of this function call as set by OpenAI
     *
     * @generated from protobuf field: string call_id = 12
     */
    callId: string;
}
/**
 * BlockOutput represents the output of a block.
 * It corresponds to a VSCode NotebookCellOutput
 * https://github.com/microsoft/vscode/blob/98332892fd2cb3c948ced33f542698e20c6279b9/src/vscode-dts/vscode.d.ts#L14835
 *
 * @generated from protobuf message BlockOutput
 */
export interface BlockOutput {
    /**
     * items is the output items. Each item is the different representation of the same output data
     *
     * @generated from protobuf field: repeated BlockOutputItem items = 1
     */
    items: BlockOutputItem[];
    /**
     * @generated from protobuf field: BlockOutputKind kind = 2
     */
    kind: BlockOutputKind;
}
/**
 * BlockOutputItem represents an item in a block output.
 * It corresponds to a VSCode NotebookCellOutputItem
 * https://github.com/microsoft/vscode/blob/98332892fd2cb3c948ced33f542698e20c6279b9/src/vscode-dts/vscode.d.ts#L14753
 *
 * @generated from protobuf message BlockOutputItem
 */
export interface BlockOutputItem {
    /**
     * mime is the mime type of the output item.
     *
     * @generated from protobuf field: string mime = 1
     */
    mime: string;
    /**
     * value of the output item.
     * We use string data type and not bytes because the JSON representation of bytes is a base64
     * string. vscode data uses a byte. We may need to add support for bytes to support non text data
     * data in the future.
     *
     * @generated from protobuf field: string text_data = 2
     */
    textData: string;
}
/**
 * @generated from protobuf message GenerateRequest
 */
export interface GenerateRequest {
    /**
     * @generated from protobuf field: repeated Block blocks = 1
     */
    blocks: Block[];
    /**
     * @generated from protobuf field: string previous_response_id = 2
     */
    previousResponseId: string;
    /**
     * openai_access_token is the OpenAI access token to use when contacting the OpenAI API.
     *
     * @generated from protobuf field: string openai_access_token = 3
     */
    openaiAccessToken: string;
}
/**
 * @generated from protobuf message GenerateResponse
 */
export interface GenerateResponse {
    /**
     * @generated from protobuf field: repeated Block blocks = 1
     */
    blocks: Block[];
    /**
     * @generated from protobuf field: string response_id = 2
     */
    responseId: string;
}
/**
 * @generated from protobuf enum BlockKind
 */
export declare enum BlockKind {
    /**
     * @generated from protobuf enum value: UNKNOWN_BLOCK_KIND = 0;
     */
    UNKNOWN_BLOCK_KIND = 0,
    /**
     * @generated from protobuf enum value: MARKUP = 1;
     */
    MARKUP = 1,
    /**
     * @generated from protobuf enum value: CODE = 2;
     */
    CODE = 2,
    /**
     * @generated from protobuf enum value: FILE_SEARCH_RESULTS = 3;
     */
    FILE_SEARCH_RESULTS = 3
}
/**
 * @generated from protobuf enum BlockRole
 */
export declare enum BlockRole {
    /**
     * @generated from protobuf enum value: BLOCK_ROLE_UNKNOWN = 0;
     */
    UNKNOWN = 0,
    /**
     * @generated from protobuf enum value: BLOCK_ROLE_USER = 1;
     */
    USER = 1,
    /**
     * @generated from protobuf enum value: BLOCK_ROLE_ASSISTANT = 2;
     */
    ASSISTANT = 2
}
/**
 * @generated from protobuf enum BlockOutputKind
 */
export declare enum BlockOutputKind {
    /**
     * @generated from protobuf enum value: UNKNOWN_BLOCK_OUTPUT_KIND = 0;
     */
    UNKNOWN_BLOCK_OUTPUT_KIND = 0,
    /**
     * @generated from protobuf enum value: STDOUT = 1;
     */
    STDOUT = 1,
    /**
     * @generated from protobuf enum value: STDERR = 2;
     */
    STDERR = 2
}
declare class Block$Type extends MessageType<Block> {
    constructor();
    create(value?: PartialMessage<Block>): Block;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: Block): Block;
    private binaryReadMap8;
    internalBinaryWrite(message: Block, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message Block
 */
export declare const Block: Block$Type;
declare class BlockOutput$Type extends MessageType<BlockOutput> {
    constructor();
    create(value?: PartialMessage<BlockOutput>): BlockOutput;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: BlockOutput): BlockOutput;
    internalBinaryWrite(message: BlockOutput, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message BlockOutput
 */
export declare const BlockOutput: BlockOutput$Type;
declare class BlockOutputItem$Type extends MessageType<BlockOutputItem> {
    constructor();
    create(value?: PartialMessage<BlockOutputItem>): BlockOutputItem;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: BlockOutputItem): BlockOutputItem;
    internalBinaryWrite(message: BlockOutputItem, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message BlockOutputItem
 */
export declare const BlockOutputItem: BlockOutputItem$Type;
declare class GenerateRequest$Type extends MessageType<GenerateRequest> {
    constructor();
    create(value?: PartialMessage<GenerateRequest>): GenerateRequest;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: GenerateRequest): GenerateRequest;
    internalBinaryWrite(message: GenerateRequest, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message GenerateRequest
 */
export declare const GenerateRequest: GenerateRequest$Type;
declare class GenerateResponse$Type extends MessageType<GenerateResponse> {
    constructor();
    create(value?: PartialMessage<GenerateResponse>): GenerateResponse;
    internalBinaryRead(reader: IBinaryReader, length: number, options: BinaryReadOptions, target?: GenerateResponse): GenerateResponse;
    internalBinaryWrite(message: GenerateResponse, writer: IBinaryWriter, options: BinaryWriteOptions): IBinaryWriter;
}
/**
 * @generated MessageType for protobuf message GenerateResponse
 */
export declare const GenerateResponse: GenerateResponse$Type;
/**
 * @generated ServiceType for protobuf service BlocksService
 */
export declare const BlocksService: any;
export {};
