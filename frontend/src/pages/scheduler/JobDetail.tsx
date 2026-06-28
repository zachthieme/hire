import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { jobs as jobsApi, applications as appsApi, candidates as candApi, type Candidate } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Plus } from 'lucide-react'

export default function JobDetail() {
  const { id } = useParams()
  const jobId = Number(id)
  const queryClient = useQueryClient()
  const { data: job } = useQuery({ queryKey: ['jobs', jobId], queryFn: () => jobsApi.get(jobId) })
  const { data: allCandidates = [] } = useQuery({ queryKey: ['candidates'], queryFn: () => candApi.list() })
  const [open, setOpen] = useState(false)
  const [selected, setSelected] = useState('')
  const [error, setError] = useState('')

  const addCandidate = useMutation({
    mutationFn: (candidateId: number) => appsApi.create(jobId, candidateId),
    onSuccess: () => { setError(''); queryClient.invalidateQueries({ queryKey: ['jobs', jobId] }); setOpen(false); setSelected('') },
    onError: (e: Error) => setError(e.message),
  })

  if (!job) return <div>Loading…</div>

  const existingCandidateIds = new Set(job.applications.map(a => a.candidate_id))
  const available = allCandidates.filter((c: Candidate) => !existingCandidateIds.has(c.id))

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{job.title}</h1>
        <p className="text-muted-foreground">{job.hiring_manager && `Hiring manager: ${job.hiring_manager}`}</p>
      </div>
      <Card>
        <CardHeader><CardTitle>Description</CardTitle></CardHeader>
        <CardContent><p className="whitespace-pre-wrap text-sm">{job.description || 'No description.'}</p></CardContent>
      </Card>

      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold">Candidates</h2>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild><Button><Plus className="h-4 w-4 mr-2" />Add Candidate</Button></DialogTrigger>
          <DialogContent>
            <DialogHeader><DialogTitle>Add Candidate to Job</DialogTitle></DialogHeader>
            <form onSubmit={e => { e.preventDefault(); if (selected) addCandidate.mutate(Number(selected)) }} className="space-y-4">
              <Select value={selected} onValueChange={setSelected}>
                <SelectTrigger><SelectValue placeholder="Select candidate" /></SelectTrigger>
                <SelectContent>
                  {available.map((c: Candidate) => <SelectItem key={c.id} value={String(c.id)}>{c.name}</SelectItem>)}
                </SelectContent>
              </Select>
              {error && <p className="text-sm text-red-600">{error}</p>}
              <Button type="submit" className="w-full" disabled={!selected || addCandidate.isPending}>Add</Button>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      <Table>
        <TableHeader><TableRow>
          <TableHead>Candidate</TableHead><TableHead>Status</TableHead><TableHead>Decision</TableHead>
        </TableRow></TableHeader>
        <TableBody>
          {job.applications.map(a => (
            <TableRow key={a.id}>
              <TableCell><Link to={`/applications/${a.id}`} className="font-medium text-primary hover:underline">{a.candidate_name}</Link></TableCell>
              <TableCell><span className="px-2 py-1 rounded text-xs font-medium bg-secondary text-primary">{a.status}</span></TableCell>
              <TableCell>{a.final_decision ?? '—'}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
