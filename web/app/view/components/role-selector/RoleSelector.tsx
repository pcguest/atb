"use client";

import * as Select from "@radix-ui/react-select";
import { Check, ChevronDown } from "lucide-react";
import React from "react";

import { dashboardRoleLabel, dashboardRoles, type DashboardRole } from "@/lib/roles";
import { useUIStore } from "@/lib/state/ui-store";

export function RoleSelector() {
  const role = useUIStore((state) => state.role);
  const setRole = useUIStore((state) => state.setRole);

  return (
    <div className="flex items-center gap-2">
      <label
        htmlFor="role-selector"
        className="text-xs uppercase tracking-wide text-muted-foreground"
      >
        Role
      </label>
      <Select.Root
        value={role}
        onValueChange={(nextValue) => {
          if (dashboardRoles.includes(nextValue as DashboardRole)) {
            setRole(nextValue as DashboardRole);
          }
        }}
      >
        <Select.Trigger
          id="role-selector"
          data-testid="role-selector"
          className="inline-flex h-9 min-w-[10rem] items-center justify-between rounded-md border border-border bg-card px-3 text-sm text-foreground shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          aria-label="Select dashboard role"
        >
          <Select.Value />
          <Select.Icon asChild>
            <ChevronDown className="h-4 w-4 text-muted-foreground" />
          </Select.Icon>
        </Select.Trigger>
        <Select.Portal>
          <Select.Content
            position="popper"
            sideOffset={6}
            className="z-50 min-w-[10rem] rounded-md border border-border bg-popover p-1 text-popover-foreground shadow-lg"
          >
            <Select.Viewport>
              {dashboardRoles.map((dashboardRole) => (
                <Select.Item
                  key={dashboardRole}
                  value={dashboardRole}
                  data-testid={`role-selector-${dashboardRole}`}
                  className="relative flex cursor-default select-none items-center rounded-sm py-2 pl-8 pr-2 text-sm outline-none data-[highlighted]:bg-muted data-[highlighted]:text-foreground"
                >
                  <Select.ItemIndicator className="absolute left-2 inline-flex h-3.5 w-3.5 items-center justify-center">
                    <Check className="h-3.5 w-3.5" />
                  </Select.ItemIndicator>
                  <Select.ItemText>{dashboardRoleLabel[dashboardRole]}</Select.ItemText>
                </Select.Item>
              ))}
            </Select.Viewport>
          </Select.Content>
        </Select.Portal>
      </Select.Root>
    </div>
  );
}
