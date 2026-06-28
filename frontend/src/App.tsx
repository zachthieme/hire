import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuth } from '@/lib/auth'
import Layout from '@/components/Layout'
import LoginPage from '@/pages/LoginPage'
import Dashboard from '@/pages/Dashboard'
import UserManagement from '@/pages/admin/UserManagement'
import CompetencyManagement from '@/pages/admin/CompetencyManagement'
import CandidatesList from '@/pages/scheduler/CandidatesList'
import JobsList from '@/pages/scheduler/JobsList'
import JobDetail from '@/pages/scheduler/JobDetail'
import ApplicationDetail from '@/pages/scheduler/ApplicationDetail'
import MyInterviews from '@/pages/interviewer/MyInterviews'
import InterviewDetail from '@/pages/interviewer/InterviewDetail'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuth()
  if (!isAuthenticated) return <Navigate to="/login" replace />
  return <>{children}</>
}

function RoleRoute({ roles, children }: { roles: string[]; children: React.ReactNode }) {
  const { user } = useAuth()
  if (!user || !roles.includes(user.role)) return <Navigate to="/" replace />
  return <>{children}</>
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        element={
          <ProtectedRoute>
            <Layout />
          </ProtectedRoute>
        }
      >
        <Route path="/" element={<Dashboard />} />
        <Route path="/my-interviews" element={<RoleRoute roles={['interviewer']}><MyInterviews /></RoleRoute>} />
        <Route path="/interviews/:id" element={<RoleRoute roles={['interviewer']}><InterviewDetail /></RoleRoute>} />
        <Route path="/jobs" element={<RoleRoute roles={['scheduler', 'admin']}><JobsList /></RoleRoute>} />
        <Route path="/jobs/:id" element={<RoleRoute roles={['scheduler', 'admin']}><JobDetail /></RoleRoute>} />
        <Route path="/applications/:id" element={<RoleRoute roles={['scheduler', 'admin']}><ApplicationDetail /></RoleRoute>} />
        <Route path="/candidates" element={<RoleRoute roles={['scheduler', 'admin']}><CandidatesList /></RoleRoute>} />
        <Route path="/admin/users" element={<RoleRoute roles={['admin']}><UserManagement /></RoleRoute>} />
        <Route path="/admin/competencies" element={<RoleRoute roles={['admin']}><CompetencyManagement /></RoleRoute>} />
      </Route>
    </Routes>
  )
}
