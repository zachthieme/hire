import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { users as usersApi, type User, type CreateUserReq } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Trash2, Plus } from 'lucide-react'

export default function UserManagement() {
  const queryClient = useQueryClient()
  const { data: userList = [] } = useQuery({ queryKey: ['users'], queryFn: () => usersApi.list() })
  const [error, setError] = useState('')
  const createUser = useMutation({
    mutationFn: (data: CreateUserReq) => usersApi.create(data),
    onSuccess: () => { setError(''); queryClient.invalidateQueries({ queryKey: ['users'] }); setOpen(false); resetForm() },
    onError: (err: Error) => setError(err.message),
  })
  const deleteUser = useMutation({
    mutationFn: usersApi.delete,
    onSuccess: () => { setError(''); queryClient.invalidateQueries({ queryKey: ['users'] }) },
    onError: (err: Error) => setError(err.message),
  })

  const [open, setOpen] = useState(false)
  const [form, setForm] = useState<CreateUserReq>({ email: '', name: '', password: '', role: 'interviewer' })
  const resetForm = () => setForm({ email: '', name: '', password: '', role: 'interviewer' })

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">User Management</h1>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button><Plus className="h-4 w-4 mr-2" />Add User</Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader><DialogTitle>Create User</DialogTitle></DialogHeader>
            <form onSubmit={e => { e.preventDefault(); createUser.mutate(form) }} className="space-y-4">
              <div className="space-y-2">
                <Label>Name</Label>
                <Input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} required />
              </div>
              <div className="space-y-2">
                <Label>Email</Label>
                <Input type="email" value={form.email} onChange={e => setForm({ ...form, email: e.target.value })} required />
              </div>
              <div className="space-y-2">
                <Label>Password</Label>
                <Input type="password" value={form.password} onChange={e => setForm({ ...form, password: e.target.value })} required />
              </div>
              <div className="space-y-2">
                <Label>Role</Label>
                <Select value={form.role} onValueChange={v => setForm({ ...form, role: v })}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="admin">Admin</SelectItem>
                    <SelectItem value="scheduler">Scheduler</SelectItem>
                    <SelectItem value="interviewer">Interviewer</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              {error && <p className="text-sm text-red-600">{error}</p>}
              <Button type="submit" className="w-full" disabled={createUser.isPending}>{createUser.isPending ? 'Creating...' : 'Create'}</Button>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Email</TableHead>
            <TableHead>Role</TableHead>
            <TableHead className="w-16"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {userList.map((u: User) => (
            <TableRow key={u.id}>
              <TableCell>{u.name}</TableCell>
              <TableCell>{u.email}</TableCell>
              <TableCell><RoleBadge role={u.role} /></TableCell>
              <TableCell>
                <Button variant="ghost" size="sm" onClick={() => deleteUser.mutate(u.id)}>
                  <Trash2 className="h-4 w-4 text-red-500" />
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}

function RoleBadge({ role }: { role: string }) {
  const colors: Record<string, string> = {
    admin: 'bg-purple-100 text-purple-800',
    scheduler: 'bg-blue-100 text-blue-800',
    interviewer: 'bg-green-100 text-green-800',
  }
  return <span className={`px-2 py-1 rounded text-xs font-medium ${colors[role] || ''}`}>{role}</span>
}
