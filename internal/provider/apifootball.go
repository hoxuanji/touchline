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

// APIFootball implements Provider using api-sports.io v3. Live polling is
// enabled via LIVE_SOURCE=real; reference reads are caller-budget-capped.
// Fixtures/Venues/Standings parsing returns not-implemented for V1 — snapshot
// remains seed/fallback until field shapes are confirmed with a paid key.
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

func (a *APIFootball) Venues(ctx context.Context) ([]model.Venue, error) {
	return nil, fmt.Errorf("apifootball venues: not yet implemented; snapshot is authoritative")
}
func (a *APIFootball) Fixtures(ctx context.Context) ([]model.Match, error) {
	return nil, fmt.Errorf("apifootball fixtures: not yet implemented; snapshot is authoritative")
}
func (a *APIFootball) Standings(ctx context.Context) ([]model.Standing, error) {
	return nil, fmt.Errorf("apifootball standings: not yet implemented; snapshot is authoritative")
}

var _ Provider = (*APIFootball)(nil)
