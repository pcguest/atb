"use client";

import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

import type { DashboardRole } from "@/lib/roles";

export type ThemeMode = "dark" | "light";

type StringStorage = {
  getItem: (name: string) => string | null;
  setItem: (name: string, value: string) => void;
  removeItem: (name: string) => void;
};

type UIStoreState = {
  role: DashboardRole;
  theme: ThemeMode;
  sidebarOpen: boolean;
  setRole: (role: DashboardRole) => void;
  setTheme: (theme: ThemeMode) => void;
  toggleTheme: () => void;
  setSidebarOpen: (open: boolean) => void;
};

const defaultState: Pick<UIStoreState, "role" | "theme" | "sidebarOpen"> = {
  role: "engineer",
  theme: "dark",
  sidebarOpen: false,
};

function createMemoryStorage(): StringStorage {
  const values = new Map<string, string>();
  return {
    getItem: (name) => values.get(name) ?? null,
    setItem: (name, value) => {
      values.set(name, value);
    },
    removeItem: (name) => {
      values.delete(name);
    },
  };
}

function resolveUIStorage(): StringStorage {
  if (typeof window === "undefined") {
    return createMemoryStorage();
  }
  return window.localStorage;
}

export const useUIStore = create<UIStoreState>()(
  persist(
    (set) => ({
      ...defaultState,
      setRole: (role) => set({ role }),
      setTheme: (theme) => set({ theme }),
      toggleTheme: () =>
        set((state) => ({
          theme: state.theme === "dark" ? "light" : "dark",
        })),
      setSidebarOpen: (open) => set({ sidebarOpen: open }),
    }),
    {
      name: "atb-ui-store-v1",
      storage: createJSONStorage(resolveUIStorage),
      partialize: (state) => ({
        role: state.role,
        theme: state.theme,
      }),
    },
  ),
);
