import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { splitwiseRequest } from '@/lib/api'
import type { SplitwiseMember } from '@/types/splitwise'

export function useSplitwiseMembers() {
  return useQuery({ queryKey: ['splitwise-members'], queryFn: () => splitwiseRequest<SplitwiseMember[]>('/members') })
}

export function useSaveSplitwiseMember() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (member: SplitwiseMember) => splitwiseRequest('/members', member),
    onSuccess: async () => { await client.invalidateQueries() },
  })
}
