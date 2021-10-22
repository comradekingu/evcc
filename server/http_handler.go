package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"text/template"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/util"
	"github.com/gorilla/mux"
)

const (
	result  = "result"
	failure = "error"
)

func indexHandler(site site.API) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")

		indexTemplate, err := fs.ReadFile(Assets, "index.html")
		if err != nil {
			log.FATAL.Print("httpd: failed to load embedded template:", err.Error())
			log.FATAL.Fatal("Make sure templates are included using the `release` build tag or use `make build`")
		}

		t, err := template.New("evcc").Delims("[[", "]]").Parse(string(indexTemplate))
		if err != nil {
			log.FATAL.Fatal("httpd: failed to create main page template:", err.Error())
		}

		if err := t.Execute(w, map[string]interface{}{
			"Version":    Version,
			"Commit":     Commit,
			"Configured": len(site.LoadPoints()),
		}); err != nil {
			log.ERROR.Println("httpd: failed to render main page:", err.Error())
		}
	})
}

// jsonHandler is a middleware that decorates responses with JSON and CORS headers
func jsonHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		h.ServeHTTP(w, r)
	})
}

func jsonResponse(w http.ResponseWriter, r *http.Request, content interface{}) {
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(content); err != nil {
		log.ERROR.Printf("httpd: failed to encode JSON: %v", err)
	}
}

// healthHandler returns current charge mode
func healthHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !site.Healthy() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	}
}

// stateHandler returns current charge mode
func stateHandler(cache *util.Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := cache.State()
		for _, k := range []string{"availableVersion", "releaseNotes"} {
			delete(res, k)
		}
		jsonResponse(w, r, res)
	}
}

// chargeModeHandler updates charge mode
func chargeModeHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		mode, err := api.ChargeModeString(vars["value"])
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			jsonResponse(w, r, map[string]interface{}{failure: err.Error()})
			return
		}

		lp.SetMode(mode)

		jsonResponse(w, r, map[string]interface{}{result: lp.GetMode()})
	}
}

// targetSoCHandler updates target soc
func targetSoCHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		soc, err := strconv.ParseInt(vars["value"], 10, 32)
		if err == nil {
			err = lp.SetTargetSoC(int(soc))
		}

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			jsonResponse(w, r, map[string]interface{}{failure: err.Error()})
			return
		}

		jsonResponse(w, r, map[string]interface{}{result: lp.GetTargetSoC()})
	}
}

// minSoCHandler updates minimum soc
func minSoCHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		soc, err := strconv.ParseInt(vars["value"], 10, 32)
		if err == nil {
			err = lp.SetMinSoC(int(soc))
		}

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			jsonResponse(w, r, map[string]interface{}{failure: err.Error()})
			return
		}

		jsonResponse(w, r, map[string]interface{}{result: lp.GetMinSoC()})
	}
}

// minCurrentHandler updates minimum current
func minCurrentHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		current, err := strconv.ParseFloat(vars["value"], 64)
		if err == nil {
			lp.SetMinCurrent(current)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			jsonResponse(w, r, map[string]interface{}{failure: err.Error()})
			return
		}

		jsonResponse(w, r, map[string]interface{}{result: lp.GetMinCurrent()})
	}
}

// maxCurrentHandler updates maximum current
func maxCurrentHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		current, err := strconv.ParseFloat(vars["value"], 64)
		if err == nil {
			lp.SetMaxCurrent(current)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			jsonResponse(w, r, map[string]interface{}{failure: err.Error()})
			return
		}

		jsonResponse(w, r, map[string]interface{}{result: lp.GetMaxCurrent()})
	}
}

// phasesHandler updates minimum soc
func phasesHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		phases, err := strconv.ParseInt(vars["value"], 10, 32)
		if err == nil {
			err = lp.SetPhases(int(phases))
		}

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			jsonResponse(w, r, map[string]interface{}{failure: err.Error()})
			return
		}

		jsonResponse(w, r, map[string]interface{}{result: lp.GetPhases()})
	}
}

// remoteDemandHandler updates minimum soc
func remoteDemandHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		demandS, ok := vars["demand"]

		var source string
		if ok {
			source, ok = vars["source"]
		}

		demand, err := loadpoint.RemoteDemandString(demandS)

		if !ok || err != nil {
			w.WriteHeader(http.StatusBadRequest)
			jsonResponse(w, r, map[string]interface{}{failure: err.Error()})
			return
		}

		lp.RemoteControl(source, demand)

		res := struct {
			Demand loadpoint.RemoteDemand `json:"demand"`
			Source string                 `json:"source"`
		}{
			Source: source,
			Demand: demand,
		}

		jsonResponse(w, r, res)
	}
}

func timezone() *time.Location {
	tz := os.Getenv("TZ")
	if tz == "" {
		tz = "Local"
	}

	loc, _ := time.LoadLocation(tz)
	return loc
}

// targetChargeHandler updates target soc
func targetChargeHandler(loadpoint loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		socS, ok := vars["soc"]
		socV, err := strconv.ParseInt(socS, 10, 32)

		if !ok || err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		timeS, ok := vars["time"]
		timeV, err := time.ParseInLocation("2006-01-02T15:04:05", timeS, timezone())

		if !ok || err != nil {
			w.WriteHeader(http.StatusBadRequest)
			jsonResponse(w, r, map[string]interface{}{failure: err.Error()})
			return
		}

		loadpoint.SetTargetCharge(timeV, int(socV))

		res := struct {
			SoC  int64     `json:"soc"`
			Time time.Time `json:"time"`
		}{
			SoC:  socV,
			Time: timeV,
		}

		jsonResponse(w, r, res)
	}
}

// socketHandler attaches websocket handler to uri
func socketHandler(hub *SocketHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ServeWebsocket(hub, w, r)
	}
}