package app

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	app "github.com/oboGameDev/leaderboard/internal/applogic"
)

type Handler struct {
	app *app.App
}

func NewHandler(a *app.App) http.Handler {
	h := &Handler{app: a}
	mux := http.NewServeMux()

	mux.HandleFunc("/league/", h.handleLeague)
	mux.HandleFunc("/user/", h.handleUser)

	return mux
}
func (h *Handler) handleUser(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/user/"), "/")
	if len(parts) < 2 {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}

	userID := parts[0]
	action := parts[1]

	// /user/{id}/add?delta=100
	if action == "add" {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		q := r.URL.Query()
		deltaStr := q.Get("delta")
		delta, err := strconv.Atoi(deltaStr)
		if err != nil {
			http.Error(w, "bad delta", http.StatusBadRequest)
			return
		}

		newPoints, league, err := h.app.AddUserPoints(context.Background(), userID, int64(delta))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp := map[string]interface{}{
			"userId":    userID,
			"newPoints": newPoints,
			"newLeague": league,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	http.Error(w, "not found", http.StatusNotFound)
}

func (h *Handler) handleLeague(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/league/"), "/")
	if len(parts) < 2 {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}

	leagueID, err := strconv.Atoi(parts[0])
	if err != nil {
		http.Error(w, "bad league id", http.StatusBadRequest)
		return
	}

	sub := parts[1]

	// /league/{id}/leaderboard
	if sub == "leaderboard" {
		if r.Method != http.MethodGet {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}

		q := r.URL.Query()
		cursor := q.Get("cursor")
		limit := 20
		if v := q.Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = n
			}
		}

		items, next, err := h.app.GetLeagueLeaderboard(context.Background(), leagueID, cursor, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp := map[string]interface{}{
			"items":      items,
			"nextCursor": next,
			"limit":      limit,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// /league/{id}/user/{userId}/rank
	if sub == "user" {
		if len(parts) < 4 {
			http.Error(w, "bad path user", http.StatusBadRequest)
			return
		}

		userID := parts[2]
		action := parts[3]

		if action == "rank" {
			rank, err := h.app.GetUserRank(context.Background(), leagueID, userID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			resp := map[string]interface{}{
				"userId": userID,
				"league": leagueID,
				"rank":   rank,
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
	}

	http.Error(w, "not found", http.StatusNotFound)
}
