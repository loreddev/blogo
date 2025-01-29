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

package blogo

import (
	"html/template"
	"io"
	"log/slog"
	"net/http"

	"forge.capytal.company/loreddev/blogo/core"
	"forge.capytal.company/loreddev/blogo/plugin"
	"forge.capytal.company/loreddev/blogo/plugins"
	"forge.capytal.company/loreddev/x/tinyssert"
)

var defaultNotFoundTemplate = template.Must(
	template.New("not-found").Parse("404: Blog post {{.Path}} not found"),
)

var defaultInternalErrTemplate = template.Must(
	template.New("internal-err").
		Parse("500: Failed to get blog post {{.Path}} due to error {{.ErrorMsg}}\n{{.Error}}"),
)

// The main function of the package. Creates a new [Blogo] implementation.
//
// This implementation automatically adds fallbacks and uses built-in [plugins] to handle
// multiple sources, renderers and error handlers.
//
// Use [Opts] to more fine grained control of what plugins are used on initialization. To have
// complete control over how plugins are managed, use the [core] package and [plugins]
// for more information and building blocks to create a custom [Blogo] implementation.
func New(opts ...Opts) Blogo {
	opt := Opts{}
	if len(opts) > 0 {
		opt = opts[0]
	}

	if opt.Assertions == nil {
		opt.Assertions = tinyssert.NewDisabledAssertions()
	}
	if opt.Logger == nil {
		opt.Logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	}

	if opt.FallbackRenderer == nil {
		opt.FallbackRenderer = plugins.NewPlainText(plugins.PlainTextOpts{
			Assertions: opt.Assertions,
		})
	}
	if opt.MultiRenderer == nil {
		opt.MultiRenderer = plugins.NewBufferedMultiRenderer(plugins.BufferedMultiRendererOpts{
			Assertions: opt.Assertions,
			Logger:     opt.Logger.WithGroup("multi-renderer"),
		})
	}

	if opt.FallbackSourcer == nil {
		opt.FallbackSourcer = plugins.NewEmptySourcer()
	}
	if opt.MultiSourcer == nil {
		opt.MultiSourcer = plugins.NewMultiSourcer(plugins.MultiSourcerOpts{
			SkipOnSourceError: true,
			SkipOnFSError:     true,

			Assertions: opt.Assertions,
			Logger:     opt.Logger.WithGroup("multi-sourcer"),
		})
	}

	if opt.FallbackErrorHandler == nil {
		logger := opt.Logger.WithGroup("errors")

		f := plugins.NewMultiErrorHandler(plugins.MultiErrorHandlerOpts{
			Assertions: opt.Assertions,
			Logger:     logger,
		})

		f.Use(plugins.NewNotFoundErrorHandler(
			*defaultNotFoundTemplate,
			plugins.TemplateErrorHandlerOpts{
				Assertions: opt.Assertions,
				Logger:     logger.WithGroup("not-found"),
			},
		))

		f.Use(plugins.NewTemplateErrorHandler(
			*defaultInternalErrTemplate,
			plugins.TemplateErrorHandlerOpts{
				Assertions: opt.Assertions,
				Logger:     logger.WithGroup("internal-err"),
			},
		))

		f.Use(plugins.NewLoggerErrorHandler(logger.WithGroup("logger"), slog.LevelError))

		opt.FallbackErrorHandler = f
	}
	if opt.MultiErrorHandler == nil {
		opt.MultiErrorHandler = plugins.NewMultiErrorHandler(plugins.MultiErrorHandlerOpts{
			Assertions: opt.Assertions,
			Logger:     opt.Logger.WithGroup("errors"),
		})
	}

	return &blogo{
		plugins: []plugin.Plugin{},

		fallbackRenderer:     opt.FallbackRenderer,
		multiRenderer:        opt.MultiRenderer,
		fallbackSourcer:      opt.FallbackSourcer,
		multiSourcer:         opt.MultiSourcer,
		fallbackErrorHandler: opt.FallbackErrorHandler,
		multiErrorHandler:    opt.MultiErrorHandler,

		assert: opt.Assertions,
		log:    opt.Logger,
	}
}

