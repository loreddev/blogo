// Copyright 2025-present Gustavo "Guz" L. de Mello
// Copyright 2025-present The Lored.dev Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metadata

// TypedMetadata expands the [Metadata] interface to add helper methods for
// Go's primitive types.
//
// [GetTyped] uses this interface for optimization.
type TypedMetadata interface {
	Metadata

	GetBool(key string) (bool, error)

	GetString(key string) (string, error)

	GetInt(key string) (int, error)
	GetInt8(key string) (int8, error)
	GetInt16(key string) (int16, error)
	GetInt32(key string) (int32, error)
	GetInt64(key string) (int64, error)

	GetUint(key string) (uint, error)
	GetUint8(key string) (uint8, error)
	GetUint16(key string) (uint16, error)
	GetUint32(key string) (uint32, error)
	GetUint64(key string) (uint64, error)
	GetUintptr(key string) (uintptr, error)

	GetByte(key string) (byte, error)

	GetRune(key string) (rune, error)

	GetFloat32(key string) (float32, error)
	GetFloat64(key string) (float64, error)

	GetComplex64(key string) (complex64, error)
	GetComplex128(key string) (complex128, error)
}

func Typed(m Metadata) TypedMetadata {
	if m, ok := m.(TypedMetadata); ok {
		return m
	}
	return &typedMetadata{m}
}

type typedMetadata struct{ Metadata }

func (m *typedMetadata) GetBool(key string) (bool, error) {
	return GetTyped[bool](m, key)
}

func (m *typedMetadata) GetString(key string) (string, error) {
	return GetTyped[string](m, key)
}

func (m *typedMetadata) GetInt(key string) (int, error) {
	return GetTyped[int](m, key)
}

func (m *typedMetadata) GetInt8(key string) (int8, error) {
	return GetTyped[int8](m, key)
}

func (m *typedMetadata) GetInt16(key string) (int16, error) {
	return GetTyped[int16](m, key)
}

func (m *typedMetadata) GetInt32(key string) (int32, error) {
	return GetTyped[int32](m, key)
}

func (m *typedMetadata) GetInt64(key string) (int64, error) {
	return GetTyped[int64](m, key)
}

func (m *typedMetadata) GetUint(key string) (uint, error) {
	return GetTyped[uint](m, key)
}

func (m *typedMetadata) GetUint8(key string) (uint8, error) {
	return GetTyped[uint8](m, key)
}

func (m *typedMetadata) GetUint16(key string) (uint16, error) {
	return GetTyped[uint16](m, key)
}

func (m *typedMetadata) GetUint32(key string) (uint32, error) {
	return GetTyped[uint32](m, key)
}

func (m *typedMetadata) GetUint64(key string) (uint64, error) {
	return GetTyped[uint64](m, key)
}

func (m *typedMetadata) GetUintptr(key string) (uintptr, error) {
	return GetTyped[uintptr](m, key)
}

func (m *typedMetadata) GetByte(key string) (byte, error) {
	return GetTyped[byte](m, key)
}

func (m *typedMetadata) GetRune(key string) (rune, error) {
	return GetTyped[rune](m, key)
}

func (m *typedMetadata) GetFloat32(key string) (float32, error) {
	return GetTyped[float32](m, key)
}

func (m *typedMetadata) GetFloat64(key string) (float64, error) {
	return GetTyped[float64](m, key)
}

func (m *typedMetadata) GetComplex64(key string) (complex64, error) {
	return GetTyped[complex64](m, key)
}

func (m *typedMetadata) GetComplex128(key string) (complex128, error) {
	return GetTyped[complex128](m, key)
}
