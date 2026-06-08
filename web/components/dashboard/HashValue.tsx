"use client";

import { useState } from "react";

import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/app/view/components/ui/tooltip";
import { copyTextToClipboard, truncateSha256 } from "@/lib/hash-display";

type HashValueProps = {
  hash: string;
  className?: string;
};

export function HashValue({ hash, className }: HashValueProps) {
  const [copied, setCopied] = useState(false);
  const display = truncateSha256(hash);

  async function handleCopy(): Promise<void> {
    const ok = await copyTextToClipboard(hash);
    if (ok) {
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    }
  }

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            type="button"
            onClick={() => void handleCopy()}
            className={`cursor-copy font-mono text-left hover:text-primary ${className ?? ""}`}
            title={copied ? "Copied" : "Click to copy full hash"}
          >
            {display}
          </button>
        </TooltipTrigger>
        <TooltipContent side="top" className="max-w-md break-all font-mono text-[10px]">
          {copied ? "Copied full hash" : hash}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
