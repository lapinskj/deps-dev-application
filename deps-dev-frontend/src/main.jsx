import './index.css';
import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App.jsx';
import DependencyDetail from './components/DependencyDetail.jsx';
import { BrowserRouter, Routes, Route } from 'react-router-dom';

ReactDOM.createRoot(document.getElementById('root')).render(
  <BrowserRouter>
    <Routes>
      <Route path="/" element={<App />} />
      <Route path="/dependencies/:system/:name/:version" element={<DependencyDetail />} />
    </Routes>
  </BrowserRouter>
);
