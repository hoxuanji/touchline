import { useLiveStream } from '../lib/useLiveStream'
import { useMeta } from '../features/meta/useMeta'
import { useNav, NavProvider } from './nav'
import MatchCenterPage from '../features/match-center/MatchCenterPage'
import MatchDetailPage from '../features/match-center/MatchDetailPage'
import SchedulePage from '../features/schedule/SchedulePage'
import StandingsPage from '../features/standings/StandingsPage'
import ExplorePage from '../features/explore/ExplorePage'
import TeamDetailPage from '../features/explore/TeamDetailPage'
import VenueDetailPage from '../features/explore/VenueDetailPage'

const TOP_VIEWS = ['Match Center', 'Schedule', 'Standings', 'Explore'] as const
type TopView = (typeof TOP_VIEWS)[number]

const PAGE_MAP: Record<TopView, string> = {
  'Match Center': 'match-center',
  Schedule: 'schedule',
  Standings: 'standings',
  Explore: 'explore',
}

function AppContent() {
  const { route, go } = useNav()
  const { connected } = useLiveStream()
  const { data: meta } = useMeta()

  const activeTab = (() => {
    if (route.page === 'match-center' || route.page === 'match') return 'Match Center'
    if (route.page === 'schedule') return 'Schedule'
    if (route.page === 'standings') return 'Standings'
    if (route.page === 'explore' || route.page === 'team' || route.page === 'venue') return 'Explore'
    return 'Match Center'
  })()

  function renderPage() {
    switch (route.page) {
      case 'match-center': return <MatchCenterPage />
      case 'match': return <MatchDetailPage matchId={route.id} />
      case 'schedule': return <SchedulePage />
      case 'standings': return <StandingsPage />
      case 'explore': return <ExplorePage />
      case 'team': return <TeamDetailPage teamId={route.id} />
      case 'venue': return <VenueDetailPage venueId={route.id} />
      default: return <MatchCenterPage />
    }
  }

  return (
    <main className="shell">
      <header className="shell__bar">
        <span className="shell__logo">
          <span className="shell__mark">▲</span>
          <h1 className="shell__title">TOUCHLINE</h1>
        </span>
        <span className="shell__tag">FIFA WORLD CUP 2026</span>
        <nav className="shell__nav">
          {TOP_VIEWS.map((v) => (
            <button
              key={v}
              className={activeTab === v ? 'is-active' : ''}
              onClick={() => go({ page: PAGE_MAP[v] as 'match-center' | 'schedule' | 'standings' | 'explore' })}
            >{v}</button>
          ))}
        </nav>
        <span className="shell__status">
          <span className={`shell__live ${connected ? 'is-on' : ''}`} title={connected ? 'Live' : 'Connecting'}>●</span>
          {meta?.source && <span className="shell__source">{meta.source}</span>}
        </span>
      </header>
      <div className="shell__body">
        {renderPage()}
      </div>
    </main>
  )
}

export default function Shell() {
  return (
    <NavProvider>
      <AppContent />
    </NavProvider>
  )
}
