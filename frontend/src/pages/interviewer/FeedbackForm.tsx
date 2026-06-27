import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { feedback as fbApi, type Competency, type FeedbackCreate } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Star } from 'lucide-react'

const RECOMMENDATIONS = [
  { value: 'strong_hire', label: 'Strong Hire' },
  { value: 'hire', label: 'Hire' },
  { value: 'no_hire', label: 'No Hire' },
  { value: 'strong_no_hire', label: 'Strong No Hire' },
]

interface Props {
  interviewId: number
  competencies: Competency[]
  onSubmitted: () => void
}

export default function FeedbackForm({ interviewId, competencies, onSubmitted }: Props) {
  const queryClient = useQueryClient()
  const [recommendation, setRecommendation] = useState('')
  const [reason, setReason] = useState('')
  const [notes, setNotes] = useState('')
  const [ratings, setRatings] = useState<Record<number, string>>({})
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async () => {
    setSubmitting(true)
    setError('')
    try {
      const data: FeedbackCreate = {
        recommendation,
        recommendation_reason: reason,
        free_form_notes: notes,
        competency_ratings: Object.entries(ratings).map(([compId, value]) => ({
          competency_id: parseInt(compId),
          rating_value: value,
        })),
      }
      await fbApi.create(interviewId, data)
      queryClient.invalidateQueries({ queryKey: ['my-interviews'] })
      onSubmitted()
    } catch (err: any) {
      setError(err.message || 'Failed to submit feedback')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Card>
      <CardHeader><CardTitle>Submit Feedback</CardTitle></CardHeader>
      <CardContent className="space-y-6">
        <div className="space-y-3">
          <Label className="text-base font-semibold">Hiring Recommendation</Label>
          <RadioGroup value={recommendation} onValueChange={setRecommendation}>
            {RECOMMENDATIONS.map(r => (
              <div key={r.value} className="flex items-center space-x-2">
                <RadioGroupItem value={r.value} id={r.value} />
                <Label htmlFor={r.value}>{r.label}</Label>
              </div>
            ))}
          </RadioGroup>
        </div>

        <div className="space-y-2">
          <Label>Reason for Recommendation</Label>
          <Textarea value={reason} onChange={e => setReason(e.target.value)} rows={3} placeholder="Why are you making this recommendation?" />
        </div>

        {competencies.map(comp => {
          const options = JSON.parse(comp.ratings_json)
          return (
            <div key={comp.id} className="space-y-2">
              <Label className="font-semibold">{comp.name}</Label>
              {comp.rating_type === 'levels' ? (
                <Select value={ratings[comp.id] || ''} onValueChange={v => setRatings({ ...ratings, [comp.id]: v })}>
                  <SelectTrigger><SelectValue placeholder="Select level" /></SelectTrigger>
                  <SelectContent>
                    {(options as string[]).map((level: string) => (
                      <SelectItem key={level} value={level}>{level}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              ) : (
                <div className="flex gap-1">
                  {Array.from({ length: options.max }, (_, i) => i + 1).map(n => (
                    <button
                      key={n}
                      type="button"
                      onClick={() => setRatings({ ...ratings, [comp.id]: String(n) })}
                      className="p-1"
                    >
                      <Star
                        className={`h-6 w-6 ${parseInt(ratings[comp.id] || '0') >= n ? 'fill-yellow-400 text-yellow-400' : 'text-gray-300'}`}
                      />
                    </button>
                  ))}
                </div>
              )}
            </div>
          )
        })}

        <div className="space-y-2">
          <Label>Additional Notes</Label>
          <Textarea value={notes} onChange={e => setNotes(e.target.value)} rows={4} placeholder="Any other observations..." />
        </div>

        {error && <p className="text-sm text-red-600">{error}</p>}
        <Button onClick={handleSubmit} disabled={!recommendation || submitting || competencies.some(c => !ratings[c.id])} className="w-full">
          {submitting ? 'Submitting...' : 'Submit Feedback'}
        </Button>
      </CardContent>
    </Card>
  )
}
