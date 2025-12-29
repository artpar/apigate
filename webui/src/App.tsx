import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { DocumentationProvider } from '@/context/DocumentationContext';
import { ThreePaneLayout } from '@/components/layout/ThreePaneLayout';
import { Dashboard } from '@/pages/Dashboard';
import { ModuleList } from '@/pages/ModuleList';
import { ModuleView } from '@/pages/ModuleView';

export function App() {
  return (
    <DocumentationProvider>
      <BrowserRouter basename="/mod/ui">
        <Routes>
          <Route path="/" element={<ThreePaneLayout />}>
            <Route index element={<Dashboard />} />
            <Route path=":module" element={<ModuleList />} />
            <Route path=":module/:id" element={<ModuleView />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </DocumentationProvider>
  );
}
