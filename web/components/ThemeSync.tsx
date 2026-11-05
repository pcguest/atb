"use client";

import { useEffect, useState } from "react";

import { resolveTheme } from "@/lib/theme";
import { useUIStore } from "@/lib/state/ui-store";

export function ThemeSync() {
  const theme = useUIStore((state) => state.theme);
  const [hydrated, setHydrated] = useState(false);

  useEffect(() => {
    void Promise.resolve(useUIStore.persist.rehydrate()).then(() => setHydrated(true));
  }, []);

  useEffect(() => {
    if (!hydrated || typeof window.matchMedia !== "function") return;
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const apply = () => {
      const resolved = resolveTheme(theme, media.matches);
      document.documentElement.dataset.theme = resolved;
      document.documentElement.style.colorScheme = resolved;
    };
    apply();
    if (theme !== "system") return;
    media.addEventListener("change", apply);
    return () => media.removeEventListener("change", apply);
  }, [theme, hydrated]);

  return null;
}
