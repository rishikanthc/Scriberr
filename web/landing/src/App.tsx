import Navbar from './components/Navbar';
import Hero from './components/Hero';
import Features from './components/Features';
import Alternating from './components/Alternating';
import Footer from './components/Footer';

export default function App() {
  return (
    <div className="min-h-screen flex flex-col">
      <Navbar />
      <main className="flex-1">
        <Hero />
        <Features />
        <Alternating />
      </main>
      <Footer />
    </div>
  );
}
