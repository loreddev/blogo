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
	"log/slog"

	"forge.capytal.company/loreddev/blogo/metadata"
	"forge.capytal.company/loreddev/blogo/plugin"
	"forge.capytal.company/loreddev/x/tinyssert"
)

const multiSourcerName = "blogo-multisourcer-sourcer"

func NewMultiSourcer(opts ...MultiSourcerOpts) MultiSourcer {
	opt := MultiSourcerOpts{}
	if len(opts) > 0 {
		opt = opts[0]
	}

	if opt.Assertions == nil {
		opt.Assertions = tinyssert.NewDisabledAssertions()
	}
	if opt.Logger == nil {
		opt.Logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	}

	return &multiSourcer{
		plugins: []plugin.Sourcer{},

		skipOnSourceError: opt.SkipOnSourceError,
		skipOnFSError:     opt.SkipOnFSError,

		log: opt.Logger,
	}
}

type MultiSourcer interface {
	plugin.Sourcer
	plugin.WithPlugins
}

type MultiSourcerOpts struct {
	SkipOnSourceError bool
	SkipOnFSError     bool

	Assertions tinyssert.Assertions
	Logger     *slog.Logger
}

type multiSourcer struct {
	plugins []plugin.Sourcer

	skipOnSourceError bool
	skipOnFSError     bool

	assert tinyssert.Assertions
	log    *slog.Logger
}

func (s *multiSourcer) Name() string {
	return multiSourcerName
}

func (s *multiSourcer) Use(p plugin.Plugin) {
	s.assert.NotNil(p)
	s.assert.NotNil(s.plugins)
	s.assert.NotNil(s.log)

	log := s.log.With(slog.String("plugin", p.Name()))

	if p, ok := p.(plugin.Group); ok {
		log.Debug("Plugin is a group, using children plugins")
		for _, p := range p.Plugins() {
			s.Use(p)
		}
	}

	if plg, ok := p.(plugin.Sourcer); ok {
		log.Debug("Added sourcer plugin")
		s.plugins = append(s.plugins, plg)
	} else {
		log.Error(fmt.Sprintf(
			"Failed to add plugin %q, since it doesn't implement plugin.Sourcer",
			p.Name(),
		))
	}
}

func (s *multiSourcer) Source() (fs.FS, error) {
	s.assert.NotNil(s.plugins)
	s.assert.NotNil(s.log)

	log := s.log.With()

	fileSystems := []fs.FS{}

	for _, ps := range s.plugins {
		log = log.With(slog.String("plugin", ps.Name()))
		log.Info("Sourcing file system of plugin")

		f, err := ps.Source()
		if err != nil && s.skipOnSourceError {
			log.Warn(
				"Failed to source file system of plugin, skipping",
				slog.String("error", err.Error()),
			)
		} else if err != nil {
			log.Error(
				"Failed to source file system of plugin, returning error",
				slog.String("error", err.Error()),
			)
			return f, err
		}

		fileSystems = append(fileSystems, f)
	}

	f := make([]fs.FS, len(fileSystems))
	for i := range f {
		f[i] = fileSystems[i]
	}

	return &multiSourcerFS{
		fileSystems: f,
		skipOnError: s.skipOnFSError,
	}, nil
}

type multiSourcerFS struct {
	fileSystems []fs.FS
	skipOnError bool
}

func (pf *multiSourcerFS) Metadata() metadata.Metadata {
	ms := []metadata.Metadata{}
	for _, v := range pf.fileSystems {
		if m, err := metadata.GetMetadata(v); err == nil {
			ms = append(ms, m)
		}
	}
	return metadata.Join(ms...)
}

func (mf *multiSourcerFS) Open(name string) (fs.File, error) {
	for _, f := range mf.fileSystems {
		file, err := f.Open(name)

		if err != nil && !errors.Is(err, fs.ErrNotExist) && !mf.skipOnError {
			return file, err
		}

		if err == nil {
			return file, err
		}
	}

	return nil, fs.ErrNotExist
}
