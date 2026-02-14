import { Play, Square, RotateCw, Server } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useAppStore } from "@/stores/app-store";
import { cn } from "@/lib/utils";
import type { ServerInstance } from "@/types";

export interface OverviewTabProps {
  instanceId: string;
}

function getStatusColor(status: ServerInstance["status"]) {
  switch (status) {
    case "running":
      return "bg-emerald-500";
    case "stopped":
    case "crashed":
      return "bg-red-500";
    case "starting":
    case "stopping":
      return "bg-amber-500";
    case "installing":
    case "updating":
      return "bg-blue-500";
    default:
      return "bg-zinc-500";
  }
}

export function OverviewTab({ instanceId }: OverviewTabProps) {
  const instances = useAppStore((s) => s.instances);
  const updateInstance = useAppStore((s) => s.updateInstance);

  const instance = instances.find((i) => i.id === instanceId);

  if (!instance) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center py-12">
          <Server className="mb-4 h-12 w-12 text-muted-foreground/50" />
          <p className="text-muted-foreground">Instance not found</p>
        </CardContent>
      </Card>
    );
  }

  const handleStart = async () => {
    try {
      await (window as any).go?.main?.App?.StartInstance?.(instanceId);
      updateInstance(instanceId, { status: "starting" });
    } catch {
      // error handling
    }
  };

  const handleStop = async () => {
    try {
      await (window as any).go?.main?.App?.StopInstance?.(instanceId);
      updateInstance(instanceId, { status: "stopping" });
    } catch {
      // error handling
    }
  };

  const handleRestart = async () => {
    try {
      await (window as any).go?.main?.App?.RestartInstance?.(instanceId);
      updateInstance(instanceId, { status: "starting" });
    } catch {
      // error handling
    }
  };

  const ipPort = `127.0.0.1:${instance.port}`;

  return (
    <div className="space-y-6">
      {/* Server info card */}
      <Card className="border-border">
        <CardHeader className="pb-3">
          <div className="flex items-start justify-between">
            <div>
              <CardTitle className="text-lg">{instance.name}</CardTitle>
              <CardDescription className="mt-1">
                {instance.current_map} • {instance.game_mode}
              </CardDescription>
            </div>
            <div className="flex items-center gap-2">
              <span
                className={cn(
                  "h-2.5 w-2.5 shrink-0 rounded-full",
                  getStatusColor(instance.status)
                )}
              />
              <Badge variant="secondary">{instance.status}</Badge>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="grid gap-2 text-sm sm:grid-cols-2">
            <div className="flex justify-between gap-2">
              <span className="text-muted-foreground">IP:Port</span>
              <span className="font-mono">{ipPort}</span>
            </div>
            <div className="flex justify-between gap-2">
              <span className="text-muted-foreground">Map</span>
              <span>{instance.current_map}</span>
            </div>
            <div className="flex justify-between gap-2">
              <span className="text-muted-foreground">Players</span>
              <span>0 / {instance.max_players}</span>
            </div>
            <div className="flex justify-between gap-2">
              <span className="text-muted-foreground">Game mode</span>
              <span>{instance.game_mode}</span>
            </div>
          </div>
          <div className="flex flex-wrap gap-2 pt-2">
            <Button
              size="sm"
              variant="outline"
              onClick={handleStart}
              disabled={
                instance.status === "running" || instance.status === "starting"
              }
            >
              <Play className="mr-1.5 h-4 w-4" />
              Start
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={handleStop}
              disabled={
                instance.status === "stopped" || instance.status === "stopping"
              }
            >
              <Square className="mr-1.5 h-4 w-4" />
              Stop
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={handleRestart}
              disabled={instance.status !== "running"}
            >
              <RotateCw className="mr-1.5 h-4 w-4" />
              Restart
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Quick stats */}
      <div className="grid gap-4 sm:grid-cols-2">
        <Card className="border-border">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Uptime
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-semibold">—</p>
            <p className="text-xs text-muted-foreground">Placeholder</p>
          </CardContent>
        </Card>
        <Card className="border-border">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Total players joined
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-semibold">—</p>
            <p className="text-xs text-muted-foreground">Placeholder</p>
          </CardContent>
        </Card>
      </div>

      {/* Metric chart placeholders */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {[
          { label: "CPU", value: "—" },
          { label: "RAM", value: "—" },
          { label: "Tick Rate", value: "—" },
          { label: "Network", value: "—" },
        ].map((m) => (
          <Card key={m.label} className="border-border">
            <CardHeader className="pb-1">
              <CardTitle className="text-sm font-medium text-muted-foreground">
                {m.label}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-xl font-semibold">{m.value}</p>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
