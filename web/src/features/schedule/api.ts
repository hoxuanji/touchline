export async function follow(teamId: number): Promise<void> {
  await fetch(`/api/follows/${teamId}`, { method: 'POST' })
}
export async function unfollow(teamId: number): Promise<void> {
  await fetch(`/api/follows/${teamId}`, { method: 'DELETE' })
}
export async function getFollows(): Promise<number[]> {
  const r = await fetch('/api/follows', { headers: { Accept: 'application/json' } })
  return r.json()
}
