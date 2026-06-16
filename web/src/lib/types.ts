export interface Team { id: number; name: string; country: string; group: string; crestUrl: string }
export interface Venue { id: number; name: string; city: string; country: string; capacity: number }
export interface Match {
  id: number; stage: string; group: string; venueId: number;
  homeId: number; awayId: number; kickoffUtc: string;
  status: string; minute: number; homeScore: number; awayScore: number;
}
export interface Standing {
  group: string; teamId: number; played: number; won: number; drawn: number;
  lost: number; gf: number; ga: number; pts: number; form: string;
}
export interface MatchEvent {
  id: number; matchId: number; minute: number; type: string;
  teamId: number; playerId: number; detail: string;
}
