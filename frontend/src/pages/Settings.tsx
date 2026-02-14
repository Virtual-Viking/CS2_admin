import { useEffect, useState } from "react";
import { Moon, Sun, Monitor, RefreshCw, ExternalLink } from "lucide-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { useAppStore } from "@/stores/app-store";
import { cn } from "@/lib/utils";
import { AuditLog } from "@/components/AuditLog";
import type { AppConfig } from "@/types";

const App = (window as any).go?.main?.App;

const THEME_OPTIONS: { value: "dark" | "light" | "system"; label: string; icon: React.ReactNode }[] = [
  { value: "dark", label: "Dark", icon: <Moon className="h-4 w-4" /> },
  { value: "light", label: "Light", icon: <Sun className="h-4 w-4" /> },
  { value: "system", label: "System", icon: <Monitor className="h-4 w-4" /> },
];

export function Settings() {
  const theme = useAppStore((s) => s.theme);
  const setTheme = useAppStore((s) => s.setTheme);
  const setAppConfig = useAppStore((s) => s.setAppConfig);

  const [config, setConfig] = useState<Partial<AppConfig>>({
    steamcmd_path: "",
    default_install_dir: "",
    backup_dir: "",
    theme: "system",
    minimize_to_tray: true,
    start_with_windows: false,
    auto_update: true,
    discord_webhook: "",
  });
  const [version, setVersion] = useState("");
  const [skinDbUpdated, setSkinDbUpdated] = useState("");
  const [saving, setSaving] = useState(false);
  const [testingDiscord, setTestingDiscord] = useState(false);
  const [updatingSkinDb, setUpdatingSkinDb] = useState(false);
  const [showAuditLog, setShowAuditLog] = useState(false);

  useEffect(() => {
    if (!App) return;
    App.GetAppConfig?.().then((cfg: AppConfig | null) => {
      if (cfg) {
        setConfig(cfg);
        setAppConfig(cfg);
      }
    });
    App?.GetVersion?.().then((v: string) => setVersion(v ?? ""));
    App?.GetSkinDatabaseLastUpdated?.().then((s: string) => setSkinDbUpdated(s ?? ""));
  }, [setAppConfig]);

  const handleSave = async () => {
    if (!App?.UpdateAppConfig || !config) return;
    setSaving(true);
    try {
      const cfg: AppConfig = {
        app_data_dir: config.app_data_dir ?? "",
        log_dir: config.log_dir ?? "",
        log_level: config.log_level ?? "info",
        steamcmd_path: config.steamcmd_path ?? "",
        default_install_dir: config.default_install_dir ?? "",
        backup_dir: config.backup_dir ?? "",
        theme: (config.theme as "dark" | "light" | "system") ?? "system",
        minimize_to_tray: config.minimize_to_tray ?? true,
        start_with_windows: config.start_with_windows ?? false,
        auto_update: config.auto_update ?? true,
        discord_webhook: config.discord_webhook ?? "",
      };
      await App.UpdateAppConfig(cfg);
      setTheme((cfg.theme as "dark" | "light" | "system") || "system");
      setAppConfig(cfg);
    } finally {
      setSaving(false);
    }
  };

  const handleTestDiscord = async () => {
    if (!App?.TestDiscordWebhook) return;
    setTestingDiscord(true);
    try {
      await App.TestDiscordWebhook();
    } finally {
      setTestingDiscord(false);
    }
  };

  const handleUpdateSkinDb = async () => {
    if (!App?.UpdateSkinDatabase) return;
    setUpdatingSkinDb(true);
    try {
      await App.UpdateSkinDatabase();
      const s = await App?.GetSkinDatabaseLastUpdated?.();
      setSkinDbUpdated(s ?? "");
    } finally {
      setUpdatingSkinDb(false);
    }
  };

  const formatDate = (s: string) => {
    if (!s) return "Never";
    try {
      const d = new Date(s);
      return isNaN(d.getTime()) ? s : d.toLocaleString();
    } catch {
      return s;
    }
  };

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Settings</h1>
          <p className="text-muted-foreground">Configure application preferences</p>
        </div>
        <Button onClick={handleSave} disabled={saving}>
          {saving ? "Saving..." : "Save changes"}
        </Button>
      </div>

      <div className="max-w-2xl space-y-6">
        <Card>
          <CardHeader>
            <CardTitle>Appearance</CardTitle>
            <CardDescription>
              Choose how CS2 Admin looks. You can select a theme or use the system default.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Label className="mb-3 block text-sm font-medium">Theme</Label>
            <div className="flex gap-2">
              {THEME_OPTIONS.map((opt) => (
                <button
                  key={opt.value}
                  type="button"
                  onClick={() => setConfig((c) => ({ ...c, theme: opt.value }))}
                  className={cn(
                    "flex items-center gap-2 rounded-lg border px-4 py-2.5 text-sm font-medium transition-colors",
                    (config.theme ?? theme) === opt.value
                      ? "border-primary bg-primary/10 text-primary"
                      : "border-border bg-card hover:bg-accent hover:text-accent-foreground"
                  )}
                >
                  {opt.icon}
                  {opt.label}
                </button>
              ))}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Paths</CardTitle>
            <CardDescription>
              Configure SteamCMD and default installation directories.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <Label htmlFor="steamcmd">SteamCMD path</Label>
              <Input
                id="steamcmd"
                value={config.steamcmd_path ?? ""}
                onChange={(e) => setConfig((c) => ({ ...c, steamcmd_path: e.target.value }))}
                placeholder="e.g. C:\SteamCMD"
                className="mt-1"
              />
            </div>
            <div>
              <Label htmlFor="install">Default install directory</Label>
              <Input
                id="install"
                value={config.default_install_dir ?? ""}
                onChange={(e) => setConfig((c) => ({ ...c, default_install_dir: e.target.value }))}
                placeholder="e.g. C:\CS2Servers"
                className="mt-1"
              />
            </div>
            <div>
              <Label htmlFor="backup">Backup directory</Label>
              <Input
                id="backup"
                value={config.backup_dir ?? ""}
                onChange={(e) => setConfig((c) => ({ ...c, backup_dir: e.target.value }))}
                placeholder="e.g. C:\CS2Admin\backups"
                className="mt-1"
              />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Startup</CardTitle>
            <CardDescription>
              Control how the application starts and behaves when closed.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <Label>Start with Windows</Label>
                <p className="text-sm text-muted-foreground">Launch CS2 Admin when Windows starts</p>
              </div>
              <Switch
                checked={config.start_with_windows ?? false}
                onChange={(e) =>
                  setConfig((c) => ({ ...c, start_with_windows: (e.target as HTMLInputElement).checked }))
                }
              />
            </div>
            <div className="flex items-center justify-between">
              <div>
                <Label>Minimize to tray</Label>
                <p className="text-sm text-muted-foreground">Close button minimizes to system tray instead of exiting</p>
              </div>
              <Switch
                checked={config.minimize_to_tray ?? true}
                onChange={(e) =>
                  setConfig((c) => ({ ...c, minimize_to_tray: (e.target as HTMLInputElement).checked }))
                }
              />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Updates</CardTitle>
            <CardDescription>
              Manage automatic updates and view current version.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <Label>Auto-update</Label>
                <p className="text-sm text-muted-foreground">Check for updates automatically</p>
              </div>
              <Switch
                checked={config.auto_update ?? true}
                onChange={(e) =>
                  setConfig((c) => ({ ...c, auto_update: (e.target as HTMLInputElement).checked }))
                }
              />
            </div>
            <div className="flex items-center gap-2">
              <Label className="shrink-0">Current version:</Label>
              <span className="text-sm text-muted-foreground">{version || "—"}</span>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Notifications</CardTitle>
            <CardDescription>
              Discord webhook for server events (start, stop, crash, performance alerts).
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <Label htmlFor="discord">Discord webhook URL</Label>
              <Input
                id="discord"
                type="url"
                value={config.discord_webhook ?? ""}
                onChange={(e) => setConfig((c) => ({ ...c, discord_webhook: e.target.value }))}
                placeholder="https://discord.com/api/webhooks/..."
                className="mt-1"
              />
            </div>
            <Button variant="outline" size="sm" onClick={handleTestDiscord} disabled={testingDiscord || !config.discord_webhook}>
              {testingDiscord ? "Sending..." : "Test notification"}
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Skin Database</CardTitle>
            <CardDescription>
              Update the weapon skin database from Valve&apos;s items_game.txt.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex flex-wrap items-center gap-4">
              <Button
                variant="outline"
                size="sm"
                onClick={handleUpdateSkinDb}
                disabled={updatingSkinDb}
              >
                <RefreshCw className={cn("mr-2 h-4 w-4", updatingSkinDb && "animate-spin")} />
                {updatingSkinDb ? "Updating..." : "Update"}
              </Button>
              <span className="text-sm text-muted-foreground">
                Last updated: {formatDate(skinDbUpdated)}
              </span>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Audit Log</CardTitle>
            <CardDescription>
              View recent administrative actions and events.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button variant="outline" size="sm" onClick={() => setShowAuditLog(true)}>
              <ExternalLink className="mr-2 h-4 w-4" />
              Show audit log
            </Button>
          </CardContent>
        </Card>
      </div>

      {showAuditLog && (
        <AuditLog open={showAuditLog} onOpenChange={setShowAuditLog} />
      )}
    </div>
  );
}
