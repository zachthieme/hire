import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuth } from '@/lib/auth'
import Layout from '@/components/Layout'
import LoginPage from '@/pages/LoginPage'
import Dashboard from '@/pages/Dashboard'
import UserManagement from '@/pages/admin/UserManagement'
import CompetencyManagement from '@/pages/admin/CompetencyManagement'
import CandidatesList from '@/pages/scheduler/CandidatesList'
import CandidateDetail from '@/pages/scheduler/CandidateDetail'
import LoopEditor from '@/pages/scheduler/LoopEditor'
import DebriefView from '@/pages/scheduler/DebriefView'
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
        <Route path="/candidates" element={<RoleRoute roles={['scheduler', 'admin']}><CandidatesList /></RoleRoute>} />
        <Route path="/candidates/:id" element={<RoleRoute roles={['scheduler', 'admin']}><CandidateDetail /></RoleRoute>} />
        <Route path="/loops/:id/edit" element={<RoleRoute roles={['scheduler', 'admin']}><LoopEditor /></RoleRoute>} />
        <Route path="/loops/:id/debrief" element={<RoleRoute roles={['scheduler', 'admin']}><DebriefView /></RoleRoute>} />
        <Route path="/admin/users" element={<RoleRoute roles={['admin']}><UserManagement /></RoleRoute>} />
        <Route path="/admin/competencies" element={<RoleRoute roles={['admin']}><CompetencyManagement /></RoleRoute>} />
      </Route>
    </Routes>
  )
}
