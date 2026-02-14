import { useEffect, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Select } from "@/components/ui/select";
import { ScrollArea } from "@/components/ui/scroll-area";
import type { AuditEntry } from "@/types";

const App = (window as any).go?.main?.App;

interface AuditLogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function AuditLog({ open, onOpenChange }: AuditLogProps) {
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [filter, setFilter] = useState<string>("all");

  useEffect(() => {
    if (!open || !App?.GetAuditLog) return;
    App.GetAuditLog(200).then((data: AuditEntry[]) => {
      setEntries(data ?? []);
    });
  }, [open]);

  const actions = Array.from(new Set(entries.map((e) => e.action))).sort();
  const filtered =
    filter === "all"
      ? entries
      : entries.filter((e) => e.action === filter);

  const formatDate = (s: string) => {
    try {
      const d = new Date(s);
      return isNaN(d.getTime()) ? s : d.toLocaleString();
    } catch {
      return s;
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[85vh] flex flex-col">
        <DialogHeader>
          <DialogTitle>Audit Log</DialogTitle>
          <DialogDescription>
            Recent administrative actions and events. Filter by action type.
          </DialogDescription>
        </DialogHeader>

        <div className="flex items-center gap-4 py-2">
          <label className="text-sm font-medium">Filter by action:</label>
          <Select
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="w-48"
          >
            <option value="all">All actions</option>
            {actions.map((a) => (
              <option key={a} value={a}>
                {a}
              </option>
            ))}
          </Select>
        </div>

        <ScrollArea className="flex-1 min-h-[200px] rounded-md border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/50">
                <th className="px-4 py-2 text-left font-medium">Timestamp</th>
                <th className="px-4 py-2 text-left font-medium">Action</th>
                <th className="px-4 py-2 text-left font-medium">Target</th>
                <th className="px-4 py-2 text-left font-medium">Details</th>
              </tr>
            </thead>
            <tbody>
              {filtered.length === 0 ? (
                <tr>
                  <td colSpan={4} className="px-4 py-8 text-center text-muted-foreground">
                    No audit entries
                  </td>
                </tr>
              ) : (
                filtered.map((e) => (
                  <tr key={e.id} className="border-b hover:bg-muted/30">
                    <td className="px-4 py-2 text-muted-foreground whitespace-nowrap">
                      {formatDate(e.created_at)}
                    </td>
                    <td className="px-4 py-2">{e.action}</td>
                    <td className="px-4 py-2">{e.target}</td>
                    <td className="px-4 py-2 max-w-md truncate" title={e.details}>
                      {e.details || "—"}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </ScrollArea>

        <div className="flex justify-end pt-2">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
