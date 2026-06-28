import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { myStages as myStagesApi, type MyStage } from '@/lib/api'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'

export default function MyInterviews() {
  const { data: stages = [] } = useQuery({ queryKey: ['my-stages'], queryFn: () => myStagesApi.list() })
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">My Interviews</h1>
      <Table>
        <TableHeader><TableRow>
          <TableHead>Type</TableHead><TableHead>Candidate</TableHead><TableHead>Job</TableHead>
          <TableHead>Scheduled</TableHead><TableHead>Status</TableHead><TableHead></TableHead>
        </TableRow></TableHeader>
        <TableBody>
          {stages.map((s: MyStage) => (
            <TableRow key={s.id}>
              <TableCell>{s.type === 'phone_screen' ? 'Phone Screen' : 'Interview'}{s.focus_area && ` — ${s.focus_area}`}</TableCell>
              <TableCell>{s.candidate_name}</TableCell>
              <TableCell>{s.job_title}</TableCell>
              <TableCell>{new Date(s.scheduled_at).toLocaleString()}</TableCell>
              <TableCell>
                {s.has_my_feedback
                  ? <Badge>Feedback Submitted</Badge>
                  : <Badge variant="outline">Pending</Badge>}
              </TableCell>
              <TableCell>
                <Link to={`/interviews/${s.id}`} className="text-primary hover:underline text-sm">
                  {s.has_my_feedback ? 'View' : 'Submit Feedback'}
                </Link>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
