import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuth } from '@/lib/auth'
import Layout from '@/components/Layout'
import LoginPage from '@/pages/LoginPage'
import Dashboard from '@/pages/Dashboard'

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
        <Route path="/candidates" element={<div className="text-gray-500">Candidates (coming soon)</div>} />
        <Route path="/candidates/:id" element={<div className="text-gray-500">Candidate Detail (coming soon)</div>} />
        <Route path="/loops/:id/edit" element={<div className="text-gray-500">Loop Editor (coming soon)</div>} />
        <Route path="/loops/:id/debrief" element={<div className="text-gray-500">Debrief (coming soon)</div>} />
        <Route path="/admin/users" element={<div className="text-gray-500">User Management (coming soon)</div>} />
        <Route path="/admin/competencies" element={<div className="text-gray-500">Competency Management (coming soon)</div>} />
      </Route>
    </Routes>
  )
}
