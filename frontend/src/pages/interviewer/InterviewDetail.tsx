import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { myStages as myStagesApi, stages as stagesApi } from '@/lib/api'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import FeedbackForm from './FeedbackForm'

export default function InterviewDetail() {
  const { id } = useParams()
  const stageId = Number(id)
  const { data: myStages = [] } = useQuery({ queryKey: ['my-stages'], queryFn: () => myStagesApi.list() })
  const { data: existing = [] } = useQuery({ queryKey: ['stage-feedback', stageId], queryFn: () => stagesApi.feedback(stageId) })
  const stage = myStages.find(s => s.id === stageId)
  if (!stage) return <div>Loading…</div>

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader><CardTitle className="flex items-center justify-between">
          <span>{stage.type === 'phone_screen' ? 'Phone Screen' : 'Interview'}{stage.focus_area && ` — ${stage.focus_area}`}</span>
          <span className="text-xs font-normal text-muted-foreground">{stage.status}</span>
        </CardTitle></CardHeader>
        <CardContent className="space-y-1 text-sm">
          <p><span className="font-semibold">Candidate:</span> {stage.candidate_name}</p>
          <p><span className="font-semibold">Job:</span> {stage.job_title}</p>
          <p><span className="font-semibold">Scheduled:</span> {new Date(stage.scheduled_at).toLocaleString()}</p>
          {stage.video_link && <p><span className="font-semibold">Video:</span> <a href={stage.video_link} target="_blank" rel="noopener noreferrer" className="text-primary">Join</a></p>}
        </CardContent>
      </Card>
      <FeedbackForm stageId={stageId} alreadySubmitted={stage.has_my_feedback} existingCount={existing.length} />
    </div>
  )
}
