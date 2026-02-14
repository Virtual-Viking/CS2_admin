import { useEffect, useState } from "react";
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Input,
  ScrollArea,
} from "@/components/ui";
import { cn } from "@/lib/utils";
import type { MapInfo } from "@/types";
import { Map as MapIcon, Download, ChevronUp, ChevronDown } from "lucide-react";

const MAP_IMAGES: Record<string, string> = {
  de_dust2: "https://cdn.cloudflare.steamstatic.com/apps/csgo/images/cs2/maps/de_dust2.jpg",
  de_mirage: "https://cdn.cloudflare.steamstatic.com/apps/csgo/images/cs2/maps/de_mirage.jpg",
  de_inferno: "https://cdn.cloudflare.steamstatic.com/apps/csgo/images/cs2/maps/de_inferno.jpg",
  de_nuke: "https://cdn.cloudflare.steamstatic.com/apps/csgo/images/cs2/maps/de_nuke.jpg",
  de_overpass: "https://cdn.cloudflare.steamstatic.com/apps/csgo/images/cs2/maps/de_overpass.jpg",
  de_ancient: "https://cdn.cloudflare.steamstatic.com/apps/csgo/images/cs2/maps/de_ancient.jpg",
  de_anubis: "https://cdn.cloudflare.steamstatic.com/apps/csgo/images/cs2/maps/de_anubis.jpg",
  de_vertigo: "https://cdn.cloudflare.steamstatic.com/apps/csgo/images/cs2/maps/de_vertigo.jpg",
  cs_office: "https://cdn.cloudflare.steamstatic.com/apps/csgo/images/cs2/maps/cs_office.jpg",
  cs_italy: "https://cdn.cloudflare.steamstatic.com/apps/csgo/images/cs2/maps/cs_italy.jpg",
  de_train: "https://cdn.cloudflare.steamstatic.com/apps/csgo/images/cs2/maps/de_train.jpg",
  de_cache: "https://cdn.cloudflare.steamstatic.com/apps/csgo/images/cs2/maps/de_cache.jpg",
};

// Mock data when Wails not available
const MOCK_MAPS: MapInfo[] = [
  { name: "de_dust2", file_name: "de_dust2.bsp", size_bytes: 0 },
  { name: "de_inferno", file_name: "de_inferno.bsp", size_bytes: 0 },
  { name: "de_mirage", file_name: "de_mirage.bsp", size_bytes: 0 },
  { name: "de_nuke", file_name: "de_nuke.bsp", size_bytes: 0 },
  { name: "de_overpass", file_name: "de_overpass.bsp", size_bytes: 0 },
];

const MOCK_ROTATION = ["de_dust2", "de_inferno", "de_mirage"];

