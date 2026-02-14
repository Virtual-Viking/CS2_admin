import { useState, useEffect, useCallback } from "react";
import {
  ChevronLeft,
  ChevronRight,
  Check,
  Download,
  SkipForward,
  X,
  FolderOpen,
} from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Slider } from "@/components/ui/slider";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app-store";
import type { ServerInstance, GameModePreset, Progress } from "@/types";

const STEPS = [
  { id: 1, title: "Basic Info", desc: "Server name and install path" },
  { id: 2, title: "Game Mode", desc: "Choose a preset" },
  { id: 3, title: "Network", desc: "Port and capacity" },
  { id: 4, title: "Install", desc: "CS2 server files" },
];

const CS2_MAPS = [
  "de_dust2",
  "de_mirage",
  "de_inferno",
  "de_nuke",
  "de_overpass",
  "de_ancient",
  "de_anubis",
  "de_vertigo",
  "cs_office",
  "cs_italy",
  "de_train",
  "de_cache",
];

function mapInstanceFromApi(r: Record<string, unknown>): ServerInstance {
  const id = String(r.ID ?? r.id ?? "");
  return {
    id,
    name: String(r.Name ?? r.name ?? ""),
    port: Number(r.Port ?? r.port ?? 27015),
    rcon_port: Number(r.RconPort ?? r.rcon_port ?? r.Port ?? r.port ?? 27015),
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
    auto_restart: Boolean(r.AutoRestart ?? r.auto_restart ?? true),
    auto_start: Boolean(r.AutoStart ?? r.auto_start ?? false),
    created_at: String(r.CreatedAt ?? r.created_at ?? ""),
    updated_at: String(r.UpdatedAt ?? r.updated_at ?? ""),
  };
}

export interface CreateInstanceWizardProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: (instance: ServerInstance) => void;
  defaultInstallPath?: string;
}

