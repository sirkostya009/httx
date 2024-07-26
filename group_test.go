package httx_test

import (
	"encoding/json"
	"github.com/sirkostya009/httx"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGroupRoutes(t *testing.T) {
	mux := httx.NewServeMux()

	g := mux.Group("/group")

	g.HandleFunc("GET /yoo", func(ctx *httx.Context) error {
		return ctx.WriteJSON(http.StatusAccepted, map[string]string{
			"boo": "yoo",
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	res, err := http.Get(server.URL + "/group/yoo")
	if err != nil {
		t.Error(err)
	}

	if res.StatusCode != http.StatusAccepted {
		t.Errorf("status was not 202")
	}

	m := map[string]string{}
	err = json.NewDecoder(res.Body).Decode(&m)
	if err != nil {
		t.Fatal(err)
	}

	if v, ok := m["boo"]; ok {
		if v != "yoo" {
			t.Errorf("'boo' isn't equal to \"yoo\"")
		}
	} else {
		t.Errorf("no 'boo'")
	}
}
