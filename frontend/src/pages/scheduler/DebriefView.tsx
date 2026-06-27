import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { loops as loopsApi, competencies as compApi, type InterviewWithFeedback, type Competency } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Separator } from '@/components/ui/separator'
import { AlertTriangle } from 'lucide-react'

const DECISIONS = ['strong_hire', 'hire', 'no_hire', 'strong_no_hire'] as const

export default function DebriefView() {
  const { id } = useParams<{ id: string }>()
  const loopId = parseInt(id!)
  const queryClient = useQueryClient()

  const { data: loop } = useQuery({ queryKey: ['loops', loopId], queryFn: () => loopsApi.get(loopId) })
  const { data: comps = [] } = useQuery({ queryKey: ['competencies'], queryFn: () => compApi.list() })

  const [decision, setDecision] = useState('')
  const [notes, setNotes] = useState('')

  useEffect(() => {
    if (loop) {
      setDecision(loop.final_decision || '')
      setNotes(loop.debrief_notes || '')
    }
  }, [loop])

  const [error, setError] = useState('')
  const updateLoop = useMutation({
    mutationFn: () => loopsApi.update(loopId, {
      status: 'complete',
      final_decision: decision,
      debrief_notes: notes,
    }),
    onSuccess: () => { setError(''); queryClient.invalidateQueries({ queryKey: ['loops', loopId] }) },
    onError: (err: Error) => setError(err.message),
  })

  if (!loop) return <div>Loading...</div>

  const allComplete = loop.interviews?.every((iv: InterviewWithFeedback) => iv.feedback != null)
  const pendingCount = loop.interviews?.filter((iv: InterviewWithFeedback) => !iv.feedback).length ?? 0

  const compMap = Object.fromEntries(comps.map((c: Competency) => [c.id, c]))

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Debrief — {loop.candidate.name}</h1>

      {!allComplete && (
        <Alert>
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>{pendingCount} interview(s) still awaiting feedback.</AlertDescription>
        </Alert>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {loop.interviews?.map((iv: InterviewWithFeedback) => (
          <Card key={iv.id}>
            <CardHeader>
              <CardTitle className="text-base">{iv.focus_area} — {iv.interviewer_name}</CardTitle>
            </CardHeader>
            <CardContent>
              {iv.feedback ? (
                <div className="space-y-3">
                  <div>
                    <span className="text-sm font-medium">Recommendation: </span>
                    <span className="font-bold">{iv.feedback.recommendation.replace(/_/g, ' ')}</span>
                  </div>
                  {iv.feedback.recommendation_reason && (
                    <div>
                      <span className="text-sm font-medium">Reason: </span>
                      <span className="text-sm">{iv.feedback.recommendation_reason}</span>
                    </div>
                  )}
                  {iv.feedback.competency_ratings?.map(cr => {
                    const comp = compMap[cr.competency_id]
                    return (
                      <div key={cr.id} className="flex justify-between text-sm">
                        <span className="text-gray-600">{comp?.name || `Competency ${cr.competency_id}`}</span>
                        <span className="font-medium">{cr.rating_value}</span>
                      </div>
                    )
                  })}
                  <Separator />
                  {iv.feedback.free_form_notes && (
                    <p className="text-sm text-gray-700 whitespace-pre-wrap">{iv.feedback.free_form_notes}</p>
                  )}
                </div>
              ) : (
                <p className="text-gray-400 text-sm">Awaiting feedback</p>
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader><CardTitle>Final Decision</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label>Decision</Label>
            <Select value={decision} onValueChange={setDecision}>
              <SelectTrigger><SelectValue placeholder="Select decision" /></SelectTrigger>
              <SelectContent>
                {DECISIONS.map(d => (
                  <SelectItem key={d} value={d}>{d.replace(/_/g, ' ')}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label>Debrief Notes</Label>
            <Textarea
              value={notes}
              onChange={e => setNotes(e.target.value)}
              rows={4}
              placeholder="Summary of debrief discussion..."
            />
          </div>
          {error && <p className="text-sm text-red-600">{error}</p>}
          <Button onClick={() => updateLoop.mutate()} disabled={updateLoop.isPending}>{updateLoop.isPending ? 'Saving...' : 'Save Decision'}</Button>
        </CardContent>
      </Card>
    </div>
  )
}
