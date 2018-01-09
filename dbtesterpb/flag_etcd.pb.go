// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: dbtesterpb/flag_etcd.proto

package dbtesterpb

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import _ "github.com/gogo/protobuf/gogoproto"

import io "io"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// See https://github.com/coreos/etcd/blob/master/etcdmain/help.go for more.
type Flag_Etcd_Tip struct {
	SnapshotCount  int64 `protobuf:"varint,1,opt,name=SnapshotCount,proto3" json:"SnapshotCount,omitempty" yaml:"snapshot_count"`
	QuotaSizeBytes int64 `protobuf:"varint,2,opt,name=QuotaSizeBytes,proto3" json:"QuotaSizeBytes,omitempty" yaml:"quota_size_bytes"`
}

func (m *Flag_Etcd_Tip) Reset()                    { *m = Flag_Etcd_Tip{} }
func (m *Flag_Etcd_Tip) String() string            { return proto.CompactTextString(m) }
func (*Flag_Etcd_Tip) ProtoMessage()               {}
func (*Flag_Etcd_Tip) Descriptor() ([]byte, []int) { return fileDescriptorFlagEtcd, []int{0} }

// See https://github.com/coreos/etcd/blob/master/etcdmain/help.go for more.
type Flag_Etcd_V3_2 struct {
	SnapshotCount  int64 `protobuf:"varint,1,opt,name=SnapshotCount,proto3" json:"SnapshotCount,omitempty" yaml:"snapshot_count"`
	QuotaSizeBytes int64 `protobuf:"varint,2,opt,name=QuotaSizeBytes,proto3" json:"QuotaSizeBytes,omitempty" yaml:"quota_size_bytes"`
}

func (m *Flag_Etcd_V3_2) Reset()                    { *m = Flag_Etcd_V3_2{} }
func (m *Flag_Etcd_V3_2) String() string            { return proto.CompactTextString(m) }
func (*Flag_Etcd_V3_2) ProtoMessage()               {}
func (*Flag_Etcd_V3_2) Descriptor() ([]byte, []int) { return fileDescriptorFlagEtcd, []int{1} }

// See https://github.com/coreos/etcd/blob/master/etcdmain/help.go for more.
type Flag_Etcd_V3_3 struct {
	SnapshotCount  int64 `protobuf:"varint,1,opt,name=SnapshotCount,proto3" json:"SnapshotCount,omitempty" yaml:"snapshot_count"`
	QuotaSizeBytes int64 `protobuf:"varint,2,opt,name=QuotaSizeBytes,proto3" json:"QuotaSizeBytes,omitempty" yaml:"quota_size_bytes"`
}

func (m *Flag_Etcd_V3_3) Reset()                    { *m = Flag_Etcd_V3_3{} }
func (m *Flag_Etcd_V3_3) String() string            { return proto.CompactTextString(m) }
func (*Flag_Etcd_V3_3) ProtoMessage()               {}
func (*Flag_Etcd_V3_3) Descriptor() ([]byte, []int) { return fileDescriptorFlagEtcd, []int{2} }

func init() {
	proto.RegisterType((*Flag_Etcd_Tip)(nil), "dbtesterpb.flag__etcd__tip")
	proto.RegisterType((*Flag_Etcd_V3_2)(nil), "dbtesterpb.flag__etcd__v3_2")
	proto.RegisterType((*Flag_Etcd_V3_3)(nil), "dbtesterpb.flag__etcd__v3_3")
}
func (m *Flag_Etcd_Tip) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Flag_Etcd_Tip) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.SnapshotCount != 0 {
		dAtA[i] = 0x8
		i++
		i = encodeVarintFlagEtcd(dAtA, i, uint64(m.SnapshotCount))
	}
	if m.QuotaSizeBytes != 0 {
		dAtA[i] = 0x10
		i++
		i = encodeVarintFlagEtcd(dAtA, i, uint64(m.QuotaSizeBytes))
	}
	return i, nil
}

