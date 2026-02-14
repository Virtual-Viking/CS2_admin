import { useEffect, useState } from "react";
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  ScrollArea,
} from "@/components/ui";
import { cn } from "@/lib/utils";
import type { Match, MatchPlayer } from "@/types";
import { BarChart3, ChevronDown, ChevronRight } from "lucide-react";

// Mock data when Wails not available
const MOCK_MATCHES: Match[] = [
  { id: "m1", instance_id: "i1", map_name: "de_dust2", game_mode: "Competitive", team1_score: 13, team2_score: 9, duration_sec: 2100, rounds_played: 22, started_at: "2024-02-10T14:00:00Z", ended_at: "2024-02-10T14:35:00Z" },
  { id: "m2", instance_id: "i1", map_name: "de_mirage", game_mode: "Competitive", team1_score: 13, team2_score: 11, duration_sec: 2400, rounds_played: 24, started_at: "2024-02-10T13:00:00Z", ended_at: "2024-02-10T13:40:00Z" },
];

const MOCK_PLAYERS: MatchPlayer[] = [
  { id: "p1", match_id: "m1", steam_id: "S1", player_name: "Player1", team: "CT", kills: 22, deaths: 16, assists: 4, headshots: 12, mvps: 2, total_damage: 2800, utility_damage: 100, enemies_flashed: 5, adr: 127, hsp: 55, score: 48 },
  { id: "p2", match_id: "m1", steam_id: "S2", player_name: "Player2", team: "T", kills: 18, deaths: 18, assists: 2, headshots: 8, mvps: 1, total_damage: 2100, utility_damage: 50, enemies_flashed: 3, adr: 95, hsp: 44, score: 38 },
];

function formatDate(s: string): string {
  try {
    return new Date(s).toLocaleDateString(undefined, {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return s;
  }
}

function formatDuration(sec: number): string {
  const m = Math.floor(sec / 60);
  return `${m} min`;
}

interface StatsTabProps {
  instanceId: string;
}

export function StatsTab({ instanceId }: StatsTabProps) {
  const [matches, setMatches] = useState<Match[]>([]);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [players, setPlayers] = useState<MatchPlayer[]>([]);
  const [loading, setLoading] = useState(true);

  const hasWails = typeof window !== "undefined" && !!(window as any).go?.main?.App;

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      try {
        if (hasWails) {
          const list = await (window as any).go?.main?.App.GetMatches?.(instanceId) ?? [];
          setMatches(Array.isArray(list) ? list : []);
        } else {
          setMatches(MOCK_MATCHES);
        }
      } catch {
        setMatches([]);
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [instanceId, hasWails]);

  const loadMatchDetail = async (matchId: string) => {
    if (expandedId === matchId) {
      setExpandedId(null);
      setPlayers([]);
      return;
    }
    setExpandedId(matchId);
    try {
      if (hasWails) {
        const pl = await (window as any).go?.main?.App.GetMatchPlayers?.(matchId) ?? [];
        setPlayers(Array.isArray(pl) ? pl : []);
      } else {
        setPlayers(MOCK_PLAYERS);
      }
    } catch {
      setPlayers([]);
    }
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BarChart3 className="h-5 w-5" />
            Match History
          </CardTitle>
          <CardDescription>View match statistics and scoreboards</CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <p className="py-8 text-center text-muted-foreground">Loading matches...</p>
          ) : matches.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <BarChart3 className="mb-4 h-12 w-12 text-muted-foreground/50" />
              <p className="text-muted-foreground">No matches recorded yet</p>
              <p className="mt-1 text-sm text-muted-foreground">
                Match stats are collected when games are played on this server.
              </p>
            </div>
          ) : (
            <div className="space-y-2">
              {matches.map((m) => (
                <div key={m.id} className="rounded-lg border border-border bg-muted/20">
                  <button
                    type="button"
                    className="flex w-full items-center gap-3 p-4 text-left transition-colors hover:bg-muted/40"
                    onClick={() => loadMatchDetail(m.id)}
                  >
                    {expandedId === m.id ? (
                      <ChevronDown className="h-4 w-4 shrink-0" />
                    ) : (
                      <ChevronRight className="h-4 w-4 shrink-0" />
                    )}
                    <span className="flex-1 font-mono text-sm">{m.map_name}</span>
                    <span className="text-sm text-muted-foreground">{formatDate(m.started_at)}</span>
                    <span className="rounded bg-muted px-2 py-0.5 font-mono text-sm">
                      {m.team1_score} - {m.team2_score}
                    </span>
                    <span className="text-xs text-muted-foreground">{formatDuration(m.duration_sec)}</span>
                  </button>
                  {expandedId === m.id && (
                    <div className="border-t border-border p-4">
                      {/* Scoreboard table */}
                      <div className="mb-4">
                        <h4 className="mb-3 text-sm font-medium">Scoreboard</h4>
                        <ScrollArea className="h-[200px]">
                          <table className="w-full text-sm">
                            <thead>
                              <tr className="border-b border-border">
                                <th className="py-2 text-left font-medium">Name</th>
                                <th className="py-2 text-center">K</th>
                                <th className="py-2 text-center">D</th>
                                <th className="py-2 text-center">A</th>
                                <th className="py-2 text-center">HS%</th>
                                <th className="py-2 text-center">ADR</th>
                                <th className="py-2 text-center">MVP</th>
                                <th className="py-2 text-right">Score</th>
                              </tr>
                            </thead>
                            <tbody>
                              {players.map((p) => (
                                <tr key={p.id} className="border-b border-border/50">
                                  <td className="py-2">{p.player_name}</td>
                                  <td className="py-2 text-center">{p.kills}</td>
                                  <td className="py-2 text-center">{p.deaths}</td>
                                  <td className="py-2 text-center">{p.assists}</td>
                                  <td className="py-2 text-center">{p.hsp}%</td>
                                  <td className="py-2 text-center">{p.adr}</td>
                                  <td className="py-2 text-center">{p.mvps}</td>
                                  <td className="py-2 text-right">{p.score}</td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </ScrollArea>
                      </div>
                      {/* Placeholders */}
                      <div className="space-y-3 rounded border border-border bg-muted/20 p-3 text-sm text-muted-foreground">
                        <p>Damage details available after match</p>
                        <p>Round timeline (placeholder)</p>
                      </div>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
