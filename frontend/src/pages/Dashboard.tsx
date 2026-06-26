import { Navigate } from 'react-router-dom'
import { useAuth } from '@/lib/auth'

export default function Dashboard() {
  const { user } = useAuth()

  switch (user?.role) {
    case 'admin':
      return <Navigate to="/admin/users" replace />
    case 'scheduler':
      return <Navigate to="/candidates" replace />
    case 'interviewer':
    default:
      return <Navigate to="/my-interviews" replace />
  }
}
