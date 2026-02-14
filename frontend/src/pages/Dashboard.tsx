import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Play, Square, RotateCw, Plus, Server, Trash2 } from "lucide-react";
import { CreateInstanceWizard } from "@/components/wizard/CreateInstanceWizard";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardFooter,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app-store";
import type { ServerInstance } from "@/types";

function getStatusBadgeVariant(
  status: ServerInstance["status"]
): "default" | "secondary" | "destructive" | "outline" {
  switch (status) {
    case "running":
      return "default";
    case "stopped":
    case "crashed":
      return "destructive";
    case "starting":
    case "stopping":
    case "installing":
    case "updating":
      return "secondary";
    default:
      return "outline";
  }
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

function mapInstanceFromApi(r: Record<string, unknown>): ServerInstance {
  return {
    id: String(r.ID ?? r.id ?? ""),
    name: String(r.Name ?? r.name ?? ""),
    port: Number(r.Port ?? r.port ?? 0),
    rcon_port: Number(r.RconPort ?? r.rcon_port ?? r.Port ?? r.port ?? 0),
    status: String(
      r.Status ?? r.status ?? "stopped"
    ).toLowerCase() as ServerInstance["status"],
    game_mode: String(r.GameMode ?? r.game_mode ?? ""),
    max_players: Number(r.MaxPlayers ?? r.max_players ?? 0),
    current_map: String(r.CurrentMap ?? r.current_map ?? ""),
    install_path: String(r.InstallPath ?? r.install_path ?? ""),
    launch_args: String(r.LaunchArgs ?? r.launch_args ?? ""),
    rcon_password: "",
    gslt_token: "",
    auto_restart: Boolean(r.AutoRestart ?? r.auto_restart ?? false),
    auto_start: Boolean(r.AutoStart ?? r.auto_start ?? false),
    created_at: String(r.CreatedAt ?? r.created_at ?? ""),
    updated_at: String(r.UpdatedAt ?? r.updated_at ?? ""),
  };
}

export function Dashboard() {
  const setInstances = useAppStore((s) => s.setInstances);
  const instances = useAppStore((s) => s.instances);
  const updateInstance = useAppStore((s) => s.updateInstance);
  const removeInstance = useAppStore((s) => s.removeInstance);
  const setSelectedInstanceId = useAppStore((s) => s.setSelectedInstanceId);
  const appConfig = useAppStore((s) => s.appConfig);
  const navigate = useNavigate();
  const [wizardOpen, setWizardOpen] = useState(false);
  const [loadingAction, setLoadingAction] = useState<Record<string, string>>(
    {}
  );

  const loadInstances = async () => {
    try {
      const fn = (window as any).go?.main?.App?.GetInstances;
      if (typeof fn === "function") {
        const result = await fn();
        if (Array.isArray(result)) {
          setInstances(result.map((r: any) => mapInstanceFromApi(r)));
        }
      }
    } catch {
      // No instances available
    }
  };

  useEffect(() => {
    loadInstances();
  }, [setInstances]);

  // Listen for status changes on all instances
  useEffect(() => {
    const cleanups: (() => void)[] = [];
    for (const inst of instances) {
      const eventName = `status:${inst.id}`;
      const cb = (status: unknown) => {
        const s = String(status).toLowerCase() as ServerInstance["status"];
        updateInstance(inst.id, { status: s });
      };
      (window as any).runtime?.EventsOn?.(eventName, cb);
      cleanups.push(() =>
        (window as any).runtime?.EventsOff?.(eventName)
      );
    }
    return () => cleanups.forEach((fn) => fn());
  }, [instances.map((i) => i.id).join(",")]);

  const handleInstanceClick = (id: string) => {
    setSelectedInstanceId(id);
    navigate(`/instances/${id}`);
  };

  const handleCreateNew = () => setWizardOpen(true);

  const handleInstanceCreated = (inst: ServerInstance) => {
    setWizardOpen(false);
    setSelectedInstanceId(inst.id);
    navigate(`/instances/${inst.id}`);
  };

  const handleStart = async (e: React.MouseEvent, id: string) => {
    e.stopPropagation();
    setLoadingAction((prev) => ({ ...prev, [id]: "starting" }));
    try {
      const fn = (window as any).go?.main?.App?.StartInstance;
      if (typeof fn === "function") {
        await fn(id);
        updateInstance(id, { status: "running" });
      }
    } catch (err) {
      console.error("Start failed:", err);
    }
    setLoadingAction((prev) => {
      const next = { ...prev };
      delete next[id];
      return next;
    });
  };

  const handleStop = async (e: React.MouseEvent, id: string) => {
    e.stopPropagation();
    setLoadingAction((prev) => ({ ...prev, [id]: "stopping" }));
    try {
      const fn = (window as any).go?.main?.App?.StopInstance;
      if (typeof fn === "function") {
        await fn(id);
        updateInstance(id, { status: "stopped" });
      }
    } catch (err) {
      console.error("Stop failed:", err);
    }
    setLoadingAction((prev) => {
      const next = { ...prev };
      delete next[id];
      return next;
    });
  };

  const handleRestart = async (e: React.MouseEvent, id: string) => {
    e.stopPropagation();
    setLoadingAction((prev) => ({ ...prev, [id]: "restarting" }));
    try {
      const fn = (window as any).go?.main?.App?.RestartInstance;
      if (typeof fn === "function") {
        await fn(id);
        updateInstance(id, { status: "running" });
      }
    } catch (err) {
      console.error("Restart failed:", err);
    }
    setLoadingAction((prev) => {
      const next = { ...prev };
      delete next[id];
      return next;
    });
  };

  const handleDelete = async (e: React.MouseEvent, id: string) => {
    e.stopPropagation();
    const inst = instances.find((i) => i.id === id);
    if (
      !confirm(
        `Delete instance "${inst?.name ?? id}"? This removes the instance from the panel but does NOT delete server files.`
      )
    ) {
      return;
    }
    try {
      const fn = (window as any).go?.main?.App?.DeleteInstance;
      if (typeof fn === "function") {
        await fn(id);
      }
      removeInstance(id);
    } catch (err) {
      alert("Delete failed: " + (err as Error)?.message);
    }
  };

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold tracking-tight">Dashboard</h1>
        <p className="text-muted-foreground">
          Manage your CS2 server instances
        </p>
      </div>

      {instances.length === 0 ? (
        <Card className="border-dashed">
          <CardContent className="flex flex-col items-center justify-center py-16">
            <Server className="mb-4 h-16 w-16 text-muted-foreground/50" />
            <p className="text-lg font-medium text-muted-foreground">
              No instances yet
            </p>
            <p className="mb-4 text-sm text-muted-foreground">
              Create your first CS2 server instance to get started
            </p>
            <Button onClick={handleCreateNew}>
              <Plus className="mr-2 h-4 w-4" />
              Create New Instance
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-3">
          {instances.map((inst) => {
            const loading = loadingAction[inst.id];
            return (
              <Card
                key={inst.id}
                className="cursor-pointer transition-colors hover:border-primary/50"
                onClick={() => handleInstanceClick(inst.id)}
              >
                <CardHeader className="pb-2">
                  <div className="flex items-start justify-between">
                    <CardTitle className="text-lg font-semibold">
                      {inst.name}
                    </CardTitle>
                    <div className="flex items-center gap-1.5">
                      <span
                        className={cn(
                          "h-2 w-2 rounded-full",
                          getStatusColor(inst.status)
                        )}
                      />
                      <Badge variant={getStatusBadgeVariant(inst.status)}>
                        {loading || inst.status}
                      </Badge>
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="pb-3">
                  <div className="space-y-1 text-sm text-muted-foreground">
                    <p>
                      <span className="font-medium text-foreground">Map:</span>{" "}
                      {inst.current_map || "—"}
                    </p>
                    <p>
                      <span className="font-medium text-foreground">
                        Port:
                      </span>{" "}
                      {inst.port}
                    </p>
                    <p>
                      <span className="font-medium text-foreground">
                        Mode:
                      </span>{" "}
                      {inst.game_mode}
                    </p>
                  </div>
                </CardContent>
                <CardFooter className="flex gap-2 pt-0">
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={(e) => handleStart(e, inst.id)}
                    disabled={
                      inst.status === "running" ||
                      inst.status === "starting" ||
                      !!loading
                    }
                  >
                    <Play className="mr-1 h-3.5 w-3.5" />
                    Start
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={(e) => handleStop(e, inst.id)}
                    disabled={
                      inst.status === "stopped" ||
                      inst.status === "stopping" ||
                      !!loading
                    }
                  >
                    <Square className="mr-1 h-3.5 w-3.5" />
                    Stop
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={(e) => handleRestart(e, inst.id)}
                    disabled={inst.status !== "running" || !!loading}
                  >
                    <RotateCw className="mr-1 h-3.5 w-3.5" />
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    className="ml-auto text-destructive hover:text-destructive hover:bg-destructive/10"
                    onClick={(e) => handleDelete(e, inst.id)}
                    disabled={
                      inst.status === "running" ||
                      inst.status === "starting" ||
                      !!loading
                    }
                    title="Delete instance"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </CardFooter>
              </Card>
            );
          })}

          {/* Create new instance card */}
          <Card
            className="cursor-pointer border-dashed border-2 transition-colors hover:border-primary/50 hover:bg-muted/30"
            onClick={handleCreateNew}
          >
            <CardContent className="flex flex-col items-center justify-center py-12">
              <Plus className="mb-3 h-12 w-12 text-muted-foreground" />
              <p className="font-medium text-muted-foreground">
                Create New Instance
              </p>
            </CardContent>
          </Card>
        </div>
      )}

      <CreateInstanceWizard
        open={wizardOpen}
        onOpenChange={setWizardOpen}
        onCreated={(inst) => handleInstanceCreated(inst)}
        defaultInstallPath={appConfig?.default_install_dir}
      />
    </div>
  );
}
