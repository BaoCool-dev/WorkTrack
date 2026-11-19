import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { HealthStatus } from './HealthStatus'
import { messages } from '@/shared/lib/messages'

function renderWithClient() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <HealthStatus />
    </QueryClientProvider>,
  )
}

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('HealthStatus', () => {
  it('hiển thị trạng thái hoạt động khi /healthz trả 200', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse(200, { status: 'ok' })))

    renderWithClient()

    expect(screen.getByText(messages.health.checking)).toBeInTheDocument()
    expect(await screen.findByText(messages.health.ok)).toBeInTheDocument()
  })

  it('hiển thị lỗi và nút thử lại khi không kết nối được', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))

    renderWithClient()

    expect(await screen.findByRole('alert')).toHaveTextContent(messages.health.error)
    expect(screen.getByRole('button', { name: messages.health.retry })).toBeInTheDocument()
  })
})
