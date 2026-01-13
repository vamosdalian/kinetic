import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'

if (process.env.NODE_ENV === 'development') {
  // const { worker } = await import('./mocks/worker.ts');
  // console.log('Starting mock service worker...');
  // worker.start();
  console.log('Start development mode');
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
