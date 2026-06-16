import { useQuery } from '@tanstack/react-query'
import { api } from './api'
import type { Team } from './types'

export function useTeamNames(): (id: number) => string {
  const { data: teams = [] } = useQuery({ queryKey: ['teams'], queryFn: api.teams, staleTime: Infinity })
  const byId = new Map<number, string>((teams as Team[]).map((t) => [t.id, t.name]))
  return (id: number) => byId.get(id) ?? `#${id}`
}
