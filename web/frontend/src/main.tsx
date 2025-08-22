import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import './App.css'
import App from './App.tsx'
import { ThemeProvider } from './contexts/ThemeContext'
import { RouterProvider } from './contexts/RouterContext'
import { TooltipProvider } from '@/components/ui/tooltip'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ThemeProvider>
      <RouterProvider>
        <TooltipProvider>
          <App />
        </TooltipProvider>
      </RouterProvider>
    </ThemeProvider>
  </StrictMode>,
)
