import type { ReactNode } from "react";

import { RoleSelector } from "@/app/view/components/role-selector/RoleSelector";
import { DashboardShell } from "@/app/view/components/shell/DashboardShell";

export default function ViewLayout({ children }: { children: ReactNode }) {
  return <DashboardShell roleControl={<RoleSelector />}>{children}</DashboardShell>;
}
