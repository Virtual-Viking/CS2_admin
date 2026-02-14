import { useEffect, useState, useRef } from "react";
import { Play, Square, RotateCw, Server, Clock, Users, Cpu, HardDrive, Activity, Wifi } from "lucide-react";
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

function formatUptime(seconds: number): string {
  if (seconds <= 0) return "—";
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  if (h > 0) return `${h}h ${m}m ${s}s`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
}

interface LiveMetrics {
  cpu_pct: number;
  ram_mb: number;
  tick_rate: number;
  players: number;
  net_in_kbps: number;
  net_out_kbps: number;
}

export function OverviewTab({ instanceId }: OverviewTabProps) {
  const instances = useAppStore((s) => s.instances);
  const updateInstance = useAppStore((s) => s.updateInstance);
  const instance = instances.find((i) => i.id === instanceId);

  const [metrics, setMetrics] = useState<LiveMetrics>({
    cpu_pct: 0, ram_mb: 0, tick_rate: 0, players: 0, net_in_kbps: 0, net_out_kbps: 0,
  });
  const [uptime, setUptime] = useState(0);
  const startedAtRef = useRef<number | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // Track uptime
  useEffect(() => {
    if (instance?.status === "running") {
      if (!startedAtRef.current) {
        startedAtRef.current = Date.now();
      }
      const timer = setInterval(() => {
        if (startedAtRef.current) {
          setUptime(Math.floor((Date.now() - startedAtRef.current) / 1000));
        }
      }, 1000);
      return () => clearInterval(timer);
    } else {
      startedAtRef.current = null;
      setUptime(0);
    }
  }, [instance?.status]);

  // Listen for metrics events
  useEffect(() => {
    const eventName = `metrics:${instanceId}`;
    const cb = (data: unknown) => {
      const d = data as Record<string, number>;
      if (d && typeof d === "object") {
        setMetrics({
          cpu_pct: d.cpu_pct ?? d.CPUPercent ?? 0,
          ram_mb: d.ram_mb ?? d.RAMMb ?? 0,
          tick_rate: d.tick_rate ?? d.TickRate ?? 0,
          players: d.players ?? d.Players ?? 0,
          net_in_kbps: d.net_in_kbps ?? d.NetInKbps ?? 0,
          net_out_kbps: d.net_out_kbps ?? d.NetOutKbps ?? 0,
        });
      }
    };
    (window as any).runtime?.EventsOn?.(eventName, cb);
    return () => {
      (window as any).runtime?.EventsOff?.(eventName);
    };
  }, [instanceId]);

  // Listen for status events
  useEffect(() => {
    const eventName = `status:${instanceId}`;
    const cb = (status: unknown) => {
      const s = String(status).toLowerCase() as ServerInstance["status"];
      updateInstance(instanceId, { status: s });
    };
    (window as any).runtime?.EventsOn?.(eventName, cb);
    return () => {
      (window as any).runtime?.EventsOff?.(eventName);
    };
  }, [instanceId, updateInstance]);

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

  const handleAction = async (action: string) => {
    setActionLoading(action);
    try {
      const fn = (window as any).go?.main?.App;
      if (action === "start") {
        await fn?.StartInstance?.(instanceId);
        updateInstance(instanceId, { status: "starting" });
      } else if (action === "stop") {
        await fn?.StopInstance?.(instanceId);
        updateInstance(instanceId, { status: "stopping" });
      } else if (action === "restart") {
        await fn?.RestartInstance?.(instanceId);
        updateInstance(instanceId, { status: "starting" });
      }
    } catch (err) {
      console.error(`${action} failed:`, err);
    } finally {
      setActionLoading(null);
    }
  };

  const isRunning = instance.status === "running";
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
                {instance.current_map} &bull; {instance.game_mode}
              </CardDescription>
            </div>
            <div className="flex items-center gap-2">
              <span className={cn("h-2.5 w-2.5 shrink-0 rounded-full animate-pulse", getStatusColor(instance.status))} />
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
              <span>{isRunning ? metrics.players : 0} / {instance.max_players}</span>
            </div>
            <div className="flex justify-between gap-2">
              <span className="text-muted-foreground">Game mode</span>
              <span className="capitalize">{instance.game_mode}</span>
            </div>
          </div>
          <div className="flex flex-wrap gap-2 pt-2">
            <Button size="sm" variant="outline" onClick={() => handleAction("start")}
              disabled={isRunning || instance.status === "starting" || !!actionLoading}>
              <Play className="mr-1.5 h-4 w-4" />
              {actionLoading === "start" ? "Starting..." : "Start"}
            </Button>
            <Button size="sm" variant="outline" onClick={() => handleAction("stop")}
              disabled={instance.status === "stopped" || instance.status === "stopping" || !!actionLoading}>
              <Square className="mr-1.5 h-4 w-4" />
              {actionLoading === "stop" ? "Stopping..." : "Stop"}
            </Button>
            <Button size="sm" variant="outline" onClick={() => handleAction("restart")}
              disabled={!isRunning || !!actionLoading}>
              <RotateCw className="mr-1.5 h-4 w-4" />
              {actionLoading === "restart" ? "Restarting..." : "Restart"}
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Live stats */}
      <div className="grid gap-4 sm:grid-cols-2">
        <Card className="border-border">
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
              <Clock className="h-4 w-4" /> Uptime
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-semibold">{isRunning ? formatUptime(uptime) : "Offline"}</p>
          </CardContent>
        </Card>
        <Card className="border-border">
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
              <Users className="h-4 w-4" /> Players Online
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-semibold">{isRunning ? metrics.players : 0}</p>
            <p className="text-xs text-muted-foreground">of {instance.max_players} max</p>
          </CardContent>
        </Card>
      </div>

      {/* Metric cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card className="border-border">
          <CardHeader className="pb-1">
            <CardTitle className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
              <Cpu className="h-4 w-4" /> CPU
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-xl font-semibold">{isRunning ? `${metrics.cpu_pct.toFixed(1)}%` : "—"}</p>
          </CardContent>
        </Card>
        <Card className="border-border">
          <CardHeader className="pb-1">
            <CardTitle className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
              <HardDrive className="h-4 w-4" /> RAM
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-xl font-semibold">{isRunning ? `${metrics.ram_mb.toFixed(0)} MB` : "—"}</p>
          </CardContent>
        </Card>
        <Card className="border-border">
          <CardHeader className="pb-1">
            <CardTitle className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
              <Activity className="h-4 w-4" /> Tick Rate
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-xl font-semibold">{isRunning && metrics.tick_rate > 0 ? `${metrics.tick_rate.toFixed(0)} Hz` : "—"}</p>
          </CardContent>
        </Card>
        <Card className="border-border">
          <CardHeader className="pb-1">
            <CardTitle className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
              <Wifi className="h-4 w-4" /> Network
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-xl font-semibold">{isRunning ? `${metrics.net_in_kbps.toFixed(0)}` : "—"}<span className="text-sm text-muted-foreground"> / {isRunning ? `${metrics.net_out_kbps.toFixed(0)}` : "—"} KB/s</span></p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
