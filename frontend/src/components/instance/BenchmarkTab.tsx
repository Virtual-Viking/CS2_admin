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
  Slider,
  ScrollArea,
} from "@/components/ui";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
} from "recharts";
import { cn } from "@/lib/utils";
import type { BenchmarkResult } from "@/types";
import { Zap, Play } from "lucide-react";

// Mock when Wails not available
const MOCK_RESULTS: BenchmarkResult[] = [
  { id: "b1", instance_id: "i1", bot_count: 10, avg_tickrate: 128, min_tickrate: 125, avg_frametime: 7.8, cpu_usage: 25, ram_usage: 1200, duration_sec: 30, created_at: "2024-02-10T12:00:00Z" },
  { id: "b2", instance_id: "i1", bot_count: 32, avg_tickrate: 115, min_tickrate: 108, avg_frametime: 8.7, cpu_usage: 55, ram_usage: 1800, duration_sec: 30, created_at: "2024-02-10T12:05:00Z" },
  { id: "b3", instance_id: "i1", bot_count: 64, avg_tickrate: 95, min_tickrate: 88, avg_frametime: 10.5, cpu_usage: 88, ram_usage: 2400, duration_sec: 30, created_at: "2024-02-10T12:10:00Z" },
];

interface BenchmarkTabProps {
  instanceId: string;
}

