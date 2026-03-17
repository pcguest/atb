"use client";

import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

import type { DashboardRole } from "@/lib/roles";

export type ThemeMode = "dark" | "light";

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
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        role: state.role,
        theme: state.theme,
      }),
    },
  ),
);
