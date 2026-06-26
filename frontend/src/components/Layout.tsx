import { Outlet, Link, useNavigate } from 'react-router-dom'
import { useAuth } from '@/lib/auth'
import { Button } from '@/components/ui/button'
import NotificationBell from './NotificationBell'

export default function Layout() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  const navLinks = () => {
    switch (user?.role) {
      case 'admin':
        return (
          <>
            <Link to="/admin/users" className="hover:underline">Users</Link>
            <Link to="/admin/competencies" className="hover:underline">Competencies</Link>
          </>
        )
      case 'scheduler':
        return (
          <>
            <Link to="/candidates" className="hover:underline">Candidates</Link>
          </>
        )
      case 'interviewer':
        return (
          <>
            <Link to="/" className="hover:underline">My Interviews</Link>
          </>
        )
      default:
        return null
    }
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white border-b px-6 py-3 flex items-center justify-between">
        <div className="flex items-center gap-6">
          <Link to="/" className="text-xl font-bold">Hire</Link>
          <div className="flex items-center gap-4 text-sm">{navLinks()}</div>
        </div>
        <div className="flex items-center gap-4">
          <NotificationBell />
          <span className="text-sm text-gray-600">{user?.name} ({user?.role})</span>
          <Button variant="ghost" size="sm" onClick={handleLogout}>Logout</Button>
        </div>
      </nav>
      <main className="max-w-6xl mx-auto p-6">
        <Outlet />
      </main>
    </div>
  )
}