function formatSize(bytes: number): string {
  if (bytes === 0) return "—";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

interface MapsTabProps {
  instanceId: string;
}

export function MapsTab({ instanceId }: MapsTabProps) {
  const [maps, setMaps] = useState<MapInfo[]>([]);
  const [rotation, setRotation] = useState<string[]>([]);
  const [workshopId, setWorkshopId] = useState("");
  const [downloadProgress, setDownloadProgress] = useState<{ percent: number; message: string } | null>(null);
  const [isDownloading, setIsDownloading] = useState(false);
  const [loading, setLoading] = useState(true);
  const [changingMap, setChangingMap] = useState<string | null>(null);

  const hasWails = typeof window !== "undefined" && !!(window as any).go?.main?.App;

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      try {
        if (hasWails) {
          const app = (window as any).go?.main?.App;
          const [mapsRes, rotRes] = await Promise.all([
            app.GetInstalledMaps(instanceId).catch(() => []),
            app.GetMapRotation(instanceId).catch(() => []),
          ]);
          setMaps(Array.isArray(mapsRes) ? mapsRes : []);
          setRotation(Array.isArray(rotRes) ? rotRes : []);
        } else {
          setMaps(MOCK_MAPS);
          setRotation(MOCK_ROTATION);
        }
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [instanceId, hasWails]);

  useEffect(() => {
    if (!hasWails || !isDownloading) return;
    const unsub = (window as any).runtime?.EventsOn?.("progress:" + instanceId, (p: { percent?: number; message?: string }) => {
      setDownloadProgress({ percent: p?.percent ?? 0, message: p?.message ?? "" });
    });
    return () => {
      if (typeof unsub === "function") unsub();
    };
  }, [instanceId, hasWails, isDownloading]);

  const handleChangeMap = async (mapName: string) => {
    setChangingMap(mapName);
    try {
      if (hasWails) {
        const app = (window as any).go?.main?.App;
        await app.ChangeMap(instanceId, mapName);
      }
    } catch (e) {
      console.error(e);
    } finally {
      setChangingMap(null);
    }
  };

  const handleMoveInRotation = (index: number, direction: "up" | "down") => {
    const newRot = [...rotation];
    const swap = direction === "up" ? index - 1 : index + 1;
    if (swap < 0 || swap >= newRot.length) return;
    [newRot[index], newRot[swap]] = [newRot[swap], newRot[index]];
    setRotation(newRot);
    if (hasWails) {
      (window as any).go?.main?.App.SetMapRotation(instanceId, newRot).catch(console.error);
    }
  };

  const handleSaveRotation = async () => {
    if (hasWails) {
      try {
        await (window as any).go?.main?.App.SetMapRotation(instanceId, rotation);
      } catch (e) {
        console.error(e);
      }
    }
  };

  const handleDownloadWorkshop = async () => {
    const id = parseInt(workshopId, 10);
    if (isNaN(id) || id <= 0) return;
    setIsDownloading(true);
    setDownloadProgress({ percent: 0, message: "Starting..." });
    try {
      if (hasWails) {
        const app = (window as any).go?.main?.App;
        await app.DownloadWorkshopMap(instanceId, id);
        const [mapsRes, rotRes] = await Promise.all([
          app.GetInstalledMaps(instanceId),
          app.GetMapRotation(instanceId),
        ]);
        setMaps(Array.isArray(mapsRes) ? mapsRes : []);
        setRotation(Array.isArray(rotRes) ? rotRes : rotation);
      }
    } catch (e) {
      console.error(e);
    } finally {
      setIsDownloading(false);
      setDownloadProgress(null);
    }
  };

  const addToRotation = (mapName: string) => {
    if (rotation.includes(mapName)) return;
    const next = [...rotation, mapName];
    setRotation(next);
    if (hasWails) (window as any).go?.main?.App.SetMapRotation(instanceId, next).catch(console.error);
  };

  const removeFromRotation = (mapName: string) => {
    const next = rotation.filter((m) => m !== mapName);
    setRotation(next);
    if (hasWails) (window as any).go?.main?.App.SetMapRotation(instanceId, next).catch(console.error);
  };

  if (loading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-12">
          <p className="text-muted-foreground">Loading maps...</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* Installed Maps Grid */}
      <Card>
        <CardHeader>
          <CardTitle>Installed Maps</CardTitle>
          <CardDescription>Maps available on this server instance</CardDescription>
        </CardHeader>
        <CardContent>
          <ScrollArea className="h-[400px]">
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
              {maps.map((m) => (
                <div
                  key={m.name}
                  className="group flex flex-col overflow-hidden rounded-lg border border-border bg-muted/20 transition-colors hover:border-primary/50"
                >
                  <div className="relative aspect-[16/10] w-full bg-muted overflow-hidden">
                    {MAP_IMAGES[m.name] ? (
                      <img
                        src={MAP_IMAGES[m.name]}
                        alt={m.name}
                        className="h-full w-full object-cover transition-transform group-hover:scale-105"
                        onError={(e) => { (e.target as HTMLImageElement).style.display = 'none'; }}
                      />
                    ) : (
                      <div className="flex h-full items-center justify-center">
                        <MapIcon className="h-8 w-8 text-muted-foreground/40" />
                      </div>
                    )}
                    <div className="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/70 to-transparent px-3 py-2">
                      <span className="font-mono text-sm font-medium text-white">{m.name}</span>
                    </div>
                  </div>
                  <div className="p-3 space-y-2">
                    <p className="text-xs text-muted-foreground">{formatSize(m.size_bytes)}</p>
                    <div className="flex gap-2">
                      <Button size="sm" variant="outline" onClick={() => handleChangeMap(m.name)} disabled={!!changingMap}>
                        {changingMap === m.name ? "Changing..." : "Play"}
                      </Button>
                      {rotation.includes(m.name) ? (
                        <Button size="sm" variant="ghost" onClick={() => removeFromRotation(m.name)}>
                          Remove
                        </Button>
                      ) : (
                        <Button size="sm" variant="ghost" onClick={() => addToRotation(m.name)}>
                          + Rotation
                        </Button>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </ScrollArea>
        </CardContent>
      </Card>

      {/* Map Rotation */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Map Rotation</CardTitle>
              <CardDescription>Order of maps in the rotation</CardDescription>
            </div>
            <Button variant="outline" size="sm" onClick={handleSaveRotation}>
              Save rotation
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            {rotation.length === 0 ? (
              <p className="py-4 text-center text-sm text-muted-foreground">No maps in rotation</p>
            ) : (
              rotation.map((mapName, i) => (
                <div
                  key={`${mapName}-${i}`}
                  className="flex items-center gap-2 rounded-md border border-border bg-muted/20 px-3 py-2"
                >
                  <span className="w-6 text-sm text-muted-foreground">{i + 1}.</span>
                  <span className="flex-1 font-mono text-sm">{mapName}</span>
                  <div className="flex gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8"
                      onClick={() => handleMoveInRotation(i, "up")}
                      disabled={i === 0}
                    >
                      <ChevronUp className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8"
                      onClick={() => handleMoveInRotation(i, "down")}
                      disabled={i === rotation.length - 1}
                    >
                      <ChevronDown className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              ))
            )}
          </div>
        </CardContent>
      </Card>

      {/* Workshop Download */}
      <Card>
        <CardHeader>
          <CardTitle>Workshop Download</CardTitle>
          <CardDescription>Download a map from Steam Workshop by ID</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap gap-2">
            <Input
              type="number"
              placeholder="Workshop ID (e.g. 1234567890)"
              value={workshopId}
              onChange={(e) => setWorkshopId(e.target.value)}
              className="max-w-xs"
            />
            <Button
              onClick={handleDownloadWorkshop}
              disabled={isDownloading || !workshopId.trim()}
            >
              <Download className="mr-2 h-4 w-4" />
              {downloadProgress ? `${downloadProgress.percent}% - ${downloadProgress.message}` : "Download"}
            </Button>
          </div>
          {downloadProgress && (
            <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
              <div
                className="h-full rounded-full bg-primary transition-all"
                style={{ width: `${downloadProgress.percent}%` }}
              />
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
