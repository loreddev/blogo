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

// [Metadata] is a simple key-value data structure that should be used by
// plugins to pass data between their processes and between plugins.
//
// This package provides a collection of types, interfaces and functions to
// help create and manipulate said data structure.
package metadata

import (
	"errors"
	"reflect"
)

var (
	ErrImmutable   = errors.New("metadata is immutable")
	ErrInvalidType = errors.New("key is not of specified type")
	ErrNotFound    = errors.New("value not found metadata")
	ErrNoMetadata  = errors.New("type does not implement or has metadata")
)

// Gets a value from a [Metadata] or [WithMetadata] objects and tries
// to convert it to the specified type. If m implements [TypedMetadata],
// tries to use the typed methods directly.
//
// For more information, see [Get].
//
// If the value is not of the specified type, returns [ErrInvalidType].
func GetTyped[T any](m any, key string) (T, error) {
	var z T

	if m, ok := m.(TypedMetadata); ok {
		v, err := getTypedFromTyped[T](m, key)
		if v, ok := v.(T); ok && err == nil {
			return v, err
		}
	}

	v, err := Get(m, key)
	if err != nil {
		return z, err
	}

	if v, ok := v.(T); ok {
		return v, nil
	}

	vv, zv := reflect.ValueOf(v), reflect.ValueOf(z)
	vt, zt := vv.Type(), zv.Type()
	if vt.ConvertibleTo(zt) {
		v = vv.Convert(zt).Interface()
		if v, ok := v.(T); ok {
			return v, nil
		}
	}

	return z, ErrInvalidType
}

func getTypedFromTyped[T any](m TypedMetadata, key string) (any, error) {
	var z T

	t := reflect.TypeOf(z)
	if t == nil {
		return z, ErrInvalidType
	}

	switch t.Kind() {
	case reflect.Bool:
		return m.GetBool(key)

	case reflect.String:
		return m.GetString(key)

	case reflect.Int:
		return m.GetInt(key)
	case reflect.Int8:
		return m.GetInt8(key)
	case reflect.Int16:
		return m.GetInt16(key)
	case reflect.Int32:
		return m.GetInt32(key)
	case reflect.Int64:
		return m.GetInt64(key)

	case reflect.Uint:
		return m.GetInt(key)
	case reflect.Uint8:
		return m.GetUint8(key)
	case reflect.Uint16:
		return m.GetUint16(key)
	case reflect.Uint32:
		return m.GetUint32(key)
	case reflect.Uint64:
		return m.GetUint64(key)
	case reflect.Uintptr:
		return m.GetUintptr(key)

	case reflect.Float32:
		return m.GetFloat32(key)
	case reflect.Float64:
		return m.GetFloat64(key)

	case reflect.Complex64:
		return m.GetComplex64(key)
	case reflect.Complex128:
		return m.GetComplex128(key)

	default:
		return m.Get(key)
	}
}

// Gets a value from m, if it implements [Metadata] or [WithMetadata], otherwise returns
// [ErrNoMetadata].
//
// If there is metadata, but there isn't any value associated with the specified key,
// returns [ErrNotFound]. More information at [Metadata]'s Get method.
func Get(m any, key string) (any, error) {
	data, err := GetMetadata(m)
	if err != nil {
		return nil, err
	}
	return data.Get(key)
}

// Sets a value of m, if it implements [Metadata] or [WithMetadata], otherwise returns
// otherwise returns [ErrNoMetadata].
//
// If the underlying metadata is [Immutable], returns [ErrImmutable]. See [Metadata]'s
// Set method for more information.
func Set(m any, key string, v any) error {
	data, err := GetMetadata(m)
	if err != nil {
		return err
	}
	return data.Set(key, v)
}

// Deletes a value of m, if it implements [Metadata] or [WithMetadata], otherwise returns
// otherwise returns [ErrNoMetadata].
//
// If the underlying metadata is [Immutable], returns [ErrImmutable]. See [Metadata]'s
// Delete method for more information.
func Delete(m any, key string) error {
	data, err := GetMetadata(m)
	if err != nil {
		return err
	}
	return data.Delete(key)
}

