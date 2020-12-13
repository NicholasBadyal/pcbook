package serializer

import (
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

func ProtobufToJSON(message proto.Message) (string, error) {
	marshaler := jsonpb.Marshaler{
		OrigName:     true,
		EnumsAsInts:  false,
		EmitDefaults: true,
		Indent:       "	",
	}

	return marshaler.MarshalToString(message)
}

func JSONToProtobuf(data string, message proto.Message) error {
	return jsonpb.UnmarshalString(data, message)
}