// The simplest interface of the [blogo] package's blogging engine.
//
// Users should use the [New] function to easily have a implementation
// that handles plugins and initialization out-of-the-box, or implement
// their own to have more fine grained control over the package.
type Blogo interface {
	// Adds a new plugin to the engine.
	//
	// Implementations may accept any type of plugin interface. The default
	// implementation accepts [plugin.Sourcer], [plugin.Renderer], [plugin.ErrorHandler],
	// and [plugin.Group], ignoring any other plugins or nil values silently.
	Use(plugin.Plugin)
	// Initialize the plugins or internal state if necessary.
	//
	// Implementations may call it on the fist request and/or panic on failed initialization.
	// The default implementation calls Init on the first call to ServeHTTP
	Init()
	// The main entry point to access all blog posts.
	//
	// Implementations may not expect to the ServeHTTP's request to have a path different from
	// "/". Users should use http.StripPrefix if they are using it on a defined path, for example:
	//
	//   http.Handle("/blog", http.StripPrefix("/blog/", blogo))
	//
	// Implementations of this interface may add other method to access the blog posts
	// besides just http requests. Plugins that register API endpoints will handle them
	// inside this handler, in other words, endpoints paths will be appended to any path
	// that this Handler is being used in.
	http.Handler
}

// Options used by [New] to better fine grain the default plugins used by the
// default [Blogo] implementation.
type Opts struct {
	// The plugin that will be used if no [plugin.Renderer] is provided.
	// Defaults to [plugins.NewPlainText].
	FallbackRenderer plugin.Renderer

	// What plugin will be used to combine multiple renderers if necessary.
	MultiRenderer interface {
		plugin.Renderer
		plugin.WithPlugins
	}

	// The plugin that will be used if no [plugin.Sourcer] is provided.
	// Defaults to [plugins.NewEmptySourcer].
	FallbackSourcer plugin.Sourcer

	// What plugin will be used to combine multiple sourcers if necessary.
	MultiSourcer interface {
		plugin.Sourcer
		plugin.WithPlugins
	}

	// The plugin that will be used if no [plugin.ErrorHandler] is provided.
	// Defaults to a MultiErrorHandler with [plugins.NewNotFoundErrorHandler],
	// [plugins.NewTemplateErrorHandler] and [plugins.NewLoggerErrorHandler].
	FallbackErrorHandler plugin.ErrorHandler

	// What plugin will be used to combine multiple error handlers.
	MultiErrorHandler interface {
		plugin.ErrorHandler
		plugin.WithPlugins
	}

	// [tinyssert.Assertions] implementation used Assertions, by default
	// uses [tinyssert.NewDisabledAssertions] to effectively disable assertions.
	// Use this if to fail-fast on incorrect states. This is also passed to the
	// default built-in plugins on initialization.
	Assertions tinyssert.Assertions

	// Logger to be used to send error, warns and debug messages, useful for plugin
	// development and debugging the pipeline of files. By default it uses a logger
	// that writes to [io.Discard], effectively disabling logging. This is passed
	// to the default built-in plugins on initialization.
	Logger *slog.Logger
}

type blogo struct {
	plugins []plugin.Plugin

	fallbackRenderer plugin.Renderer
	multiRenderer    interface {
		plugin.Renderer
		plugin.WithPlugins
	}
	fallbackSourcer plugin.Sourcer
	multiSourcer    interface {
		plugin.Sourcer
		plugin.WithPlugins
	}
	fallbackErrorHandler plugin.ErrorHandler
	multiErrorHandler    interface {
		plugin.ErrorHandler
		plugin.WithPlugins
	}

	server http.Handler

	assert tinyssert.Assertions
	log    *slog.Logger
}

func (b *blogo) Use(p plugin.Plugin) {
	b.assert.NotNil(p, "Plugin definition should not be nil")
	b.assert.NotNil(b.plugins, "Plugins needs to be not-nil")
	b.assert.NotNil(b.log)

	log := b.log.With(slog.String("plugin", p.Name()))

	if p, ok := p.(plugin.Group); ok {
		log.Debug("Plugin group found, adding it's plugins")
		for _, p := range p.Plugins() {
			b.Use(p)
		}
	}

	if p != nil {
		b.plugins = append(b.plugins, p)
	}
}

func (b *blogo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.assert.NotNil(b.log)
	b.assert.NotNil(w)
	b.assert.NotNil(r)

	if b.server != nil {
		b.server.ServeHTTP(w, r)
		return
	}

	log := b.log.With()
	log.Debug("Core server not initialized")

	b.Init()

	b.server.ServeHTTP(w, r)
}

func (b *blogo) Init() {
	b.assert.NotNil(b.plugins, "Plugins needs to be not-nil")
	b.assert.NotNil(b.log)

	log := b.log.With()
	log.Debug("Initializing Blogo plugins")

	sourcer := b.initSourcer()
	renderer := b.initRenderer()
	errorHandler := b.initErrorHandler()

	log.Debug("Constructing Blogo server")

	b.server = core.NewServer(sourcer, renderer, errorHandler, core.ServerOpts{
		Assertions: b.assert,
		Logger:     b.log.WithGroup("server"),
	})

	log.Debug("Server constructed")
}

