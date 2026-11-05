"use client";

import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

import { isThemePreference, type ThemePreference } from "@/lib/theme";

export type ThemeMode = ThemePreference;

type StringStorage = {
  getItem: (name: string) => string | null;
  setItem: (name: string, value: string) => void;
  removeItem: (name: string) => void;
};

type UIStoreState = {
  theme: ThemeMode;
  setTheme: (theme: ThemeMode) => void;
  toggleTheme: () => void;
};

const defaultState: Pick<UIStoreState, "theme"> = {
  theme: "system",
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
      setTheme: (theme) => set({ theme }),
      toggleTheme: () =>
        set((state) => ({
          theme: state.theme === "system" ? "dark" : state.theme === "dark" ? "light" : "system",
        })),
    }),
    {
      name: "atb-ui-store-v1",
      version: 3,
      skipHydration: true,
      migrate: (persisted, version) => {
        const state = persisted as Record<string, unknown>;
        if (version < 3 && state.theme === "dark") {
          return { ...state, theme: "system" } as Partial<UIStoreState>;
        }
        if (!isThemePreference(state.theme)) {
          return { ...state, theme: "system" } as Partial<UIStoreState>;
        }
        return state as Partial<UIStoreState>;
      },
      storage: createJSONStorage(resolveUIStorage),
      partialize: (state) => ({
        theme: state.theme,
      }),
    },
  ),
);
