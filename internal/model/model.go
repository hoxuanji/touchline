// Package model holds Touchline's domain types. Pure data, JSON-tagged for the API.
package model

import "time"

type Team struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Country  string `json:"country"`
	Group    string `json:"group"`
	CrestURL string `json:"crestUrl"`
}

type Venue struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	City     string `json:"city"`
	Country  string `json:"country"`
	Capacity int    `json:"capacity"`
}

type Player struct {
	ID       int    `json:"id"`
	TeamID   int    `json:"teamId"`
	Name     string `json:"name"`
	Position string `json:"position"`
	Number   int    `json:"number"`
}

type Match struct {
	ID         int       `json:"id"`
	Stage      string    `json:"stage"` // group | r32 | r16 | qf | sf | third | final
	Group      string    `json:"group"`
	VenueID    int       `json:"venueId"`
	HomeID     int       `json:"homeId"`
	AwayID     int       `json:"awayId"`
	KickoffUTC time.Time `json:"kickoffUtc"`
	Status     string    `json:"status"` // scheduled | live | ht | finished
	Minute     int       `json:"minute"`
	HomeScore  int       `json:"homeScore"`
	AwayScore  int       `json:"awayScore"`
}

type MatchEvent struct {
	ID       int    `json:"id"`
	MatchID  int    `json:"matchId"`
	Minute   int    `json:"minute"`
	Type     string `json:"type"` // goal | card | sub | var
	TeamID   int    `json:"teamId"`
	PlayerID int    `json:"playerId"`
	Detail   string `json:"detail"`
}

type Standing struct {
	Group  string `json:"group"`
	TeamID int    `json:"teamId"`
	Played int    `json:"played"`
	Won    int    `json:"won"`
	Drawn  int    `json:"drawn"`
	Lost   int    `json:"lost"`
	GF     int    `json:"gf"`
	GA     int    `json:"ga"`
	Pts    int    `json:"pts"`
	Form   string `json:"form"`
}

type Note struct {
	ID          int       `json:"id"`
	SubjectType string    `json:"subjectType"` // match | team | player
	SubjectID   int       `json:"subjectId"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Follow struct {
	TeamID int `json:"teamId"`
}

type Reminder struct {
	ID          int  `json:"id"`
	MatchID     int  `json:"matchId"`
	LeadMinutes int  `json:"leadMinutes"`
	Fired       bool `json:"fired"`
}

type Pref struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
