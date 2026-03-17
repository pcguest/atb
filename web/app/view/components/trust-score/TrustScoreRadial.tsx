"use client";

import { PolarAngleAxis, RadialBar, RadialBarChart, ResponsiveContainer } from "recharts";

function scoreColor(score: number): string {
  if (score >= 80) {
    return "hsl(154 70% 42%)";
  }
  if (score >= 50) {
    return "hsl(36 92% 52%)";
  }
  return "hsl(2 83% 58%)";
}

export function TrustScoreRadial({ score }: { score: number }) {
  return (
    <div className="h-44 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <RadialBarChart
          data={[{ name: "Trust", value: score }]}
          startAngle={205}
          endAngle={-25}
          innerRadius="68%"
          outerRadius="100%"
        >
          <PolarAngleAxis type="number" domain={[0, 100]} tick={false} />
          <RadialBar dataKey="value" cornerRadius={14} fill={scoreColor(score)} background />
        </RadialBarChart>
      </ResponsiveContainer>
    </div>
  );
}
