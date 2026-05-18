import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import RSVPPage from './pages/RSVPPage'
import CreatePage from './pages/CreatePage'
import './App.css'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/events/:slug" element={<RSVPPage />} />
        <Route path="/create" element={<CreatePage />} />
        <Route path="*" element={<Navigate to="/create" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