export function BenchmarkTab({ instanceId }: BenchmarkTabProps) {
  const [maxBots, setMaxBots] = useState(32);
  const [stepSize, setStepSize] = useState(4);
  const [duration, setDuration] = useState(30);
  const [results, setResults] = useState<BenchmarkResult[]>([]);
  const [loading, setLoading] = useState(true);
  const [running, setRunning] = useState(false);
  const [progress, setProgress] = useState<{ step: number; total: number; botCount?: number; tickRate?: number } | null>(null);
  const [livePoints, setLivePoints] = useState<{ bot_count: number; tick_rate: number }[]>([]);

  const hasWails = typeof window !== "undefined" && !!(window as any).go?.main?.App;

  const loadResults = useCallback(async () => {
    try {
      if (hasWails) {
        const list = await (window as any).go?.main?.App?.GetBenchmarkResults?.(instanceId) ?? [];
        setResults(Array.isArray(list) ? list : []);
      } else {
        setResults(MOCK_RESULTS);
      }
    } catch {
      setResults([]);
    } finally {
      setLoading(false);
    }
  }, [instanceId, hasWails]);

  useEffect(() => {
    loadResults();
  }, [loadResults]);

  useEffect(() => {
    if (!hasWails || !instanceId) return;
    const onProgress = (p: unknown) => {
      const obj = p as Record<string, number>;
      if (obj && typeof obj === "object") {
        setProgress({
          step: (obj.step as number) ?? 0,
          total: (obj.total as number) ?? 0,
          botCount: obj.bot_count,
          tickRate: obj.tick_rate,
        });
        if (typeof obj.bot_count === "number" && typeof obj.tick_rate === "number") {
          setLivePoints((prev) => [...prev, { bot_count: obj.bot_count!, tick_rate: obj.tick_rate! }]);
        }
      }
    };
    const onComplete = () => {
      setRunning(false);
      setProgress(null);
      setLivePoints([]);
      loadResults();
    };
    const onError = () => {
      setRunning(false);
      setProgress(null);
    };
    try {
      (window as any).runtime?.EventsOn?.("benchmark:" + instanceId, onProgress);
      (window as any).runtime?.EventsOn?.("benchmark:" + instanceId + ":complete", onComplete);
      (window as any).runtime?.EventsOn?.("benchmark:" + instanceId + ":error", onError);
    } catch {
      /* no-op */
    }
    return () => {
      try {
        (window as any).runtime?.EventsOff?.("benchmark:" + instanceId);
        (window as any).runtime?.EventsOff?.("benchmark:" + instanceId + ":complete");
        (window as any).runtime?.EventsOff?.("benchmark:" + instanceId + ":error");
      } catch {
        /* no-op */
      }
    };
  }, [instanceId, hasWails, loadResults]);

  const handleRun = async () => {
    setRunning(true);
    setLivePoints([]);
    try {
      if (hasWails) {
        await (window as any).go?.main?.App?.RunBenchmark?.(instanceId, maxBots, stepSize, duration);
      } else {
        setProgress({ step: 1, total: 3, botCount: 10, tickRate: 128 });
        setTimeout(() => setRunning(false), 2000);
      }
    } catch (e) {
      console.error(e);
      setRunning(false);
    }
  };

  const chartData = livePoints.length > 0
    ? livePoints
    : results
        .map((r) => ({ bot_count: r.bot_count, tick_rate: r.avg_tickrate }))
        .sort((a, b) => a.bot_count - b.bot_count);

  const formatDate = (s: string) => {
    try {
      return new Date(s).toLocaleString(undefined, { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });
    } catch {
      return s;
    }
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Zap className="h-5 w-5" />
            Benchmark Config
          </CardTitle>
          <CardDescription>Test server performance with varying bot counts</CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="space-y-4">
            <div>
              <div className="flex justify-between">
                <Label>Max Bots (10–64)</Label>
                <span className="font-mono text-sm text-muted-foreground">{maxBots}</span>
              </div>
              <Slider
                min={10}
                max={64}
                value={maxBots}
                onChange={(e) => setMaxBots(parseInt((e.target as HTMLInputElement).value, 10) || 10)}
                className="mt-2"
              />
            </div>
            <div>
              <Label>Step Size</Label>
              <Input
                type="number"
                min={1}
                max={16}
                value={stepSize}
                onChange={(e) => setStepSize(parseInt(e.target.value, 10) || 1)}
                className="mt-2 w-24"
              />
            </div>
            <div>
              <Label>Duration per step (seconds)</Label>
              <Input
                type="number"
                min={10}
                max={120}
                value={duration}
                onChange={(e) => setDuration(parseInt(e.target.value, 10) || 30)}
                className="mt-2 w-24"
              />
            </div>
          </div>
          <Button onClick={handleRun} disabled={running}>
            <Play className="mr-2 h-4 w-4" />
            Run Benchmark
          </Button>
        </CardContent>
      </Card>

      {running && progress && (
        <Card>
          <CardHeader>
            <CardTitle>Progress</CardTitle>
            <CardDescription>
              Step {progress.step} of {progress.total}
              {progress.botCount != null && ` • ${progress.botCount} bots`}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
              <div
                className="h-full rounded-full bg-primary transition-all"
                style={{ width: progress.total > 0 ? `${(progress.step / progress.total) * 100}%` : "0%" }}
              />
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Results: Tick Rate vs Bot Count</CardTitle>
          <CardDescription>Higher tick rate with more bots indicates better performance</CardDescription>
        </CardHeader>
        <CardContent>
          {chartData.length > 0 ? (
            <ResponsiveContainer width="100%" height={280}>
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                <XAxis dataKey="bot_count" stroke="currentColor" className="text-xs" />
                <YAxis stroke="currentColor" className="text-xs" />
                <Tooltip formatter={(v: number | undefined) => [v != null ? v.toFixed(1) : "0", "Tick Rate"]} />
                <Line type="monotone" dataKey="tick_rate" stroke="#3b82f6" strokeWidth={2} dot name="Tick Rate" />
              </LineChart>
            </ResponsiveContainer>
          ) : (
            <p className="py-12 text-center text-muted-foreground">Run a benchmark to see results</p>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>History</CardTitle>
          <CardDescription>Previous benchmark runs</CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <p className="py-6 text-center text-muted-foreground">Loading...</p>
          ) : results.length === 0 ? (
            <p className="py-6 text-center text-sm text-muted-foreground">No benchmark results yet</p>
          ) : (
            <ScrollArea className="h-[200px]">
              <div className="space-y-2">
                {results.map((r) => (
                  <div
                    key={r.id}
                    className="flex items-center justify-between rounded-md border border-border bg-muted/20 px-4 py-3"
                  >
                    <span className="font-mono text-sm">
                      {r.bot_count} bots → {r.avg_tickrate.toFixed(1)} Hz
                    </span>
                    <span className="text-xs text-muted-foreground">{formatDate(r.created_at)}</span>
                  </div>
                ))}
              </div>
            </ScrollArea>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
