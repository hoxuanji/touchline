package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"touchline/internal/model"
)

// FootballData implements Provider using the football-data.org v4 API.
// Free tier includes FIFA World Cup. Register at football-data.org for a token.
type FootballData struct {
	token  string
	client *http.Client
}

func NewFootballData(token string) *FootballData {
	return &FootballData{
		token:  token,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (f *FootballData) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.football-data.org/v4"+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", f.token)
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		// Extract message from response if possible
		var errBody struct{ Message string `json:"message"` }
		if json.Unmarshal(body, &errBody) == nil && errBody.Message != "" {
			return nil, fmt.Errorf("football-data %s: %s", path, errBody.Message)
		}
		return nil, fmt.Errorf("football-data %s: HTTP %d", path, resp.StatusCode)
	}
	return body, nil
}

// statusMap converts football-data status strings to our model values.
var fdStatusMap = map[string]string{
	"SCHEDULED": "scheduled", "TIMED": "scheduled",
	"IN_PLAY": "live", "LIVE": "live",
	"PAUSED":    "ht",
	"FINISHED":  "finished",
	"SUSPENDED": "scheduled", "POSTPONED": "scheduled", "CANCELLED": "scheduled",
}

func (f *FootballData) Teams(ctx context.Context) ([]model.Team, error) {
	b, err := f.get(ctx, "/competitions/WC/teams?season=2026")
	if err != nil {
		return nil, err
	}
	var doc struct {
		Teams []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Area struct{ Name string `json:"name"` } `json:"area"`
			Crest string `json:"crest"`
		} `json:"teams"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	out := make([]model.Team, 0, len(doc.Teams))
	for _, t := range doc.Teams {
		out = append(out, model.Team{
			ID:       t.ID,
			Name:     t.Name,
			Country:  t.Area.Name,
			CrestURL: t.Crest,
		})
	}
	return out, nil
}

func (f *FootballData) Fixtures(ctx context.Context) ([]model.Match, error) {
	b, err := f.get(ctx, "/competitions/WC/matches?season=2026")
	if err != nil {
		return nil, err
	}
	var doc struct {
		Matches []struct {
			ID      int    `json:"id"`
			UTCDate string `json:"utcDate"`
			Status  string `json:"status"`
			Minute  *int   `json:"minute"`
			Stage   string `json:"stage"`
			Group   string `json:"group"`
			HomeTeam struct{ ID int `json:"id"` } `json:"homeTeam"`
			AwayTeam struct{ ID int `json:"id"` } `json:"awayTeam"`
			Score struct {
				FullTime struct {
					Home *int `json:"home"`
					Away *int `json:"away"`
				} `json:"fullTime"`
				HalfTime struct {
					Home *int `json:"home"`
					Away *int `json:"away"`
				} `json:"halfTime"`
			} `json:"score"`
		} `json:"matches"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}

	out := make([]model.Match, 0, len(doc.Matches))
	for _, m := range doc.Matches {
		kickoff, _ := time.Parse(time.RFC3339, m.UTCDate)
		status := fdStatusMap[m.Status]
		if status == "" {
			status = "scheduled"
		}
		minute := 0
		if m.Minute != nil {
			minute = *m.Minute
		}
		homeScore, awayScore := 0, 0
		if m.Score.FullTime.Home != nil {
			homeScore = *m.Score.FullTime.Home
		}
		if m.Score.FullTime.Away != nil {
			awayScore = *m.Score.FullTime.Away
		}

		stage := stageFromFD(m.Stage)
		group := groupFromFD(m.Group)

		out = append(out, model.Match{
			ID:         m.ID,
			Stage:      stage,
			Group:      group,
			HomeID:     m.HomeTeam.ID,
			AwayID:     m.AwayTeam.ID,
			KickoffUTC: kickoff,
			Status:     status,
			Minute:     minute,
			HomeScore:  homeScore,
			AwayScore:  awayScore,
		})
	}
	return out, nil
}

func (f *FootballData) Standings(ctx context.Context) ([]model.Standing, error) {
	b, err := f.get(ctx, "/competitions/WC/standings?season=2026")
	if err != nil {
		return nil, err
	}
	var doc struct {
		Standings []struct {
			Type  string `json:"type"`
			Group string `json:"group"`
			Table []struct {
				Team struct{ ID int `json:"id"` } `json:"team"`
				PlayedGames    int    `json:"playedGames"`
				Won            int    `json:"won"`
				Draw           int    `json:"draw"`
				Lost           int    `json:"lost"`
				GoalsFor       int    `json:"goalsFor"`
				GoalsAgainst   int    `json:"goalsAgainst"`
				Points         int    `json:"points"`
				Form           string `json:"form"`
			} `json:"table"`
		} `json:"standings"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	var out []model.Standing
	for _, s := range doc.Standings {
		if s.Type != "TOTAL" {
			continue
		}
		group := groupFromFD(s.Group)
		for _, row := range s.Table {
			out = append(out, model.Standing{
				Group:  group,
				TeamID: row.Team.ID,
				Played: row.PlayedGames,
				Won:    row.Won,
				Drawn:  row.Draw,
				Lost:   row.Lost,
				GF:     row.GoalsFor,
				GA:     row.GoalsAgainst,
				Pts:    row.Points,
				Form:   row.Form,
			})
		}
	}
	return out, nil
}

func (f *FootballData) Venues(ctx context.Context) ([]model.Venue, error) {
	return nil, fmt.Errorf("football-data: venues not available; using snapshot")
}

var _ Provider = (*FootballData)(nil)

// stageFromFD maps football-data.org stage strings to our model values.
func stageFromFD(s string) string {
	switch {
	case s == "GROUP_STAGE":
		return "group"
	case strings.Contains(s, "ROUND_OF_32"):
		return "r32"
	case strings.Contains(s, "ROUND_OF_16"):
		return "r16"
	case strings.Contains(s, "QUARTER"):
		return "qf"
	case strings.Contains(s, "SEMI"):
		return "sf"
	case strings.Contains(s, "THIRD"):
		return "third"
	case strings.Contains(s, "FINAL"):
		return "final"
	default:
		return "group"
	}
}

// groupFromFD converts "GROUP_A" → "A".
func groupFromFD(g string) string {
	g = strings.TrimPrefix(g, "GROUP_")
	if len(g) == 1 {
		return g
	}
	return ""
}
