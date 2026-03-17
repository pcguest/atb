export const dashboardRoles = ["engineer", "auditor", "executive"] as const;

export type DashboardRole = (typeof dashboardRoles)[number];
export type Role = DashboardRole;

export const dashboardRoleLabel: Record<DashboardRole, string> = {
  engineer: "Engineer",
  auditor: "Auditor",
  executive: "Executive",
};

export function canViewRawData(role: DashboardRole): boolean {
  return role === "engineer";
}

export function canExportEvidence(role: DashboardRole): boolean {
  return role === "auditor" || role === "engineer";
}

export function canViewExecutiveSummary(role: DashboardRole): boolean {
  return role === "executive";
}

export function canViewRawEventData(role: DashboardRole): boolean {
  return canViewRawData(role);
}

export function canRevealMaskedFields(role: DashboardRole): boolean {
  return canViewRawData(role);
}
