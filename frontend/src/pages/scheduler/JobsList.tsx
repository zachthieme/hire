import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { jobs as jobsApi, type Job } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Plus } from 'lucide-react'

const statusColor: Record<string, string> = {
  open: 'bg-secondary text-primary',
  closed: 'bg-muted text-muted-foreground',
  filled: 'bg-green-100 text-green-800',
}

export default function JobsList() {
  const queryClient = useQueryClient()
  const { data: jobs = [] } = useQuery({ queryKey: ['jobs'], queryFn: () => jobsApi.list() })
  const [open, setOpen] = useState(false)
  const [error, setError] = useState('')
  const [form, setForm] = useState({ title: '', description: '', hiring_manager: '' })
  const reset = () => setForm({ title: '', description: '', hiring_manager: '' })

  const create = useMutation({
    mutationFn: (data: Partial<Job>) => jobsApi.create(data),
    onSuccess: () => { setError(''); queryClient.invalidateQueries({ queryKey: ['jobs'] }); setOpen(false); reset() },
    onError: (e: Error) => setError(e.message),
  })

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Jobs</h1>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button><Plus className="h-4 w-4 mr-2" />New Job</Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader><DialogTitle>New Job</DialogTitle></DialogHeader>
            <form onSubmit={e => { e.preventDefault(); create.mutate({ ...form, status: 'open' }) }} className="space-y-4">
              <div className="space-y-2"><Label>Title</Label>
                <Input value={form.title} onChange={e => setForm({ ...form, title: e.target.value })} required /></div>
              <div className="space-y-2"><Label>Description</Label>
                <Textarea value={form.description} onChange={e => setForm({ ...form, description: e.target.value })} /></div>
              <div className="space-y-2"><Label>Hiring Manager</Label>
                <Input value={form.hiring_manager} onChange={e => setForm({ ...form, hiring_manager: e.target.value })} /></div>
              {error && <p className="text-sm text-red-600">{error}</p>}
              <Button type="submit" className="w-full" disabled={create.isPending}>{create.isPending ? 'Creating…' : 'Create'}</Button>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      <Table>
        <TableHeader><TableRow>
          <TableHead>Title</TableHead><TableHead>Hiring Manager</TableHead><TableHead>Status</TableHead>
        </TableRow></TableHeader>
        <TableBody>
          {jobs.map((j: Job) => (
            <TableRow key={j.id}>
              <TableCell><Link to={`/jobs/${j.id}`} className="font-medium text-primary hover:underline">{j.title}</Link></TableCell>
              <TableCell>{j.hiring_manager || '—'}</TableCell>
              <TableCell><span className={`px-2 py-1 rounded text-xs font-medium ${statusColor[j.status] || ''}`}>{j.status}</span></TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
