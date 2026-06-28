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

  const linkClass = 'text-white/80 hover:text-white transition-colors'

  const navLinks = () => {
    switch (user?.role) {
      case 'admin':
        return (
          <>
            <Link to="/admin/users" className={linkClass}>Users</Link>
            <Link to="/admin/competencies" className={linkClass}>Competencies</Link>
          </>
        )
      case 'scheduler':
        return (
          <>
            <Link to="/jobs" className={linkClass}>Jobs</Link>
            <Link to="/candidates" className={linkClass}>Candidates</Link>
          </>
        )
      case 'interviewer':
        return (
          <>
            <Link to="/my-interviews" className={linkClass}>My Interviews</Link>
          </>
        )
      default:
        return null
    }
  }

  return (
    <div className="min-h-screen bg-background">
      <nav className="bg-house-green text-white px-6 py-3 flex items-center justify-between shadow-sm">
        <div className="flex items-center gap-6">
          <Link to="/" className="flex items-center gap-2 text-xl font-bold tracking-tight">
            <span className="flex size-7 items-center justify-center rounded-full bg-primary text-primary-foreground text-sm">H</span>
            Hire
          </Link>
          <div className="flex items-center gap-5 text-sm">{navLinks()}</div>
        </div>
        <div className="flex items-center gap-4">
          <NotificationBell />
          <span className="text-sm text-white/70">{user?.name} ({user?.role})</span>
          <Button variant="ghost" size="sm" onClick={handleLogout} className="text-white hover:bg-white/10 hover:text-white">Logout</Button>
        </div>
      </nav>
      <main className="max-w-6xl mx-auto p-6">
        <Outlet />
      </main>
    </div>
  )
}
