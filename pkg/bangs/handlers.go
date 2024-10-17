package bangs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"bangs/pkg/middleware"

	"github.com/sett17/prettyslog"
)

func Handler(doAllowNoBang bool) http.Handler {
	allowNoBang = doAllowNoBang

	router := http.NewServeMux()

	router.HandleFunc("/list", listAll)
	router.HandleFunc("/", searchByQuery)
	router.HandleFunc("/{bang}/{query...}", searchByPath)

	logOptions := make([]prettyslog.Option, 0)
	if slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		logOptions = append(logOptions, prettyslog.WithLevel(slog.LevelDebug))
	}
	logger := slog.New(prettyslog.NewPrettyslogHandler("HTTP", logOptions...))
	stack := middleware.CreateStack(
		middleware.Logger(logger, "bang"),
	)

	return stack(router)
}

func listAll(w http.ResponseWriter, r *http.Request) {
	asJSON, err := json.Marshal(All().Entries)
	if err != nil {
		slog.Error("Error converting registry to json", "err", err)
		http.Error(w, fmt.Sprintf("Internal JSON error -.-\n%v\n", err), http.StatusInternalServerError)
		return
	}
	_, err = w.Write(asJSON)
	if err != nil {
		slog.Error("Error writing response", "err", err)
		return
	}
}

func searchByQuery(w http.ResponseWriter, r *http.Request) {
	queries := r.URL.Query()
	q := queries.Get("q")
	if strings.TrimSpace(q) == "" {
		msg := "No query provided for search"
		slog.Error(msg, "url", r.URL)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	entry, query, err := registry.Entries.PrepareInput(q)
	if err != nil {
		if _, ok := err.(InputHasNoBangError); ok {
			slog.Debug("No bang found in input, forwarding to default", "query", q)
			_ = registry.DefaultForward(q, w, r)
			return
		}
		slog.Error("Error preparing input", "err", err)
		http.Error(w, fmt.Sprintf("Error preparing input: %v", err), http.StatusBadRequest)
		return
	}

	_ = entry.Forward(query, w, r)
}

func searchByPath(w http.ResponseWriter, r *http.Request) {
	bang := r.URL.Query().Get("bang")
	bang = strings.TrimSpace(bang)
	if len(bang) == 0 {
		msg := "No bang provided for search"
		slog.Error(msg, "url", r.URL)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	query := r.URL.Query().Get("query")
	query = strings.TrimSpace(query)
	if len(query) == 0 {
		msg := "No query provided for search"
		slog.Error(msg, "url", r.URL)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	entry, ok := registry.Entries.byBang[bang]
	if !ok {
		msg := fmt.Sprintf("Unknown bang: '%s'", bang)
		slog.Error(msg, "url", r.URL)
		http.Error(w, msg, http.StatusNotFound)
		return
	}

	_ = entry.Forward(query, w, r)
}
