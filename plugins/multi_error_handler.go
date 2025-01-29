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
	"io"
	"log/slog"

	"forge.capytal.company/loreddev/blogo/plugin"
	"forge.capytal.company/loreddev/x/tinyssert"
)

const multiErrorHandlerName = "blogo-multierrorhandler-errorhandler"

func NewMultiErrorHandler(opts ...MultiErrorHandlerOpts) MultiErrorHandler {
	opt := MultiErrorHandlerOpts{}
	if len(opts) > 0 {
		opt = opts[0]
	}

	if opt.Assertions == nil {
		opt.Assertions = tinyssert.NewDisabledAssertions()
	}
	if opt.Logger == nil {
		opt.Logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	}

	return &multiErrorHandler{
		handlers: []plugin.ErrorHandler{},

		assert: opt.Assertions,
		log:    opt.Logger,
	}
}

type MultiErrorHandler interface {
	plugin.ErrorHandler
	plugin.WithPlugins
}

type MultiErrorHandlerOpts struct {
	Assertions tinyssert.Assertions
	Logger     *slog.Logger
}

type multiErrorHandler struct {
	handlers []plugin.ErrorHandler

	assert tinyssert.Assertions
	log    *slog.Logger
}

func (h *multiErrorHandler) Name() string {
	return multiErrorHandlerName
}

func (h *multiErrorHandler) Use(p plugin.Plugin) {
	h.assert.NotNil(h.handlers, "Error handlers slice should not be nil")
	h.assert.NotNil(h.log)

	log := h.log.With(slog.String("plugin", p.Name()))
	log.Debug("Adding plugin")

	if p, ok := p.(plugin.Group); ok {
		log.Debug("Plugin is a group, using children plugins")
		for _, p := range p.Plugins() {
			h.Use(p)
		}
	}

	if p, ok := p.(plugin.ErrorHandler); ok {
		h.handlers = append(h.handlers, p)
	} else {
		log.Debug("Plugin does not implement ErrorHandler, ignoring")
	}
}

func (h *multiErrorHandler) Handle(err error) (recovr any, handled bool) {
	h.assert.NotNil(h.handlers, "Error handlers slice should not be nil")
	h.assert.NotNil(h.log)

	log := h.log.With(slog.String("err", err.Error()))
	log.Debug("Handling error")

	for _, handler := range h.handlers {
		log := log.With(slog.String("plugin", handler.Name()))
		log.Debug("Handling error with plugin")

		recovr, ok := handler.Handle(err)
		if ok {
			log.Debug("Error successfully handled with plugin")
			return recovr, ok
		}
	}

	log.Debug("Failed to handle error with any plugin")
	return nil, false
}
