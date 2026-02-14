import { useEffect, useState, useCallback } from "react";
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
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
import type { MetricSnapshot } from "@/types";
import { Activity, Play, Square } from "lucide-react";

const MAX_POINTS = 300; // 5 min at 1/s
const CHART_HEIGHT = 180;

interface MetricsData {
  cpu_pct?: number;
  ram_mb?: number;
  tick_rate?: number;
  net_in_kbps?: number;
  net_out_kbps?: number;
  timestamp: string;
}

interface MonitoringTabProps {
  instanceId: string;
}

export function MonitoringTab({ instanceId }: MonitoringTabProps) {
  const [history, setHistory] = useState<MetricsData[]>([]);
  const [running, setRunning] = useState(false);
  const [loading, setLoading] = useState(false);
  const hasWails = typeof window !== "undefined" && !!(window as any).go?.main?.App;

  const addPoint = useCallback((m: { cpu_pct?: number; ram_mb?: number; tick_rate?: number; net_in_kbps?: number; net_out_kbps?: number }) => {
    const ts = new Date().toISOString();
    setHistory((prev) => {
      const next = [...prev, { ...m, timestamp: ts }];
      if (next.length > MAX_POINTS) return next.slice(-MAX_POINTS);
      return next;
    });
  }, []);

  useEffect(() => {
    if (!hasWails) return;
    const loadHistory = async () => {
      try {
        const list = await (window as any).go?.main?.App?.GetMetricsHistory?.(instanceId, 5) ?? [];
        const arr = Array.isArray(list) ? list : [];
        const data: MetricsData[] = arr.map((s: MetricSnapshot) => ({
          cpu_pct: s.cpu_pct,
          ram_mb: s.ram_mb,
          tick_rate: s.tick_rate,
          net_in_kbps: s.net_in_kbps,
          net_out_kbps: s.net_out_kbps,
          timestamp: s.timestamp || "",
        }));
        setHistory(data);
      } catch {
        /* ignore */
      }
    };
    loadHistory();
  }, [instanceId, hasWails]);

  useEffect(() => {
    if (!hasWails || !instanceId) return;
    const eventName = `metrics:${instanceId}`;
    const cb = (m: unknown) => {
      const obj = m as Record<string, number>;
      if (obj && typeof obj === "object") {
        addPoint({
          cpu_pct: obj.cpu_pct ?? obj.CPUPercent,
          ram_mb: obj.ram_mb ?? obj.RAMMb,
          tick_rate: obj.tick_rate ?? obj.TickRate,
          net_in_kbps: obj.net_in_kbps ?? obj.NetInKbps,
          net_out_kbps: obj.net_out_kbps ?? obj.NetOutKbps,
        });
      }
    };
    try {
      (window as any).runtime?.EventsOn?.(eventName, cb);
    } catch {
      /* no-op */
    }
    return () => {
      try {
        (window as any).runtime?.EventsOff?.(eventName);
      } catch {
        /* no-op */
      }
    };
  }, [instanceId, hasWails, addPoint]);

  const handleStart = async () => {
    setLoading(true);
    try {
      if (hasWails) {
        await (window as any).go?.main?.App?.StartMetrics?.(instanceId);
        setRunning(true);
      }
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  const handleStop = async () => {
    setLoading(true);
    try {
      if (hasWails) {
        (window as any).go?.main?.App?.StopMetrics?.(instanceId);
        setRunning(false);
      }
    } finally {
      setLoading(false);
    }
  };

  const chartData = history.length > 0 ? history : [{ timestamp: "", cpu_pct: 0, ram_mb: 0, tick_rate: 0, net_in_kbps: 0, net_out_kbps: 0 }];

  const formatTime = (ts: string) => {
    if (!ts) return "";
    try {
      const d = new Date(ts);
      return d.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit", second: "2-digit" });
    } catch {
      return ts;
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h3 className="text-lg font-semibold">Live Metrics</h3>
          <p className="text-sm text-muted-foreground">
            Last 5 minutes of data • Start metrics collection when server is running
          </p>
        </div>
        <div className="flex gap-2">
          <Button size="sm" variant="outline" onClick={handleStart} disabled={running || loading}>
            <Play className="mr-1.5 h-4 w-4" />
            Start
          </Button>
          <Button size="sm" variant="outline" onClick={handleStop} disabled={!running || loading}>
            <Square className="mr-1.5 h-4 w-4" />
            Stop
          </Button>
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        {/* CPU Usage */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2 text-sm">
              <Activity className="h-4 w-4" />
              CPU Usage (%)
            </CardTitle>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={CHART_HEIGHT}>
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                <XAxis dataKey="timestamp" tickFormatter={formatTime} stroke="currentColor" className="text-xs" />
                <YAxis stroke="currentColor" className="text-xs" />
                <Tooltip formatter={(v: number | undefined) => [v != null ? v.toFixed(1) : "0", "CPU %"]} labelFormatter={(label) => formatTime(String(label ?? ""))} />
                <Line type="monotone" dataKey="cpu_pct" stroke="#22c55e" strokeWidth={2} dot={false} name="CPU %" />
              </LineChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        {/* RAM Usage */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm">RAM Usage (MB)</CardTitle>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={CHART_HEIGHT}>
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                <XAxis dataKey="timestamp" tickFormatter={formatTime} stroke="currentColor" className="text-xs" />
                <YAxis stroke="currentColor" className="text-xs" />
                <Tooltip formatter={(v: number | undefined) => [v != null ? v.toFixed(0) : "0", "MB"]} labelFormatter={(label) => formatTime(String(label ?? ""))} />
                <Line type="monotone" dataKey="ram_mb" stroke="#3b82f6" strokeWidth={2} dot={false} name="RAM MB" />
              </LineChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        {/* Tick Rate */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm">Tick Rate (Hz)</CardTitle>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={CHART_HEIGHT}>
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                <XAxis dataKey="timestamp" tickFormatter={formatTime} stroke="currentColor" className="text-xs" />
                <YAxis stroke="currentColor" className="text-xs" />
                <Tooltip formatter={(v: number | undefined) => [v != null ? v.toFixed(1) : "0", "Hz"]} labelFormatter={(label) => formatTime(String(label ?? ""))} />
                <Line type="monotone" dataKey="tick_rate" stroke="#eab308" strokeWidth={2} dot={false} name="Tick" />
              </LineChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        {/* Network I/O */}
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm">Network I/O (Kbps)</CardTitle>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={CHART_HEIGHT}>
              <LineChart data={chartData}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                <XAxis dataKey="timestamp" tickFormatter={formatTime} stroke="currentColor" className="text-xs" />
                <YAxis stroke="currentColor" className="text-xs" />
                <Tooltip
                  formatter={(v: number | undefined, n?: string) => [v != null ? v.toFixed(1) : "0", (n ?? "") === "net_in_kbps" ? "In" : "Out"]}
                  labelFormatter={(label) => formatTime(String(label ?? ""))}
                />
                <Line type="monotone" dataKey="net_in_kbps" stroke="#22c55e" strokeWidth={2} dot={false} name="In" />
                <Line type="monotone" dataKey="net_out_kbps" stroke="#f97316" strokeWidth={2} dot={false} name="Out" />
              </LineChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
