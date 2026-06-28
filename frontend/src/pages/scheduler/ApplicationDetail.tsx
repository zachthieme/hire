import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { applications as appsApi, stages as stagesApi, users as usersApi, type Stage } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'

const DECISIONS = ['strong_hire', 'hire', 'no_hire', 'strong_no_hire'] as const

export default function ApplicationDetail() {
  const { id } = useParams()
  const appId = Number(id)
  const queryClient = useQueryClient()
  const invalidate = () => queryClient.invalidateQueries({ queryKey: ['application', appId] })
  const { data: app } = useQuery({ queryKey: ['application', appId], queryFn: () => appsApi.get(appId) })
  const { data: interviewers = [] } = useQuery({ queryKey: ['users'], queryFn: () => usersApi.list() })

  const addStage = useMutation({
    mutationFn: (data: Partial<Stage>) => stagesApi.create(appId, data),
    onSuccess: invalidate,
  })
  const assign = useMutation({
    mutationFn: ({ stageId, interviewerId }: { stageId: number; interviewerId: number }) =>
      stagesApi.addInterviewer(stageId, interviewerId),
    onSuccess: invalidate,
  })
  const saveDecision = useMutation({
    mutationFn: (data: { status: string; final_decision: string | null; final_interview_notes: string | null }) =>
      appsApi.update(appId, data as never),
    onSuccess: invalidate,
  })

  const [notes, setNotes] = useState<string | null>(null)
  const [decision, setDecision] = useState<string>('')

  if (!app) return <div>Loading…</div>
  const onlyInterviewers = interviewers.filter(u => u.role === 'interviewer')

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{app.candidate.name}</h1>
        <p className="text-muted-foreground">{app.job.title}</p>
      </div>

      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold">Stages</h2>
        <AddStageButton onAdd={(data) => addStage.mutate(data)} />
      </div>

      {app.stages.map(st => (
        <Card key={st.id}>
          <CardHeader>
            <CardTitle className="flex items-center justify-between">
              <span>{st.type === 'phone_screen' ? 'Phone Screen' : 'Interview'}{st.focus_area && ` — ${st.focus_area}`}</span>
              <span className="text-xs font-normal text-muted-foreground">{st.status}</span>
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {st.participants.map(p => (
              <div key={p.interviewer_id} className="border-b pb-2 last:border-b-0">
                <div className="flex items-center justify-between">
                  <span className="font-medium">{p.interviewer_name}</span>
                  {p.feedback
                    ? <span className="text-sm font-semibold text-primary">{p.feedback.recommendation}</span>
                    : <span className="text-sm text-muted-foreground">awaiting feedback</span>}
                </div>
                {p.feedback?.recommendation_reason && <p className="text-sm text-muted-foreground">{p.feedback.recommendation_reason}</p>}
              </div>
            ))}
            <div className="flex items-center gap-2 pt-2">
              <Select onValueChange={(v) => assign.mutate({ stageId: st.id, interviewerId: Number(v) })}>
                <SelectTrigger className="w-56"><SelectValue placeholder="Assign interviewer" /></SelectTrigger>
                <SelectContent>
                  {onlyInterviewers.map(u => <SelectItem key={u.id} value={String(u.id)}>{u.name}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>
      ))}

      <Card>
        <CardHeader><CardTitle>Final Decision</CardTitle></CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label>Decision</Label>
            <Select value={decision || app.final_decision || ''} onValueChange={setDecision}>
              <SelectTrigger className="w-56"><SelectValue placeholder="Select decision" /></SelectTrigger>
              <SelectContent>
                {DECISIONS.map(d => <SelectItem key={d} value={d}>{d}</SelectItem>)}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label>Final Interview Notes</Label>
            <Textarea value={notes ?? app.final_interview_notes ?? ''} onChange={e => setNotes(e.target.value)} placeholder="Summary of the debrief…" />
          </div>
          <Button onClick={() => saveDecision.mutate({
            status: app.status,
            final_decision: (decision || app.final_decision) || null,
            final_interview_notes: (notes ?? app.final_interview_notes) || null,
          })} disabled={saveDecision.isPending}>Save Decision</Button>
        </CardContent>
      </Card>
    </div>
  )
}

function AddStageButton({ onAdd }: { onAdd: (data: Partial<Stage>) => void }) {
  const [open, setOpen] = useState(false)
  const [form, setForm] = useState({ type: 'interview', focus_area: '', scheduled_at: '', video_link: '' })
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild><Button>Add Stage</Button></DialogTrigger>
      <DialogContent>
        <DialogHeader><DialogTitle>Add Stage</DialogTitle></DialogHeader>
        <form onSubmit={(e: React.FormEvent) => {
          e.preventDefault()
          onAdd({
            type: form.type as Stage['type'],
            focus_area: form.focus_area,
            scheduled_at: form.scheduled_at ? new Date(form.scheduled_at).toISOString() : new Date().toISOString(),
            video_link: form.video_link,
            status: 'pending',
          })
          setOpen(false)
        }} className="space-y-4">
          <div className="space-y-2"><Label>Type</Label>
            <Select value={form.type} onValueChange={v => setForm({ ...form, type: v })}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="phone_screen">Phone Screen</SelectItem>
                <SelectItem value="interview">Interview</SelectItem>
              </SelectContent>
            </Select></div>
          <div className="space-y-2"><Label>Focus Area</Label>
            <Input value={form.focus_area} onChange={e => setForm({ ...form, focus_area: e.target.value })} /></div>
          <div className="space-y-2"><Label>Scheduled At</Label>
            <Input type="datetime-local" value={form.scheduled_at} onChange={e => setForm({ ...form, scheduled_at: e.target.value })} /></div>
          <div className="space-y-2"><Label>Video Link</Label>
            <Input value={form.video_link} onChange={e => setForm({ ...form, video_link: e.target.value })} /></div>
          <Button type="submit" className="w-full">Add Stage</Button>
        </form>
      </DialogContent>
    </Dialog>
  )
}
