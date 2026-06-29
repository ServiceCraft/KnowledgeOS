import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { bootstrapPersistedStores } from '@/lib/bootstrapStores'
import './index.css'
import App from './App.tsx'

bootstrapPersistedStores()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
