package main

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirdivealot/ingrid/api"
	"github.com/sirdivealot/ingrid/ext/osrm"
)

type handlers struct {
	osrmclient *osrm.Client
}

func newHandlers() (h *handlers, err error) {
	h = &handlers{}
	h.osrmclient, err = osrm.NewClient("https://router.project-osrm.org")
	return
}

// FindRoutes computes the fastest route from src to each dst.
func (h handlers) FindRoutes(e echo.Context, params api.FindRoutesParams) error {
	// load shedding
	// caching & invalidation
	// auth

	ctx, cancel := context.WithTimeout(e.Request().Context(), 5*time.Second)
	defer cancel()

	routes, err := h.osrmclient.FindRoutes(ctx, []api.LngLat{params.Src}, params.Dst)
	if err != nil {
		return &echo.HTTPError{
			Code:     http.StatusInternalServerError,
			Message:  "could not find routes",
			Internal: err,
		}
	}

	return e.JSON(http.StatusOK, &api.Routes{
		Source: params.Src,
		Routes: routes,
	})
}
