import { useEffect, useState } from "react";
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Select,
  ScrollArea,
} from "@/components/ui";
import { cn } from "@/lib/utils";
import type { Backup } from "@/types";
import { HardDrive, Plus, RotateCcw, Trash2 } from "lucide-react";

const BACKUP_TYPES = [
  { value: "full", label: "Full" },
  { value: "config", label: "Config Only" },
  { value: "maps", label: "Maps Only" },
  { value: "plugins", label: "Plugins Only" },
];

// Mock when Wails not available
const MOCK_BACKUPS: Backup[] = [
  { id: "b1", instance_id: "i1", path: "/backups/full_1.zip", size_bytes: 1024 * 1024 * 512, backup_type: "full", created_at: "2024-02-10T10:00:00Z" },
  { id: "b2", instance_id: "i1", path: "/backups/config_1.zip", size_bytes: 1024 * 64, backup_type: "config", created_at: "2024-02-09T14:00:00Z" },
];

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}

function formatDate(s: string): string {
  try {
    return new Date(s).toLocaleString(undefined, {
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

interface BackupsTabProps {
  instanceId: string;
}

export function BackupsTab({ instanceId }: BackupsTabProps) {
  const [backups, setBackups] = useState<Backup[]>([]);
  const [backupType, setBackupType] = useState("full");
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [actionId, setActionId] = useState<string | null>(null);

  const hasWails = typeof window !== "undefined" && !!(window as any).go?.main?.App;

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      try {
        if (hasWails) {
          const list = await (window as any).go?.main?.App?.GetBackups?.(instanceId) ?? [];
          setBackups(Array.isArray(list) ? list : []);
        } else {
          setBackups(MOCK_BACKUPS);
        }
      } catch {
        setBackups([]);
      } finally {
        setLoading(false);
      }
    };
    load();
  }, [instanceId, hasWails]);

  const handleCreate = async () => {
    setCreating(true);
    try {
      if (hasWails) {
        await (window as any).go?.main?.App?.CreateBackup?.(instanceId, backupType);
        const list = await (window as any).go?.main?.App?.GetBackups?.(instanceId) ?? [];
        setBackups(Array.isArray(list) ? list : []);
      }
    } catch (e) {
      console.error(e);
    } finally {
      setCreating(false);
    }
  };

  const handleRestore = async (backupId: string) => {
    setActionId(backupId);
    try {
      if (hasWails) {
        await (window as any).go?.main?.App?.RestoreBackup?.(backupId);
      }
    } catch (e) {
      console.error(e);
    } finally {
      setActionId(null);
    }
  };

  const handleDelete = async (backupId: string) => {
    if (!confirm("Delete this backup?")) return;
    setActionId(backupId);
    try {
      if (hasWails) {
        await (window as any).go?.main?.App?.DeleteBackup?.(backupId);
        setBackups((prev) => prev.filter((b) => b.id !== backupId));
      }
    } catch (e) {
      console.error(e);
    } finally {
      setActionId(null);
    }
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <HardDrive className="h-5 w-5" />
            Create Backup
          </CardTitle>
          <CardDescription>Create a new backup for this instance</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-wrap items-end gap-4">
          <div>
            <label className="mb-2 block text-sm font-medium">Type</label>
            <Select value={backupType} onChange={(e) => setBackupType(e.target.value)} className="w-40">
              {BACKUP_TYPES.map((t) => (
                <option key={t.value} value={t.value}>
                  {t.label}
                </option>
              ))}
            </Select>
          </div>
          <Button onClick={handleCreate} disabled={creating}>
            <Plus className="mr-2 h-4 w-4" />
            Create
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Backup List</CardTitle>
          <CardDescription>Restore or delete backups</CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <p className="py-8 text-center text-muted-foreground">Loading...</p>
          ) : backups.length === 0 ? (
            <p className="py-8 text-center text-sm text-muted-foreground">No backups found</p>
          ) : (
            <ScrollArea>
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border">
                    <th className="py-3 text-left font-medium">Date</th>
                    <th className="py-3 text-left font-medium">Type</th>
                    <th className="py-3 text-right font-medium">Size</th>
                    <th className="py-3 text-right font-medium">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {backups.map((b) => (
                    <tr key={b.id} className="border-b border-border/50">
                      <td className="py-3">{formatDate(b.created_at)}</td>
                      <td className="py-3 capitalize">{b.backup_type}</td>
                      <td className="py-3 text-right">{formatSize(b.size_bytes)}</td>
                      <td className="py-3 text-right">
                        <div className="flex justify-end gap-2">
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => handleRestore(b.id)}
                            disabled={actionId !== null}
                          >
                            <RotateCcw className="mr-1 h-4 w-4" />
                            Restore
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => handleDelete(b.id)}
                            disabled={actionId !== null}
                          >
                            <Trash2 className="mr-1 h-4 w-4" />
                            Delete
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </ScrollArea>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
