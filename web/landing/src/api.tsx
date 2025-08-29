import React from 'react';
import { createRoot } from 'react-dom/client';
import ApiReference from './components/ApiReference';
import './styles.css';

const root = createRoot(document.getElementById('root')!);
root.render(
  <React.StrictMode>
    <ApiReference />
  </React.StrictMode>
);

