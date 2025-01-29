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
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log/slog"

	"forge.capytal.company/loreddev/blogo/plugin"
	"forge.capytal.company/loreddev/x/tinyssert"
)

const foldingRendererPluginName = "blogo-foldingrenderer-renderer"

func NewFoldingRenderer(opts ...FoldingRendererOpts) FoldingRenderer {
	opt := FoldingRendererOpts{}
	if len(opts) > 0 {
		opt = opts[0]
	}

	if opt.Assertions == nil {
		opt.Assertions = tinyssert.NewDisabledAssertions()
	}
	if opt.Logger == nil {
		opt.Logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	}

	return &foldingRenderer{
		plugins: []plugin.Renderer{},

		assert: opt.Assertions,
		log:    opt.Logger,
	}
}

type FoldingRenderer interface {
	plugin.WithPlugins
	plugin.Renderer
}

type FoldingRendererOpts struct {
	PanicOnInit bool

	Assertions tinyssert.Assertions
	Logger     *slog.Logger
}

type foldingRenderer struct {
	plugins []plugin.Renderer

	assert tinyssert.Assertions
	log    *slog.Logger
}

func (r *foldingRenderer) Name() string {
	return foldingRendererPluginName
}

func (r *foldingRenderer) Use(p plugin.Plugin) {
	r.assert.NotNil(p)
	r.assert.NotNil(r.plugins)
	r.assert.NotNil(r.log)

	log := r.log.With(slog.String("plugin", p.Name()))

	if pr, ok := p.(plugin.Renderer); ok {
		r.plugins = append(r.plugins, pr)
	} else {
		log.Error(fmt.Sprintf(
			"Failed to add plugin %q, since it doesn't implement plugin.Renderer",
			p.Name(),
		))
	}
}

func (r *foldingRenderer) Render(src fs.File, w io.Writer) error {
	r.assert.NotNil(r.plugins)
	r.assert.NotNil(r.log)
	r.assert.NotNil(src)
	r.assert.NotNil(w)

	log := r.log.With()

	if len(r.plugins) == 0 {
		log.Debug("No renderers found, copying file contents to writer")

		_, err := io.Copy(w, src)
		return err
	}

	log.Debug("Creating folding file")

	f, err := newFoldignFile(src)
	if err != nil {
		log.Error("Failed to create folding file", slog.String("err", err.Error()))

		return err
	}

	for _, p := range r.plugins {
		log := log.With(slog.String("plugin", p.Name()))

		log.Debug("Rendering with plugin")

		err := p.Render(f, f)
		if err != nil {
			log.Error("Failed to render with plugin", slog.String("err", err.Error()))
			return err
		}

		log.Debug("Folding file to next render")

		if err := f.Fold(); err != nil {
			log.Error("Failed to fold file", slog.String("err", err.Error()))
			return err
		}
	}

	log.Debug("Writing final file to Writer")

	_, err = io.Copy(w, f)
	return err
}

type foldingFile struct {
	fs.File
	read   *bytes.Buffer
	writer *bytes.Buffer
}

func newFoldignFile(f fs.File) (*foldingFile, error) {
	var r, w bytes.Buffer

	if _, err := io.Copy(&r, f); err != nil {
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}

	return &foldingFile{File: f, read: &r, writer: &w}, nil
}

func (f *foldingFile) Close() error {
	return nil
}

func (f *foldingFile) Read(p []byte) (int, error) {
	return f.read.Read(p)
}

func (f *foldingFile) Write(p []byte) (int, error) {
	return f.writer.Write(p)
}

func (f *foldingFile) Fold() error {
	f.read.Reset()
	if _, err := io.Copy(f.read, f.writer); err != nil {
		return err
	}
	f.writer.Reset()
	return nil
}
