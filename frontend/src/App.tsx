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

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuth()
  if (!isAuthenticated) return <Navigate to="/login" replace />
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
        <Route path="/my-interviews" element={<div className="text-gray-500">My Interviews (coming soon)</div>} />
        <Route path="/interviews/:id" element={<div className="text-gray-500">Interview Detail (coming soon)</div>} />
        <Route path="/candidates" element={<CandidatesList />} />
        <Route path="/candidates/:id" element={<CandidateDetail />} />
        <Route path="/loops/:id/edit" element={<LoopEditor />} />
        <Route path="/loops/:id/debrief" element={<DebriefView />} />
        <Route path="/admin/users" element={<UserManagement />} />
        <Route path="/admin/competencies" element={<CompetencyManagement />} />
      </Route>
    </Routes>
  )
}
