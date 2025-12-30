import { lazy, Suspense } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'

// Lazy load route components for better performance
const Dashboard = lazy(() => import("@/features/transcription/components/Dashboard").then(module => ({ default: module.Dashboard })));
const AudioDetailView = lazy(() => import("@/features/transcription/components/AudioDetailView").then(module => ({ default: module.AudioDetailView })));
const Settings = lazy(() => import('@/features/settings/pages/SettingsPage').then(module => ({ default: module.Settings })))
const CLISettings = lazy(() => import('@/features/settings/pages/CLISettingsPage').then(module => ({ default: module.CLISettings })))
const CLIAuthConfirmation = lazy(() => import('./features/auth/components/CLIAuthConfirmation').then(module => ({ default: module.CLIAuthConfirmation })))


// Loading component
const PageLoader = () => (
  <div className="flex items-center justify-center min-h-screen">
    <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
  </div>
)

function App() {
  return (
    <Suspense fallback={<PageLoader />}>
      <Routes>
        <Route path="/" element={<Dashboard />} />
        <Route path="/audio/:audioId" element={<AudioDetailView />} />

        <Route path="/settings" element={<Settings />} />
        <Route path="/settings/cli" element={<CLISettings />} />
        <Route path="/auth/cli/authorize" element={<CLIAuthConfirmation />} />

        {/* Fallback */}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Suspense>
  )
}

export default App
