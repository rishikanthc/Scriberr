import { lazy, Suspense } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'

// Lazy load route components for better performance
const HomePage = lazy(() => import("@/features/home/components/HomePage").then(module => ({ default: module.HomePage })));
const AudioDetailView = lazy(() => import("@/features/transcription/components/AudioDetailView").then(module => ({ default: module.AudioDetailView })));
const Settings = lazy(() => import('@/features/settings/pages/SettingsPage').then(module => ({ default: module.Settings })))
const CLISettings = lazy(() => import('@/features/settings/pages/CLISettingsPage').then(module => ({ default: module.CLISettings })))
const CLIAuthConfirmation = lazy(() => import('./features/auth/components/CLIAuthConfirmation').then(module => ({ default: module.CLIAuthConfirmation })))


// Loading component
const PageLoader = () => (
  <div className="scr-app flex items-center justify-center">
    <div className="h-8 w-8 animate-spin rounded-full border-2 border-[var(--scr-brand-muted)] border-b-[var(--scr-brand-solid)]"></div>
  </div>
)

function App() {
  return (
    <Suspense fallback={<PageLoader />}>
      <Routes>
        <Route path="/" element={<HomePage />} />
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
