import { useEffect, useState, useMemo } from "react";
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Input,
  Label,
  Select,
  Switch,
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
  Tooltip,
} from "@/components/ui";
import { cn } from "@/lib/utils";
import type {
  CvarDef,
  GameModePreset,
  ConfigProfile,
} from "@/types";
import { Save, FileCode, LayoutGrid, Search, Wifi, Info } from "lucide-react";

const CATEGORIES = [
  "server",
  "gameplay",
  "network",
  "bots",
  "performance",
] as const;

// Convert cvars map to raw cfg text
function cvarsToRaw(cvars: Record<string, string>): string {
  return Object.entries(cvars)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([k, v]) => (v.includes(" ") || v === "" ? `${k} "${v}"` : `${k} ${v}`))
    .join("\n");
}

// Parse raw cfg text to cvars map
function rawToCvars(raw: string): Record<string, string> {
  const result: Record<string, string> = {};
  raw.split("\n").forEach((line) => {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("//")) return;
    const spaceIdx = trimmed.indexOf(" ");
    if (spaceIdx < 0) {
      result[trimmed] = "";
      return;
    }
    const key = trimmed.slice(0, spaceIdx).trim();
    let value = trimmed.slice(spaceIdx + 1).trim();
    if (value.startsWith('"') && value.endsWith('"')) {
      value = value.slice(1, -1);
    }
    result[key] = value;
  });
  return result;
}

// Mock data for when Wails is not available
const MOCK_CVAR_DB: CvarDef[] = [
  { name: "hostname", type: "string", default: "CS2 Server", description: "Server name", category: "server" },
  { name: "sv_maxplayers", type: "int", default: "10", description: "Max players", category: "server", min: "1", max: "64" },
  { name: "mp_maxrounds", type: "int", default: "24", description: "Max rounds", category: "gameplay", min: "0", max: "999" },
  { name: "sv_maxrate", type: "int", default: "0", description: "Max bandwidth", category: "network", min: "0", max: "786432" },
  { name: "bot_quota", type: "int", default: "0", description: "Number of bots", category: "bots", min: "0", max: "64" },
  { name: "fps_max", type: "int", default: "300", description: "Max framerate", category: "performance", min: "30", max: "1000" },
];

const MOCK_PRESETS: GameModePreset[] = [
  { name: "Competitive", game_type: 0, game_mode: 1, description: "MR12 competitive", cvars: { mp_maxrounds: "24" }, default_map: "de_dust2" },
  { name: "Casual", game_type: 0, game_mode: 0, description: "Casual 10v10", cvars: { mp_maxrounds: "15" }, default_map: "de_dust2" },
];

const MOCK_LAN_CVARS: Record<string, string> = {
  sv_maxrate: "786432",
  sv_minrate: "786432",
  sv_maxupdaterate: "128",
  fps_max: "512",
};

interface ConfigTabProps {
  instanceId: string;
}

