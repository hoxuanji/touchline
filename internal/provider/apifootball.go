package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"touchline/internal/model"
)

// APIFootball implements Provider using api-sports.io v3.
type APIFootball struct {
	key    string
	base   string
	client *http.Client
}

func NewAPIFootball(key string) *APIFootball {
	return &APIFootball{
		key:    key,
		base:   "https://v3.football.api-sports.io",
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// apiResponse is the common API-Football envelope.
type apiResponse struct {
	Errors  interface{}     `json:"errors"`
	Results int             `json:"results"`
	Raw     json.RawMessage `json:"response"`
}

func (a *APIFootball) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.base+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-apisports-key", a.key)
	req.Header.Set("x-rapidapi-host", "v3.football.api-sports.io")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("apifootball %s: HTTP %d: %s", path, resp.StatusCode, string(body))
	}

	// Check for API-level errors (returned as HTTP 200 with error body)
	var env apiResponse
	if err := json.Unmarshal(body, &env); err == nil {
		// errors is {} (empty object) or [] (empty array) when no errors
		switch e := env.Errors.(type) {
		case map[string]interface{}:
			if len(e) > 0 {
				// Extract first error message
				for k, v := range e {
					return nil, fmt.Errorf("apifootball %s: %s: %v", path, k, v)
				}
			}
		}
		if env.Results == 0 && path != "/status" {
			// Could be a valid empty response or a domain block
		}
	}

	return body, nil
}

// Status calls /status and returns the raw response for diagnostics.
func (a *APIFootball) Status(ctx context.Context) (map[string]interface{}, error) {
	b, err := a.get(ctx, "/status")
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(b, &result)
	return result, err
}

// ── Teams ──

func parseTeams(b []byte) ([]model.Team, error) {
	var env struct {
		Response []struct {
			Team struct {
				ID      int    `json:"id"`
				Name    string `json:"name"`
				Country string `json:"country"`
				Logo    string `json:"logo"`
			} `json:"team"`
		} `json:"response"`
	}
	if err := json.Unmarshal(b, &env); err != nil {
		return nil, err
	}
	out := make([]model.Team, 0, len(env.Response))
	for _, r := range env.Response {
		out = append(out, model.Team{
			ID:       r.Team.ID,
			Name:     r.Team.Name,
			Country:  r.Team.Country,
			CrestURL: r.Team.Logo,
		})
	}
	return out, nil
}

func (a *APIFootball) Teams(ctx context.Context) ([]model.Team, error) {
	b, err := a.get(ctx, "/teams?league=1&season=2026")
	if err != nil {
		return nil, err
	}
	return parseTeams(b)
}

// ── Fixtures ──

