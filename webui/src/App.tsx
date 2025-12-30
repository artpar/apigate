import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { DocumentationProvider } from '@/context/DocumentationContext';
import { AuthProvider, useAuth } from '@/context/AuthContext';
import { ThreePaneLayout } from '@/components/layout/ThreePaneLayout';
import { Dashboard } from '@/pages/Dashboard';
import { ModuleList } from '@/pages/ModuleList';
import { ModuleView } from '@/pages/ModuleView';
import { UsageDashboard } from '@/pages/UsageDashboard';
import { Login } from '@/pages/Login';
import { Register } from '@/pages/Register';
import { Setup } from '@/pages/Setup';

// Loading spinner component
function LoadingScreen() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="text-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
        <p className="mt-4 text-gray-600">Loading...</p>
      </div>
    </div>
  );
}

// Protected route wrapper
function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading, setupRequired } = useAuth();

  if (isLoading) {
    return <LoadingScreen />;
  }

  if (setupRequired) {
    return <Navigate to="/setup" replace />;
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
}

// Auth route wrapper (login/register - redirect if already authenticated)
function AuthRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading, setupRequired } = useAuth();

  if (isLoading) {
    return <LoadingScreen />;
  }

  if (setupRequired) {
    return <Navigate to="/setup" replace />;
  }

  if (isAuthenticated) {
    return <Navigate to="/" replace />;
  }

  return <>{children}</>;
}

// Setup route wrapper
function SetupRoute({ children }: { children: React.ReactNode }) {
  const { isLoading, setupRequired, isAuthenticated } = useAuth();

  if (isLoading) {
    return <LoadingScreen />;
  }

  if (!setupRequired) {
    if (isAuthenticated) {
      return <Navigate to="/" replace />;
    }
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
}

function AppRoutes() {
  return (
    <Routes>
      {/* Setup route - first time only */}
      <Route
        path="/setup"
        element={
          <SetupRoute>
            <Setup />
          </SetupRoute>
        }
      />

      {/* Auth routes */}
      <Route
        path="/login"
        element={
          <AuthRoute>
            <Login />
          </AuthRoute>
        }
      />
      <Route
        path="/register"
        element={
          <AuthRoute>
            <Register />
          </AuthRoute>
        }
      />

      {/* Protected routes */}
      <Route
        path="/"
        element={
          <ProtectedRoute>
            <ThreePaneLayout />
          </ProtectedRoute>
        }
      >
        <Route index element={<Dashboard />} />
        <Route path="usage" element={<UsageDashboard />} />
        <Route path=":module" element={<ModuleList />} />
        <Route path=":module/:id" element={<ModuleView />} />
      </Route>

      {/* Catch all - redirect to home */}
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}

export function App() {
  return (
    <AuthProvider>
      <DocumentationProvider>
        <BrowserRouter basename="/mod/ui">
          <AppRoutes />
        </BrowserRouter>
      </DocumentationProvider>
    </AuthProvider>
  );
}