// Gets the underlying [Metadata] of m. If m implements [Metadata], returns it unchanged,
// otherwise uses the Metadata method if it implements [WithMetadata].
//
// If m doesn't implement any of the interfaces, returns [ErrNoMetadata].
func GetMetadata(m any) (Metadata, error) {
	var data Metadata

	if mt, ok := m.(Metadata); ok {
		data = mt
	} else if mfile, ok := m.(WithMetadata); ok {
		data = mfile.Metadata()
	} else {
		return nil, ErrNoMetadata
	}

	return data, nil
}

// Types may implement this interface to add [Metadata] to their objects that can
// be easily accessed via [Get], [Set], [Delete] and [GetMetadata].
type WithMetadata interface {
	// Returns the underlying [Metadata] of the type.
	//
	// If the [Metadata] is empty and/or the type doesn't have any data associated with
	// it, implementations should return a empty [Metadata] (such as Map(map[string]any{}))
	// and should never return a nil interface.
	Metadata() Metadata
}

// Minimal interface for the Metadata data. This data is used to easily pass information
// between [plugin.Plugin]'s files and file systems.
//
// Implementations of this interface can store their metadata in any way possible,
// may it be with a simple underlying map[string]any or a network accessed storage.
//
// Plugins may add prefixed to their keys to minimize conflicts between other plugins'
// data. The convention for key strings is "<prefix>.<key-name>", all lowercase and
// kabeb-cased.
//
// Other objects and interfaces, such as [fs.FS] and [fs.File] may implement this
// interface directly or use the [WithMetadata] interface.
//
// [TypedMetadata] may also be implemented to optimize calls via [GetTyped].
type Metadata interface {
	// Gets the value of the specified key.
	//
	// Implementations should return [ErrNotFound] if the provided key doesn't have
	// any associated value with it.
	Get(key string) (any, error)
	// Sets the value of the specified key.
	//
	// If the key cannot be created or have it's underlying value changed,
	// implementations should return [ErrImmutable].
	//
	// If the type of v is different from the stored value, implementations may return
	// [ErrInvalidType] if they can't accept different types.
	Set(key string, v any) error
	// Deletes the key from metadata. Implementations that cannot delete the key directly,
	// should set the value a appropriated zero or implementations' default value for that key.
	//
	// If the key cannot be created or have it's underlying value changed,
	// implementations should return [ErrImmutable].
	Delete(key string) error
}

// Type adapter to allow the use of ordinary maps as [Metadata] implementations.
//
// If map is nil, Get always returns [ErrNotFound], Set and Delete return [ErrImmutable].
type Map map[string]any

func (m Map) Get(key string) (any, error) {
	if m == nil {
		return nil, ErrNotFound
	}

	if v, ok := m[key]; ok {
		return v, nil
	} else {
		return nil, ErrNotFound
	}
}

func (m Map) Set(key string, v any) error {
	if m == nil {
		return ErrImmutable
	}
	m[key] = v
	return nil
}

func (m Map) Delete(key string) error {
	if m == nil {
		return ErrImmutable
	}
	delete(m, key)
	return nil
}

// Joins multiple [Metadata] objects together so their values can be easily
// accessed using just one call.
//
// [Get]:
// Iterates over all Metadatas until it finds one that returns a nil-error.
//
// [Set]:
// Sets the specified key on all underlying Metadatas. Ignores errors.
//
// [Delete]:
// Deletes the specified key on all underlying Metadatas. Ignores errors.
func Join(ms ...Metadata) Metadata {
	ms = append([]Metadata{Map(make(map[string]any))}, ms...)
	return joined(ms)
}

type joined []Metadata

func (m joined) Get(key string) (any, error) {
	for _, m := range m {
		v, err := m.Get(key)
		if err == nil {
			return v, nil
		}
	}
	return nil, ErrNotFound
}

func (m joined) Set(key string, v any) error {
	for _, m := range m {
		_ = m.Set(key, v)
	}
	return nil
}

func (m joined) Delete(key string) error {
	for _, m := range m {
		_ = m.Delete(key)
	}
	return nil
}

type immutable struct{ Metadata }

// Converts all keys from m to immutable. All calls to Set and Delete
// are responded with [ErrImmutable].
func Immutable(m Metadata) Metadata {
	return &immutable{m}
}

func (m *immutable) Set(key string, v any) error {
	return ErrImmutable
}

func (m *immutable) Delete(key string) error {
	return ErrImmutable
}
