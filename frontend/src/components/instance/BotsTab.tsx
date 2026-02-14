import { useEffect, useState, useCallback } from "react";
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Label,
  Select,
  Slider,
} from "@/components/ui";
import { cn } from "@/lib/utils";
import type { BotConfig } from "@/types";
import { Bot } from "lucide-react";

const QUOTA_MODES = [
  { value: "normal", label: "Normal" },
  { value: "fill", label: "Fill" },
  { value: "match", label: "Match" },
];

const DIFFICULTY_LABELS = ["Easy", "Normal", "Hard", "Expert"];

const DEFAULT_BOT_CONFIG: BotConfig = {
  quota: 0,
  quota_mode: "normal",
  difficulty: 1,
};

interface BotsTabProps {
  instanceId: string;
}

export function BotsTab({ instanceId }: BotsTabProps) {
  const [config, setConfig] = useState<BotConfig>(DEFAULT_BOT_CONFIG);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  const loadConfig = useCallback(async () => {
    setLoading(true);
    try {
      const cfg = await (window as any).go?.main?.App?.GetBotConfig?.(instanceId);
      if (cfg && typeof cfg === "object") {
        setConfig({
          quota: typeof cfg.quota === "number" ? cfg.quota : (typeof cfg.Quota === "number" ? cfg.Quota : 0),
          quota_mode: cfg.quota_mode ?? cfg.QuotaMode ?? "normal",
          difficulty: typeof cfg.difficulty === "number" ? cfg.difficulty : (typeof cfg.Difficulty === "number" ? cfg.Difficulty : 1),
        });
      }
      setLoadError(null);
    } catch {
      setLoadError("Could not fetch bot config. Server may not be running.");
    } finally {
      setLoading(false);
    }
  }, [instanceId]);

  useEffect(() => {
    loadConfig();
  }, [loadConfig]);

  const handleApply = async () => {
    setSaving(true);
    try {
      await (window as any).go?.main?.App?.UpdateBotConfig?.(instanceId, config);
    } catch (e) {
      console.error(e);
    } finally {
      setSaving(false);
    }
  };

  if (loadError) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center py-12 gap-3">
          <Bot className="h-12 w-12 text-muted-foreground/50" />
          <p className="text-muted-foreground text-center">{loadError}</p>
          <Button variant="outline" size="sm" onClick={() => { setLoadError(null); loadConfig(); }}>
            Retry
          </Button>
        </CardContent>
      </Card>
    );
  }

  if (loading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-12">
          <p className="text-muted-foreground">Loading bot config...</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Bot className="h-5 w-5" />
          Bot Configuration
        </CardTitle>
        <CardDescription>Configure bots for this server instance</CardDescription>
      </CardHeader>
      <CardContent className="space-y-8">
        {/* Bot Quota */}
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <Label>Bot Quota (0–64)</Label>
            <span className="font-mono text-sm text-muted-foreground">{config.quota}</span>
          </div>
          <div className="flex items-center gap-4">
            <Slider
              min={0}
              max={64}
              value={config.quota}
              onChange={(e) => setConfig((c) => ({ ...c, quota: parseInt(e.target.value, 10) || 0 }))}
              className="flex-1"
            />
            <span className="w-12 text-right text-sm tabular-nums">{config.quota}</span>
          </div>
        </div>

        {/* Quota Mode */}
        <div className="space-y-2">
          <Label>Quota Mode</Label>
          <Select
            value={config.quota_mode}
            onChange={(e) => setConfig((c) => ({ ...c, quota_mode: e.target.value }))}
          >
            {QUOTA_MODES.map((m) => (
              <option key={m.value} value={m.value}>
                {m.label}
              </option>
            ))}
          </Select>
          <p className="text-xs text-muted-foreground">
            {config.quota_mode === "fill" && "Bots fill empty slots until quota reached"}
            {config.quota_mode === "match" && "Bots match player count per team"}
            {config.quota_mode === "normal" && "Standard bot behavior"}
          </p>
        </div>

        {/* Difficulty */}
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <Label>Difficulty</Label>
            <span className="text-sm font-medium">{DIFFICULTY_LABELS[config.difficulty] ?? config.difficulty}</span>
          </div>
          <div className="flex items-center gap-4">
            <Slider
              min={0}
              max={3}
              step={1}
              value={config.difficulty}
              onChange={(e) => setConfig((c) => ({ ...c, difficulty: parseInt(e.target.value, 10) || 0 }))}
              className="flex-1"
            />
          </div>
          <div className="flex justify-between text-xs text-muted-foreground">
            {DIFFICULTY_LABELS.map((label, i) => (
              <span key={label}>{label}</span>
            ))}
          </div>
        </div>

        <Button onClick={handleApply} disabled={saving}>
          {saving ? "Applying..." : "Apply"}
        </Button>
      </CardContent>
    </Card>
  );
}
