import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { competencies as compApi, type Competency } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Trash2, Plus } from 'lucide-react'

export default function CompetencyManagement() {
  const queryClient = useQueryClient()
  const { data: comps = [] } = useQuery({ queryKey: ['competencies'], queryFn: () => compApi.list() })
  const [error, setError] = useState('')
  const createComp = useMutation({
    mutationFn: (data: Partial<Competency>) => compApi.create(data),
    onSuccess: () => { setError(''); queryClient.invalidateQueries({ queryKey: ['competencies'] }); setOpen(false) },
    onError: (err: Error) => setError(err.message),
  })
  const deleteComp = useMutation({
    mutationFn: compApi.delete,
    onSuccess: () => { setError(''); queryClient.invalidateQueries({ queryKey: ['competencies'] }) },
    onError: (err: Error) => setError(err.message),
  })

  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [ratingType, setRatingType] = useState<'levels' | 'stars'>('levels')
  const [levelsInput, setLevelsInput] = useState('Learning, Owning, Advising')
  const [starsMax, setStarsMax] = useState('5')

  const handleCreate = () => {
    const ratingsJson = ratingType === 'levels'
      ? JSON.stringify(levelsInput.split(',').map(s => s.trim()).filter(Boolean))
      : JSON.stringify({ min: 1, max: parseInt(starsMax) })
    createComp.mutate({ name, rating_type: ratingType, ratings_json: ratingsJson })
  }

  const parseRatings = (c: Competency) => {
    try {
      const parsed = JSON.parse(c.ratings_json)
      if (c.rating_type === 'levels') return (parsed as string[]).join(', ')
      return `1-${parsed.max} stars`
    } catch { return c.ratings_json }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Competency Management</h1>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button><Plus className="h-4 w-4 mr-2" />Add Competency</Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader><DialogTitle>Create Competency</DialogTitle></DialogHeader>
            <div className="space-y-4">
              <div className="space-y-2">
                <Label>Name</Label>
                <Input value={name} onChange={e => setName(e.target.value)} placeholder="e.g. Problem Solving" />
              </div>
              <div className="space-y-2">
                <Label>Rating Type</Label>
                <Select value={ratingType} onValueChange={v => setRatingType(v as 'levels' | 'stars')}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="levels">Levels (custom labels)</SelectItem>
                    <SelectItem value="stars">Stars (numeric scale)</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              {ratingType === 'levels' ? (
                <div className="space-y-2">
                  <Label>Levels (comma-separated)</Label>
                  <Input value={levelsInput} onChange={e => setLevelsInput(e.target.value)} placeholder="Learning, Owning, Advising" />
                </div>
              ) : (
                <div className="space-y-2">
                  <Label>Max Stars</Label>
                  <Input type="number" value={starsMax} onChange={e => setStarsMax(e.target.value)} min="2" max="10" />
                </div>
              )}
              {error && <p className="text-sm text-red-600">{error}</p>}
              <Button onClick={handleCreate} className="w-full" disabled={createComp.isPending}>{createComp.isPending ? 'Creating...' : 'Create'}</Button>
            </div>
          </DialogContent>
        </Dialog>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Type</TableHead>
            <TableHead>Ratings</TableHead>
            <TableHead className="w-16"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {comps.map((c: Competency) => (
            <TableRow key={c.id}>
              <TableCell className="font-medium">{c.name}</TableCell>
              <TableCell>{c.rating_type}</TableCell>
              <TableCell>{parseRatings(c)}</TableCell>
              <TableCell>
                <Button variant="ghost" size="sm" onClick={() => deleteComp.mutate(c.id)}>
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
