import { useQuery } from '@tanstack/react-query'

export interface Meta { source: string }

async function getMeta(): Promise<Meta> {
  const r = await fetch('/api/meta', { headers: { Accept: 'application/json' } })
  return r.json()
}

export function useMeta() {
  return useQuery({ queryKey: ['meta'], queryFn: getMeta, staleTime: Infinity })
}
