import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { loops as loopsApi, competencies as compApi, type Interview, type InterviewWithFeedback, type Competency } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { ExternalLink } from 'lucide-react'
import { interviews as ivApi } from '@/lib/api'
import FeedbackForm from './FeedbackForm'

export default function InterviewDetail() {
  const { id } = useParams<{ id: string }>()
  const interviewId = parseInt(id!)
  const [feedbackSubmitted, setFeedbackSubmitted] = useState(false)

  const { data: myInterviews = [] } = useQuery({ queryKey: ['my-interviews'], queryFn: ivApi.listMine })
  const interview = myInterviews.find((iv: Interview) => iv.id === interviewId)

  const { data: loop, refetch: refetchLoop } = useQuery({
    queryKey: ['loops', interview?.loop_id],
    queryFn: () => loopsApi.get(interview!.loop_id),
    enabled: !!interview,
  })

  const { data: competenciesList = [] } = useQuery({ queryKey: ['competencies'], queryFn: compApi.list })

  if (!interview) return <div>Loading...</div>

  const myInterview = loop?.interviews?.find((iv: InterviewWithFeedback) => iv.id === interviewId)
  const hasFeedback = myInterview?.feedback != null || feedbackSubmitted

  const compMap = Object.fromEntries(competenciesList.map((c: Competency) => [c.id, c]))

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span>{interview.focus_area} Interview</span>
            <Badge variant={hasFeedback ? 'default' : 'outline'}>
              {hasFeedback ? 'Feedback Submitted' : 'Pending'}
            </Badge>
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          {loop && <p><strong>Candidate:</strong> {loop.candidate.name} ({loop.candidate.email})</p>}
          <p><strong>Scheduled:</strong> {new Date(interview.scheduled_at).toLocaleString()}</p>
          {interview.video_link && (
            <p>
              <strong>Video: </strong>
              <a href={interview.video_link} target="_blank" rel="noopener noreferrer" className="text-blue-600 inline-flex items-center gap-1">
                Join <ExternalLink className="h-3 w-3" />
              </a>
            </p>
          )}
          {interview.notes_for_interviewer && (
            <div>
              <strong>Notes from scheduler:</strong>
              <p className="text-sm text-gray-600 mt-1">{interview.notes_for_interviewer}</p>
            </div>
          )}
          {loop?.candidate.resume_url && (
            <p>
              <strong>Resume: </strong>
              <a href={loop.candidate.resume_url} target="_blank" rel="noopener noreferrer" className="text-blue-600 inline-flex items-center gap-1">
                View <ExternalLink className="h-3 w-3" />
              </a>
            </p>
          )}
        </CardContent>
      </Card>

      {!hasFeedback ? (
        <FeedbackForm
          interviewId={interviewId}
          competencies={competenciesList}
          onSubmitted={() => { setFeedbackSubmitted(true); refetchLoop() }}
        />
      ) : (
        <>
          {myInterview?.feedback && (
            <Card>
              <CardHeader><CardTitle>Your Feedback</CardTitle></CardHeader>
              <CardContent className="space-y-2">
                <p><strong>Recommendation:</strong> {myInterview.feedback.recommendation.replace(/_/g, ' ')}</p>
                {myInterview.feedback.recommendation_reason && <p><strong>Reason:</strong> {myInterview.feedback.recommendation_reason}</p>}
                {myInterview.feedback.competency_ratings?.map(cr => (
                  <div key={cr.id} className="flex justify-between text-sm">
                    <span>{compMap[cr.competency_id]?.name || `Competency ${cr.competency_id}`}</span>
                    <span className="font-medium">{cr.rating_value}</span>
                  </div>
                ))}
                {myInterview.feedback.free_form_notes && (
                  <>
                    <Separator />
                    <p className="text-sm whitespace-pre-wrap">{myInterview.feedback.free_form_notes}</p>
                  </>
                )}
              </CardContent>
            </Card>
          )}

          {loop?.interviews?.filter((iv: InterviewWithFeedback) => iv.id !== interviewId && iv.feedback).map((iv: InterviewWithFeedback) => (
            <Card key={iv.id}>
              <CardHeader><CardTitle className="text-base">{iv.focus_area} — {iv.interviewer_name}</CardTitle></CardHeader>
              <CardContent className="space-y-2">
                <p><strong>Recommendation:</strong> {iv.feedback!.recommendation.replace(/_/g, ' ')}</p>
                {iv.feedback!.recommendation_reason && <p className="text-sm">{iv.feedback!.recommendation_reason}</p>}
                {iv.feedback!.competency_ratings?.map(cr => (
                  <div key={cr.id} className="flex justify-between text-sm">
                    <span>{compMap[cr.competency_id]?.name || `Competency ${cr.competency_id}`}</span>
                    <span className="font-medium">{cr.rating_value}</span>
                  </div>
                ))}
                {iv.feedback!.free_form_notes && (
                  <>
                    <Separator />
                    <p className="text-sm whitespace-pre-wrap">{iv.feedback!.free_form_notes}</p>
                  </>
                )}
              </CardContent>
            </Card>
          ))}
        </>
      )}
    </div>
  )
}
