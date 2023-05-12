// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.21.12
// source: proto/basic_leaderless.proto

package proto

import (
	reflect "reflect"
	sync "sync"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// ResolvableKV is a key-value pair that can be conflict-resolved with other ResolvableKV's. The
// resolution strategy is done via the clock field, which could be as simple as a timestamp or as
// complex as a dotted version vector.
type ResolvableKV struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Key   string `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value string `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	Clock *Clock `protobuf:"bytes,3,opt,name=clock,proto3" json:"clock,omitempty"`
}

func (x *ResolvableKV) Reset() {
	*x = ResolvableKV{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_basic_leaderless_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ResolvableKV) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ResolvableKV) ProtoMessage() {}

func (x *ResolvableKV) ProtoReflect() protoreflect.Message {
	mi := &file_proto_basic_leaderless_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ResolvableKV.ProtoReflect.Descriptor instead.
func (*ResolvableKV) Descriptor() ([]byte, []int) {
	return file_proto_basic_leaderless_proto_rawDescGZIP(), []int{0}
}

func (x *ResolvableKV) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *ResolvableKV) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

func (x *ResolvableKV) GetClock() *Clock {
	if x != nil {
		return x.Clock
	}
	return nil
}

type Key struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Key   string `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Clock *Clock `protobuf:"bytes,2,opt,name=clock,proto3" json:"clock,omitempty"`
}

func (x *Key) Reset() {
	*x = Key{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_basic_leaderless_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Key) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Key) ProtoMessage() {}

func (x *Key) ProtoReflect() protoreflect.Message {
	mi := &file_proto_basic_leaderless_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Key.ProtoReflect.Descriptor instead.
func (*Key) Descriptor() ([]byte, []int) {
	return file_proto_basic_leaderless_proto_rawDescGZIP(), []int{1}
}

func (x *Key) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *Key) GetClock() *Clock {
	if x != nil {
		return x.Clock
	}
	return nil
}

type HandlePeerWriteReply struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Whether the node accepted the KV trying to be replicated
	Accepted bool `protobuf:"varint,1,opt,name=accepted,proto3" json:"accepted,omitempty"`
	// If the node did not accept the KV trying to be replicated, then it should respond with a
	// more up-to-date value
	ResolvableKv *ResolvableKV `protobuf:"bytes,2,opt,name=resolvable_kv,json=resolvableKv,proto3,oneof" json:"resolvable_kv,omitempty"`
}

func (x *HandlePeerWriteReply) Reset() {
	*x = HandlePeerWriteReply{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_basic_leaderless_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HandlePeerWriteReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HandlePeerWriteReply) ProtoMessage() {}

func (x *HandlePeerWriteReply) ProtoReflect() protoreflect.Message {
	mi := &file_proto_basic_leaderless_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HandlePeerWriteReply.ProtoReflect.Descriptor instead.
func (*HandlePeerWriteReply) Descriptor() ([]byte, []int) {
	return file_proto_basic_leaderless_proto_rawDescGZIP(), []int{2}
}

func (x *HandlePeerWriteReply) GetAccepted() bool {
	if x != nil {
		return x.Accepted
	}
	return false
}

func (x *HandlePeerWriteReply) GetResolvableKv() *ResolvableKV {
	if x != nil {
		return x.ResolvableKv
	}
	return nil
}

type HandlePeerReadReply struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Found        bool          `protobuf:"varint,1,opt,name=found,proto3" json:"found,omitempty"`
	ResolvableKv *ResolvableKV `protobuf:"bytes,2,opt,name=resolvable_kv,json=resolvableKv,proto3,oneof" json:"resolvable_kv,omitempty"`
}

func (x *HandlePeerReadReply) Reset() {
	*x = HandlePeerReadReply{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_basic_leaderless_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *HandlePeerReadReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*HandlePeerReadReply) ProtoMessage() {}

func (x *HandlePeerReadReply) ProtoReflect() protoreflect.Message {
	mi := &file_proto_basic_leaderless_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use HandlePeerReadReply.ProtoReflect.Descriptor instead.
func (*HandlePeerReadReply) Descriptor() ([]byte, []int) {
	return file_proto_basic_leaderless_proto_rawDescGZIP(), []int{3}
}

func (x *HandlePeerReadReply) GetFound() bool {
	if x != nil {
		return x.Found
	}
	return false
}

func (x *HandlePeerReadReply) GetResolvableKv() *ResolvableKV {
	if x != nil {
		return x.ResolvableKv
	}
	return nil
}

var File_proto_basic_leaderless_proto protoreflect.FileDescriptor

var file_proto_basic_leaderless_proto_rawDesc = []byte{
	0x0a, 0x1c, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x62, 0x61, 0x73, 0x69, 0x63, 0x5f, 0x6c, 0x65,
	0x61, 0x64, 0x65, 0x72, 0x6c, 0x65, 0x73, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x0f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x61, 0x70, 0x69,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x5a, 0x0a, 0x0c, 0x52, 0x65, 0x73, 0x6f, 0x6c, 0x76,
	0x61, 0x62, 0x6c, 0x65, 0x4b, 0x56, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x12, 0x22,
	0x0a, 0x05, 0x63, 0x6c, 0x6f, 0x63, 0x6b, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0c, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x43, 0x6c, 0x6f, 0x63, 0x6b, 0x52, 0x05, 0x63, 0x6c, 0x6f,
	0x63, 0x6b, 0x22, 0x3b, 0x0a, 0x03, 0x4b, 0x65, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x22, 0x0a, 0x05, 0x63,
	0x6c, 0x6f, 0x63, 0x6b, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0c, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2e, 0x43, 0x6c, 0x6f, 0x63, 0x6b, 0x52, 0x05, 0x63, 0x6c, 0x6f, 0x63, 0x6b, 0x22,
	0x83, 0x01, 0x0a, 0x14, 0x48, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x50, 0x65, 0x65, 0x72, 0x57, 0x72,
	0x69, 0x74, 0x65, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x12, 0x1a, 0x0a, 0x08, 0x61, 0x63, 0x63, 0x65,
	0x70, 0x74, 0x65, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x08, 0x61, 0x63, 0x63, 0x65,
	0x70, 0x74, 0x65, 0x64, 0x12, 0x3d, 0x0a, 0x0d, 0x72, 0x65, 0x73, 0x6f, 0x6c, 0x76, 0x61, 0x62,
	0x6c, 0x65, 0x5f, 0x6b, 0x76, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2e, 0x52, 0x65, 0x73, 0x6f, 0x6c, 0x76, 0x61, 0x62, 0x6c, 0x65, 0x4b, 0x56,
	0x48, 0x00, 0x52, 0x0c, 0x72, 0x65, 0x73, 0x6f, 0x6c, 0x76, 0x61, 0x62, 0x6c, 0x65, 0x4b, 0x76,
	0x88, 0x01, 0x01, 0x42, 0x10, 0x0a, 0x0e, 0x5f, 0x72, 0x65, 0x73, 0x6f, 0x6c, 0x76, 0x61, 0x62,
	0x6c, 0x65, 0x5f, 0x6b, 0x76, 0x22, 0x7c, 0x0a, 0x13, 0x48, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x50,
	0x65, 0x65, 0x72, 0x52, 0x65, 0x61, 0x64, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x12, 0x14, 0x0a, 0x05,
	0x66, 0x6f, 0x75, 0x6e, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x05, 0x66, 0x6f, 0x75,
	0x6e, 0x64, 0x12, 0x3d, 0x0a, 0x0d, 0x72, 0x65, 0x73, 0x6f, 0x6c, 0x76, 0x61, 0x62, 0x6c, 0x65,
	0x5f, 0x6b, 0x76, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2e, 0x52, 0x65, 0x73, 0x6f, 0x6c, 0x76, 0x61, 0x62, 0x6c, 0x65, 0x4b, 0x56, 0x48, 0x00,
	0x52, 0x0c, 0x72, 0x65, 0x73, 0x6f, 0x6c, 0x76, 0x61, 0x62, 0x6c, 0x65, 0x4b, 0x76, 0x88, 0x01,
	0x01, 0x42, 0x10, 0x0a, 0x0e, 0x5f, 0x72, 0x65, 0x73, 0x6f, 0x6c, 0x76, 0x61, 0x62, 0x6c, 0x65,
	0x5f, 0x6b, 0x76, 0x32, 0x9e, 0x01, 0x0a, 0x19, 0x42, 0x61, 0x73, 0x69, 0x63, 0x4c, 0x65, 0x61,
	0x64, 0x65, 0x72, 0x6c, 0x65, 0x73, 0x73, 0x52, 0x65, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x6f,
	0x72, 0x12, 0x45, 0x0a, 0x0f, 0x48, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x50, 0x65, 0x65, 0x72, 0x57,
	0x72, 0x69, 0x74, 0x65, 0x12, 0x13, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x52, 0x65, 0x73,
	0x6f, 0x6c, 0x76, 0x61, 0x62, 0x6c, 0x65, 0x4b, 0x56, 0x1a, 0x1b, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2e, 0x48, 0x61, 0x6e, 0x64, 0x6c, 0x65, 0x50, 0x65, 0x65, 0x72, 0x57, 0x72, 0x69, 0x74,
	0x65, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x22, 0x00, 0x12, 0x3a, 0x0a, 0x0e, 0x48, 0x61, 0x6e, 0x64,
	0x6c, 0x65, 0x50, 0x65, 0x65, 0x72, 0x52, 0x65, 0x61, 0x64, 0x12, 0x0a, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2e, 0x4b, 0x65, 0x79, 0x1a, 0x1a, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x48,
	0x61, 0x6e, 0x64, 0x6c, 0x65, 0x50, 0x65, 0x65, 0x72, 0x52, 0x65, 0x61, 0x64, 0x52, 0x65, 0x70,
	0x6c, 0x79, 0x22, 0x00, 0x42, 0x0e, 0x5a, 0x0c, 0x6d, 0x6f, 0x64, 0x69, 0x73, 0x74, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_proto_basic_leaderless_proto_rawDescOnce sync.Once
	file_proto_basic_leaderless_proto_rawDescData = file_proto_basic_leaderless_proto_rawDesc
)

func file_proto_basic_leaderless_proto_rawDescGZIP() []byte {
	file_proto_basic_leaderless_proto_rawDescOnce.Do(func() {
		file_proto_basic_leaderless_proto_rawDescData = protoimpl.X.CompressGZIP(file_proto_basic_leaderless_proto_rawDescData)
	})
	return file_proto_basic_leaderless_proto_rawDescData
}

var file_proto_basic_leaderless_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_proto_basic_leaderless_proto_goTypes = []interface{}{
	(*ResolvableKV)(nil),         // 0: proto.ResolvableKV
	(*Key)(nil),                  // 1: proto.Key
	(*HandlePeerWriteReply)(nil), // 2: proto.HandlePeerWriteReply
	(*HandlePeerReadReply)(nil),  // 3: proto.HandlePeerReadReply
	(*Clock)(nil),                // 4: proto.Clock
}
var file_proto_basic_leaderless_proto_depIdxs = []int32{
	4, // 0: proto.ResolvableKV.clock:type_name -> proto.Clock
	4, // 1: proto.Key.clock:type_name -> proto.Clock
	0, // 2: proto.HandlePeerWriteReply.resolvable_kv:type_name -> proto.ResolvableKV
	0, // 3: proto.HandlePeerReadReply.resolvable_kv:type_name -> proto.ResolvableKV
	0, // 4: proto.BasicLeaderlessReplicator.HandlePeerWrite:input_type -> proto.ResolvableKV
	1, // 5: proto.BasicLeaderlessReplicator.HandlePeerRead:input_type -> proto.Key
	2, // 6: proto.BasicLeaderlessReplicator.HandlePeerWrite:output_type -> proto.HandlePeerWriteReply
	3, // 7: proto.BasicLeaderlessReplicator.HandlePeerRead:output_type -> proto.HandlePeerReadReply
	6, // [6:8] is the sub-list for method output_type
	4, // [4:6] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_proto_basic_leaderless_proto_init() }
func file_proto_basic_leaderless_proto_init() {
	if File_proto_basic_leaderless_proto != nil {
		return
	}
	file_proto_api_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_proto_basic_leaderless_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ResolvableKV); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_basic_leaderless_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Key); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_basic_leaderless_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HandlePeerWriteReply); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_basic_leaderless_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*HandlePeerReadReply); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_proto_basic_leaderless_proto_msgTypes[2].OneofWrappers = []interface{}{}
	file_proto_basic_leaderless_proto_msgTypes[3].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_proto_basic_leaderless_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_proto_basic_leaderless_proto_goTypes,
		DependencyIndexes: file_proto_basic_leaderless_proto_depIdxs,
		MessageInfos:      file_proto_basic_leaderless_proto_msgTypes,
	}.Build()
	File_proto_basic_leaderless_proto = out.File
	file_proto_basic_leaderless_proto_rawDesc = nil
	file_proto_basic_leaderless_proto_goTypes = nil
	file_proto_basic_leaderless_proto_depIdxs = nil
}