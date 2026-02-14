import { create } from "zustand";
import type { ServerInstance, AppConfig } from "@/types";

const MAX_CONSOLE_LINES = 5000;

function applyThemeToDocument(theme: "dark" | "light" | "system") {
  const root = document.documentElement;
  const isDark =
    theme === "dark" ||
    (theme === "system" &&
      window.matchMedia("(prefers-color-scheme: dark)").matches);
  if (isDark) {
    root.classList.add("dark");
  } else {
    root.classList.remove("dark");
  }
}

interface AppState {
  // Instances
  instances: ServerInstance[];
  setInstances: (instances: ServerInstance[]) => void;
  updateInstance: (id: string, updates: Partial<ServerInstance>) => void;
  addInstance: (instance: ServerInstance) => void;
  removeInstance: (id: string) => void;

  // Selected instance
  selectedInstanceId: string | null;
  setSelectedInstanceId: (id: string | null) => void;

  // App config
  appConfig: AppConfig | null;
  setAppConfig: (config: AppConfig) => void;

  // Theme
  theme: "dark" | "light" | "system";
  setTheme: (theme: "dark" | "light" | "system") => void;

  // Sidebar
  sidebarCollapsed: boolean;
  toggleSidebar: () => void;

  // Console logs per instance
  consoleLogs: Record<string, string[]>;
  appendConsoleLog: (instanceId: string, line: string) => void;
  clearConsoleLogs: (instanceId: string) => void;
}

export const useAppStore = create<AppState>((set, get) => ({
  instances: [],
  setInstances: (instances) => set({ instances }),

  updateInstance: (id, updates) =>
    set((state) => ({
      instances: state.instances.map((inst) =>
        inst.id === id ? { ...inst, ...updates } : inst,
      ),
    })),

  addInstance: (instance) =>
    set((state) => ({
      instances: [...state.instances, instance],
    })),

  removeInstance: (id) =>
    set((state) => ({
      instances: state.instances.filter((inst) => inst.id !== id),
      selectedInstanceId:
        state.selectedInstanceId === id ? null : state.selectedInstanceId,
      consoleLogs: (() => {
        const { [id]: _, ...rest } = state.consoleLogs;
        return rest;
      })(),
    })),

  selectedInstanceId: null,
  setSelectedInstanceId: (id) => set({ selectedInstanceId: id }),

  appConfig: null,
  setAppConfig: (config) => set({ appConfig: config }),

  theme: "system",
  setTheme: (theme) => {
    set({ theme });
    applyThemeToDocument(theme);
  },

  sidebarCollapsed: false,
  toggleSidebar: () =>
    set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),

  consoleLogs: {},
  appendConsoleLog: (instanceId, line) =>
    set((state) => {
      const current = state.consoleLogs[instanceId] ?? [];
      const next = [...current, line];
      const trimmed =
        next.length > MAX_CONSOLE_LINES ? next.slice(-MAX_CONSOLE_LINES) : next;
      return {
        consoleLogs: {
          ...state.consoleLogs,
          [instanceId]: trimmed,
        },
      };
    }),

  clearConsoleLogs: (instanceId) =>
    set((state) => ({
      consoleLogs: {
        ...state.consoleLogs,
        [instanceId]: [],
      },
    })),
}));

// Apply theme on store init and when system preference changes
applyThemeToDocument(useAppStore.getState().theme);
if (typeof window !== "undefined") {
  window
    .matchMedia("(prefers-color-scheme: dark)")
    .addEventListener("change", () => {
      const { theme } = useAppStore.getState();
      if (theme === "system") applyThemeToDocument(theme);
    });
}
