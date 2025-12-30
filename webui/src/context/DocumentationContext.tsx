/**
 * Documentation Context
 *
 * Provides global state for the documentation panel.
 * Tracks currently focused field, action, or module to show
 * contextual documentation in the right pane.
 */

import React, { createContext, useContext, useState, useCallback, useMemo, useRef } from 'react';
import type { DocFocus, ModuleSchema, FieldSchema, ActionSchema } from '@/types/schema';

interface DocumentationContextValue {
  /** Current focus target for documentation */
  focus: DocFocus;

  /** Set focus to a module */
  focusModule: (module: ModuleSchema) => void;

  /** Set focus to a field within current module */
  focusField: (field: FieldSchema, module: ModuleSchema) => void;

  /** Set focus to an action within current module */
  focusAction: (action: ActionSchema, module: ModuleSchema) => void;

  /** Clear focus (show default documentation) - delayed to allow hovering on docs panel */
  clearFocus: () => void;

  /** Cancel pending clear focus (call when hovering on docs panel) */
  cancelClearFocus: () => void;

  /** Whether documentation panel is expanded */
  isExpanded: boolean;

  /** Toggle documentation panel */
  toggleExpanded: () => void;

  /** Set expanded state */
  setExpanded: (expanded: boolean) => void;
}

const DocumentationContext = createContext<DocumentationContextValue | null>(null);

export function DocumentationProvider({ children }: { children: React.ReactNode }) {
  const [focus, setFocus] = useState<DocFocus>({ type: 'none' });
  const [isExpanded, setIsExpanded] = useState(true);
  const clearTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Cancel any pending clear
  const cancelClearFocus = useCallback(() => {
    if (clearTimeoutRef.current) {
      clearTimeout(clearTimeoutRef.current);
      clearTimeoutRef.current = null;
    }
  }, []);

  const focusModule = useCallback((module: ModuleSchema) => {
    cancelClearFocus();
    setFocus({ type: 'module', module });
  }, [cancelClearFocus]);

  const focusField = useCallback((field: FieldSchema, module: ModuleSchema) => {
    cancelClearFocus();
    setFocus({ type: 'field', field, module });
  }, [cancelClearFocus]);

  const focusAction = useCallback((action: ActionSchema, module: ModuleSchema) => {
    cancelClearFocus();
    setFocus({ type: 'action', action, module });
  }, [cancelClearFocus]);

  // Delayed clear to allow hovering on documentation panel
  const clearFocus = useCallback(() => {
    cancelClearFocus();
    clearTimeoutRef.current = setTimeout(() => {
      setFocus({ type: 'none' });
    }, 300); // 300ms delay gives time to move to docs panel
  }, [cancelClearFocus]);

  const toggleExpanded = useCallback(() => {
    setIsExpanded(prev => !prev);
  }, []);

  const setExpanded = useCallback((expanded: boolean) => {
    setIsExpanded(expanded);
  }, []);

  const value = useMemo(() => ({
    focus,
    focusModule,
    focusField,
    focusAction,
    clearFocus,
    cancelClearFocus,
    isExpanded,
    toggleExpanded,
    setExpanded,
  }), [focus, focusModule, focusField, focusAction, clearFocus, cancelClearFocus, isExpanded, toggleExpanded, setExpanded]);

  return (
    <DocumentationContext.Provider value={value}>
      {children}
    </DocumentationContext.Provider>
  );
}

export function useDocumentation() {
  const context = useContext(DocumentationContext);
  if (!context) {
    throw new Error('useDocumentation must be used within a DocumentationProvider');
  }
  return context;
}

/**
 * Hook to track field focus on hover/focus events.
 * Returns event handlers to attach to form fields.
 */
export function useFieldDocumentation(field: FieldSchema, module: ModuleSchema) {
  const { focusField, clearFocus } = useDocumentation();

  const handlers = useMemo(() => ({
    onFocus: () => focusField(field, module),
    onMouseEnter: () => focusField(field, module),
    onBlur: clearFocus,
    onMouseLeave: clearFocus,
  }), [field, module, focusField, clearFocus]);

  return handlers;
}
