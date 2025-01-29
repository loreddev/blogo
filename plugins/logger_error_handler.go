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
	"log/slog"

	"forge.capytal.company/loreddev/blogo/plugin"
)

const loggerErrorHandlerName = "blogo-loggererrorhandler-errorhandler"

func NewLoggerErrorHandler(logger *slog.Logger, level ...slog.Level) plugin.ErrorHandler {
	l := slog.LevelError
	if len(level) > 0 {
		l = level[0]
	}

	if logger == nil {
		panic(fmt.Sprintf("%s: Failed to construct LoggerErrorHandler, logger needs to be non-nil",
			loggerErrorHandlerName))
	}

	return &loggerErrorHandler{logger: logger, level: l}
}

type loggerErrorHandler struct {
	logger *slog.Logger
	level  slog.Level
}

func (h *loggerErrorHandler) Name() string {
	return loggerErrorHandlerName
}

func (h *loggerErrorHandler) log(msg string, args ...any) {
	switch h.level {
	case slog.LevelDebug:
		h.logger.Debug(msg, args...)
	case slog.LevelInfo:
		h.logger.Info(msg, args...)
	case slog.LevelWarn:
		h.logger.Warn(msg, args...)
	default:
		h.logger.Error(msg, args...)
	}
}

func (h *loggerErrorHandler) Handle(err error) (recovr any, handled bool) {
	h.log("BLOGO ERROR", slog.String("err", err.Error()))
	return nil, true
}
