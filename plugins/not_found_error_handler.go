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
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"

	"forge.capytal.company/loreddev/blogo/core"
	"forge.capytal.company/loreddev/blogo/plugin"
	"forge.capytal.company/loreddev/x/tinyssert"
)

const notFoundErrorHandlerName = "blogo-notfounderrorhandler-errorhandler"

func NewNotFoundErrorHandler(
	templt template.Template,
	opts ...TemplateErrorHandlerOpts,
) plugin.ErrorHandler {
	opt := TemplateErrorHandlerOpts{}
	if len(opts) > 0 {
		opt = opts[0]
	}

	if opt.Assertions == nil {
		opt.Assertions = tinyssert.NewDisabledAssertions()
	}
	if opt.Logger == nil {
		opt.Logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	}

	return &notFoundErrorHandler{
		templt: templt,

		assert: opt.Assertions,
		log:    opt.Logger,
	}
}

type NotFoundErrorHandlerOpts struct {
	Assertions tinyssert.Assertions
	Logger     *slog.Logger
}

type NotFoundErrorHandlerInfo struct {
	Plugin   string
	Path     string
	FilePath string
	Error    error
	ErrorMsg string
}

type notFoundErrorHandler struct {
	templt template.Template

	assert tinyssert.Assertions
	log    *slog.Logger
}

func (h *notFoundErrorHandler) Name() string {
	return notFoundErrorHandlerName
}

func (h *notFoundErrorHandler) Handle(err error) (recovr any, handled bool) {
	h.assert.NotNil(err, "Error should not be nil")
	h.assert.NotNil(h.templt, "notFound should not be nil")
	h.assert.NotNil(h.log)

	log := h.log.With(slog.String("err", err.Error()))

	var serr core.ServeError
	if !errors.As(err, &serr) {
		log.Debug("Error is not a core.ServeError, ignoring error")
		return nil, false
	}

	log = h.log.With(slog.String("serveerr", serr.Error()))

	var sourceErr core.SourceError
	if !errors.As(serr.Err, &sourceErr) {
		log.Debug("Error is not a core.SourceError, ignoring error")
		return nil, false
	}

	log = h.log.With(slog.String("sourceerr", sourceErr.Error()))

	pathErr, ok := sourceErr.Err.(*fs.PathError)
	if !ok {
		log.Debug("Error is not a *fs.PathError, ignoring error")
		return nil, false
	} else if pathErr.Err != fs.ErrNotExist {
		log.Debug("Error is not fs.ErrNotExist, ignoring error")
		return nil, false
	}

	log = h.log.With(slog.String("patherr", pathErr.Error()))

	log.Debug("Handling error")

	w, r := serr.Res, serr.Req

	w.WriteHeader(http.StatusNotFound)
	if err := h.templt.Execute(w, NotFoundErrorHandlerInfo{
		Plugin:   sourceErr.Sourcer.Name(),
		Path:     r.URL.Path,
		FilePath: pathErr.Path,
		Error:    serr.Err,
		ErrorMsg: serr.Err.Error(),
	}); err != nil {
		log.Error("Failed to execute notFound and respond error")
		return nil, false
	}

	return nil, true
}
