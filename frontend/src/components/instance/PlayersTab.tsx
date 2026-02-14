import { useEffect, useState, useCallback } from "react";
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Input,
  Label,
  Dialog,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogFooter,
  DialogTitle,
  DialogDescription,
  DialogClose,
} from "@/components/ui";
import { cn } from "@/lib/utils";
import type { Player, BanEntry } from "@/types";
import { UserMinus, Ban, MicOff, Trash2, Users } from "lucide-react";

// Mock data when Wails not available
const MOCK_PLAYERS: Player[] = [
  { name: "Player1", steam_id: "STEAM_1:0:12345", ping: 42, score: 12, team: "CT", ip: "192.168.1.1" },
  { name: "Player2", steam_id: "STEAM_1:0:67890", ping: 55, score: 8, team: "T", ip: "192.168.1.2" },
];

const MOCK_BANS: BanEntry[] = [
  {
    id: "b1",
    instance_id: "inst1",
    steam_id: "STEAM_1:0:99999",
    ip_address: "10.0.0.1",
    reason: "Cheating",
    expires_at: null,
    is_permanent: true,
    created_at: "2024-01-15T12:00:00Z",
  },
];

function formatDate(s: string | null): string {
  if (!s) return "Permanent";
  try {
    return new Date(s).toLocaleDateString(undefined, {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return String(s);
  }
}

interface PlayersTabProps {
  instanceId: string;
}

export function PlayersTab({ instanceId }: PlayersTabProps) {
  const [players, setPlayers] = useState<Player[]>([]);
  const [bans, setBans] = useState<BanEntry[]>([]);
  const [loading, setLoading] = useState(true);

  const hasWails = typeof window !== "undefined" && !!(window as any).go?.main?.App;

  const fetchPlayers = useCallback(async () => {
    if (hasWails) {
      try {
        const list = await (window as any).go?.main?.App.GetPlayers(instanceId);
        setPlayers(Array.isArray(list) ? list : []);
      } catch {
        setPlayers([]);
      }
    } else {
      setPlayers(MOCK_PLAYERS);
    }
  }, [instanceId, hasWails]);

  const fetchBans = useCallback(async () => {
    if (hasWails) {
      try {
        const list = await (window as any).go?.main?.App.GetBanList(instanceId);
        setBans(Array.isArray(list) ? list : []);
      } catch {
        setBans([]);
      }
    } else {
      setBans(MOCK_BANS);
    }
  }, [instanceId, hasWails]);

  useEffect(() => {
    setLoading(true);
    const load = async () => {
      await Promise.all([fetchPlayers(), fetchBans()]);
      setLoading(false);
    };
    load();
  }, [fetchPlayers, fetchBans]);

  useEffect(() => {
    const interval = setInterval(fetchPlayers, 3000);
    return () => clearInterval(interval);
  }, [fetchPlayers]);

  const handleKick = async (steamId: string, reason: string) => {
    if (!hasWails) return;
    try {
      await (window as any).go?.main?.App.KickPlayer(instanceId, steamId, reason);
      await fetchPlayers();
    } catch (e) {
      console.error(e);
    }
  };

  const handleBan = async (steamId: string, durationMinutes: number, reason: string) => {
    if (!hasWails) return;
    try {
      await (window as any).go?.main?.App.BanPlayer(instanceId, steamId, durationMinutes, reason);
      await Promise.all([fetchPlayers(), fetchBans()]);
    } catch (e) {
      console.error(e);
    }
  };

  const handleMute = async (steamId: string) => {
    if (!hasWails) return;
    try {
      await (window as any).go?.main?.App.MutePlayer(instanceId, steamId);
    } catch (e) {
      console.error(e);
    }
  };

  const handleRemoveBan = async (banId: string) => {
    if (!hasWails) return;
    try {
      await (window as any).go?.main?.App.RemoveBan(banId);
      await fetchBans();
    } catch (e) {
      console.error(e);
    }
  };

  if (loading && players.length === 0 && bans.length === 0) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-12">
          <p className="text-muted-foreground">Loading...</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* Live Players */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Users className="h-5 w-5" />
            Live Players ({players.length})
          </CardTitle>
          <CardDescription>Auto-refreshes every 3 seconds</CardDescription>
        </CardHeader>
        <CardContent>
          {players.length === 0 ? (
            <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border py-12">
              <Users className="mb-4 h-12 w-12 text-muted-foreground/50" />
              <p className="text-sm font-medium text-muted-foreground">No players connected</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border">
                    <th className="pb-3 text-left font-medium">Name</th>
                    <th className="pb-3 text-left font-medium">SteamID</th>
                    <th className="pb-3 text-right font-medium">Ping</th>
                    <th className="pb-3 text-right font-medium">Score</th>
                    <th className="pb-3 text-left font-medium">Team</th>
                    <th className="pb-3 text-right font-medium">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {players.map((p) => (
                    <tr key={p.steam_id} className="border-b border-border/50">
                      <td className="py-3">{p.name}</td>
                      <td className="py-3 font-mono text-xs text-muted-foreground">{p.steam_id}</td>
                      <td className="py-3 text-right">{p.ping}</td>
                      <td className="py-3 text-right">{p.score}</td>
                      <td className="py-3">
                        <span
                          className={cn(
                            "rounded px-1.5 py-0.5 text-xs font-medium",
                            p.team === "CT" ? "bg-sky-500/20 text-sky-400" : "bg-amber-500/20 text-amber-400"
                          )}
                        >
                          {p.team || "—"}
                        </span>
                      </td>
                      <td className="py-3 text-right">
                        <div className="flex justify-end gap-1">
                          <KickDialog steamId={p.steam_id} onKick={handleKick} />
                          <BanDialog steamId={p.steam_id} onBan={handleBan} />
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-8 w-8"
                            title="Mute"
                            onClick={() => handleMute(p.steam_id)}
                            disabled={!hasWails}
                          >
                            <MicOff className="h-4 w-4" />
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Ban List */}
      <Card>
        <CardHeader>
          <CardTitle>Ban List</CardTitle>
          <CardDescription>Banned players on this instance</CardDescription>
        </CardHeader>
        <CardContent>
          {bans.length === 0 ? (
            <p className="py-4 text-center text-sm text-muted-foreground">No bans</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border">
                    <th className="pb-3 text-left font-medium">SteamID</th>
                    <th className="pb-3 text-left font-medium">Reason</th>
                    <th className="pb-3 text-left font-medium">Banned</th>
                    <th className="pb-3 text-left font-medium">Expiry</th>
                    <th className="pb-3 text-right font-medium">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {bans.map((b) => (
                    <tr key={b.id} className="border-b border-border/50">
                      <td className="py-3 font-mono text-xs">{b.steam_id}</td>
                      <td className="py-3">{b.reason || "—"}</td>
                      <td className="py-3 text-muted-foreground">{formatDate(b.created_at)}</td>
                      <td className="py-3">
                        {b.is_permanent ? (
                          <span className="text-destructive">Permanent</span>
                        ) : (
                          formatDate(b.expires_at)
                        )}
                      </td>
                      <td className="py-3 text-right">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleRemoveBan(b.id)}
                          disabled={!hasWails}
                        >
                          <Trash2 className="mr-1 h-4 w-4" />
                          Remove
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function KickDialog({
  steamId,
  onKick,
}: {
  steamId: string;
  onKick: (steamId: string, reason: string) => void;
}) {
  const [reason, setReason] = useState("Kicked by admin");

  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button variant="ghost" size="icon" className="h-8 w-8" title="Kick">
          <UserMinus className="h-4 w-4" />
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Kick Player</DialogTitle>
          <DialogDescription>Enter a reason for kicking this player.</DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label>Reason</Label>
            <Input value={reason} onChange={(e) => setReason(e.target.value)} placeholder="Kicked by admin" />
          </div>
        </div>
        <DialogFooter>
          <DialogClose className="inline-flex h-10 items-center justify-center rounded-md border border-input bg-background px-4 py-2 text-sm font-medium hover:bg-accent hover:text-accent-foreground">
            Cancel
          </DialogClose>
          <Button onClick={() => onKick(steamId, reason)}>Kick</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function BanDialog({
  steamId,
  onBan,
}: {
  steamId: string;
  onBan: (steamId: string, durationMinutes: number, reason: string) => void;
}) {
  const [reason, setReason] = useState("Banned by admin");
  const [duration, setDuration] = useState("60");
  const [permanent, setPermanent] = useState(false);

  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button variant="ghost" size="icon" className="h-8 w-8 text-destructive" title="Ban">
          <Ban className="h-4 w-4" />
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Ban Player</DialogTitle>
          <DialogDescription>Ban this player. Duration 0 = permanent.</DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label>Duration (minutes)</Label>
            <Input
              type="number"
              min={0}
              value={duration}
              onChange={(e) => setDuration(e.target.value)}
              disabled={permanent}
            />
          </div>
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="perm"
              checked={permanent}
              onChange={(e) => setPermanent(e.target.checked)}
              className="rounded"
            />
            <Label htmlFor="perm">Permanent ban</Label>
          </div>
          <div className="space-y-2">
            <Label>Reason</Label>
            <Input value={reason} onChange={(e) => setReason(e.target.value)} placeholder="Banned by admin" />
          </div>
        </div>
        <DialogFooter>
          <DialogClose className="inline-flex h-10 items-center justify-center rounded-md border border-input bg-background px-4 py-2 text-sm font-medium hover:bg-accent hover:text-accent-foreground">
            Cancel
          </DialogClose>
          <Button
            variant="destructive"
            onClick={() =>
              onBan(steamId, permanent ? 0 : parseInt(duration, 10) || 60, reason)
            }
          >
            Ban
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
