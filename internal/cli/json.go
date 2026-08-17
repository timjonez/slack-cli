package cli

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/timjonez/slack-cli/internal/config"
	"github.com/timjonez/slack-cli/internal/slackx"
)

type jsonError struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func (a *App) writeJSON(v any) error {
	return a.writeJSONTo(a.Stdout, v)
}

func (a *App) writeJSONError(err error) {
	_ = a.writeJSONTo(a.Stderr, jsonError{
		Error: err.Error(),
		Code:  errorCode(err),
	})
}

func (a *App) writeJSONTo(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func errorCode(err error) string {
	switch {
	case errors.Is(err, slackx.ErrNotFound):
		return "not_found"
	case errors.Is(err, config.ErrMissingBotToken), errors.Is(err, config.ErrMissingAppToken),
		errors.Is(err, config.ErrBadBotPrefix), errors.Is(err, config.ErrBadAppPrefix):
		return "invalid"
	default:
		if code := slackx.SlackErrCode(err); code != "" {
			return code
		}
		return "error"
	}
}

func (a *App) emit(jsonVal any, quietOK bool, human func()) error {
	if a.JSON {
		return a.writeJSON(jsonVal)
	}
	if quietOK && a.Quiet {
		return nil
	}
	human()
	return nil
}

func (a *App) emitAlways(jsonVal any, human func()) error {
	if a.JSON {
		return a.writeJSON(jsonVal)
	}
	human()
	return nil
}