func (b *blogo) initRenderer() plugin.Renderer {
	b.assert.NotNil(b.plugins, "Plugins needs to be not-nil")
	b.assert.NotNil(b.fallbackRenderer, "FallbackRenderer needs to be not-nil")
	b.assert.NotNil(b.multiRenderer, "MultiRenderer needs to be not-nil")
	b.assert.NotNil(b.log)

	log := b.log.With()
	log.Debug("Initializing Blogo Renderer plugins")

	renderers := []plugin.Renderer{}

	for _, p := range b.plugins {
		if r, ok := p.(plugin.Renderer); ok {
			log.Debug("Adding Renderer", slog.String("sourcer", r.Name()))

			renderers = append(renderers, r)
		}
	}

	if len(renderers) == 0 {
		log.Debug("No Renderer avaiable, using %q as fallback",
			slog.String("renderer", b.fallbackRenderer.Name()))

		return b.fallbackRenderer
	}

	if len(renderers) == 1 {
		log.Debug("Just one Renderer found, using it directly",
			slog.String("renderer", renderers[0].Name()))

		return renderers[0]
	}

	log.Debug("Multiple Renderers found, using MultiRenderer to combine them",
		slog.String("renderer", b.multiRenderer.Name()),
	)
	for _, r := range renderers {
		b.multiRenderer.Use(r)
	}

	return b.multiRenderer
}

func (b *blogo) initSourcer() plugin.Sourcer {
	b.assert.NotNil(b.plugins, "Plugins needs to be not-nil")
	b.assert.NotNil(b.fallbackSourcer, "FallbackSourcer needs to be not-nil")
	b.assert.NotNil(b.multiSourcer, "MultiSourcer needs to be not-nil")
	b.assert.NotNil(b.log)

	log := b.log.With()
	log.Debug("Initializing Blogo Sourcer plugins")

	sourcers := []plugin.Sourcer{}

	for _, p := range b.plugins {
		if s, ok := p.(plugin.Sourcer); ok {
			log.Debug("Adding Sourcer", slog.String("sourcer", s.Name()))

			sourcers = append(sourcers, s)
		}
	}

	if len(sourcers) == 0 {
		log.Debug("No Sourcer avaiable, using %q as fallback",
			slog.String("sourcer", b.fallbackSourcer.Name()))

		return b.fallbackSourcer
	}

	if len(sourcers) == 1 {
		log.Debug("Just one Sourcer found, using it directly",
			slog.String("sourcer", sourcers[0].Name()))

		return sourcers[0]
	}

	log.Debug("Multiple Sourcers found, using MultiSourcer to combine them",
		slog.String("sourcer", b.multiSourcer.Name()),
	)
	for _, s := range sourcers {
		b.multiSourcer.Use(s)
	}

	return b.multiSourcer
}

func (b *blogo) initErrorHandler() plugin.ErrorHandler {
	b.assert.NotNil(b.plugins, "Plugins needs to be not-nil")
	b.assert.NotNil(b.fallbackErrorHandler, "FallbackErrorHandler needs to be not-nil")
	b.assert.NotNil(b.multiErrorHandler, "MultiErrorHandler needs to be not-nil")
	b.assert.NotNil(b.log)

	log := b.log.With()
	log.Debug("Initializing Blogo ErrorHandler plugins")

	errorHandlers := []plugin.ErrorHandler{}

	for _, p := range b.plugins {
		if s, ok := p.(plugin.ErrorHandler); ok {
			log.Debug("Adding ErrorHandler", slog.String("errorHandler", s.Name()))

			errorHandlers = append(errorHandlers, s)
		}
	}

	if len(errorHandlers) == 0 {
		log.Debug("No ErrorHandler avaiable, using %q as fallback",
			slog.String("errorHandler", b.fallbackErrorHandler.Name()))

		return b.fallbackErrorHandler
	}

	if len(errorHandlers) == 1 {
		log.Debug("Just one ErrorHandler found, using it directly",
			slog.String("errorHandler", errorHandlers[0].Name()))

		return errorHandlers[0]
	}

	log.Debug("Multiple ErrorHandlers found, using MultiSourcer to combine them",
		slog.String("errorHandler", b.multiErrorHandler.Name()),
	)
	for _, s := range errorHandlers {
		b.multiErrorHandler.Use(s)
	}

	return b.multiErrorHandler
}
