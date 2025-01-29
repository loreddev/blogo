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

package core

import (
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"forge.capytal.company/loreddev/blogo/plugin"
	"forge.capytal.company/loreddev/x/tinyssert"
)

// Creates a implementation of [http.Handler] that maps the [(*http.Request).Path] to a file of the
// same name in the file system provided by the sourcer. Use [Opts] to have more fine grained control
// over some additional behaviour of the implementation.
func NewServer(
	sourcer plugin.Sourcer,
	renderer plugin.Renderer,
	onerror plugin.ErrorHandler,
	opts ...ServerOpts,
) http.Handler {
	opt := ServerOpts{}
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.Assertions == nil {
		opt.Assertions = tinyssert.NewDisabledAssertions()
	}
	if opt.Logger == nil {
		opt.Logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	}

	var filesystem fs.FS
	if opt.SourceOnInit {
		fs, err := sourcer.Source()
		if err != nil {
			panic(fmt.Sprintf("Failed to source files on initialization due to error: %s",
				err.Error(),
			))
		}
		filesystem = fs
	}

	return &server{
		files: filesystem,

		sourcer:  sourcer,
		renderer: renderer,
		onerror:  onerror,

		assert: opt.Assertions,
		log:    opt.Logger,
	}
}

// Options used in the construction of the server/[http.Handler] in [NewServer] to better
// control additional behaviour of the implementation.
type ServerOpts struct {
	// Call [(plugin.Sourcer).Source] on construction of the implementation on [NewServer]?
	// Panics if the it returns a error. By default sourcing of files is done on the first
	// request.
	SourceOnInit bool
	// [tinyssert.Assertions] implementation used by server for it's Assertions, by default
	// uses [tinyssert.NewDisabledAssertions] to effectively disable assertions. Use this
	// if you want to the server to fail-fast on incorrect states.
	Assertions tinyssert.Assertions
	// Logger to be used to send error, warns and debug messages, useful for plugin development
	// and debugging the pipeline of files. By default it uses a logger that writes to [io.Discard],
	// effectively disabling logging.
	Logger *slog.Logger
}

type server struct {
	files fs.FS

	sourcer  plugin.Sourcer
	renderer plugin.Renderer
	onerror  plugin.ErrorHandler

	assert tinyssert.Assertions
	log    *slog.Logger
}

func (srv *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	srv.assert.NotNil(srv.log)
	srv.assert.NotNil(w)
	srv.assert.NotNil(r)

	log := srv.log.With(slog.String("path", r.URL.Path))
	log.Debug("Serving endpoint")

	if srv.files == nil {
		err := srv.serveHTTPSource(w, r)
		if err != nil {
			return
		}
	}

	path := strings.Trim(r.URL.Path, "/")
	if path == "" || path == "/" {
		path = "."
	}

	file, err := srv.serveHTTPOpenFile(path, w, r)
	if err != nil {
		return
	}

	// Defers the closing of the file to prevent memory being held if a renderer
	// does not properly closes the file.
	defer file.Close()

	err = srv.serveHTTPRender(file, w, r)
	if err != nil {
		return
	}

	log.Debug("Finished serving endpoint")
}

func (srv *server) serveHTTPSource(w http.ResponseWriter, r *http.Request) error {
	srv.assert.NotNil(srv.sourcer, "A sourcer needs to be available")
	srv.assert.NotNil(srv.onerror, "An error handler needs to be available in cases of errors")
	srv.assert.NotNil(srv.log)
	srv.assert.NotNil(w)
	srv.assert.NotNil(r)

	log := srv.log.With(slog.String("path", r.URL.Path), slog.String("sourcer", srv.sourcer.Name()))
	log.Debug("Initializing file system")

	fs, err := srv.sourcer.Source()
	if err != nil {
		log := log.With(
			slog.String("err", err.Error()),
			slog.String("errorhandler", srv.onerror.Name()),
		)

		log.Error(
			"Failed to get file system, handling error to ErrorHandler",
		)

		recovr, ok := srv.onerror.Handle(&ServeError{
			Res: w,
			Req: r,
			Err: &SourceError{
				Sourcer: srv.sourcer,
				Err:     err,
			},
		})

		if !ok {
			log.Error("Failed to handle error with plugin")

			w.WriteHeader(http.StatusInternalServerError)
			_, err = w.Write([]byte(fmt.Sprintf(
				"Failed to handle error %q with plugin %q",
				err.Error(),
				srv.onerror.Name(),
			)))
			srv.assert.Nil(err)

			return err
		}

		r, ok := recovr.(plugin.Sourcer)

		if !ok {
			return err
		}

		fs, err = r.Source()
		srv.assert.Nil(err)
	}

	srv.files = fs

	return nil
}

