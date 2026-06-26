import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { interviews as ivApi, type Interview } from '@/lib/api'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'

export default function MyInterviews() {
  const { data: myInterviews = [] } = useQuery({ queryKey: ['my-interviews'], queryFn: ivApi.listMine })

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">My Interviews</h1>

      {myInterviews.length === 0 && <p className="text-gray-500">No interviews assigned yet.</p>}

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Focus Area</TableHead>
            <TableHead>Scheduled</TableHead>
            <TableHead>Status</TableHead>
            <TableHead></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {myInterviews.map((iv: Interview) => (
            <TableRow key={iv.id}>
              <TableCell className="font-medium">{iv.focus_area}</TableCell>
              <TableCell>{new Date(iv.scheduled_at).toLocaleString()}</TableCell>
              <TableCell>
                <Badge variant={iv.status === 'complete' ? 'default' : 'outline'}>
                  {iv.status === 'complete' ? 'Feedback Submitted' : 'Pending'}
                </Badge>
              </TableCell>
              <TableCell>
                <Link to={`/interviews/${iv.id}`} className="text-blue-600 hover:underline text-sm">
                  {iv.status === 'complete' ? 'View' : 'Submit Feedback'}
                </Link>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
