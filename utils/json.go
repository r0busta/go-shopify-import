package utils

import (
	"encoding/json"
	"io"
)

func WriteFormatedJSON(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "    ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}
