import { Card, CardContent, CardHeader, CardTitle } from "@/app/view/components/ui/card";
import { Skeleton } from "@/app/view/components/ui/skeleton";

export default function ViewLoading() {
  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>Loading Trust Dashboard</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          <Skeleton className="h-4 w-1/3" />
          <Skeleton className="h-4 w-2/3" />
        </CardContent>
      </Card>

      <section className="grid gap-4 xl:grid-cols-5">
        <Card className="xl:col-span-2">
          <CardContent className="space-y-3 py-5">
            <Skeleton className="h-44 w-full" />
            <Skeleton className="h-6 w-2/5" />
          </CardContent>
        </Card>
        <Card className="xl:col-span-3">
          <CardContent className="space-y-3 py-5">
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-16 w-full" />
          </CardContent>
        </Card>
      </section>
    </div>
  );
}
