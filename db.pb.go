// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.0
// 	protoc        v3.21.2
// source: db.proto

package bitcask

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Entry struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Crc       uint64     `protobuf:"varint,1,opt,name=crc,proto3" json:"crc,omitempty"`
	EntryData *EntryData `protobuf:"bytes,2,opt,name=entry_data,json=entryData,proto3" json:"entry_data,omitempty"`
}

func (x *Entry) Reset() {
	*x = Entry{}
	if protoimpl.UnsafeEnabled {
		mi := &file_db_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Entry) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Entry) ProtoMessage() {}

func (x *Entry) ProtoReflect() protoreflect.Message {
	mi := &file_db_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Entry.ProtoReflect.Descriptor instead.
func (*Entry) Descriptor() ([]byte, []int) {
	return file_db_proto_rawDescGZIP(), []int{0}
}

func (x *Entry) GetCrc() uint64 {
	if x != nil {
		return x.Crc
	}
	return 0
}

func (x *Entry) GetEntryData() *EntryData {
	if x != nil {
		return x.EntryData
	}
	return nil
}

type EntryData struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Timestamp int32  `protobuf:"varint,1,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	KeySize   int64  `protobuf:"varint,2,opt,name=key_size,json=keySize,proto3" json:"key_size,omitempty"`
	ValueSize int64  `protobuf:"varint,3,opt,name=value_size,json=valueSize,proto3" json:"value_size,omitempty"`
	Key       string `protobuf:"bytes,4,opt,name=key,proto3" json:"key,omitempty"`
	Value     []byte `protobuf:"bytes,5,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *EntryData) Reset() {
	*x = EntryData{}
	if protoimpl.UnsafeEnabled {
		mi := &file_db_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EntryData) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EntryData) ProtoMessage() {}

func (x *EntryData) ProtoReflect() protoreflect.Message {
	mi := &file_db_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EntryData.ProtoReflect.Descriptor instead.
func (*EntryData) Descriptor() ([]byte, []int) {
	return file_db_proto_rawDescGZIP(), []int{1}
}

func (x *EntryData) GetTimestamp() int32 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

func (x *EntryData) GetKeySize() int64 {
	if x != nil {
		return x.KeySize
	}
	return 0
}

func (x *EntryData) GetValueSize() int64 {
	if x != nil {
		return x.ValueSize
	}
	return 0
}

func (x *EntryData) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *EntryData) GetValue() []byte {
	if x != nil {
		return x.Value
	}
	return nil
}

var File_db_proto protoreflect.FileDescriptor

var file_db_proto_rawDesc = []byte{
	0x0a, 0x08, 0x64, 0x62, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x07, 0x62, 0x69, 0x74, 0x63,
	0x61, 0x73, 0x6b, 0x22, 0x4c, 0x0a, 0x05, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03,
	0x63, 0x72, 0x63, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x03, 0x63, 0x72, 0x63, 0x12, 0x31,
	0x0a, 0x0a, 0x65, 0x6e, 0x74, 0x72, 0x79, 0x5f, 0x64, 0x61, 0x74, 0x61, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x12, 0x2e, 0x62, 0x69, 0x74, 0x63, 0x61, 0x73, 0x6b, 0x2e, 0x45, 0x6e, 0x74,
	0x72, 0x79, 0x44, 0x61, 0x74, 0x61, 0x52, 0x09, 0x65, 0x6e, 0x74, 0x72, 0x79, 0x44, 0x61, 0x74,
	0x61, 0x22, 0x8b, 0x01, 0x0a, 0x09, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x44, 0x61, 0x74, 0x61, 0x12,
	0x1c, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x05, 0x52, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x12, 0x19, 0x0a,
	0x08, 0x6b, 0x65, 0x79, 0x5f, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x07, 0x6b, 0x65, 0x79, 0x53, 0x69, 0x7a, 0x65, 0x12, 0x1d, 0x0a, 0x0a, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x5f, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x53, 0x69, 0x7a, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x42,
	0x2a, 0x5a, 0x28, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x72, 0x61,
	0x73, 0x6b, 0x63, 0x68, 0x61, 0x6e, 0x6b, 0x79, 0x2f, 0x67, 0x6f, 0x2d, 0x62, 0x69, 0x74, 0x63,
	0x61, 0x73, 0x6b, 0x2f, 0x62, 0x69, 0x74, 0x63, 0x61, 0x73, 0x6b, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_db_proto_rawDescOnce sync.Once
	file_db_proto_rawDescData = file_db_proto_rawDesc
)

func file_db_proto_rawDescGZIP() []byte {
	file_db_proto_rawDescOnce.Do(func() {
		file_db_proto_rawDescData = protoimpl.X.CompressGZIP(file_db_proto_rawDescData)
	})
	return file_db_proto_rawDescData
}

var file_db_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_db_proto_goTypes = []interface{}{
	(*Entry)(nil),     // 0: bitcask.Entry
	(*EntryData)(nil), // 1: bitcask.EntryData
}
var file_db_proto_depIdxs = []int32{
	1, // 0: bitcask.Entry.entry_data:type_name -> bitcask.EntryData
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_db_proto_init() }
func file_db_proto_init() {
	if File_db_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_db_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Entry); i {
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
		file_db_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EntryData); i {
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
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_db_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_db_proto_goTypes,
		DependencyIndexes: file_db_proto_depIdxs,
		MessageInfos:      file_db_proto_msgTypes,
	}.Build()
	File_db_proto = out.File
	file_db_proto_rawDesc = nil
	file_db_proto_goTypes = nil
	file_db_proto_depIdxs = nil
}