import { useQuery } from '@tanstack/react-query'
import { http } from '@/shared/api/http'

export interface HealthResponse {
  status: string
}

export const healthKeys = {
  all: ['health'] as const,
}

export function fetchHealth(): Promise<HealthResponse> {
  return http<HealthResponse>('/healthz')
}

export function useHealth() {
  return useQuery({
    queryKey: healthKeys.all,
    queryFn: fetchHealth,
    retry: false,
  })
}
