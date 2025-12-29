/**
 * Three-Pane Layout Component
 *
 * Main application layout with:
 * - Left: Navigation sidebar
 * - Center: Main content area
 * - Right: Documentation panel
 */

import { NavLink, Outlet } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { fetchModules } from '@/api/schema';
import { DocumentationPanel } from '@/components/docs/DocumentationPanel';
import { useDocumentation } from '@/context/DocumentationContext';

export function ThreePaneLayout() {
  const { isExpanded } = useDocumentation();

  return (
    <div className="h-screen flex flex-col bg-gray-50">
      {/* Header */}
      <header className="bg-white border-b border-gray-200 px-4 py-3 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <svg className="w-8 h-8 text-primary-600" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" />
          </svg>
          <h1 className="text-xl font-bold text-gray-900">APIGate</h1>
          <span className="text-xs bg-primary-100 text-primary-700 px-2 py-0.5 rounded-full">Admin</span>
        </div>
        <div className="flex items-center gap-4">
          <a
            href="/mod/_schema"
            target="_blank"
            className="text-sm text-gray-500 hover:text-gray-700"
          >
            Schema API
          </a>
          <a
            href="/swagger"
            target="_blank"
            className="text-sm text-gray-500 hover:text-gray-700"
          >
            Swagger
          </a>
        </div>
      </header>

      {/* Main Layout */}
      <div className="flex-1 flex overflow-hidden">
        {/* Left Sidebar */}
        <Sidebar />

        {/* Main Content */}
        <main className={`flex-1 overflow-y-auto p-6 transition-all ${isExpanded ? 'mr-80' : ''}`}>
          <Outlet />
        </main>

        {/* Right Documentation Panel */}
        <div className="fixed right-0 top-[57px] bottom-0">
          <DocumentationPanel />
        </div>
      </div>
    </div>
  );
}

function Sidebar() {
  const { data: modules, isLoading, error } = useQuery({
    queryKey: ['modules'],
    queryFn: fetchModules,
  });

  return (
    <aside className="w-56 bg-white border-r border-gray-200 flex flex-col">
      {/* Overview */}
      <nav className="p-4 border-b border-gray-100">
        <NavLink
          to="/"
          end
          className={({ isActive }) =>
            `flex items-center gap-2 px-3 py-2 rounded-lg text-sm transition-colors ${
              isActive
                ? 'bg-primary-50 text-primary-700 font-medium'
                : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
            }`
          }
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
          </svg>
          Dashboard
        </NavLink>
      </nav>

      {/* Modules */}
      <div className="flex-1 overflow-y-auto p-4">
        <h2 className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-3">
          Modules
        </h2>

        {isLoading && (
          <div className="text-sm text-gray-500 px-3">Loading...</div>
        )}

        {error && (
          <div className="text-sm text-red-500 px-3">Failed to load modules</div>
        )}

        {modules && (
          <nav className="space-y-1">
            {modules.map((mod) => (
              <NavLink
                key={mod.module}
                to={`/${mod.plural}`}
                className={({ isActive }) =>
                  `flex items-center gap-2 px-3 py-2 rounded-lg text-sm transition-colors ${
                    isActive
                      ? 'bg-primary-50 text-primary-700 font-medium'
                      : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
                  }`
                }
              >
                <ModuleIcon module={mod.module} />
                <span className="capitalize">{mod.plural}</span>
              </NavLink>
            ))}
          </nav>
        )}
      </div>

      {/* Footer */}
      <div className="p-4 border-t border-gray-100">
        <p className="text-xs text-gray-400 text-center">
          APIGate v1.0.0
        </p>
      </div>
    </aside>
  );
}

function ModuleIcon({ module }: { module: string }) {
  // Map common module names to icons
  const icons: Record<string, JSX.Element> = {
    user: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
      </svg>
    ),
    plan: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
      </svg>
    ),
    key: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
      </svg>
    ),
    route: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
      </svg>
    ),
    upstream: (
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />
      </svg>
    ),
  };

  return icons[module] || (
    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 10h16M4 14h16M4 18h16" />
    </svg>
  );
}
