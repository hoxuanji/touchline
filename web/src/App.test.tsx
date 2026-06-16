import { render, screen } from '@testing-library/react'
import App from './App'

test('renders the Touchline shell title and tournament tag', () => {
  render(<App />)
  expect(screen.getByText('TOUCHLINE')).toBeInTheDocument()
  expect(screen.getByText(/WORLD CUP 2026/i)).toBeInTheDocument()
})
