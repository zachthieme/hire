import { useParams, Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { candidates as candApi, loops as loopsApi } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Plus, ExternalLink } from 'lucide-react'

export default function CandidateDetail() {
  const { id } = useParams<{ id: string }>()
  const candidateId = parseInt(id!)
  const queryClient = useQueryClient()

  const { data: candidate } = useQuery({ queryKey: ['candidates', candidateId], queryFn: () => candApi.get(candidateId) })
  const { data: candidateLoops = [] } = useQuery({
    queryKey: ['loops', { candidate_id: candidateId }],
    queryFn: () => loopsApi.list({ candidate_id: candidateId }),
  })

  const createLoop = useMutation({
    mutationFn: () => loopsApi.create({ candidate_id: candidateId }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['loops'] }),
  })

  if (!candidate) return <div>Loading...</div>

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{candidate.name}</h1>
          <p className="text-gray-500">{candidate.email}</p>
          {candidate.resume_url && (
            <a href={candidate.resume_url} target="_blank" rel="noopener noreferrer" className="text-blue-600 text-sm flex items-center gap-1">
              Resume <ExternalLink className="h-3 w-3" />
            </a>
          )}
        </div>
        <Button onClick={() => createLoop.mutate()}>
          <Plus className="h-4 w-4 mr-2" />New Interview Loop
        </Button>
      </div>

      {candidateLoops.length === 0 && (
        <p className="text-gray-500">No interview loops yet. Create one to get started.</p>
      )}

      {candidateLoops.map(loop => (
        <Card key={loop.id}>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="text-lg">Loop #{loop.id}</CardTitle>
            <div className="flex items-center gap-2">
              <Badge variant={loop.status === 'complete' ? 'default' : 'secondary'}>{loop.status}</Badge>
              <Link to={`/loops/${loop.id}/edit`}>
                <Button variant="outline" size="sm">Edit Loop</Button>
              </Link>
              <Link to={`/loops/${loop.id}/debrief`}>
                <Button variant="outline" size="sm">Debrief</Button>
              </Link>
            </div>
          </CardHeader>
          <CardContent>
            {loop.final_decision && (
              <p className="text-sm">Final Decision: <strong>{loop.final_decision.replace(/_/g, ' ')}</strong></p>
            )}
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
