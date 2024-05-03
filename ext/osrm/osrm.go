package osrm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/sirdivealot/ingrid/api"
)

type (
	Client struct {
		serverhost string
	}

	qoption func(url.Values)

	service string
	profile string
)

var (
	serviceTable service = "table"
	serviceRoute service = "route"

	profileDriving profile = "driving"
)

// NewClient returns a OSRM client targeting host.
func NewClient(host string) (*Client, error) {
	return &Client{host}, nil
}

// FindRoutes computes the fastest route from each src to each dst. Routes are
// sorted by duration with distance as tie-breaker, in ascending order.
func (osrm Client) FindRoutes(
	ctx context.Context,
	src []api.LngLat,
	dst []api.LngLat,
) ([]api.Route, error) {
	coords := make([]string, 0, len(src)+len(dst))
	coords = append(coords, src...)
	coords = append(coords, dst...)

	req, err := request(ctx, osrm.serverhost, serviceTable, profileDriving, coords,
		withSources(indexlist(0, len(src))),
		withDestinations(indexlist(len(src), len(src)+len(dst))),
		withAnnotations("distance,duration"),
	)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("osrm: do request: %w", err)
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("osrm: read response: %w", err)
	}

	var payload struct {
		Distances [][]float32
		Durations [][]float32
		Code      string
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("osrm: decode payload: %w", err)
	}

	if payload.Code != "Ok" {
		return nil, fmt.Errorf("osrm: err: %s", payload.Code)
	}

	var r []api.Route
	for i := range src {
		for j := range dst {
			r = append(r, api.Route{
				Destination: dst[j],
				Distance:    payload.Distances[i][j],
				Duration:    payload.Durations[i][j],
			})
		}
	}

	sort.Slice(r, func(i, j int) bool {
		if r[i].Duration != r[j].Duration {
			return r[i].Duration < r[j].Duration
		}
		return r[i].Distance < r[j].Distance
	})

	return r, nil
}

// indexlist returns the string representation of the index range from a to b,
// where a < b.
//
//	a=0, b=3 -> []string{"0", "1", "2"}
func indexlist(a, b int) (s []string) {
	s = make([]string, 0, b-a)
	for a < b {
		s = append(s, strconv.Itoa(a))
		a++
	}
	return s
}

func request(ctx context.Context, host string, svc service, profile profile, coordinates []string, opts ...qoption) (*http.Request, error) {
	endpoint := fmt.Sprintf("%s/%s/v1/%s/%s", host, svc, profile, url.PathEscape(strings.Join(coordinates, ";")))

	requrl, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("osrm: make url: %w", err)
	}

	var q = make(url.Values)
	for _, o := range opts {
		o(q)
	}
	requrl.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", requrl.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("osrm: new request: %w", err)
	}

	return req, nil
}

func withAnnotations(w string) qoption {
	return func(q url.Values) {
		q.Set("annotations", string(w))
	}
}
func withOverview(w string) qoption {
	return func(q url.Values) {
		q.Set("overview", string(w))
	}
}
func withSources(w []string) qoption {
	return func(q url.Values) {
		q.Set("sources", strings.Join(w, ";"))
	}
}
func withDestinations(w []string) qoption {
	return func(q url.Values) {
		q.Set("destinations", strings.Join(w, ";"))
	}
}