func (m *Flag_Etcd_V3_2) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Flag_Etcd_V3_2) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.SnapshotCount != 0 {
		dAtA[i] = 0x8
		i++
		i = encodeVarintFlagEtcd(dAtA, i, uint64(m.SnapshotCount))
	}
	if m.QuotaSizeBytes != 0 {
		dAtA[i] = 0x10
		i++
		i = encodeVarintFlagEtcd(dAtA, i, uint64(m.QuotaSizeBytes))
	}
	return i, nil
}

func (m *Flag_Etcd_V3_3) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalTo(dAtA)
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Flag_Etcd_V3_3) MarshalTo(dAtA []byte) (int, error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.SnapshotCount != 0 {
		dAtA[i] = 0x8
		i++
		i = encodeVarintFlagEtcd(dAtA, i, uint64(m.SnapshotCount))
	}
	if m.QuotaSizeBytes != 0 {
		dAtA[i] = 0x10
		i++
		i = encodeVarintFlagEtcd(dAtA, i, uint64(m.QuotaSizeBytes))
	}
	return i, nil
}

func encodeVarintFlagEtcd(dAtA []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return offset + 1
}
func (m *Flag_Etcd_Tip) Size() (n int) {
	var l int
	_ = l
	if m.SnapshotCount != 0 {
		n += 1 + sovFlagEtcd(uint64(m.SnapshotCount))
	}
	if m.QuotaSizeBytes != 0 {
		n += 1 + sovFlagEtcd(uint64(m.QuotaSizeBytes))
	}
	return n
}

func (m *Flag_Etcd_V3_2) Size() (n int) {
	var l int
	_ = l
	if m.SnapshotCount != 0 {
		n += 1 + sovFlagEtcd(uint64(m.SnapshotCount))
	}
	if m.QuotaSizeBytes != 0 {
		n += 1 + sovFlagEtcd(uint64(m.QuotaSizeBytes))
	}
	return n
}

func (m *Flag_Etcd_V3_3) Size() (n int) {
	var l int
	_ = l
	if m.SnapshotCount != 0 {
		n += 1 + sovFlagEtcd(uint64(m.SnapshotCount))
	}
	if m.QuotaSizeBytes != 0 {
		n += 1 + sovFlagEtcd(uint64(m.QuotaSizeBytes))
	}
	return n
}

