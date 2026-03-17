"use client";

import * as Switch from "@radix-ui/react-switch";
import { MoonStar, SunMedium } from "lucide-react";
import { useEffect } from "react";

import { useUIStore } from "@/lib/state/ui-store";

export function ThemeToggle() {
  const theme = useUIStore((state) => state.theme);
  const setTheme = useUIStore((state) => state.setTheme);

  useEffect(() => {
    const root = document.documentElement;
    root.classList.toggle("dark", theme === "dark");
    root.classList.toggle("light", theme === "light");
  }, [theme]);

  return (
    <label className="flex items-center gap-2 text-sm text-muted-foreground">
      <SunMedium className="h-4 w-4" aria-hidden="true" />
      <Switch.Root
        checked={theme === "dark"}
        onCheckedChange={(checked) => setTheme(checked ? "dark" : "light")}
        className="relative h-6 w-11 rounded-full border border-border bg-muted data-[state=checked]:bg-primary"
        aria-label="Toggle dashboard theme"
      >
        <Switch.Thumb className="block h-5 w-5 translate-x-0.5 rounded-full bg-foreground transition-transform data-[state=checked]:translate-x-[1.25rem] data-[state=checked]:bg-primary-foreground" />
      </Switch.Root>
      <MoonStar className="h-4 w-4" aria-hidden="true" />
    </label>
  );
}
