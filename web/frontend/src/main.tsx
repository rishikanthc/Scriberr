import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import './App.css'
import App from './App.tsx'
import { ThemeProvider } from './contexts/ThemeContext'
import { RouterProvider } from './contexts/RouterContext'
import { AuthProvider } from './contexts/AuthContext'
import { ProtectedRoute } from './components/ProtectedRoute'
import { TooltipProvider } from '@/components/ui/tooltip'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ThemeProvider>
      <AuthProvider>
        <RouterProvider>
          <TooltipProvider>
            <ProtectedRoute>
              <App />
            </ProtectedRoute>
          </TooltipProvider>
        </RouterProvider>
      </AuthProvider>
    </ThemeProvider>
  </StrictMode>,
)
