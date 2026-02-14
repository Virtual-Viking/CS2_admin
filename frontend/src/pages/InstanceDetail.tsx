import { useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import {
  Play,
  Square,
  RotateCw,
  ArrowLeft,
  Server,
  Terminal,
  Settings,
  Map,
  Users,
  Bot,
  Palette,
  BarChart3,
  Puzzle,
  HardDrive,
  FileText,
  Calendar,
  Activity,
  Zap,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ConsoleTab } from "@/components/instance/ConsoleTab";
import { OverviewTab } from "@/components/instance/OverviewTab";
import {
  ConfigTab,
  MapsTab,
  PlayersTab,
  BotsTab,
  SkinsTab,
  StatsTab,
  PluginsTab,
  MonitoringTab,
  BenchmarkTab,
  BackupsTab,
  FilesTab,
  SchedulerTab,
} from "@/components/instance";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app-store";
import type { ServerInstance } from "@/types";

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

const TAB_CONFIG: { id: string; label: string; icon: React.ReactNode }[] = [
  { id: "overview", label: "Overview", icon: <Server className="h-4 w-4" /> },
  { id: "console", label: "Console", icon: <Terminal className="h-4 w-4" /> },
  { id: "config", label: "Config", icon: <Settings className="h-4 w-4" /> },
  { id: "maps", label: "Maps", icon: <Map className="h-4 w-4" /> },
  { id: "players", label: "Players", icon: <Users className="h-4 w-4" /> },
  { id: "bots", label: "Bots", icon: <Bot className="h-4 w-4" /> },
  { id: "skins", label: "Skins", icon: <Palette className="h-4 w-4" /> },
  { id: "stats", label: "Stats", icon: <BarChart3 className="h-4 w-4" /> },
  { id: "plugins", label: "Plugins", icon: <Puzzle className="h-4 w-4" /> },
  { id: "monitoring", label: "Monitoring", icon: <Activity className="h-4 w-4" /> },
  { id: "benchmark", label: "Benchmark", icon: <Zap className="h-4 w-4" /> },
  { id: "backups", label: "Backups", icon: <HardDrive className="h-4 w-4" /> },
  { id: "files", label: "Files", icon: <FileText className="h-4 w-4" /> },
  { id: "schedule", label: "Schedule", icon: <Calendar className="h-4 w-4" /> },
];

export function InstanceDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const instances = useAppStore((s) => s.instances);
  const updateInstance = useAppStore((s) => s.updateInstance);

  const instance = instances.find((i) => i.id === id);

  // Subscribe to status updates from backend
  useEffect(() => {
    if (!id) return;
    const eventName = `status:${id}`;
    const cb = (status: unknown) => {
      updateInstance(id, { status: String(status ?? "").toLowerCase() as ServerInstance["status"] });
    };
    try {
      (window as any).runtime?.EventsOn?.(eventName, cb);
    } catch {
      // no-op
    }
    return () => {
      try {
        (window as any).runtime?.EventsOff?.(eventName);
      } catch {
        // no-op
      }
    };
  }, [id, updateInstance]);

  if (!id) {
    return (
      <div className="flex flex-col items-center justify-center p-12">
        <p className="text-muted-foreground">No instance selected</p>
        <Button variant="link" onClick={() => navigate("/")}>
          Back to Dashboard
        </Button>
      </div>
    );
  }

  if (!instance) {
    return (
      <div className="flex flex-col items-center justify-center p-12">
        <Server className="mb-4 h-16 w-16 text-muted-foreground/50" />
        <p className="mb-2 text-lg font-medium">Instance not found</p>
        <p className="mb-4 text-sm text-muted-foreground">
          Instance &quot;{id}&quot; may have been removed or doesn&apos;t exist.
        </p>
        <Button onClick={() => navigate("/")}>
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to Dashboard
        </Button>
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="border-b border-border bg-card/50 px-6 py-4">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => navigate("/")}
            className="shrink-0"
          >
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-3">
              <h1 className="truncate text-xl font-bold">{instance.name}</h1>
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
            <p className="mt-0.5 text-sm text-muted-foreground">
              {instance.current_map} • {instance.game_mode} •{" "}
              {instance.max_players} slots
            </p>
          </div>
          <div className="flex shrink-0 gap-2">
            <Button
              size="sm"
              variant="outline"
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
              disabled={instance.status !== "running"}
            >
              <RotateCw className="mr-1.5 h-4 w-4" />
              Restart
            </Button>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex-1 overflow-auto p-6">
        <Tabs defaultValue="overview" className="w-full">
          <TabsList className="mb-4 flex h-auto flex-wrap gap-1 bg-transparent p-0">
            {TAB_CONFIG.map((tab) => (
              <TabsTrigger
                key={tab.id}
                value={tab.id}
                className="gap-1.5 data-[state=active]:bg-muted"
              >
                {tab.icon}
                {tab.label}
              </TabsTrigger>
            ))}
          </TabsList>

          <TabsContent value="overview">
            <OverviewTab instanceId={id!} />
          </TabsContent>
          <TabsContent value="console">
            <ConsoleTab instanceId={id!} />
          </TabsContent>
          {TAB_CONFIG.filter((t) => !["overview", "console"].includes(t.id)).map((tab) => (
            <TabsContent key={tab.id} value={tab.id}>
              {tab.id === "config" && <ConfigTab instanceId={id!} />}
              {tab.id === "maps" && <MapsTab instanceId={id!} />}
              {tab.id === "players" && <PlayersTab instanceId={id!} />}
              {tab.id === "bots" && <BotsTab instanceId={id!} />}
              {tab.id === "skins" && <SkinsTab instanceId={id!} />}
              {tab.id === "stats" && <StatsTab instanceId={id!} />}
              {tab.id === "plugins" && <PluginsTab instanceId={id!} />}
              {tab.id === "monitoring" && <MonitoringTab instanceId={id!} />}
              {tab.id === "benchmark" && <BenchmarkTab instanceId={id!} />}
              {tab.id === "backups" && <BackupsTab instanceId={id!} />}
              {tab.id === "files" && <FilesTab instanceId={id!} />}
              {tab.id === "schedule" && <SchedulerTab instanceId={id!} />}
            </TabsContent>
          ))}
        </Tabs>
      </div>
    </div>
  );
}
