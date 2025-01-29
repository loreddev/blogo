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

package plugins

import (
	"errors"
	"fmt"
	"io"
	"io/fs"

	"forge.capytal.company/loreddev/blogo/plugin"
	"forge.capytal.company/loreddev/x/tinyssert"
)

const plainTextName = "blogo-plaintext-renderer"

func NewPlainText(opts ...PlainTextOpts) plugin.Renderer {
	opt := PlainTextOpts{}
	if len(opts) > 0 {
		opt = opts[0]
	}

	if opt.Assertions == nil {
		opt.Assertions = tinyssert.NewDisabledAssertions()
	}

	return &painText{
		assert: opt.Assertions,
	}
}

type PlainTextOpts struct {
	Assertions tinyssert.Assertions
}

type painText struct {
	assert tinyssert.Assertions
}

func (p *painText) Name() string {
	return plainTextName
}

func (p *painText) Render(src fs.File, w io.Writer) error {
	p.assert.NotNil(src)
	p.assert.NotNil(w)

	if d, ok := src.(fs.ReadDirFile); ok {
		return p.renderDirectory(d, w)
	}

	_, err := io.Copy(w, src)
	return err
}

func (p *painText) renderDirectory(f fs.ReadDirFile, w io.Writer) error {
	es, err := f.ReadDir(-1)
	if err != nil {
		return err
	}

	for _, e := range es {
		_, err := w.Write([]byte(fmt.Sprintf("%s\n", e.Name())))
		if err != nil {
			return errors.Join(
				fmt.Errorf("failed to write directory file list, file %s", e.Name()),
				err,
			)
		}
	}

	return nil
}
