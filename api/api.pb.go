// Code generated by protoc-gen-go. DO NOT EDIT.
// source: api.proto

package api

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import _ "google.golang.org/genproto/googleapis/api/annotations"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// EventStore is used for NATS Publish-Subscribe
type EventStore struct {
	AggregateId          string   `protobuf:"bytes,1,opt,name=aggregate_id,json=aggregateId,proto3" json:"aggregate_id,omitempty"`
	AggregateType        string   `protobuf:"bytes,2,opt,name=aggregate_type,json=aggregateType,proto3" json:"aggregate_type,omitempty"`
	EventId              string   `protobuf:"bytes,3,opt,name=event_id,json=eventId,proto3" json:"event_id,omitempty"`
	EventType            string   `protobuf:"bytes,4,opt,name=event_type,json=eventType,proto3" json:"event_type,omitempty"`
	EventData            string   `protobuf:"bytes,5,opt,name=event_data,json=eventData,proto3" json:"event_data,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *EventStore) Reset()         { *m = EventStore{} }
func (m *EventStore) String() string { return proto.CompactTextString(m) }
func (*EventStore) ProtoMessage()    {}
func (*EventStore) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{0}
}
func (m *EventStore) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_EventStore.Unmarshal(m, b)
}
func (m *EventStore) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_EventStore.Marshal(b, m, deterministic)
}
func (dst *EventStore) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EventStore.Merge(dst, src)
}
func (m *EventStore) XXX_Size() int {
	return xxx_messageInfo_EventStore.Size(m)
}
func (m *EventStore) XXX_DiscardUnknown() {
	xxx_messageInfo_EventStore.DiscardUnknown(m)
}

var xxx_messageInfo_EventStore proto.InternalMessageInfo

func (m *EventStore) GetAggregateId() string {
	if m != nil {
		return m.AggregateId
	}
	return ""
}

func (m *EventStore) GetAggregateType() string {
	if m != nil {
		return m.AggregateType
	}
	return ""
}

func (m *EventStore) GetEventId() string {
	if m != nil {
		return m.EventId
	}
	return ""
}

func (m *EventStore) GetEventType() string {
	if m != nil {
		return m.EventType
	}
	return ""
}

func (m *EventStore) GetEventData() string {
	if m != nil {
		return m.EventData
	}
	return ""
}

type DevMeta struct {
	Type                 string   `protobuf:"bytes,1,opt,name=type,proto3" json:"type,omitempty"`
	Name                 string   `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Mac                  string   `protobuf:"bytes,3,opt,name=mac,proto3" json:"mac,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DevMeta) Reset()         { *m = DevMeta{} }
func (m *DevMeta) String() string { return proto.CompactTextString(m) }
func (*DevMeta) ProtoMessage()    {}
func (*DevMeta) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{1}
}
func (m *DevMeta) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DevMeta.Unmarshal(m, b)
}
func (m *DevMeta) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DevMeta.Marshal(b, m, deterministic)
}
func (dst *DevMeta) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DevMeta.Merge(dst, src)
}
func (m *DevMeta) XXX_Size() int {
	return xxx_messageInfo_DevMeta.Size(m)
}
func (m *DevMeta) XXX_DiscardUnknown() {
	xxx_messageInfo_DevMeta.DiscardUnknown(m)
}

var xxx_messageInfo_DevMeta proto.InternalMessageInfo

func (m *DevMeta) GetType() string {
	if m != nil {
		return m.Type
	}
	return ""
}

func (m *DevMeta) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *DevMeta) GetMac() string {
	if m != nil {
		return m.Mac
	}
	return ""
}

type SetDevInitConfigRequest struct {
	Time                 int64    `protobuf:"varint,1,opt,name=time,proto3" json:"time,omitempty"`
	Meta                 *DevMeta `protobuf:"bytes,2,opt,name=meta,proto3" json:"meta,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SetDevInitConfigRequest) Reset()         { *m = SetDevInitConfigRequest{} }
func (m *SetDevInitConfigRequest) String() string { return proto.CompactTextString(m) }
func (*SetDevInitConfigRequest) ProtoMessage()    {}
func (*SetDevInitConfigRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{2}
}
func (m *SetDevInitConfigRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SetDevInitConfigRequest.Unmarshal(m, b)
}
func (m *SetDevInitConfigRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SetDevInitConfigRequest.Marshal(b, m, deterministic)
}
func (dst *SetDevInitConfigRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SetDevInitConfigRequest.Merge(dst, src)
}
func (m *SetDevInitConfigRequest) XXX_Size() int {
	return xxx_messageInfo_SetDevInitConfigRequest.Size(m)
}
func (m *SetDevInitConfigRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_SetDevInitConfigRequest.DiscardUnknown(m)
}

var xxx_messageInfo_SetDevInitConfigRequest proto.InternalMessageInfo

func (m *SetDevInitConfigRequest) GetTime() int64 {
	if m != nil {
		return m.Time
	}
	return 0
}

func (m *SetDevInitConfigRequest) GetMeta() *DevMeta {
	if m != nil {
		return m.Meta
	}
	return nil
}

type SetDevInitConfigResponse struct {
	Config               []byte   `protobuf:"bytes,1,opt,name=config,proto3" json:"config,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SetDevInitConfigResponse) Reset()         { *m = SetDevInitConfigResponse{} }
func (m *SetDevInitConfigResponse) String() string { return proto.CompactTextString(m) }
func (*SetDevInitConfigResponse) ProtoMessage()    {}
func (*SetDevInitConfigResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{3}
}
func (m *SetDevInitConfigResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SetDevInitConfigResponse.Unmarshal(m, b)
}
func (m *SetDevInitConfigResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SetDevInitConfigResponse.Marshal(b, m, deterministic)
}
func (dst *SetDevInitConfigResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SetDevInitConfigResponse.Merge(dst, src)
}
func (m *SetDevInitConfigResponse) XXX_Size() int {
	return xxx_messageInfo_SetDevInitConfigResponse.Size(m)
}
func (m *SetDevInitConfigResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_SetDevInitConfigResponse.DiscardUnknown(m)
}

var xxx_messageInfo_SetDevInitConfigResponse proto.InternalMessageInfo

func (m *SetDevInitConfigResponse) GetConfig() []byte {
	if m != nil {
		return m.Config
	}
	return nil
}

type SaveDevDataRequest struct {
	Time                 int64    `protobuf:"varint,1,opt,name=time,proto3" json:"time,omitempty"`
	Meta                 *DevMeta `protobuf:"bytes,2,opt,name=meta,proto3" json:"meta,omitempty"`
	Data                 []byte   `protobuf:"bytes,3,opt,name=data,proto3" json:"data,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SaveDevDataRequest) Reset()         { *m = SaveDevDataRequest{} }
func (m *SaveDevDataRequest) String() string { return proto.CompactTextString(m) }
func (*SaveDevDataRequest) ProtoMessage()    {}
func (*SaveDevDataRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{4}
}
func (m *SaveDevDataRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SaveDevDataRequest.Unmarshal(m, b)
}
func (m *SaveDevDataRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SaveDevDataRequest.Marshal(b, m, deterministic)
}
func (dst *SaveDevDataRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SaveDevDataRequest.Merge(dst, src)
}
func (m *SaveDevDataRequest) XXX_Size() int {
	return xxx_messageInfo_SaveDevDataRequest.Size(m)
}
func (m *SaveDevDataRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_SaveDevDataRequest.DiscardUnknown(m)
}

var xxx_messageInfo_SaveDevDataRequest proto.InternalMessageInfo

func (m *SaveDevDataRequest) GetTime() int64 {
	if m != nil {
		return m.Time
	}
	return 0
}

func (m *SaveDevDataRequest) GetMeta() *DevMeta {
	if m != nil {
		return m.Meta
	}
	return nil
}

func (m *SaveDevDataRequest) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

type SaveDevDataResponse struct {
	Status               string   `protobuf:"bytes,1,opt,name=status,proto3" json:"status,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SaveDevDataResponse) Reset()         { *m = SaveDevDataResponse{} }
func (m *SaveDevDataResponse) String() string { return proto.CompactTextString(m) }
func (*SaveDevDataResponse) ProtoMessage()    {}
func (*SaveDevDataResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_00212fb1f9d3bf1c, []int{5}
}
func (m *SaveDevDataResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SaveDevDataResponse.Unmarshal(m, b)
}
func (m *SaveDevDataResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SaveDevDataResponse.Marshal(b, m, deterministic)
}
func (dst *SaveDevDataResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SaveDevDataResponse.Merge(dst, src)
}
func (m *SaveDevDataResponse) XXX_Size() int {
	return xxx_messageInfo_SaveDevDataResponse.Size(m)
}
func (m *SaveDevDataResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_SaveDevDataResponse.DiscardUnknown(m)
}

var xxx_messageInfo_SaveDevDataResponse proto.InternalMessageInfo

func (m *SaveDevDataResponse) GetStatus() string {
	if m != nil {
		return m.Status
	}
	return ""
}

func init() {
	proto.RegisterType((*EventStore)(nil), "api.EventStore")
	proto.RegisterType((*DevMeta)(nil), "api.DevMeta")
	proto.RegisterType((*SetDevInitConfigRequest)(nil), "api.SetDevInitConfigRequest")
	proto.RegisterType((*SetDevInitConfigResponse)(nil), "api.SetDevInitConfigResponse")
	proto.RegisterType((*SaveDevDataRequest)(nil), "api.SaveDevDataRequest")
	proto.RegisterType((*SaveDevDataResponse)(nil), "api.SaveDevDataResponse")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// CenterServiceClient is the client API for CenterService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type CenterServiceClient interface {
	SetDevInitConfig(ctx context.Context, in *SetDevInitConfigRequest, opts ...grpc.CallOption) (*SetDevInitConfigResponse, error)
	SaveDevData(ctx context.Context, in *SaveDevDataRequest, opts ...grpc.CallOption) (*SaveDevDataResponse, error)
}

type centerServiceClient struct {
	cc *grpc.ClientConn
}

func NewCenterServiceClient(cc *grpc.ClientConn) CenterServiceClient {
	return &centerServiceClient{cc}
}

func (c *centerServiceClient) SetDevInitConfig(ctx context.Context, in *SetDevInitConfigRequest, opts ...grpc.CallOption) (*SetDevInitConfigResponse, error) {
	out := new(SetDevInitConfigResponse)
	err := c.cc.Invoke(ctx, "/api.CenterService/SetDevInitConf", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *centerServiceClient) SaveDevData(ctx context.Context, in *SaveDevDataRequest, opts ...grpc.CallOption) (*SaveDevDataResponse, error) {
	out := new(SaveDevDataResponse)
	err := c.cc.Invoke(ctx, "/api.CenterService/SaveDevData", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// CenterServiceServer is the server API for CenterService service.
type CenterServiceServer interface {
	SetDevInitConfig(context.Context, *SetDevInitConfigRequest) (*SetDevInitConfigResponse, error)
	SaveDevData(context.Context, *SaveDevDataRequest) (*SaveDevDataResponse, error)
}

func RegisterCenterServiceServer(s *grpc.Server, srv CenterServiceServer) {
	s.RegisterService(&_CenterService_serviceDesc, srv)
}

func _CenterService_SetDevInitConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SetDevInitConfigRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CenterServiceServer).SetDevInitConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.CenterService/SetDevInitConf",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CenterServiceServer).SetDevInitConfig(ctx, req.(*SetDevInitConfigRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _CenterService_SaveDevData_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SaveDevDataRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CenterServiceServer).SaveDevData(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.CenterService/SaveDevData",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CenterServiceServer).SaveDevData(ctx, req.(*SaveDevDataRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _CenterService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "api.CenterService",
	HandlerType: (*CenterServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SetDevInitConf",
			Handler:    _CenterService_SetDevInitConfig_Handler,
		},
		{
			MethodName: "SaveDevData",
			Handler:    _CenterService_SaveDevData_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api.proto",
}

func init() { proto.RegisterFile("api.proto", fileDescriptor_00212fb1f9d3bf1c) }

var fileDescriptor_00212fb1f9d3bf1c = []byte{
	// 376 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xa4, 0x92, 0xcf, 0x6e, 0xda, 0x40,
	0x10, 0xc6, 0xeb, 0x9a, 0x42, 0x19, 0xa0, 0x42, 0x5b, 0xa9, 0xb8, 0x08, 0x24, 0x6a, 0xa9, 0x52,
	0x2f, 0x05, 0x89, 0xbe, 0x41, 0x71, 0x0f, 0x1c, 0xaa, 0xaa, 0x76, 0xce, 0x89, 0x26, 0x78, 0x62,
	0xad, 0x14, 0xef, 0x3a, 0xf6, 0x60, 0x89, 0x17, 0xca, 0x35, 0xaf, 0x18, 0x79, 0x6c, 0xfe, 0x24,
	0x88, 0x53, 0x6e, 0x33, 0xf3, 0xcd, 0xfc, 0xe6, 0xdb, 0xd1, 0x42, 0x17, 0x33, 0x3d, 0xcf, 0x72,
	0xcb, 0x56, 0xb9, 0x98, 0xe9, 0xf1, 0x24, 0xb1, 0x36, 0xb9, 0xa7, 0x05, 0x66, 0x7a, 0x81, 0xc6,
	0x58, 0x46, 0xd6, 0xd6, 0x14, 0x75, 0x8b, 0xff, 0xe4, 0x00, 0xfc, 0x29, 0xc9, 0x70, 0xc4, 0x36,
	0x27, 0xf5, 0x0d, 0xfa, 0x98, 0x24, 0x39, 0x25, 0xc8, 0x74, 0xa3, 0x63, 0xcf, 0x99, 0x39, 0x3f,
	0xba, 0x61, 0xef, 0x50, 0x5b, 0xc7, 0xea, 0x3b, 0x7c, 0x3a, 0xb6, 0xf0, 0x2e, 0x23, 0xef, 0xbd,
	0x34, 0x0d, 0x0e, 0xd5, 0xab, 0x5d, 0x46, 0xea, 0x2b, 0x7c, 0xa4, 0x8a, 0x5b, 0x51, 0x5c, 0x69,
	0xe8, 0x48, 0xbe, 0x8e, 0xd5, 0x14, 0xa0, 0x96, 0x64, 0xba, 0x25, 0x62, 0x57, 0x2a, 0x32, 0x79,
	0x90, 0x63, 0x64, 0xf4, 0x3e, 0x9c, 0xc8, 0x01, 0x32, 0xfa, 0x2b, 0xe8, 0x04, 0x54, 0xfe, 0x25,
	0x46, 0xa5, 0xa0, 0x25, 0x88, 0xda, 0xa5, 0xc4, 0x55, 0xcd, 0x60, 0xba, 0x37, 0x25, 0xb1, 0x1a,
	0x82, 0x9b, 0xe2, 0xa6, 0xb1, 0x51, 0x85, 0xfe, 0x3f, 0x18, 0x45, 0xc4, 0x01, 0x95, 0x6b, 0xa3,
	0x79, 0x65, 0xcd, 0x9d, 0x4e, 0x42, 0x7a, 0xd8, 0x52, 0xc1, 0x02, 0xd5, 0x69, 0x0d, 0x75, 0x43,
	0x89, 0xd5, 0x0c, 0x5a, 0x29, 0x31, 0x0a, 0xb4, 0xb7, 0xec, 0xcf, 0xab, 0x13, 0x37, 0x26, 0x42,
	0x51, 0xfc, 0x25, 0x78, 0xe7, 0xc0, 0x22, 0xb3, 0xa6, 0x20, 0xf5, 0x05, 0xda, 0x1b, 0xa9, 0x08,
	0xb3, 0x1f, 0x36, 0x99, 0x7f, 0x0d, 0x2a, 0xc2, 0x92, 0x02, 0x2a, 0xab, 0x87, 0xbd, 0x69, 0x7f,
	0x35, 0x25, 0xe7, 0x72, 0x65, 0x83, 0xc4, 0xfe, 0x4f, 0xf8, 0xfc, 0x82, 0x7f, 0xb4, 0x53, 0x30,
	0xf2, 0xb6, 0x68, 0xee, 0xd6, 0x64, 0xcb, 0x47, 0x07, 0x06, 0x2b, 0x32, 0x4c, 0x79, 0x44, 0x79,
	0xa9, 0x37, 0xa4, 0xfe, 0xc3, 0xf0, 0xf5, 0xa3, 0xd4, 0x44, 0x96, 0x5f, 0x38, 0xde, 0x78, 0x7a,
	0x41, 0xad, 0x57, 0xfb, 0xef, 0xd4, 0x6f, 0xe8, 0x9d, 0x78, 0x52, 0xa3, 0xba, 0xff, 0xec, 0x0a,
	0x63, 0xef, 0x5c, 0xd8, 0x33, 0x6e, 0xdb, 0xf2, 0x75, 0x7f, 0x3d, 0x07, 0x00, 0x00, 0xff, 0xff,
	0xc2, 0x51, 0x2f, 0xf1, 0xea, 0x02, 0x00, 0x00,
}
