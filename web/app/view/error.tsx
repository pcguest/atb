"use client";

import { AlertTriangle } from "lucide-react";

import { Button } from "@/app/view/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/app/view/components/ui/card";

export default function ViewError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <div className="mx-auto max-w-3xl py-8">
      <Card className="border-destructive/40 bg-card">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-foreground">
            <AlertTriangle className="h-5 w-5 text-destructive" aria-hidden="true" />
            Viewer API Error
          </CardTitle>
          <CardDescription className="text-foreground">
            Dashboard data could not be verified or loaded from the local viewer API.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          <p className="text-sm text-foreground">{error.message || "Unknown error"}</p>
          <Button type="button" variant="outline" onClick={reset}>
            Verify bundle first
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
