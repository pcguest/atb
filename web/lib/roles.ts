export const dashboardRoles = ["engineer", "security", "auditor"] as const;

export type DashboardRole = (typeof dashboardRoles)[number];
export type Role = DashboardRole;

export const dashboardRoleLabel: Record<DashboardRole, string> = {
  engineer: "Engineer",
  security: "Security",
  auditor: "Auditor",
};

export const dashboardRoleDescription: Record<DashboardRole, string> = {
  engineer: "Dense records, identifiers, schemas, tools, and model details",
  security: "Findings, policy decisions, approvals, timeline, and context provenance",
  auditor: "Integrity, profile obligations, coverage, custody, anchors, and exports",
};

// These modes control presentation only. API authorisation is enforced by the
// server and does not change when this selector changes.
export function canViewRawData(_role: DashboardRole): boolean {
  return true;
}

export function canExportEvidence(_role: DashboardRole): boolean {
  return true;
}

export function canViewRawEventData(role: DashboardRole): boolean {
  return canViewRawData(role);
}

export function canRevealMaskedFields(_role: DashboardRole): boolean {
  return true;
}

export function isDensePresentation(role: DashboardRole): boolean {
  return role === "engineer";
}