func parseFixtures(b []byte) ([]model.Match, error) {
	var env struct {
		Response []struct {
			Fixture struct {
				ID     int `json:"id"`
				Status struct {
					Short   string `json:"short"`
					Elapsed *int   `json:"elapsed"`
				} `json:"status"`
				Date string `json:"date"`
				Venue struct {
					ID *int `json:"id"`
				} `json:"venue"`
			} `json:"fixture"`
			League struct {
				Round string `json:"round"`
				Group string `json:"group"`
			} `json:"league"`
			Teams struct {
				Home struct{ ID int `json:"id"` } `json:"home"`
				Away struct{ ID int `json:"id"` } `json:"away"`
			} `json:"teams"`
			Goals struct {
				Home *int `json:"home"`
				Away *int `json:"away"`
			} `json:"goals"`
		} `json:"response"`
	}
	if err := json.Unmarshal(b, &env); err != nil {
		return nil, err
	}

	statusMap := map[string]string{
		"NS": "scheduled", "TBD": "scheduled",
		"1H": "live", "2H": "live", "ET": "live", "P": "live", "LIVE": "live",
		"HT":  "ht",
		"FT":  "finished", "AET": "finished", "PEN": "finished",
		"PST": "scheduled", "CANC": "scheduled", "ABD": "scheduled",
	}

	out := make([]model.Match, 0, len(env.Response))
	for _, r := range env.Response {
		kickoff, _ := time.Parse(time.RFC3339, r.Fixture.Date)
		status := statusMap[r.Fixture.Status.Short]
		if status == "" {
			status = "scheduled"
		}
		minute := 0
		if r.Fixture.Status.Elapsed != nil {
			minute = *r.Fixture.Status.Elapsed
		}
		homeScore, awayScore := 0, 0
		if r.Goals.Home != nil {
			homeScore = *r.Goals.Home
		}
		if r.Goals.Away != nil {
			awayScore = *r.Goals.Away
		}
		venueID := 0
		if r.Fixture.Venue.ID != nil {
			venueID = *r.Fixture.Venue.ID
		}

		// Derive stage from round string
		stage := "group"
		round := r.League.Round
		switch {
		case contains(round, "Final") && contains(round, "3rd"):
			stage = "third"
		case contains(round, "Final") && !contains(round, "Semi") && !contains(round, "Quarter"):
			stage = "final"
		case contains(round, "Semi"):
			stage = "sf"
		case contains(round, "Quarter"):
			stage = "qf"
		case contains(round, "16"):
			stage = "r16"
		case contains(round, "32"):
			stage = "r32"
		}

		out = append(out, model.Match{
			ID:         r.Fixture.ID,
			Stage:      stage,
			Group:      r.League.Group,
			VenueID:    venueID,
			HomeID:     r.Teams.Home.ID,
			AwayID:     r.Teams.Away.ID,
			KickoffUTC: kickoff,
			Status:     status,
			Minute:     minute,
			HomeScore:  homeScore,
			AwayScore:  awayScore,
		})
	}
	return out, nil
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}

func (a *APIFootball) Fixtures(ctx context.Context) ([]model.Match, error) {
	b, err := a.get(ctx, "/fixtures?league=1&season=2026")
	if err != nil {
		return nil, err
	}
	return parseFixtures(b)
}

// ── Standings ──

func parseStandings(b []byte) ([]model.Standing, error) {
	var env struct {
		Response []struct {
			League struct {
				Standings [][]struct {
					Team struct{ ID int `json:"id"` } `json:"team"`
					Points    int    `json:"points"`
					Group     string `json:"group"`
					All struct {
						Played int `json:"played"`
						Win    int `json:"win"`
						Draw   int `json:"draw"`
						Lose   int `json:"lose"`
						Goals  struct {
							For     int `json:"for"`
							Against int `json:"against"`
						} `json:"goals"`
					} `json:"all"`
					Form string `json:"form"`
				} `json:"standings"`
			} `json:"league"`
		} `json:"response"`
	}
	if err := json.Unmarshal(b, &env); err != nil {
		return nil, err
	}
	var out []model.Standing
	for _, resp := range env.Response {
		for _, group := range resp.League.Standings {
			for _, row := range group {
				out = append(out, model.Standing{
					Group:  row.Group,
					TeamID: row.Team.ID,
					Played: row.All.Played,
					Won:    row.All.Win,
					Drawn:  row.All.Draw,
					Lost:   row.All.Lose,
					GF:     row.All.Goals.For,
					GA:     row.All.Goals.Against,
					Pts:    row.Points,
					Form:   row.Form,
				})
			}
		}
	}
	return out, nil
}

func (a *APIFootball) Standings(ctx context.Context) ([]model.Standing, error) {
	b, err := a.get(ctx, "/standings?league=1&season=2026")
	if err != nil {
		return nil, err
	}
	return parseStandings(b)
}

func (a *APIFootball) Venues(ctx context.Context) ([]model.Venue, error) {
	return nil, fmt.Errorf("apifootball venues: use snapshot")
}

var _ Provider = (*APIFootball)(nil)