func (srv *server) serveHTTPOpenFile(
	name string,
	w http.ResponseWriter,
	r *http.Request,
) (fs.File, error) {
	srv.assert.NotZero(name, "Name of file should not be empty")
	srv.assert.NotNil(srv.files, "A file system needs to be present to open a file")
	srv.assert.NotNil(srv.onerror, "An error handler needs to be available in cases of errors")
	srv.assert.NotNil(srv.log)
	srv.assert.NotNil(w)
	srv.assert.NotNil(r)

	log := srv.log.With(
		slog.String("path", r.URL.Path),
		slog.String("filename", name),
		slog.String("sourcer", srv.sourcer.Name()),
	)
	log.Debug("Opening file")

	f, err := srv.files.Open(name)

	if err != nil || f == nil {
		if err == nil && f == nil {
			err = fmt.Errorf(
				"file system returned a nil file using sourcer %q",
				srv.sourcer.Name(),
			)
		}

		log := log.With(
			slog.String("err", err.Error()),
			slog.String("errorhandler", srv.onerror.Name()),
		)

		log.Warn(
			"Failed to open file, handling error to ErrorHandler",
		)

		recovr, ok := srv.onerror.Handle(ServeError{
			Res: w,
			Req: r,
			Err: SourceError{
				Sourcer: srv.sourcer,
				Err:     err,
			},
		})

		if !ok {
			log.Error("Failed to handle error with plugin")
			w.WriteHeader(http.StatusInternalServerError)
			_, err = w.Write([]byte(fmt.Sprintf(
				"Failed to handle error %q with plugin %q",
				err.Error(),
				srv.onerror.Name(),
			)))
			srv.assert.Nil(err)

			return nil, err
		}

		r, ok := recovr.(fs.FS)

		if !ok {
			return nil, err
		}

		f, err = r.Open(name)
		srv.assert.Nil(err)

	}

	return f, err
}

func (srv *server) serveHTTPRender(file fs.File, w http.ResponseWriter, r *http.Request) error {
	srv.assert.NotNil(file, "A file needs to be present to it to be rendered")
	srv.assert.NotNil(srv.renderer, "A renderer needs to be present to render a file")
	srv.assert.NotNil(srv.onerror, "An error handler needs to be available in cases of errors")
	srv.assert.NotNil(srv.log)
	srv.assert.NotNil(w)
	srv.assert.NotNil(r)

	log := srv.log.With(
		slog.String("path", r.URL.Path),
		slog.String("renderer", srv.renderer.Name()),
	)
	log.Debug("Rendering file")

	err := srv.renderer.Render(file, w)
	if err != nil {
		log := log.With(
			slog.String("err", err.Error()),
			slog.String("errorhandler", srv.onerror.Name()),
		)

		log.Error(
			"Failed to render file, handling error to ErrorHandler",
		)

		recovr, ok := srv.onerror.Handle(ServeError{
			Res: w,
			Req: r,
			Err: RenderError{
				Renderer: srv.renderer,
				File:     file,
				Err:      err,
			},
		})

		if !ok {
			log.Error("Failed to handle error with plugin")

			w.WriteHeader(http.StatusInternalServerError)
			_, err = w.Write([]byte(fmt.Sprintf(
				"Failed to handle error %q with plugin %q",
				err.Error(),
				srv.onerror.Name(),
			)))
			srv.assert.Nil(err)

			return err
		}

		r, ok := recovr.(plugin.Renderer)

		if !ok {
			return err
		}

		err = r.Render(file, w)
		srv.assert.Nil(err)

	}

	return nil
}
