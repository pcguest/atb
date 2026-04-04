"use client";

import { LayoutDashboard, Menu, ShieldCheck, SlidersHorizontal, X } from "lucide-react";
import type { ReactNode } from "react";

import packageMeta from "@/package.json";
import { Button } from "@/app/view/components/ui/button";
import { ThemeToggle } from "@/app/view/components/ui/theme-toggle";
import { dashboardRoleLabel } from "@/lib/roles";
import { useUIStore } from "@/lib/state/ui-store";
import { cn } from "@/lib/utils";

const navigationByRole = {
  engineer: [
    { id: "overview", label: "Overview", icon: LayoutDashboard },
    { id: "trace", label: "Trace Data", icon: SlidersHorizontal },
    { id: "security", label: "Security", icon: ShieldCheck },
  ],
  auditor: [
    { id: "overview", label: "Overview", icon: LayoutDashboard },
    { id: "evidence", label: "Evidence", icon: ShieldCheck },
    { id: "controls", label: "Controls", icon: SlidersHorizontal },
  ],
  executive: [
    { id: "overview", label: "Overview", icon: LayoutDashboard },
    { id: "risk", label: "Risk Posture", icon: ShieldCheck },
    { id: "strategy", label: "Strategy", icon: SlidersHorizontal },
  ],
} as const;

export function DashboardShell({
  roleControl,
  children,
}: {
  roleControl: ReactNode;
  children: ReactNode;
}) {
  const role = useUIStore((state) => state.role);
  const sidebarOpen = useUIStore((state) => state.sidebarOpen);
  const setSidebarOpen = useUIStore((state) => state.setSidebarOpen);
  const navItems = navigationByRole[role];

  return (
    <div className="relative min-h-screen overflow-hidden">
      <a
        href="#dashboard-content"
        className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-50 focus:rounded-md focus:bg-primary focus:px-3 focus:py-2 focus:text-primary-foreground"
      >
        Skip to dashboard content
      </a>

      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_left,hsl(193_99%_45%_/_0.12),transparent_42%),radial-gradient(circle_at_top_right,hsl(32_95%_57%_/_0.1),transparent_40%)]" />

      <div className="relative z-10 flex min-h-screen">
        <aside
          className={cn(
            "fixed inset-y-0 left-0 z-40 w-64 border-r border-border bg-card/90 px-4 py-5 backdrop-blur-sm transition-transform duration-200 ease-out md:static md:translate-x-0",
            sidebarOpen ? "translate-x-0" : "-translate-x-full md:translate-x-0",
          )}
        >
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs uppercase tracking-[0.2em] text-muted-foreground">ATB</p>
              <h1 className="text-lg font-semibold text-foreground">Trust Dashboard</h1>
            </div>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="md:hidden"
              aria-label="Close sidebar"
              onClick={() => setSidebarOpen(false)}
            >
              <X className="h-4 w-4" />
            </Button>
          </div>

          <div className="mt-6 rounded-lg border border-border bg-background/60 p-3">
            <p className="text-xs uppercase tracking-wide text-muted-foreground">Active role</p>
            <p className="mt-1 font-semibold text-foreground">{dashboardRoleLabel[role]}</p>
          </div>

          <nav className="mt-6 space-y-1" aria-label="Dashboard sections">
            {navItems.map((item) => {
              const Icon = item.icon;
              return (
                <button
                  key={item.id}
                  type="button"
                  className="flex w-full items-center gap-2 rounded-md border border-transparent px-3 py-2 text-left text-sm text-muted-foreground transition hover:border-border hover:bg-muted/40 hover:text-foreground"
                >
                  <Icon className="h-4 w-4" aria-hidden="true" />
                  {item.label}
                </button>
              );
            })}
          </nav>
        </aside>

        {sidebarOpen && (
          <button
            type="button"
            className="fixed inset-0 z-30 bg-background/60 md:hidden"
            aria-label="Close sidebar overlay"
            onClick={() => setSidebarOpen(false)}
          />
        )}

        <div className="flex min-h-screen flex-1 flex-col md:ml-0">
          <header className="sticky top-0 z-20 border-b border-border bg-background/80 px-4 py-3 backdrop-blur md:px-8">
            <div className="flex items-center justify-between gap-3">
              <div className="flex items-center gap-3">
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  className="md:hidden"
                  aria-label="Open sidebar"
                  onClick={() => setSidebarOpen(true)}
                >
                  <Menu className="h-4 w-4" />
                </Button>
                <div>
                  <p className="text-xs uppercase tracking-[0.16em] text-muted-foreground">
                    Version {packageMeta.version}
                  </p>
                  <p className="text-sm font-medium text-foreground">
                    Enterprise Trust Control Plane
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-4">
                {roleControl}
                <ThemeToggle />
              </div>
            </div>
          </header>

          <main id="dashboard-content" className="flex-1 px-4 py-4 md:px-8 md:py-6">
            {children}
          </main>
        </div>
      </div>
    </div>
  );
}
