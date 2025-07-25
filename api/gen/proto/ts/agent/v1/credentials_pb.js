/* eslint-disable */
// @generated by protobuf-ts 2.11.1 with parameter output_javascript,optimize_code_size,long_type_string,add_pb_suffix,ts_nocheck,eslint_disable
// @generated from protobuf file "agent/v1/credentials.proto" (package "agent.v1", syntax proto3)
// tslint:disable
// @ts-nocheck
import { WireType } from "@protobuf-ts/runtime";
import { UnknownFieldHandler } from "@protobuf-ts/runtime";
import { reflectionMergePartial } from "@protobuf-ts/runtime";
import { MessageType } from "@protobuf-ts/runtime";
import { Timestamp } from "../../google/protobuf/timestamp_pb";
// @generated message type with reflection information, may provide speed optimized methods
class OAuthToken$Type extends MessageType {
    constructor() {
        super("agent.v1.OAuthToken", [
            { no: 1, name: "access_token", kind: "scalar", T: 9 /*ScalarType.STRING*/ },
            { no: 2, name: "token_type", kind: "scalar", T: 9 /*ScalarType.STRING*/ },
            { no: 3, name: "refresh_token", kind: "scalar", T: 9 /*ScalarType.STRING*/ },
            { no: 4, name: "expires_at", kind: "scalar", T: 3 /*ScalarType.INT64*/ },
            { no: 5, name: "expiry", kind: "message", T: () => Timestamp },
            { no: 6, name: "expires_in", kind: "scalar", T: 3 /*ScalarType.INT64*/ }
        ]);
    }
    create(value) {
        const message = globalThis.Object.create((this.messagePrototype));
        message.accessToken = "";
        message.tokenType = "";
        message.refreshToken = "";
        message.expiresAt = "0";
        message.expiresIn = "0";
        if (value !== undefined)
            reflectionMergePartial(this, message, value);
        return message;
    }
    internalBinaryRead(reader, length, options, target) {
        let message = target ?? this.create(), end = reader.pos + length;
        while (reader.pos < end) {
            let [fieldNo, wireType] = reader.tag();
            switch (fieldNo) {
                case /* string access_token */ 1:
                    message.accessToken = reader.string();
                    break;
                case /* string token_type */ 2:
                    message.tokenType = reader.string();
                    break;
                case /* string refresh_token */ 3:
                    message.refreshToken = reader.string();
                    break;
                case /* int64 expires_at */ 4:
                    message.expiresAt = reader.int64().toString();
                    break;
                case /* google.protobuf.Timestamp expiry */ 5:
                    message.expiry = Timestamp.internalBinaryRead(reader, reader.uint32(), options, message.expiry);
                    break;
                case /* int64 expires_in */ 6:
                    message.expiresIn = reader.int64().toString();
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
        /* string access_token = 1; */
        if (message.accessToken !== "")
            writer.tag(1, WireType.LengthDelimited).string(message.accessToken);
        /* string token_type = 2; */
        if (message.tokenType !== "")
            writer.tag(2, WireType.LengthDelimited).string(message.tokenType);
        /* string refresh_token = 3; */
        if (message.refreshToken !== "")
            writer.tag(3, WireType.LengthDelimited).string(message.refreshToken);
        /* int64 expires_at = 4; */
        if (message.expiresAt !== "0")
            writer.tag(4, WireType.Varint).int64(message.expiresAt);
        /* google.protobuf.Timestamp expiry = 5; */
        if (message.expiry)
            Timestamp.internalBinaryWrite(message.expiry, writer.tag(5, WireType.LengthDelimited).fork(), options).join();
        /* int64 expires_in = 6; */
        if (message.expiresIn !== "0")
            writer.tag(6, WireType.Varint).int64(message.expiresIn);
        let u = options.writeUnknownFields;
        if (u !== false)
            (u == true ? UnknownFieldHandler.onWrite : u)(this.typeName, message, writer);
        return writer;
    }
}
/**
 * @generated MessageType for protobuf message agent.v1.OAuthToken
 */
export const OAuthToken = new OAuthToken$Type();
