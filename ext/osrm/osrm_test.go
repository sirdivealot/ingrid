package osrm

import (
	"context"
	"testing"
)

func TestOSRM(t *testing.T) {
	o, _ := NewClient("https://router.project-osrm.org")

	routes, err := o.FindRoutes(context.TODO(), []string{"13.388860,52.517037"}, []string{"13.397634,52.529407", "13.428555,52.523219"})
	if err != nil {
		t.Fatalf("find routes: %v", err)
	}

	if len(routes) != 2 {
		t.Errorf("num routes: expected 2 routes but got %v", len(routes))
	}

	for i := 1; i < len(routes); i++ {
		if a, b := routes[i-1].Duration, routes[i].Duration; a > b {
			t.Errorf("order fail: expected route[%v].Duration %v <= route[%v].Duration %v", i-1, a, i, b)
		}
		if a, b := routes[i-1].Distance, routes[i].Distance; a > b {
			t.Errorf("order fail: expected route[%v].Distance %v <= route[%v].Distance %v", i-1, a, i, b)
		}
	}
}
