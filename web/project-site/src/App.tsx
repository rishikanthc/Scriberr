import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { Layout } from './Layout';
import { DocsLayout } from './layouts/DocsLayout';
import { Home } from './pages/Home';
import Introduction from './docs/Introduction.mdx';
import Features from './docs/Features.mdx';
import Usage from './docs/Usage.mdx';
import Diarization from './docs/Diarization.mdx';
import Installation from './docs/Installation.mdx';
import Troubleshooting from './docs/Troubleshooting.mdx';
import ApiPage from './pages/ApiPage';

function App() {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<Layout><Home /></Layout>} />
        <Route path="/api" element={<ApiPage />} />
        <Route path="/docs/*" element={
          <DocsLayout>
            <Routes>
              <Route path="intro" element={<Introduction />} />
              <Route path="features" element={<Features />} />
              <Route path="usage" element={<Usage />} />
              <Route path="diarization" element={<Diarization />} />
              <Route path="installation" element={<Installation />} />
              <Route path="troubleshooting" element={<Troubleshooting />} />
              <Route path="*" element={<Introduction />} />
            </Routes>
          </DocsLayout>
        } />
      </Routes>
    </Router>
  )
}

export default App
