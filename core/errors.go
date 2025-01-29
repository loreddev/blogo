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

package core

import (
	"fmt"
	"io/fs"
	"net/http"

	"forge.capytal.company/loreddev/blogo/plugin"
)

type ServeError struct {
	Res http.ResponseWriter
	Req *http.Request
	Err error
}

func (e ServeError) Error() string {
	return fmt.Sprintf("failed to serve file on path %q", e.Req.URL.Path)
}

func (e ServeError) Unwrap() error {
	return e.Err
}

type SourceError struct {
	Sourcer plugin.Sourcer
	Err     error
}

func (e SourceError) Error() string {
	return fmt.Sprintf("failed to source files with sourcer %q", e.Sourcer.Name())
}

func (e SourceError) Unwrap() error {
	return e.Err
}

type RenderError struct {
	Renderer plugin.Renderer
	File     fs.File
	Err      error
}

func (e RenderError) Error() string {
	return fmt.Sprintf("failed to source files with renderer %q", e.Renderer.Name())
}

func (e RenderError) Unwrap() error {
	return e.Err
}
