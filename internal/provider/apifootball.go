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
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *APIFootball) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.base+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-apisports-key", a.key)
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("apifootball %s: status %d", path, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// ── Teams ──

func parseTeams(b []byte) ([]model.Team, error) {
	var doc struct {
		Response []struct {
			Team struct {
				ID      int    `json:"id"`
				Name    string `json:"name"`
				Country string `json:"country"`
				Logo    string `json:"logo"`
			} `json:"team"`
		} `json:"response"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	out := make([]model.Team, 0, len(doc.Response))
	for _, r := range doc.Response {
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
	var doc struct {
		Response []struct {
			Fixture struct {
				ID     int    `json:"id"`
				Status struct {
					Short  string `json:"short"`
					Elapsed *int  `json:"elapsed"`
				} `json:"status"`
				Date string `json:"date"`
			} `json:"fixture"`
			League struct {
				Round string `json:"round"`
			} `json:"league"`
			Teams struct {
				Home struct{ ID int `json:"id"` } `json:"home"`
				Away struct{ ID int `json:"id"` } `json:"away"`
			} `json:"teams"`
			Goals struct {
				Home *int `json:"home"`
				Away *int `json:"away"`
			} `json:"goals"`
			Score struct {
				Halftime struct {
					Home *int `json:"home"`
					Away *int `json:"away"`
				} `json:"halftime"`
			} `json:"score"`
		} `json:"response"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}

	statusMap := map[string]string{
		"NS": "scheduled", "TBD": "scheduled",
		"1H": "live", "2H": "live", "ET": "live", "P": "live", "LIVE": "live",
		"HT": "ht",
		"FT": "finished", "AET": "finished", "PEN": "finished",
		"PST": "scheduled", "CANC": "scheduled", "ABD": "scheduled",
	}

	out := make([]model.Match, 0, len(doc.Response))
	for _, r := range doc.Response {
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
		out = append(out, model.Match{
			ID:         r.Fixture.ID,
			Stage:      "group", // simplified; round parsing can be added
			Group:      "",
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

func (a *APIFootball) Fixtures(ctx context.Context) ([]model.Match, error) {
	b, err := a.get(ctx, "/fixtures?league=1&season=2026")
	if err != nil {
		return nil, err
	}
	return parseFixtures(b)
}

// ── Standings ──

func parseStandings(b []byte) ([]model.Standing, error) {
	var doc struct {
		Response []struct {
			League struct {
				Standings [][]struct {
					Rank int    `json:"rank"`
					Team struct{ ID int `json:"id"` } `json:"team"`
					Points int `json:"points"`
					GoalsDiff int `json:"goalsDiff"`
					Group string `json:"group"`
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
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	var out []model.Standing
	for _, resp := range doc.Response {
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

// ── Venues ──

func (a *APIFootball) Venues(ctx context.Context) ([]model.Venue, error) {
	// Venues are structural WC2026 data not available per-league on free tier;
	// the bundled snapshot is authoritative.
	return nil, fmt.Errorf("apifootball venues: use snapshot")
}

var _ Provider = (*APIFootball)(nil)
