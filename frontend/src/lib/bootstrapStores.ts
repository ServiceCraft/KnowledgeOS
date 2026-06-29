import { useAuthStore } from '@/stores/authStore';
import { useUIStore } from '@/stores/uiStore';

type PersistedPayload = { state?: Record<string, unknown> };

/** Restore persisted client state before the first React render to avoid auth/layout flashes. */
export function bootstrapPersistedStores() {
  bootstrapAuthStore();
  bootstrapUIStore();
}

function bootstrapAuthStore() {
  try {
    const raw = localStorage.getItem('auth-storage');
    if (!raw) {
      useAuthStore.setState({ _hasHydrated: true });
      return;
    }
    const parsed = JSON.parse(raw) as PersistedPayload;
    if (parsed.state) {
      useAuthStore.setState({ ...parsed.state, _hasHydrated: true });
    } else {
      useAuthStore.setState({ _hasHydrated: true });
    }
  } catch {
    useAuthStore.setState({ _hasHydrated: true });
  }
}

function bootstrapUIStore() {
  try {
    const raw = localStorage.getItem('ui-storage');
    if (!raw) return;
    const parsed = JSON.parse(raw) as PersistedPayload;
    if (!parsed.state) return;
    useUIStore.setState({
      sidebarCollapsed: Boolean(parsed.state.sidebarCollapsed),
      darkMode: Boolean(parsed.state.darkMode),
    });
    if (parsed.state.darkMode) {
      document.documentElement.classList.add('dark');
    }
  } catch {
    // ignore corrupt storage
  }
}
