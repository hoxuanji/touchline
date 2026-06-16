package provider

import "testing"

const sampleTeams = `{"response":[
  {"team":{"id":21,"name":"Brazil","country":"Brazil","logo":"https://x/21.png"}},
  {"team":{"id":17,"name":"France","country":"France","logo":"https://x/17.png"}}
]}`

func TestParseTeamsResponse(t *testing.T) {
	teams, err := parseTeams([]byte(sampleTeams))
	if err != nil {
		t.Fatal(err)
	}
	if len(teams) != 2 || teams[0].ID != 21 || teams[0].Name != "Brazil" || teams[0].CrestURL == "" {
		t.Fatalf("parsed = %+v", teams)
	}
}
