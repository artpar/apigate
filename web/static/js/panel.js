/**
 * Context Panel - Global documentation and reference panel
 * Provides consistent help across all pages
 */
(function() {
    'use strict';

    const STORAGE_KEY = 'apigate_panel_open';

    class ContextPanel {
        constructor() {
            this.panel = null;
            this.isOpen = false;
            this.activeTab = 'docs';
            this.init();
        }

        init() {
            // Wait for DOM
            if (document.readyState === 'loading') {
                document.addEventListener('DOMContentLoaded', () => this.setup());
            } else {
                this.setup();
            }
        }

        setup() {
            this.panel = document.getElementById('context-panel');
            if (!this.panel) return;

            // Restore state from localStorage
            const savedState = localStorage.getItem(STORAGE_KEY);
            if (savedState === 'true') {
                this.open();
            }

            // Setup tab switching
            this.panel.querySelectorAll('.panel-tab').forEach(tab => {
                tab.addEventListener('click', () => this.switchTab(tab.dataset.tab));
            });

            // Keyboard shortcuts
            document.addEventListener('keydown', (e) => this.handleKeydown(e));

            // Close on click outside (optional - commented out for now)
            // document.addEventListener('click', (e) => this.handleOutsideClick(e));
        }

        handleKeydown(e) {
            // ? key to toggle panel (when not in input)
            if (e.key === '?' && !this.isInputFocused()) {
                e.preventDefault();
                this.toggle();
            }
            // Escape to close
            if (e.key === 'Escape' && this.isOpen) {
                this.close();
            }
        }

        isInputFocused() {
            const active = document.activeElement;
            if (!active) return false;
            const tag = active.tagName.toLowerCase();
            return tag === 'input' || tag === 'textarea' || tag === 'select' || active.isContentEditable;
        }

        toggle() {
            if (this.isOpen) {
                this.close();
            } else {
                this.open();
            }
        }

        open() {
            if (!this.panel) return;
            this.panel.classList.add('open');
            document.body.classList.add('panel-open');
            this.isOpen = true;
            localStorage.setItem(STORAGE_KEY, 'true');
        }

        close() {
            if (!this.panel) return;
            this.panel.classList.remove('open');
            document.body.classList.remove('panel-open');
            this.isOpen = false;
            localStorage.setItem(STORAGE_KEY, 'false');
        }

        switchTab(tabName) {
            if (!this.panel) return;
            this.activeTab = tabName;

            // Update tab buttons
            this.panel.querySelectorAll('.panel-tab').forEach(tab => {
                tab.classList.toggle('active', tab.dataset.tab === tabName);
            });

            // Update content
            this.panel.querySelectorAll('.panel-content').forEach(content => {
                content.classList.toggle('active', content.id === `panel-${tabName}`);
            });
        }

        // Method to programmatically set content (for dynamic pages)
        setContent(tab, html) {
            const content = document.getElementById(`panel-${tab}`);
            if (content) {
                content.innerHTML = html;
            }
        }

        // Show a specific section by scrolling to it
        showSection(sectionId) {
            this.open();
            const section = document.getElementById(sectionId);
            if (section) {
                section.scrollIntoView({ behavior: 'smooth', block: 'start' });
            }
        }
    }

    // Create global instance
    window.ContextPanel = new ContextPanel();
})();
