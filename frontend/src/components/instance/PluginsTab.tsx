import { useEffect, useState } from "react";
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui";
import { cn } from "@/lib/utils";
import type { PluginInfo } from "@/types";
import { Puzzle, Download, Check } from "lucide-react";

const AVAILABLE_PLUGINS = [
  { name: "Metamod:Source", installName: "metamod", description: "Metamod plugin loader" },
  { name: "CounterStrikeSharp", installName: "counterstrikesharp", description: "C# plugin framework" },
  { name: "WeaponPaints", installName: "weaponpaints", description: "Weapon skin plugin" },
];

// Mock when Wails not available
const MOCK_PLUGINS: PluginInfo[] = [
  { name: "metamod", installed: true, version: "2.0", path: "", enabled: true },
  { name: "counterstrikesharp", installed: false, version: "", path: "", enabled: false },
];

interface PluginsTabProps {
  instanceId: string;
}

export function PluginsTab({ instanceId }: PluginsTabProps) {
  const [plugins, setPlugins] = useState<PluginInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [installing, setInstalling] = useState<string | null>(null);

  const hasWails = typeof window !== "undefined" && !!(window as any).go?.main?.App;

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      try {
        if (hasWails) {
          const list = await (window as any).go?.main?.App.GetPlugins?.(instanceId) ?? [];
          setPlugins(Array.isArray(list) ? list : []);
        } else {
          setPlugins(MOCK_PLUGINS);
        }
      } catch {
        setPlugins([]);
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [instanceId, hasWails]);

  const isInstalled = (installName: string) => {
    const found = plugins.find((p) => p.name.toLowerCase() === installName.toLowerCase());
    return !!found?.installed;
  };

  const getPluginInfo = (installName: string): PluginInfo | undefined => {
    return plugins.find((p) => p.name.toLowerCase() === installName.toLowerCase());
  };

  const handleInstall = async (installName: string) => {
    setInstalling(installName);
    try {
      if (hasWails) {
        await (window as any).go?.main?.App?.InstallPlugin?.(instanceId, installName);
        const list = await (window as any).go?.main?.App.GetPlugins?.(instanceId) ?? [];
        setPlugins(Array.isArray(list) ? list : []);
      }
    } catch (e) {
      console.error(e);
    } finally {
      setInstalling(null);
    }
  };

  const installedPlugins = plugins.filter((p) => p.installed);

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Puzzle className="h-5 w-5" />
            Available Plugins
          </CardTitle>
          <CardDescription>Install plugins to extend server functionality</CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <p className="py-8 text-center text-muted-foreground">Loading...</p>
          ) : (
            <div className="space-y-3">
              {AVAILABLE_PLUGINS.map((p) => {
                const info = getPluginInfo(p.installName);
                const installed = isInstalled(p.installName);
                return (
                  <div
                    key={p.installName}
                    className={cn(
                      "flex flex-wrap items-center justify-between gap-4 rounded-lg border border-border bg-muted/20 p-4",
                      "sm:flex-nowrap"
                    )}
                  >
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-medium">{p.name}</span>
                        {installed && (
                          <span className="flex items-center gap-1 rounded bg-emerald-500/20 px-2 py-0.5 text-xs text-emerald-400">
                            <Check className="h-3 w-3" />
                            Installed
                          </span>
                        )}
                      </div>
                      <p className="mt-0.5 text-sm text-muted-foreground">{p.description}</p>
                      {info?.version && (
                        <p className="mt-1 text-xs text-muted-foreground">Version: {info.version}</p>
                      )}
                    </div>
                    <Button
                      size="sm"
                      variant={installed ? "outline" : "default"}
                      onClick={() => handleInstall(p.installName)}
                      disabled={installing !== null}
                    >
                      <Download className="mr-1.5 h-4 w-4" />
                      {installing === p.installName ? "Installing..." : installed ? "Update" : "Install"}
                    </Button>
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Installed Plugins</CardTitle>
          <CardDescription>Plugins currently installed on this instance</CardDescription>
        </CardHeader>
        <CardContent>
          {installedPlugins.length === 0 ? (
            <p className="py-6 text-center text-sm text-muted-foreground">No plugins installed</p>
          ) : (
            <div className="space-y-2">
              {installedPlugins.map((p) => (
                <div
                  key={p.name}
                  className="flex items-center justify-between rounded-md border border-border bg-muted/20 px-4 py-3"
                >
                  <span className="font-medium capitalize">{p.name.replace(/([A-Z])/g, " $1").trim()}</span>
                  <span className="text-sm text-muted-foreground">{p.version || "—"}</span>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
