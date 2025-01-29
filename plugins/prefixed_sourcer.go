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
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"strings"

	"forge.capytal.company/loreddev/blogo/metadata"
	"forge.capytal.company/loreddev/blogo/plugin"
	"forge.capytal.company/loreddev/x/tinyssert"
)

const prefixedSourcerName = "blogo-prefixedsourcer-sourcer"

func NewPrefixedSourcer(opts ...PrefixedSourcerOpts) PrefixedSourcer {
	opt := PrefixedSourcerOpts{}
	if len(opts) > 0 {
		opt = opts[0]
	}

	if opt.PrefixSeparator == "" {
		opt.PrefixSeparator = "/"
	}

	if opt.Assertions == nil {
		opt.Assertions = tinyssert.NewDisabledAssertions()
	}
	if opt.Logger == nil {
		opt.Logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	}

	return &prefixedSourcer{
		plugins: map[string]plugin.Sourcer{},

		prefixSeparator:  opt.PrefixSeparator,
		acceptDuplicated: opt.AcceptDuplicated,

		skipOnSourceError: opt.SkipOnSourceError,
		skipOnFSError:     opt.SkipOnFSError,

		assert: opt.Assertions,
		log:    opt.Logger,
	}
}

type PrefixedSourcerOpts struct {
	PrefixSeparator  string
	AcceptDuplicated bool

	SkipOnSourceError bool
	SkipOnFSError     bool

	Assertions tinyssert.Assertions
	Logger     *slog.Logger
}

type PrefixedSourcer interface {
	plugin.Sourcer
	plugin.WithPlugins
	UseNamed(string, plugin.Plugin)
}

type prefixedSourcer struct {
	plugins map[string]plugin.Sourcer

	prefixSeparator  string
	acceptDuplicated bool

	skipOnSourceError bool
	skipOnFSError     bool

	assert tinyssert.Assertions
	log    *slog.Logger
}

func (s *prefixedSourcer) Name() string {
	return prefixedSourcerName
}

func (s *prefixedSourcer) Use(plugin plugin.Plugin) {
	s.UseNamed(plugin.Name(), plugin)
}

func (s *prefixedSourcer) UseNamed(prefix string, p plugin.Plugin) {
	s.assert.NotZero(prefix, "Prefix of plugin should not be empty")
	s.assert.NotNil(p)
	s.assert.NotNil(s.plugins)
	s.assert.NotNil(s.log)

	log := s.log.With(slog.String("plugin", p.Name()), slog.String("prefix", prefix))
	log.Debug("Adding plugin")

	var sourcer plugin.Sourcer
	if ps, ok := p.(plugin.Sourcer); ok {
		sourcer = ps
	} else {
		log.Error(fmt.Sprintf(
			"Failed to add plugin %q (with prefix %q), since it doesn't implement SourcerPlugin",
			p.Name(), prefix,
		))
		return
	}

	if _, ok := s.plugins[prefix]; ok && !s.acceptDuplicated {
		log.Error("Duplicated prefix, skipping plugin")
		return
	}

	s.plugins[prefix] = sourcer
}

func (s *prefixedSourcer) Source() (fs.FS, error) {
	s.assert.NotNil(s.plugins)
	s.assert.NotNil(s.log)

	log := s.log.With()

	fileSystems := make(map[string]fs.FS, len(s.plugins))

	for a, ps := range s.plugins {
		log = log.With(slog.String("plugin", ps.Name()), slog.String("prefix", a))
		log.Info("Sourcing file system of plugin")

		f, err := ps.Source()
		if err != nil && s.skipOnSourceError {
			log.Warn("Failed to source file system of plugin, skipping",
				slog.String("error", err.Error()))
		} else if err != nil {
			log.Error("Failed to source file system of plugin, returning error",
				slog.String("error", err.Error()))
			return f, err
		}

		fileSystems[a] = f
	}

	return &prefixedSourcerFS{
		fileSystems:     fileSystems,
		prefixSeparator: s.prefixSeparator,
	}, nil
}

type prefixedSourcerFS struct {
	fileSystems     map[string]fs.FS
	prefixSeparator string
}

func (pf *prefixedSourcerFS) Metadata() metadata.Metadata {
	ms := []metadata.Metadata{}
	for _, v := range pf.fileSystems {
		if m, err := metadata.GetMetadata(v); err == nil {
			ms = append(ms, m)
		}
	}
	return metadata.Join(ms...)
}

func (pf *prefixedSourcerFS) Open(name string) (fs.File, error) {
	prefix, path, found := strings.Cut(name, pf.prefixSeparator)
	if !found {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	if f, ok := pf.fileSystems[prefix]; ok {
		return f.Open(path)
	}

	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}
