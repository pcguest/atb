"use client";

import { useId } from "react";

import { useUIStore } from "@/lib/state/ui-store";
import { type ThemePreference } from "@/lib/theme";

export function ThemeToggle() {
  const id = useId();
  const theme = useUIStore((state) => state.theme);
  const setTheme = useUIStore((state) => state.setTheme);

  return (
    <div className="flex items-center gap-2">
      <label htmlFor={id} className="text-xs uppercase tracking-wide text-muted-foreground">
        Theme
      </label>
      <select
        id={id}
        aria-label="Theme"
        value={theme}
        onChange={(event) => {
          if (event.target.value === "system" || event.target.value === "light" || event.target.value === "dark") {
            setTheme(event.target.value as ThemePreference);
          }
        }}
        className="h-9 rounded-md border border-border bg-card px-2 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        <option value="system">System</option>
        <option value="light">Light</option>
        <option value="dark">Dark</option>
      </select>
    </div>
  );
}
