import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { loops as loopsApi, interviews as ivApi, users as usersApi, type InterviewWithFeedback } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Plus, Trash2 } from 'lucide-react'

export default function LoopEditor() {
  const { id } = useParams<{ id: string }>()
  const loopId = parseInt(id!)
  const queryClient = useQueryClient()

  const { data: loop } = useQuery({ queryKey: ['loops', loopId], queryFn: () => loopsApi.get(loopId) })
  const { data: userList = [] } = useQuery({ queryKey: ['users'], queryFn: usersApi.list })
  const interviewers = userList.filter(u => u.role === 'interviewer')

  const createInterview = useMutation({
    mutationFn: (data: any) => ivApi.createInLoop(loopId, data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['loops', loopId] }),
  })
  const deleteInterview = useMutation({
    mutationFn: ivApi.delete,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['loops', loopId] }),
  })

  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({
    interviewer_id: 0,
    focus_area: '',
    scheduled_at: '',
    video_link: '',
    notes_for_interviewer: '',
  })

  const handleAdd = () => {
    createInterview.mutate({
      ...form,
      interviewer_id: Number(form.interviewer_id),
      scheduled_at: new Date(form.scheduled_at).toISOString(),
      status: 'pending',
    })
    setForm({ interviewer_id: 0, focus_area: '', scheduled_at: '', video_link: '', notes_for_interviewer: '' })
    setShowForm(false)
  }

  if (!loop) return <div>Loading...</div>

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Edit Loop — {loop.candidate.name}</h1>
          <p className="text-gray-500">{loop.candidate.email}</p>
        </div>
        <Badge variant={loop.status === 'complete' ? 'default' : 'secondary'}>{loop.status}</Badge>
      </div>

      <div className="space-y-3">
        {loop.interviews?.map((iv: InterviewWithFeedback) => (
          <Card key={iv.id}>
            <CardContent className="flex items-center justify-between py-4">
              <div>
                <p className="font-medium">{iv.focus_area}</p>
                <p className="text-sm text-gray-500">
                  {iv.interviewer_name} &middot; {new Date(iv.scheduled_at).toLocaleString()}
                </p>
                {iv.video_link && <p className="text-sm text-blue-600">{iv.video_link}</p>}
                {iv.notes_for_interviewer && <p className="text-sm text-gray-400 mt-1">{iv.notes_for_interviewer}</p>}
              </div>
              <div className="flex items-center gap-2">
                <Badge variant={iv.status === 'complete' ? 'default' : 'outline'}>{iv.status}</Badge>
                <Button variant="ghost" size="sm" onClick={() => deleteInterview.mutate(iv.id)}>
                  <Trash2 className="h-4 w-4 text-red-500" />
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {showForm ? (
        <Card>
          <CardHeader><CardTitle>Add Interview</CardTitle></CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>Interviewer</Label>
                <Select value={String(form.interviewer_id)} onValueChange={v => setForm({ ...form, interviewer_id: parseInt(v) })}>
                  <SelectTrigger><SelectValue placeholder="Select interviewer" /></SelectTrigger>
                  <SelectContent>
                    {interviewers.map(u => (
                      <SelectItem key={u.id} value={String(u.id)}>{u.name}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>Focus Area</Label>
                <Input value={form.focus_area} onChange={e => setForm({ ...form, focus_area: e.target.value })} placeholder="e.g. Coding, System Design" />
              </div>
              <div className="space-y-2">
                <Label>Scheduled At</Label>
                <Input type="datetime-local" value={form.scheduled_at} onChange={e => setForm({ ...form, scheduled_at: e.target.value })} />
              </div>
              <div className="space-y-2">
                <Label>Video Link</Label>
                <Input value={form.video_link} onChange={e => setForm({ ...form, video_link: e.target.value })} placeholder="https://meet.example.com/..." />
              </div>
            </div>
            <div className="space-y-2">
              <Label>Notes for Interviewer</Label>
              <Textarea value={form.notes_for_interviewer} onChange={e => setForm({ ...form, notes_for_interviewer: e.target.value })} />
            </div>
            <div className="flex gap-2">
              <Button onClick={handleAdd}>Add Interview</Button>
              <Button variant="outline" onClick={() => setShowForm(false)}>Cancel</Button>
            </div>
          </CardContent>
        </Card>
      ) : (
        <Button variant="outline" onClick={() => setShowForm(true)}>
          <Plus className="h-4 w-4 mr-2" />Add Interview
        </Button>
      )}
    </div>
  )
}
