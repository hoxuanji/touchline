import { QueryClientProvider } from '@tanstack/react-query'
import { queryClient } from './app/queryClient'
import Shell from './app/Shell'

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <Shell />
    </QueryClientProvider>
  )
}
