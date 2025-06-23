package logs

import (
	"encoding/json"

	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	Debug = 1
)

// ZapProto is a helper function to be able to log protos as JSON objects.
// We want protos to be logged using the proto json format so we can deserialize them from the logs.
// If you just log a proto with zap it will use the json serialization of the GoLang struct which will not match
// the proto json format. So we serialize the request to JSON and then deserialize it to a map so we can log it as a
// JSON object. A more efficient solution would be to use https://github.com/kazegusuri/go-proto-zap-marshaler
// to generate a custom zapcore.ObjectMarshaler implementation for each proto message.
func ZapProto(key string, pb proto.Message) zap.Field {
	log := NewLogger()
	reqObj := map[string]interface{}{}
	reqJSON, err := protojson.Marshal(pb)
	if err != nil {
		log.Error(err, "failed to marshal request")
		reqObj["error"] = err.Error()
		return zap.Any(key, reqObj)
	}

	if err := json.Unmarshal(reqJSON, &reqObj); err != nil {
		log.Error(err, "failed to unmarshal request")
		reqObj["error"] = err.Error()
	}

	f := zap.Any(key, reqObj)
	return f
}