func sovFlagEtcd(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozFlagEtcd(x uint64) (n int) {
	return sovFlagEtcd(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Flag_Etcd_Tip) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowFlagEtcd
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: flag__etcd__tip: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: flag__etcd__tip: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field SnapshotCount", wireType)
			}
			m.SnapshotCount = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFlagEtcd
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.SnapshotCount |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field QuotaSizeBytes", wireType)
			}
			m.QuotaSizeBytes = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFlagEtcd
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.QuotaSizeBytes |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipFlagEtcd(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthFlagEtcd
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *Flag_Etcd_V3_2) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowFlagEtcd
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: flag__etcd__v3_2: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: flag__etcd__v3_2: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field SnapshotCount", wireType)
			}
			m.SnapshotCount = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFlagEtcd
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.SnapshotCount |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field QuotaSizeBytes", wireType)
			}
			m.QuotaSizeBytes = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFlagEtcd
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.QuotaSizeBytes |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipFlagEtcd(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthFlagEtcd
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *Flag_Etcd_V3_3) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowFlagEtcd
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: flag__etcd__v3_3: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: flag__etcd__v3_3: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field SnapshotCount", wireType)
			}
			m.SnapshotCount = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFlagEtcd
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.SnapshotCount |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field QuotaSizeBytes", wireType)
			}
			m.QuotaSizeBytes = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFlagEtcd
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.QuotaSizeBytes |= (int64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipFlagEtcd(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthFlagEtcd
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipFlagEtcd(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowFlagEtcd
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowFlagEtcd
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
			return iNdEx, nil
		case 1:
			iNdEx += 8
			return iNdEx, nil
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowFlagEtcd
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			iNdEx += length
			if length < 0 {
				return 0, ErrInvalidLengthFlagEtcd
			}
			return iNdEx, nil
		case 3:
			for {
				var innerWire uint64
				var start int = iNdEx
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return 0, ErrIntOverflowFlagEtcd
					}
					if iNdEx >= l {
						return 0, io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					innerWire |= (uint64(b) & 0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				innerWireType := int(innerWire & 0x7)
				if innerWireType == 4 {
					break
				}
				next, err := skipFlagEtcd(dAtA[start:])
				if err != nil {
					return 0, err
				}
				iNdEx = start + next
			}
			return iNdEx, nil
		case 4:
			return iNdEx, nil
		case 5:
			iNdEx += 4
			return iNdEx, nil
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
	}
	panic("unreachable")
}

var (
	ErrInvalidLengthFlagEtcd = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowFlagEtcd   = fmt.Errorf("proto: integer overflow")
)

func init() { proto.RegisterFile("dbtesterpb/flag_etcd.proto", fileDescriptorFlagEtcd) }

var fileDescriptorFlagEtcd = []byte{
	// 239 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0x4a, 0x49, 0x2a, 0x49,
	0x2d, 0x2e, 0x49, 0x2d, 0x2a, 0x48, 0xd2, 0x4f, 0xcb, 0x49, 0x4c, 0x8f, 0x4f, 0x2d, 0x49, 0x4e,
	0xd1, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x42, 0xc8, 0x49, 0xe9, 0xa6, 0x67, 0x96, 0x64,
	0x94, 0x26, 0xe9, 0x25, 0xe7, 0xe7, 0xea, 0xa7, 0xe7, 0xa7, 0xe7, 0xeb, 0x83, 0x95, 0x24, 0x95,
	0xa6, 0x81, 0x79, 0x60, 0x0e, 0x98, 0x05, 0xd1, 0xaa, 0x34, 0x9d, 0x91, 0x8b, 0x1f, 0x6c, 0x1c,
	0xd8, 0xbc, 0xf8, 0xf8, 0x92, 0xcc, 0x02, 0x21, 0x7b, 0x2e, 0xde, 0xe0, 0xbc, 0xc4, 0x82, 0xe2,
	0x8c, 0xfc, 0x12, 0xe7, 0xfc, 0xd2, 0xbc, 0x12, 0x09, 0x46, 0x05, 0x46, 0x0d, 0x66, 0x27, 0xc9,
	0x4f, 0xf7, 0xe4, 0x45, 0x2b, 0x13, 0x73, 0x73, 0xac, 0x94, 0x8a, 0xa1, 0xd2, 0xf1, 0xc9, 0x20,
	0x79, 0xa5, 0x20, 0x54, 0xf5, 0x42, 0xce, 0x5c, 0x7c, 0x81, 0xa5, 0xf9, 0x25, 0x89, 0xc1, 0x99,
	0x55, 0xa9, 0x4e, 0x95, 0x25, 0xa9, 0xc5, 0x12, 0x4c, 0x60, 0x13, 0xa4, 0x3f, 0xdd, 0x93, 0x17,
	0x87, 0x98, 0x50, 0x08, 0x92, 0x8f, 0x2f, 0xce, 0xac, 0x4a, 0x8d, 0x4f, 0x02, 0xa9, 0x50, 0x0a,
	0x42, 0xd3, 0xa2, 0x34, 0x83, 0x91, 0x4b, 0x00, 0xd9, 0x65, 0x65, 0xc6, 0xf1, 0x46, 0x83, 0xd7,
	0x69, 0xc6, 0x83, 0xc3, 0x69, 0x4e, 0x22, 0x27, 0x1e, 0xca, 0x31, 0x9c, 0x78, 0x24, 0xc7, 0x78,
	0xe1, 0x91, 0x1c, 0xe3, 0x83, 0x47, 0x72, 0x8c, 0x33, 0x1e, 0xcb, 0x31, 0x24, 0xb1, 0x81, 0x23,
	0xdb, 0x18, 0x10, 0x00, 0x00, 0xff, 0xff, 0x11, 0xc5, 0x8d, 0xd9, 0x45, 0x02, 0x00, 0x00,
}
