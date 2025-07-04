/* eslint-disable */
// @generated by protobuf-ts 2.11.1 with parameter output_javascript,optimize_code_size,long_type_string,add_pb_suffix,ts_nocheck,eslint_disable
// @generated from protobuf file "agent/filesearch.proto" (syntax proto3)
// tslint:disable
// @ts-nocheck
import { WireType } from "@protobuf-ts/runtime";
import { UnknownFieldHandler } from "@protobuf-ts/runtime";
import { reflectionMergePartial } from "@protobuf-ts/runtime";
import { MessageType } from "@protobuf-ts/runtime";
// @generated message type with reflection information, may provide speed optimized methods
class FileSearchResult$Type extends MessageType {
    constructor() {
        super("FileSearchResult", [
            { no: 1, name: "FileID", kind: "scalar", jsonName: "FileID", T: 9 /*ScalarType.STRING*/ },
            { no: 2, name: "FileName", kind: "scalar", jsonName: "FileName", T: 9 /*ScalarType.STRING*/ },
            { no: 3, name: "Score", kind: "scalar", jsonName: "Score", T: 1 /*ScalarType.DOUBLE*/ },
            { no: 4, name: "Link", kind: "scalar", jsonName: "Link", T: 9 /*ScalarType.STRING*/ }
        ]);
    }
    create(value) {
        const message = globalThis.Object.create((this.messagePrototype));
        message.fileID = "";
        message.fileName = "";
        message.score = 0;
        message.link = "";
        if (value !== undefined)
            reflectionMergePartial(this, message, value);
        return message;
    }
    internalBinaryRead(reader, length, options, target) {
        let message = target ?? this.create(), end = reader.pos + length;
        while (reader.pos < end) {
            let [fieldNo, wireType] = reader.tag();
            switch (fieldNo) {
                case /* string FileID */ 1:
                    message.fileID = reader.string();
                    break;
                case /* string FileName */ 2:
                    message.fileName = reader.string();
                    break;
                case /* double Score */ 3:
                    message.score = reader.double();
                    break;
                case /* string Link */ 4:
                    message.link = reader.string();
                    break;
                default:
                    let u = options.readUnknownField;
                    if (u === "throw")
                        throw new globalThis.Error(`Unknown field ${fieldNo} (wire type ${wireType}) for ${this.typeName}`);
                    let d = reader.skip(wireType);
                    if (u !== false)
                        (u === true ? UnknownFieldHandler.onRead : u)(this.typeName, message, fieldNo, wireType, d);
            }
        }
        return message;
    }
    internalBinaryWrite(message, writer, options) {
        /* string FileID = 1; */
        if (message.fileID !== "")
            writer.tag(1, WireType.LengthDelimited).string(message.fileID);
        /* string FileName = 2; */
        if (message.fileName !== "")
            writer.tag(2, WireType.LengthDelimited).string(message.fileName);
        /* double Score = 3; */
        if (message.score !== 0)
            writer.tag(3, WireType.Bit64).double(message.score);
        /* string Link = 4; */
        if (message.link !== "")
            writer.tag(4, WireType.LengthDelimited).string(message.link);
        let u = options.writeUnknownFields;
        if (u !== false)
            (u == true ? UnknownFieldHandler.onWrite : u)(this.typeName, message, writer);
        return writer;
    }
}
/**
 * @generated MessageType for protobuf message FileSearchResult
 */
export const FileSearchResult = new FileSearchResult$Type();
