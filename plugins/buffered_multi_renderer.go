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
	"io"
	"io/fs"
	"log/slog"

	"forge.capytal.company/loreddev/blogo/plugin"
	"forge.capytal.company/loreddev/x/tinyssert"
)

const bufferedMultiRendererName = "blogo-buffer-renderer"

func NewBufferedMultiRenderer(opts ...BufferedMultiRendererOpts) BufferedMultiRenderer {
	opt := BufferedMultiRendererOpts{}
	if len(opts) > 0 {
		opt = opts[0]
	}

	if opt.Assertions == nil {
		opt.Assertions = tinyssert.NewDisabledAssertions()
	}
	if opt.Logger == nil {
		opt.Logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	}

	return &bufferedMultiRenderer{
		plugins: []plugin.Renderer{},

		assert: opt.Assertions,
		log:    opt.Logger,
	}
}

type BufferedMultiRenderer interface {
	plugin.Renderer
	plugin.WithPlugins
}

type BufferedMultiRendererOpts struct {
	Assertions tinyssert.Assertions
	Logger     *slog.Logger
}

type bufferedMultiRenderer struct {
	plugins []plugin.Renderer

	assert tinyssert.Assertions
	log    *slog.Logger
}

func (r *bufferedMultiRenderer) Name() string {
	return bufferedMultiRendererName
}

func (r *bufferedMultiRenderer) Use(p plugin.Plugin) {
	r.assert.NotNil(r.plugins, "Plugins slice needs to be not-nil")
	r.assert.NotNil(r.log)

	log := r.log.With(slog.String("plugin", p.Name()))
	log.Debug("Adding plugin")

	if p, ok := p.(plugin.Group); ok {
		log.Debug("Plugin implements plugin.Group, using it's plugins")
		for _, p := range p.Plugins() {
			r.Use(p)
		}
	}

	if p, ok := p.(plugin.Renderer); ok {
		log.Debug("Adding plugin")
		r.plugins = append(r.plugins, p)
	} else {
		log.Warn("Plugin does not implement Renderer, ignoring")
	}
}

func (r *bufferedMultiRenderer) Render(src fs.File, w io.Writer) error {
	r.assert.NotNil(r.plugins, "Plugins slice needs to be not-nil")
	r.assert.NotNil(r.log)

	log := r.log.With()

	log.Debug("Creating buffered file")
	bf := newBufferedFile(src)

	var buf bytes.Buffer

	for _, p := range r.plugins {
		log := log.With(slog.String("plugin", p.Name()))
		log.Debug("Trying to render with plugin")

		err := p.Render(bf, &buf)
		if err == nil {
			log.Debug("Successfully rendered with plugin")
			break
		}

		log.Debug("Unable to render with plugin, resetting file and writer")

		if err = bf.Reset(); err != nil {
			log.Error("Failed to reset file", slog.String("err", err.Error()))
			return errors.Join(errors.New("failed to reset buffered file"), err)
		}

		buf.Reset()
	}

	log.Debug("Copying response to final writer")
	if _, err := io.Copy(w, &buf); err != nil {
		log.Error("Failed to copy response to final writer")
		return errors.Join(errors.New("failed to copy response to final writer"), err)
	}

	return nil
}

func newBufferedFile(src fs.File) bufferedFile {
	var buf bytes.Buffer
	r := io.TeeReader(src, &buf)

	if d, ok := src.(fs.ReadDirFile); ok {
		return &bufDirFile{
			file:   d,
			buffer: &buf,
			reader: r,

			entries: []fs.DirEntry{},
			eof:     false,
			n:       0,
		}
	}

	return &bufFile{
		file:   src,
		buffer: &buf,
		reader: r,
	}
}

type bufferedFile interface {
	fs.File
	Reset() error
}

type bufFile struct {
	file   fs.File
	buffer *bytes.Buffer
	reader io.Reader
}

func (f *bufFile) Read(p []byte) (int, error) {
	return f.reader.Read(p)
}

func (f *bufFile) Close() error {
	return nil
}

func (f *bufFile) Stat() (fs.FileInfo, error) {
	return f.file.Stat()
}

func (f *bufFile) Reset() error {
	_, err := io.ReadAll(f.reader)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	r := io.TeeReader(f.buffer, &buf)

	f.buffer = &buf
	f.reader = r

	return nil
}

type bufDirFile struct {
	file fs.ReadDirFile

	buffer *bytes.Buffer
	reader io.Reader

	entries []fs.DirEntry
	eof     bool
	n       int
}

func (f *bufDirFile) Read(p []byte) (int, error) {
	return f.reader.Read(p)
}

func (f *bufDirFile) Close() error {
	return nil
}

func (f *bufDirFile) Stat() (fs.FileInfo, error) {
	return f.file.Stat()
}

func (f *bufDirFile) ReadDir(n int) ([]fs.DirEntry, error) {
	start, end := f.n, f.n+n

	var err error

	// If EOF is true, it means we already read all the content from the
	// source directory, so we can use the entries slice directly. Otherwise, we
	// may need to read more from the source directly if the provided end index
	// is bigger than what we already have.
	if end > len(f.entries) && !f.eof {
		e, err := f.file.ReadDir(n)

		if e != nil {
			// Add the entries to our buffer so we can access them even after a reset.
			f.entries = append(f.entries, e...)
		}

		if err != nil && !errors.Is(err, io.EOF) {
			return []fs.DirEntry{}, err
		}

		// If we reached EOF, we don't need to call the source directory anymore
		// and can just use the slice directly
		if errors.Is(err, io.EOF) {
			f.eof = true
		}
	}

	// Reading all contents from the directory needs us to have all values inside
	// our buffer/slice, if EOF isn't already reached, we need to get the rest of
	// the content from the source directory.
	if n <= 0 && !f.eof {
		e, err := f.file.ReadDir(n)

		if e != nil {
			f.entries = append(f.entries, e...)
		}

		if err != nil && !errors.Is(err, io.EOF) {
			return []fs.DirEntry{}, err
		}

		if errors.Is(err, io.EOF) {
			f.eof = true
		}
	}

	if n <= 0 {
		start, end = 0, len(f.entries)
	} else if end > len(f.entries) {
		end = len(f.entries)
		err = io.EOF
	}

	e := f.entries[start:end]

	f.n = end

	return e, err
}

func (f *bufDirFile) Reset() error {
	// To reset the ReadDir of the file, we pretty much just need to set
	// the start offset to 0, so any subsequent reads will start at the first
	// item, that is probably already buffered on f.entries.
	f.n = 0

	// Reset the Read buffer, the directory file implementation may have contents
	// on it's Read function.
	_, err := io.ReadAll(f.reader)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	r := io.TeeReader(f.buffer, &buf)

	f.buffer = &buf
	f.reader = r

	return nil
}