export function ConfigTab({ instanceId }: ConfigTabProps) {
  const [mode, setMode] = useState<"visual" | "raw">("visual");
  const [cvars, setCvars] = useState<Record<string, string>>({});
  const [rawText, setRawText] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [profiles, setProfiles] = useState<ConfigProfile[]>([]);
  const [selectedProfile, setSelectedProfile] = useState("");
  const [cvarDb, setCvarDb] = useState<CvarDef[]>([]);
  const [presets, setPresets] = useState<GameModePreset[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const hasWails = typeof window !== "undefined" && !!(window as any).go?.main?.App;

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      try {
        if (hasWails) {
          const app = (window as any).go?.main?.App;
          const [cfgRes, dbRes, presetsRes, profilesRes] = await Promise.all([
            app.GetServerConfig(instanceId).catch(() => ({})),
            app.GetCvarDatabase?.() ?? Promise.resolve(MOCK_CVAR_DB),
            app.GetGameModePresets?.() ?? Promise.resolve(MOCK_PRESETS),
            app.GetConfigProfiles?.(instanceId) ?? Promise.resolve([]),
          ]);
          const cfgMap = typeof cfgRes === "object" && cfgRes !== null ? (cfgRes as Record<string, string>) : {};
          setCvars(cfgMap);
          setRawText(cvarsToRaw(cfgMap));
          setCvarDb(Array.isArray(dbRes) ? dbRes : MOCK_CVAR_DB);
          setPresets(Array.isArray(presetsRes) ? presetsRes : MOCK_PRESETS);
          setProfiles(Array.isArray(profilesRes) ? profilesRes : []);
        } else {
          setCvars({ hostname: "CS2 Server", sv_maxplayers: "10" });
          setRawText('hostname "CS2 Server"\nsv_maxplayers 10');
          setCvarDb(MOCK_CVAR_DB);
          setPresets(MOCK_PRESETS);
          setProfiles([]);
        }
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [instanceId, hasWails]);

  const filteredCvars = useMemo(() => {
    const q = searchQuery.toLowerCase();
    return cvarDb.filter(
      (c) =>
        c.category &&
        c.name.toLowerCase().includes(q) &&
        CATEGORIES.includes(c.category as (typeof CATEGORIES)[number])
    );
  }, [cvarDb, searchQuery]);

  const cvarsByCategory = useMemo(() => {
    const map: Record<string, CvarDef[]> = {};
    filteredCvars.forEach((c) => {
      const cat = c.category || "other";
      if (!map[cat]) map[cat] = [];
      map[cat].push(c);
    });
    return map;
  }, [filteredCvars]);

  const handleApplyLAN = async () => {
    if (hasWails) {
        const app = (window as any).go?.main?.App;
        const lan = await app?.GetLANOptimizedCvars?.().catch(() => MOCK_LAN_CVARS);
      const merged = { ...cvars, ...(lan || MOCK_LAN_CVARS) };
      setCvars(merged);
      setRawText(cvarsToRaw(merged));
    } else {
      const merged = { ...cvars, ...MOCK_LAN_CVARS };
      setCvars(merged);
      setRawText(cvarsToRaw(merged));
    }
  };

  const handlePresetChange = async (presetName: string) => {
    if (!presetName) return;
    if (hasWails) {
      const app = (window as any).go?.main?.App;
      try {
        await app?.ApplyGameModePreset?.(instanceId, presetName);
        const cfg = await app.GetServerConfig(instanceId);
        const cfgMap = typeof cfg === "object" && cfg !== null ? (cfg as Record<string, string>) : {};
        setCvars(cfgMap);
        setRawText(cvarsToRaw(cfgMap));
      } catch {
        const preset = presets.find((p) => p.name === presetName);
        if (preset) {
          const merged = { ...cvars, ...preset.cvars };
          setCvars(merged);
          setRawText(cvarsToRaw(merged));
        }
      }
    } else {
      const preset = presets.find((p) => p.name === presetName);
      if (preset) {
        const merged = { ...cvars, ...preset.cvars };
        setCvars(merged);
        setRawText(cvarsToRaw(merged));
      }
    }
  };

  const handleLoadProfile = async (profileId: string) => {
    setSelectedProfile(profileId);
    if (!profileId || !hasWails) return;
    try {
      const app = (window as any).go?.main?.App;
      const loaded = await app?.LoadConfigProfile?.(profileId);
      if (loaded && typeof loaded === "object") {
        setCvars(loaded as Record<string, string>);
        setRawText(cvarsToRaw(loaded as Record<string, string>));
      }
    } catch {
      /* ignore */
    }
  };

  const handleSaveProfile = async () => {
    const name = prompt("Profile name:");
    if (!name?.trim()) return;
    if (hasWails) {
      const app = (window as any).go?.main?.App;
      try {
        await app?.SaveConfigProfile?.(instanceId, name.trim(), cvars);
        const list = await app?.GetConfigProfiles?.(instanceId);
        setProfiles(Array.isArray(list) ? list : []);
      } catch (e) {
        console.error(e);
      }
    }
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      const toSave = mode === "raw" ? rawToCvars(rawText) : cvars;
      if (hasWails) {
        const app = (window as any).go?.main?.App;
        await app.UpdateServerConfig(instanceId, toSave);
      }
      if (mode === "raw") setCvars(toSave);
      else setRawText(cvarsToRaw(toSave));
    } catch (e) {
      console.error(e);
    } finally {
      setSaving(false);
    }
  };

  const updateCvar = (name: string, value: string) => {
    const next = { ...cvars, [name]: value };
    setCvars(next);
    setRawText(cvarsToRaw(next));
  };

  if (loading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-12">
          <p className="text-muted-foreground">Loading config...</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-4">
      {/* Mode toggle & Profile dropdown */}
      <div className="flex flex-wrap items-center gap-4">
        <div className="flex rounded-lg border border-border bg-muted/30 p-1">
          <button
            type="button"
            onClick={() => setMode("visual")}
            className={cn(
              "flex items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium transition-colors",
              mode === "visual" ? "bg-background shadow-sm" : "text-muted-foreground hover:text-foreground"
            )}
          >
            <LayoutGrid className="h-4 w-4" />
            Visual Editor
          </button>
          <button
            type="button"
            onClick={() => setMode("raw")}
            className={cn(
              "flex items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium transition-colors",
              mode === "raw" ? "bg-background shadow-sm" : "text-muted-foreground hover:text-foreground"
            )}
          >
            <FileCode className="h-4 w-4" />
            Raw Editor
          </button>
        </div>

        <div className="flex items-center gap-2">
          <Label className="text-muted-foreground text-sm">Profile:</Label>
          <Select
            value={selectedProfile}
            onChange={(e) => handleLoadProfile(e.target.value)}
            className="w-48"
          >
            <option value="">— Select —</option>
            {profiles.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </Select>
          <Button variant="outline" size="sm" onClick={handleSaveProfile}>
            Save as Profile
          </Button>
        </div>

        <Button onClick={handleSave} disabled={saving}>
          <Save className="mr-2 h-4 w-4" />
          {saving ? "Saving..." : "Save"}
        </Button>
      </div>

      {mode === "visual" ? (
        <Card>
          <CardHeader className="pb-4">
            <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <CardTitle>Server Configuration</CardTitle>
                <CardDescription>Edit CS2 server cvars by category</CardDescription>
              </div>
              <div className="flex flex-wrap gap-2">
                <div className="relative flex-1 sm:w-64">
                  <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    placeholder="Search cvars..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="pl-9"
                  />
                </div>
                <Button variant="outline" size="sm" onClick={handleApplyLAN}>
                  <Wifi className="mr-1.5 h-4 w-4" />
                  Apply LAN Optimization
                </Button>
                <div className="flex items-center gap-2">
                  <Label className="text-sm text-muted-foreground">Preset:</Label>
                  <Select
                    defaultValue=""
                    onChange={(e) => handlePresetChange(e.target.value)}
                    className="w-40"
                  >
                    <option value="">— None —</option>
                    {presets.map((p) => (
                      <option key={p.name} value={p.name}>
                        {p.name}
                      </option>
                    ))}
                  </Select>
                </div>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <Tabs defaultValue={CATEGORIES[0]} className="w-full">
              <TabsList className="mb-4">
                {CATEGORIES.map((cat) => (
                  <TabsTrigger key={cat} value={cat}>
                    {cat.charAt(0).toUpperCase() + cat.slice(1)}
                  </TabsTrigger>
                ))}
              </TabsList>
              {CATEGORIES.map((cat) => (
                <TabsContent key={cat} value={cat} className="space-y-4">
                  {(cvarsByCategory[cat] || []).map((c) => (
                    <CvarRow
                      key={c.name}
                      def={c}
                      value={cvars[c.name] ?? c.default}
                      onChange={(v) => updateCvar(c.name, v)}
                    />
                  ))}
                  {(!cvarsByCategory[cat] || cvarsByCategory[cat].length === 0) && (
                    <p className="text-sm text-muted-foreground">No cvars in this category</p>
                  )}
                </TabsContent>
              ))}
            </Tabs>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle>Raw server.cfg</CardTitle>
            <CardDescription>Edit config as plain text (key "value" format)</CardDescription>
          </CardHeader>
          <CardContent>
            <textarea
              value={rawText}
              onChange={(e) => setRawText(e.target.value)}
              className="h-96 w-full rounded-md border border-input bg-zinc-950 px-4 py-3 font-mono text-sm text-zinc-100"
              spellCheck={false}
              placeholder="hostname &quot;My Server&quot;&#10;sv_maxplayers 10"
            />
            <div className="mt-4">
              <Button onClick={handleSave} disabled={saving}>
                <Save className="mr-2 h-4 w-4" />
                {saving ? "Saving..." : "Save"}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

function CvarRow({
  def,
  value,
  onChange,
}: {
  def: CvarDef;
  value: string;
  onChange: (v: string) => void;
}) {
  const isBool = def.type === "bool";
  const isInt = def.type === "int";
  const isFloat = def.type === "float";
  const min = def.min != null ? parseFloat(def.min) : undefined;
  const max = def.max != null ? parseFloat(def.max) : undefined;

  return (
    <div className="flex flex-wrap items-center gap-4 rounded-lg border border-border/50 bg-muted/20 p-4">
      <div className="flex min-w-0 flex-1 items-center gap-2">
        <span className="font-mono text-sm font-medium">{def.name}</span>
        <Tooltip content={def.description} side="right">
          <Info className="h-4 w-4 shrink-0 text-muted-foreground" />
        </Tooltip>
      </div>
      <div className="flex items-center gap-2">
        {isBool ? (
          <Switch
            checked={value === "1"}
            onChange={(e) => onChange((e.target as HTMLInputElement).checked ? "1" : "0")}
          />
        ) : isInt || isFloat ? (
          <Input
            type="number"
            value={value}
            onChange={(e) => onChange(e.target.value)}
            min={min}
            max={max}
            step={isFloat ? 0.01 : 1}
            className="w-32"
          />
        ) : (
          <Input
            value={value}
            onChange={(e) => onChange(e.target.value)}
            placeholder={def.default}
            className="min-w-[200px]"
          />
        )}
      </div>
    </div>
  );
}
