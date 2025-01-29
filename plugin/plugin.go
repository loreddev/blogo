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

package plugin

import (
	"io"
	"io/fs"
)

type Plugin interface {
	Name() string
}

type WithPlugins interface {
	Plugin
	Use(Plugin)
}

type Renderer interface {
	Plugin
	Render(src fs.File, out io.Writer) error
}

type Sourcer interface {
	Plugin
	Source() (fs.FS, error)
}

type ErrorHandler interface {
	Plugin
	Handle(error) (recovr any, handled bool)
}
