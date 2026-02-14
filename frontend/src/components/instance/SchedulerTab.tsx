import { useEffect, useState } from "react";
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
} from "@/components/ui";
import { cn } from "@/lib/utils";
import type { ScheduledTask } from "@/types";
import { useAppStore } from "@/stores/app-store";
import { Calendar, Plus, Trash2 } from "lucide-react";

const ACTIONS = [
  { value: "rcon", label: "RCON Command" },
  { value: "backup", label: "Backup" },
  { value: "restart", label: "Restart Server" },
];

// Mock when Wails not available
const MOCK_TASKS: ScheduledTask[] = [
  { id: "t1", instance_id: "i1", cron_expr: "0 */6 * * *", action: "backup", payload: "", enabled: true, last_run: null, next_run: "2024-02-10T18:00:00Z", created_at: "2024-02-01T00:00:00Z" },
  { id: "t2", instance_id: "i1", cron_expr: "0 3 * * *", action: "rcon", payload: "sv_restart 1", enabled: true, last_run: null, next_run: "2024-02-11T03:00:00Z", created_at: "2024-02-01T00:00:00Z" },
];

function formatDate(s: string | null): string {
  if (!s) return "—";
  try {
    return new Date(s).toLocaleString(undefined, {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return String(s);
  }
}

interface SchedulerTabProps {
  instanceId: string;
}

export function SchedulerTab({ instanceId }: SchedulerTabProps) {
  const instances = useAppStore((s) => s.instances);
  const [tasks, setTasks] = useState<ScheduledTask[]>([]);
  const [loading, setLoading] = useState(true);
  const [action, setAction] = useState("rcon");
  const [payload, setPayload] = useState("");
  const [cronExpr, setCronExpr] = useState("0 * * * *");
  const [enabled, setEnabled] = useState(true);

  const hasWails = typeof window !== "undefined" && !!(window as any).go?.main?.App;

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      try {
        if (hasWails) {
          const list = await (window as any).go?.main?.App?.GetScheduledTasks?.(instanceId) ?? [];
          setTasks(Array.isArray(list) ? list : []);
        } else {
          setTasks(MOCK_TASKS);
        }
      } catch {
        setTasks([]);
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [instanceId, hasWails]);

  const handleAdd = async () => {
    try {
      if (hasWails) {
        const task: Partial<ScheduledTask> = {
          instance_id: instanceId,
          cron_expr: cronExpr,
          action,
          payload,
          enabled,
        };
        await (window as any).go?.main?.App?.CreateScheduledTask?.(task);
        const list = await (window as any).go?.main?.App?.GetScheduledTasks?.(instanceId) ?? [];
        setTasks(Array.isArray(list) ? list : []);
        setPayload("");
      }
    } catch (e) {
      console.error(e);
    }
  };

  const handleDelete = async (taskId: string) => {
    if (!confirm("Delete this scheduled task?")) return;
    try {
      if (hasWails) {
        await (window as any).go?.main?.App?.DeleteScheduledTask?.(taskId);
        setTasks((prev) => prev.filter((t) => t.id !== taskId));
      }
    } catch (e) {
      console.error(e);
    }
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Calendar className="h-5 w-5" />
            Add Task
          </CardTitle>
          <CardDescription>Schedule recurring actions for this instance</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <Label>Instance</Label>
            <Select value={instanceId} className="mt-1 w-full max-w-xs" disabled>
              {instances.map((i) => (
                <option key={i.id} value={i.id}>
                  {i.name}
                </option>
              ))}
            </Select>
          </div>
          <div>
            <Label>Action Type</Label>
            <Select value={action} onChange={(e) => setAction(e.target.value)} className="mt-1 w-full max-w-xs">
              {ACTIONS.map((a) => (
                <option key={a.value} value={a.value}>
                  {a.label}
                </option>
              ))}
            </Select>
          </div>
          {action === "rcon" && (
            <div>
              <Label>Payload (RCON command)</Label>
              <Input
                placeholder="e.g. sv_restart 1"
                value={payload}
                onChange={(e) => setPayload(e.target.value)}
                className="mt-1 max-w-md font-mono"
              />
            </div>
          )}
          <div>
            <Label>Cron Expression</Label>
            <Input
              placeholder="0 * * * *"
              value={cronExpr}
              onChange={(e) => setCronExpr(e.target.value)}
              className="mt-1 max-w-xs font-mono"
            />
            <p className="mt-1 text-xs text-muted-foreground">
              Format: minute hour day month weekday (e.g. 0 */6 * * * = every 6 hours)
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Switch checked={enabled} onChange={(e) => setEnabled((e.target as HTMLInputElement).checked)} />
            <Label>Enabled</Label>
          </div>
          <Button onClick={handleAdd}>
            <Plus className="mr-2 h-4 w-4" />
            Add Task
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Scheduled Tasks</CardTitle>
          <CardDescription>Cron-scheduled actions</CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <p className="py-8 text-center text-muted-foreground">Loading...</p>
          ) : tasks.length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">No scheduled tasks</p>
          ) : (
            <div className="space-y-2">
              {tasks.map((t) => (
                <div
                  key={t.id}
                  className="flex flex-wrap items-center justify-between gap-4 rounded-lg border border-border bg-muted/20 p-4"
                >
                  <div className="flex flex-wrap items-center gap-4">
                    <span className="rounded bg-muted px-2 py-0.5 font-mono text-sm">{t.action}</span>
                    <span className="font-mono text-sm">{t.cron_expr}</span>
                    {t.payload && <span className="text-sm text-muted-foreground">{t.payload}</span>}
                    <span className="text-xs text-muted-foreground">Next: {formatDate(t.next_run)}</span>
                    <Switch checked={t.enabled} disabled />
                  </div>
                  <Button size="sm" variant="outline" onClick={() => handleDelete(t.id)}>
                    <Trash2 className="mr-1 h-4 w-4" />
                    Delete
                  </Button>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
