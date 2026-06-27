import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { candidates as candApi, type Candidate } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Plus } from 'lucide-react'

export default function CandidatesList() {
  const queryClient = useQueryClient()
  const { data: cands = [] } = useQuery({ queryKey: ['candidates'], queryFn: () => candApi.list() })
  const [error, setError] = useState('')
  const createCand = useMutation({
    mutationFn: (data: Partial<Candidate>) => candApi.create(data),
    onSuccess: () => { setError(''); queryClient.invalidateQueries({ queryKey: ['candidates'] }); setOpen(false); resetForm() },
    onError: (err: Error) => setError(err.message),
  })

  const [open, setOpen] = useState(false)
  const [form, setForm] = useState({ name: '', email: '', resume_url: '' })
  const resetForm = () => setForm({ name: '', email: '', resume_url: '' })

  const statusColor: Record<string, string> = {
    active: 'bg-blue-100 text-blue-800',
    hired: 'bg-green-100 text-green-800',
    rejected: 'bg-red-100 text-red-800',
    withdrawn: 'bg-gray-100 text-gray-800',
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Candidates</h1>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button><Plus className="h-4 w-4 mr-2" />Add Candidate</Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader><DialogTitle>Add Candidate</DialogTitle></DialogHeader>
            <form onSubmit={e => { e.preventDefault(); createCand.mutate({ ...form, status: 'active' }) }} className="space-y-4">
              <div className="space-y-2">
                <Label>Name</Label>
                <Input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} required />
              </div>
              <div className="space-y-2">
                <Label>Email</Label>
                <Input type="email" value={form.email} onChange={e => setForm({ ...form, email: e.target.value })} required />
              </div>
              <div className="space-y-2">
                <Label>Resume URL</Label>
                <Input value={form.resume_url} onChange={e => setForm({ ...form, resume_url: e.target.value })} />
              </div>
              {error && <p className="text-sm text-red-600">{error}</p>}
              <Button type="submit" className="w-full" disabled={createCand.isPending}>{createCand.isPending ? 'Creating...' : 'Create'}</Button>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Email</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Added</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {cands.map((c: Candidate) => (
            <TableRow key={c.id}>
              <TableCell>
                <Link to={`/candidates/${c.id}`} className="font-medium text-blue-600 hover:underline">{c.name}</Link>
              </TableCell>
              <TableCell>{c.email}</TableCell>
              <TableCell>
                <span className={`px-2 py-1 rounded text-xs font-medium ${statusColor[c.status] || ''}`}>{c.status}</span>
              </TableCell>
              <TableCell className="text-sm text-gray-500">{new Date(c.created_at).toLocaleDateString()}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
