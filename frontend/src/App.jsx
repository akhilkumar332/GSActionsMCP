import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import Landing from './pages/Landing';
import './App.css';

function App() {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<Landing />} />
        {/* Future routes like /dashboard or /login can be added here */}
      </Routes>
    </Router>
  );
}

export default App;
