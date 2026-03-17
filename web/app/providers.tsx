"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useEffect, useState, type ReactNode } from "react";

function useAxeInDev(): void {
  useEffect(() => {
    if (process.env.NODE_ENV !== "development") {
      return;
    }

    void Promise.all([import("react"), import("react-dom"), import("@axe-core/react")])
      .then(([React, ReactDOM, axe]) => {
        axe.default(React, ReactDOM, 1000);
      })
      .catch(() => undefined);
  }, []);
}

export function Providers({ children }: { children: ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            refetchOnWindowFocus: false,
          },
        },
      }),
  );

  useAxeInDev();

  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}
