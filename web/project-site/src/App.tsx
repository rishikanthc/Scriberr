import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { Layout } from './Layout';
import { DocsLayout } from './layouts/DocsLayout';
import { Home } from './pages/Home';
import Introduction from './docs/Introduction.mdx';
import Features from './docs/Features.mdx';
import Usage from './docs/Usage.mdx';
import Configuration from './docs/Configuration.mdx';
import Installation from './docs/Installation.mdx';

function App() {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<Layout><Home /></Layout>} />
        <Route path="/docs/*" element={
          <DocsLayout>
            <Routes>
              <Route path="intro" element={<Introduction />} />
              <Route path="features" element={<Features />} />
              <Route path="usage" element={<Usage />} />
              <Route path="configuration" element={<Configuration />} />
              <Route path="installation" element={<Installation />} />
              <Route path="*" element={<Introduction />} />
            </Routes>
          </DocsLayout>
        } />
      </Routes>
    </Router>
  )
}

export default App
