package app

import (
	"fmt"
	"io"

	"github.com/maximseshuk/snapr/internal/api"
)

// DumpOpenAPI writes the snapr API's OpenAPI 3.1 spec as JSON to w. Used by the
// `snapr openapi` CLI subcommand so the static docs site can ship a versioned
// spec without depending on a running snapr instance.
func DumpOpenAPI(w io.Writer, version string) error {
	spec, err := api.BuildOpenAPIJSON(version)
	if err != nil {
		return fmt.Errorf("build openapi: %w", err)
	}
	if _, err := w.Write(spec); err != nil {
		return fmt.Errorf("write openapi: %w", err)
	}
	return nil
}
