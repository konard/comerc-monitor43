// Package api provides API definitions for monitor services
package api

// This file exists to make the api package valid without causing import cycles
// Proto files import this package, so it must not import proto

// Empty is a placeholder message type
// Re-exported from common.proto to fix proto generation issues
// TODO: regenerate proto files with correct go_package options
type Empty struct{}

// ProtoReflect implements proto.Message
func (x *Empty) ProtoReflect() any { return x }

// String returns string representation
func (x *Empty) String() string { return "" }

// Reset resets the message
func (x *Empty) Reset() {}

// ProtoMessage returns true
func (*Empty) ProtoMessage() {}