export function CreateInstanceWizard({
  open,
  onOpenChange,
  onCreated,
  defaultInstallPath = "",
}: CreateInstanceWizardProps) {
  const appConfig = useAppStore((s) => s.appConfig);
  const addInstance = useAppStore((s) => s.addInstance);

  const [step, setStep] = useState(0);
  const [name, setName] = useState("");
  const [installPath, setInstallPath] = useState(
    defaultInstallPath || appConfig?.default_install_dir || ""
  );
  const [rconPassword, setRconPassword] = useState("");
  const [preset, setPreset] = useState<GameModePreset | null>(null);
  const [presets, setPresets] = useState<GameModePreset[]>([]);
  const [port, setPort] = useState(27015);
  const [maxPlayers, setMaxPlayers] = useState(10);
  const [map, setMap] = useState("de_dust2");
  const [createdInstance, setCreatedInstance] =
    useState<ServerInstance | null>(null);
  const [progress, setProgress] = useState<Progress | null>(null);
  const [installError, setInstallError] = useState<string | null>(null);
  const [isCreating, setIsCreating] = useState(false);
  const [isInstalling, setIsInstalling] = useState(false);

  // Load presets
  useEffect(() => {
    const load = async () => {
      try {
        const fn = (window as any).go?.main?.App?.GetGameModePresets;
        if (typeof fn === "function") {
          const result = await fn();
          if (Array.isArray(result)) {
            setPresets(
              result.map((p: any) => ({
                name: p.Name ?? p.name ?? "",
                game_type: Number(p.GameType ?? p.game_type ?? 0),
                game_mode: Number(p.GameMode ?? p.game_mode ?? 0),
                description: p.Description ?? p.description ?? "",
                cvars: p.Cvars ?? p.cvars ?? {},
                default_map: p.DefaultMap ?? p.default_map ?? "de_dust2",
              }))
            );
          }
        }
      } catch {
        // ignore
      }
      // Always ensure fallback presets exist
      setPresets((prev) =>
        prev.length > 0
          ? prev
          : [
              {
                name: "Competitive",
                game_type: 0,
                game_mode: 1,
                description: "5v5 matchmaking-style competitive",
                cvars: {},
                default_map: "de_dust2",
              },
              {
                name: "Casual",
                game_type: 0,
                game_mode: 0,
                description: "Casual 10v10 with relaxed rules",
                cvars: {},
                default_map: "de_dust2",
              },
              {
                name: "Deathmatch",
                game_type: 1,
                game_mode: 2,
                description: "Free-for-all deathmatch",
                cvars: {},
                default_map: "de_dust2",
              },
              {
                name: "Wingman",
                game_type: 0,
                game_mode: 2,
                description: "2v2 competitive",
                cvars: {},
                default_map: "de_inferno",
              },
              {
                name: "Retake",
                game_type: 0,
                game_mode: 0,
                description: "Retake practice",
                cvars: {},
                default_map: "de_dust2",
              },
              {
                name: "Surf",
                game_type: 3,
                game_mode: 0,
                description: "Surf maps with custom physics",
                cvars: {},
                default_map: "surf_beginner",
              },
              {
                name: "KZ (Climb)",
                game_type: 3,
                game_mode: 0,
                description: "KZ/Climb maps",
                cvars: {},
                default_map: "kz_beginner",
              },
            ]
      );
    };
    if (open) load();
  }, [open]);

  // Suggest port
  useEffect(() => {
    const suggest = async () => {
      try {
        const fn = (window as any).go?.main?.App?.GetInstances;
        if (typeof fn === "function") {
          const result = await fn();
          if (Array.isArray(result) && result.length > 0) {
            const ports = result.map(
              (r: any) => Number(r.Port ?? r.port ?? 0)
            );
            const maxP = Math.max(...ports, 27015);
            setPort(maxP + 10);
          }
        }
      } catch {
        // keep default 27015
      }
    };
    if (open && step === 2) suggest();
  }, [open, step]);

  // Update map when preset changes
  useEffect(() => {
    if (preset) setMap(preset.default_map);
  }, [preset]);

  const reset = useCallback(() => {
    setStep(0);
    setName("");
    setInstallPath(defaultInstallPath || appConfig?.default_install_dir || "");
    setRconPassword("");
    setPreset(null);
    setPort(27015);
    setMaxPlayers(10);
    setMap("de_dust2");
    setCreatedInstance(null);
    setProgress(null);
    setInstallError(null);
    setIsCreating(false);
    setIsInstalling(false);
  }, [defaultInstallPath, appConfig?.default_install_dir]);

  const handleClose = useCallback(() => {
    reset();
    onOpenChange(false);
  }, [onOpenChange, reset]);

  const handleBrowse = async () => {
    try {
      // Use Wails native directory dialog
      const fn = (window as any).runtime?.OpenDirectoryDialog;
      if (typeof fn === "function") {
        const result = await fn({
          Title: "Select Install Directory",
        });
        if (result) {
          setInstallPath(result);
        }
      }
    } catch {
      // fallback: do nothing, user types manually
    }
  };

  const handleNext = async () => {
    if (step === 2) {
      // Create instance and move to install step
      setIsCreating(true);
      setInstallError(null);
      try {
        const fn = (window as any).go?.main?.App?.CreateInstance;
        if (typeof fn !== "function")
          throw new Error("CreateInstance not available");

        // Build install path with server name subfolder
        let finalPath = installPath;
        if (name.trim()) {
          const safeName = name.trim().replace(/[^a-zA-Z0-9_-]/g, "_");
          finalPath = installPath.replace(/[/\\]$/, "") + "\\" + safeName;
        }

        const cfg = {
          name,
          install_path: finalPath,
          port: port || undefined,
          max_players: maxPlayers,
          game_mode: preset?.name?.toLowerCase() ?? "competitive",
          map,
          rcon_password: rconPassword || "",
          launch_args: "",
          auto_restart: true,
          auto_start: false,
        };
        const result = await fn(cfg);
        const inst = mapInstanceFromApi(result ?? {});
        setCreatedInstance(inst);
        addInstance(inst);
        setStep(3);
      } catch (err) {
        setInstallError(
          (err as Error)?.message ?? "Failed to create instance"
        );
        setIsCreating(false);
        return;
      }
      setIsCreating(false);
      return;
    }
    setStep((s) => Math.min(s + 1, STEPS.length - 1));
  };

  const handleStartInstall = async () => {
    if (!createdInstance) return;
    setIsInstalling(true);
    setInstallError(null);
    setProgress({ stage: "preparing", percent: 0, message: "Starting SteamCMD..." });
    try {
      const installFn = (window as any).go?.main?.App?.InstallCS2Server;
      if (typeof installFn === "function") {
        installFn(createdInstance.id).catch((err: Error) => {
          setInstallError(err?.message ?? "Install failed");
          setIsInstalling(false);
        });
      } else {
        setInstallError("Install function not available");
        setIsInstalling(false);
      }
    } catch (err) {
      setInstallError((err as Error)?.message ?? "Install failed");
      setIsInstalling(false);
    }
  };

  const handleBack = () => {
    setStep((s) => Math.max(0, s - 1));
  };

  const handleSkipInstall = () => {
    if (createdInstance) {
      onCreated(createdInstance);
      handleClose();
    }
  };

  const handleFinish = () => {
    if (createdInstance) {
      onCreated(createdInstance);
      handleClose();
    }
  };

  // Progress listener for step 4
  useEffect(() => {
    if (step !== 3 || !createdInstance?.id) return;
    const eventName = `progress:${createdInstance.id}`;
    const cb = (data: unknown) => {
      const d = data as Record<string, unknown>;
      const stage = String(d?.Stage ?? d?.stage ?? "").toLowerCase();
      setProgress({
        stage,
        percent: Number(d?.Percent ?? d?.percent ?? 0),
        message: String(d?.Message ?? d?.message ?? ""),
      });
      if (stage === "complete") {
        setIsInstalling(false);
      }
    };
    (window as any).runtime?.EventsOn?.(eventName, cb);
    return () => {
      (window as any).runtime?.EventsOff?.(eventName);
    };
  }, [step, createdInstance?.id]);

  const canNext =
    (step === 0 && name.trim() && installPath.trim()) ||
    (step === 1 && preset) ||
    step === 2 ||
    step === 3;

  const isComplete =
    progress?.stage === "complete" || progress?.percent === 100;
  const hasError = progress?.stage === "error" || !!installError;

  // Don't let overlay click or Escape close the dialog — use the explicit X button
  return (
    <Dialog open={open} onOpenChange={() => {}}>
      <DialogContent
        className="max-w-2xl border-border bg-background"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Close button — always visible */}
        <button
          onClick={handleClose}
          className="absolute right-4 top-4 rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 z-10"
          aria-label="Close"
        >
          <X className="h-4 w-4" />
        </button>

        <DialogHeader>
          <DialogTitle className="text-xl">
            Create New Instance — {STEPS[step]?.title}
          </DialogTitle>
          <DialogDescription>{STEPS[step]?.desc}</DialogDescription>
        </DialogHeader>

        {/* Step indicator */}
        <div className="flex gap-1 rounded-lg bg-muted/50 p-1">
          {STEPS.map((s, i) => (
            <div
              key={s.id}
              className={cn(
                "h-1.5 flex-1 rounded-full transition-colors",
                i <= step ? "bg-primary" : "bg-muted-foreground/30"
              )}
            />
          ))}
        </div>

        {/* Step content */}
        <div className="min-h-[280px]">
          {/* Step 1: Basic Info */}
          {step === 0 && (
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="wizard-name">Server Name</Label>
                <Input
                  id="wizard-name"
                  placeholder="My CS2 Server"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  onFocus={(e) => e.target.select()}
                  className="bg-background"
                  autoFocus
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="wizard-path">Install Directory</Label>
                <div className="flex gap-2">
                  <Input
                    id="wizard-path"
                    placeholder="C:\CS2Servers"
                    value={installPath}
                    onChange={(e) => setInstallPath(e.target.value)}
                    onFocus={(e) => e.target.select()}
                    className="font-mono text-sm bg-background flex-1"
                  />
                  <Button
                    variant="outline"
                    size="default"
                    onClick={handleBrowse}
                    title="Browse for folder"
                  >
                    <FolderOpen className="h-4 w-4" />
                  </Button>
                </div>
                <p className="text-xs text-muted-foreground">
                  A subfolder with the server name will be created here
                </p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="wizard-rcon">
                  RCON Password{" "}
                  <span className="text-muted-foreground font-normal">
                    (optional)
                  </span>
                </Label>
                <Input
                  id="wizard-rcon"
                  type="password"
                  placeholder="Leave empty for auto-generated"
                  value={rconPassword}
                  onChange={(e) => setRconPassword(e.target.value)}
                  onFocus={(e) => e.target.select()}
                  className="bg-background"
                />
              </div>
            </div>
          )}

          {/* Step 2: Game Mode */}
          {step === 1 && (
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
              {presets.map((p) => (
                <Card
                  key={p.name}
                  className={cn(
                    "cursor-pointer transition-all hover:border-primary/50",
                    preset?.name === p.name
                      ? "border-primary ring-2 ring-primary/30"
                      : "border-border"
                  )}
                  onClick={() => setPreset(p)}
                >
                  <CardContent className="p-4">
                    <p className="font-semibold">{p.name}</p>
                    <p className="mt-1 text-xs text-muted-foreground line-clamp-2">
                      {p.description}
                    </p>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}

          {/* Step 3: Network — port and max players only, no map selection */}
          {step === 2 && (
            <div className="space-y-6">
              <div className="space-y-2">
                <Label>Port</Label>
                <Input
                  type="number"
                  min={1024}
                  max={65535}
                  value={port || ""}
                  onChange={(e) =>
                    setPort(parseInt(e.target.value, 10) || 27015)
                  }
                  onFocus={(e) => e.target.select()}
                  className="bg-background"
                />
                <p className="text-xs text-muted-foreground">
                  Default: 27015. Each server needs a unique port.
                </p>
              </div>
              <div className="space-y-2">
                <Label>Max Players: {maxPlayers}</Label>
                <Slider
                  min={2}
                  max={64}
                  step={1}
                  value={maxPlayers}
                  onChange={(e) =>
                    setMaxPlayers(
                      parseInt((e.target as HTMLInputElement).value, 10) || 2
                    )
                  }
                  className="py-2"
                />
              </div>
              <div className="space-y-2">
                <Label>Default Map</Label>
                <select
                  value={map}
                  onChange={(e) => setMap(e.target.value)}
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                >
                  {CS2_MAPS.map((m) => (
                    <option key={m} value={m}>
                      {m}
                    </option>
                  ))}
                </select>
                <p className="text-xs text-muted-foreground">
                  You can manage maps and download workshop maps from the Maps
                  tab after creation.
                </p>
              </div>
            </div>
          )}

          {/* Step 4: Install */}
          {step === 3 && (
            <div className="space-y-4">
              {createdInstance && (
                <>
                  <div className="rounded-lg border border-border bg-muted/30 p-4">
                    <p className="text-sm font-medium">
                      {createdInstance.name}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {createdInstance.install_path}
                    </p>
                    <p className="text-xs text-muted-foreground mt-1">
                      Mode: {createdInstance.game_mode} | Port:{" "}
                      {createdInstance.port} | Players:{" "}
                      {createdInstance.max_players}
                    </p>
                  </div>

                  {!isInstalling && !progress && !installError && (
                    <div className="flex flex-col items-center justify-center py-8 gap-4">
                      <p className="text-sm text-muted-foreground text-center">
                        Instance created. Click below to download and install
                        CS2 dedicated server files (~35 GB).
                      </p>
                      <div className="flex gap-3">
                        <Button onClick={handleStartInstall}>
                          <Download className="mr-2 h-4 w-4" />
                          Install CS2 Server
                        </Button>
                        <Button variant="outline" onClick={handleSkipInstall}>
                          <SkipForward className="mr-2 h-4 w-4" />
                          Skip — I&apos;ll install later
                        </Button>
                      </div>
                    </div>
                  )}

                  {(isInstalling || progress) && (
                    <div className="space-y-2">
                      <div className="flex justify-between text-sm">
                        <span className="text-muted-foreground capitalize">
                          {progress?.stage || "Preparing..."}
                        </span>
                        <span>{Math.round(progress?.percent ?? 0)}%</span>
                      </div>
                      <div className="h-2.5 w-full overflow-hidden rounded-full bg-muted">
                        <div
                          className={cn(
                            "h-full transition-all duration-300",
                            isComplete ? "bg-emerald-500" : "bg-primary"
                          )}
                          style={{
                            width: `${Math.max(progress?.percent ?? 0, isInstalling && !progress?.percent ? 1 : 0)}%`,
                          }}
                        />
                      </div>
                      {progress?.message && (
                        <p className="font-mono text-xs text-muted-foreground truncate">
                          {progress.message}
                        </p>
                      )}
                    </div>
                  )}

                  {installError && (
                    <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-3">
                      <p className="text-sm text-destructive">{installError}</p>
                      <Button
                        variant="outline"
                        size="sm"
                        className="mt-2"
                        onClick={handleStartInstall}
                      >
                        Retry
                      </Button>
                    </div>
                  )}
                </>
              )}
            </div>
          )}
        </div>

        {/* Footer */}
        <DialogFooter className="flex-row justify-between sm:justify-between">
          <div>
            {step > 0 && step < 3 && (
              <Button variant="ghost" onClick={handleBack}>
                <ChevronLeft className="mr-2 h-4 w-4" />
                Back
              </Button>
            )}
            {step === 0 && (
              <Button variant="ghost" onClick={handleClose}>
                Cancel
              </Button>
            )}
          </div>
          <div className="flex gap-2">
            {step === 3 && createdInstance && (isInstalling || isComplete) && (
              <Button
                onClick={handleFinish}
                disabled={isInstalling && !isComplete}
              >
                {isComplete ? (
                  <>
                    <Check className="mr-2 h-4 w-4" />
                    Finish
                  </>
                ) : (
                  <>
                    <SkipForward className="mr-2 h-4 w-4" />
                    Finish Anyway
                  </>
                )}
              </Button>
            )}
            {step < 3 && (
              <Button onClick={handleNext} disabled={!canNext || isCreating}>
                {isCreating ? (
                  <span className="flex items-center gap-2">
                    <span className="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
                    Creating...
                  </span>
                ) : step === 2 ? (
                  <>
                    Create Instance
                    <ChevronRight className="ml-2 h-4 w-4" />
                  </>
                ) : (
                  <>
                    Next
                    <ChevronRight className="ml-2 h-4 w-4" />
                  </>
                )}
              </Button>
            )}
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
