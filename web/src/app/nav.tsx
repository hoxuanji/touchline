import { createContext, useContext, useState, useCallback, type ReactNode } from 'react'

export type Route =
  | { page: 'match-center' }
  | { page: 'match'; id: number }
  | { page: 'schedule' }
  | { page: 'standings' }
  | { page: 'explore' }
  | { page: 'team'; id: number }
  | { page: 'venue'; id: number }

interface NavCtx { route: Route; go: (r: Route) => void; back: () => void }

const Ctx = createContext<NavCtx>({
  route: { page: 'match-center' },
  go: () => {},
  back: () => {},
})

export function NavProvider({ children }: { children: ReactNode }) {
  const [stack, setStack] = useState<Route[]>([{ page: 'match-center' }])
  const go = useCallback((r: Route) => setStack(s => [...s, r]), [])
  const back = useCallback(() => setStack(s => s.length > 1 ? s.slice(0, -1) : s), [])
  return <Ctx.Provider value={{ route: stack[stack.length - 1], go, back }}>{children}</Ctx.Provider>
}

export function useNav() { return useContext(Ctx) }
