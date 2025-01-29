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
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"

	"forge.capytal.company/loreddev/blogo/plugin"
	"forge.capytal.company/loreddev/x/tinyssert"
)

const multiRendererName = "blogo-multirenderer-renderer"

func NewMultiRenderer(opts ...MultiRendererOpts) MultiRenderer {
	opt := MultiRendererOpts{}
	if len(opts) > 0 {
		opt = opts[0]
	}

	if opt.Assertions == nil {
		opt.Assertions = tinyssert.NewDisabledAssertions()
	}
	if opt.Logger == nil {
		opt.Logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	}

	return &multiRenderer{
		plugins: []plugin.Renderer{},

		assert: opt.Assertions,
		log:    opt.Logger,
	}
}

type MultiRenderer interface {
	plugin.Renderer
	plugin.WithPlugins
}

type MultiRendererOpts struct {
	Assertions tinyssert.Assertions
	Logger     *slog.Logger
}

type multiRenderer struct {
	plugins []plugin.Renderer

	assert tinyssert.Assertions
	log    *slog.Logger
}

func (r *multiRenderer) Name() string {
	return multiRendererName
}

func (r *multiRenderer) Use(p plugin.Plugin) {
	r.assert.NotNil(p)
	r.assert.NotNil(r.plugins)
	r.assert.NotNil(r.log)

	log := r.log.With(slog.String("plugin", p.Name()))

	if p, ok := p.(plugin.Group); ok {
		log.Debug("Plugin is a group, using children plugins")
		for _, p := range p.Plugins() {
			r.Use(p)
		}
	}

	if pr, ok := p.(plugin.Renderer); ok {
		log.Debug("Added renderer plugin")
		r.plugins = append(r.plugins, pr)
	} else {
		log.Error(fmt.Sprintf(
			"Failed to add plugin %q, since it doesn't implement plugin.Renderer",
			p.Name(),
		))
	}
}

func (r *multiRenderer) Render(src fs.File, w io.Writer) error {
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

	mf := newMultiRendererFile(src)

	for _, pr := range r.plugins {
		log := log.With(slog.String("plugin", pr.Name()))

		log.Debug("Trying to render with plugin")
		err := pr.Render(src, w)

		if err == nil {
			break
		}

		log.Debug("Unable to render using plugin", slog.String("error", err.Error()))
		log.Debug("Resetting file for next read")

		if err := mf.Reset(); err != nil {
			log.Error("Failed to reset file read offset", slog.String("error", err.Error()))
			return errors.Join(fmt.Errorf("failed to reset file read offset"), err)
		}
	}

	return nil
}

type multiRendererFile struct {
	fs.File
	buf    *bytes.Buffer
	reader io.Reader
}

func newMultiRendererFile(f fs.File) *multiRendererFile {
	if _, ok := f.(io.Seeker); ok {
		return &multiRendererFile{
			File:   f,
			reader: f,
		}
	}

	var buf bytes.Buffer
	return &multiRendererFile{
		File:   f,
		reader: io.TeeReader(f, &buf),
		buf:    &buf,
	}
}

func (f *multiRendererFile) Read(p []byte) (int, error) {
	return f.reader.Read(p)
}

func (f *multiRendererFile) Reset() error {
	if s, ok := f.File.(io.Seeker); ok {
		_, err := s.Seek(0, io.SeekStart)
		return err
	}
	var buf bytes.Buffer
	r := io.MultiReader(f.buf, f.File)

	f.reader = io.TeeReader(r, &buf)
	f.buf = &buf

	return nil
}
