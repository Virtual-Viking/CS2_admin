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
import type { FileEntry } from "@/types";
import { FileText, Folder, FolderOpen, ChevronRight, Save } from "lucide-react";

// Mock when Wails not available
const MOCK_FILES: FileEntry[] = [
  { name: "cfg", path: "cfg", is_dir: true, size: 0, modified: "2024-02-10T12:00:00Z" },
  { name: "addons", path: "addons", is_dir: true, size: 0, modified: "2024-02-10T12:00:00Z" },
  { name: "server.cfg", path: "cfg/server.cfg", is_dir: false, size: 2048, modified: "2024-02-10T12:00:00Z" },
  { name: "autoexec.cfg", path: "cfg/autoexec.cfg", is_dir: false, size: 512, modified: "2024-02-09T10:00:00Z" },
];

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

interface FilesTabProps {
  instanceId: string;
}

export function FilesTab({ instanceId }: FilesTabProps) {
  const [entries, setEntries] = useState<FileEntry[]>([]);
  const [currentPath, setCurrentPath] = useState("");
  const [breadcrumb, setBreadcrumb] = useState<string[]>([]);
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [content, setContent] = useState("");
  const [editContent, setEditContent] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const hasWails = typeof window !== "undefined" && !!(window as any).go?.main?.App;

  const loadDir = async (path: string) => {
    setLoading(true);
    try {
      if (hasWails) {
        const list = await (window as any).go?.main?.App?.ListFiles?.(instanceId, path) ?? [];
        setEntries(Array.isArray(list) ? list : []);
      } else {
        if (path === "") {
          const topLevel = MOCK_FILES.filter((e) => {
            const segments = e.path.split("/");
            return segments.length === 1;
          });
          setEntries(topLevel);
        } else {
          const prefix = path + "/";
          const children = MOCK_FILES.filter((e) => {
            if (!e.path.startsWith(prefix)) return false;
            const rest = e.path.slice(prefix.length);
            return !rest.includes("/");
          });
          setEntries(children);
        }
      }
    } catch {
      setEntries([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadDir(currentPath);
    setBreadcrumb(currentPath ? currentPath.split("/").filter(Boolean) : []);
  }, [instanceId, currentPath, hasWails]);

  const handleSelectEntry = async (e: FileEntry) => {
    if (e.is_dir) {
      setCurrentPath(e.path);
      setSelectedFile(null);
      setContent("");
      return;
    }
    setSelectedFile(e.path);
    try {
      if (hasWails) {
        const text = await (window as any).go?.main?.App?.ReadServerFile?.(instanceId, e.path) ?? "";
        setContent(String(text));
        setEditContent(String(text));
      } else {
        setContent("// Mock file content for " + e.name);
        setEditContent("// Mock file content for " + e.name);
      }
    } catch {
      setContent("");
      setEditContent("");
    }
  };

  const handleBreadcrumb = (idx: number) => {
    const path = breadcrumb.slice(0, idx + 1).join("/");
    setCurrentPath(path);
    if (idx < breadcrumb.length - 1) setSelectedFile(null);
  };

  const handleSave = async () => {
    if (!selectedFile) return;
    setSaving(true);
    try {
      if (hasWails) {
        await (window as any).go?.main?.App?.WriteServerFile?.(instanceId, selectedFile, editContent);
        setContent(editContent);
      }
    } catch (e) {
      console.error(e);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="flex h-[500px] gap-4">
      {/* Tree / List */}
      <Card className="flex w-64 shrink-0 flex-col">
        <CardHeader className="pb-2">
          <CardTitle className="text-sm">Files</CardTitle>
          <CardDescription>Browse server files</CardDescription>
        </CardHeader>
        <CardContent className="flex-1 overflow-hidden p-0">
          {/* Breadcrumb */}
          <div className="flex flex-wrap items-center gap-1 border-b border-border px-4 py-2">
            <button
              type="button"
              className="text-xs text-muted-foreground hover:text-foreground"
              onClick={() => { setCurrentPath(""); setSelectedFile(null); }}
            >
              csgo/
            </button>
            {breadcrumb.map((part, i) => (
              <span key={part} className="flex items-center gap-1">
                <ChevronRight className="h-3 w-3 text-muted-foreground" />
                <button
                  type="button"
                  className="text-xs text-muted-foreground hover:text-foreground"
                  onClick={() => handleBreadcrumb(i)}
                >
                  {part}
                </button>
              </span>
            ))}
          </div>
          <ScrollArea className="h-[300px]">
            <div className="space-y-0.5 p-2">
              {currentPath && (
                <button
                  type="button"
                  className="flex w-full items-center gap-2 rounded px-2 py-1.5 text-left text-sm hover:bg-muted"
                  onClick={() => {
                    const parts = currentPath.split("/").filter(Boolean);
                    setCurrentPath(parts.slice(0, -1).join("/"));
                  }}
                >
                  <FolderOpen className="h-4 w-4 text-amber-500" />
                  ..
                </button>
              )}
              {loading ? (
                <p className="py-4 text-center text-xs text-muted-foreground">Loading...</p>
              ) : (
                entries
                  .sort((a, b) => (a.is_dir === b.is_dir ? a.name.localeCompare(b.name) : a.is_dir ? -1 : 1))
                  .map((e) => (
                    <button
                      key={e.path}
                      type="button"
                      className={cn(
                        "flex w-full items-center gap-2 rounded px-2 py-1.5 text-left text-sm hover:bg-muted",
                        selectedFile === e.path && "bg-muted"
                      )}
                      onClick={() => handleSelectEntry(e)}
                    >
                      {e.is_dir ? (
                        <Folder className="h-4 w-4 text-amber-500" />
                      ) : (
                        <FileText className="h-4 w-4 text-muted-foreground" />
                      )}
                      <span className="truncate">{e.name}</span>
                      {!e.is_dir && <span className="ml-auto text-xs text-muted-foreground">{formatSize(e.size)}</span>}
                    </button>
                  ))
              )}
            </div>
          </ScrollArea>
        </CardContent>
      </Card>

      {/* File content viewer */}
      <Card className="flex min-w-0 flex-1 flex-col">
        <CardHeader className="pb-2">
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2 text-sm">
                <FileText className="h-4 w-4" />
                {selectedFile ? selectedFile.split("/").pop() : "Select a file"}
              </CardTitle>
              <CardDescription>{selectedFile || "Click a file to view or edit"}</CardDescription>
            </div>
            {selectedFile && (
              <Button size="sm" onClick={handleSave} disabled={saving}>
                <Save className="mr-1.5 h-4 w-4" />
                Save
              </Button>
            )}
          </div>
        </CardHeader>
        <CardContent className="flex-1 overflow-hidden p-0">
          {selectedFile ? (
            <textarea
              value={editContent}
              onChange={(e) => setEditContent(e.target.value)}
              className="h-full w-full resize-none rounded-b-lg border-0 bg-transparent px-4 pb-4 font-mono text-sm tabular-nums"
              spellCheck={false}
              placeholder="File content..."
            />
          ) : (
            <div className="flex h-full items-center justify-center text-muted-foreground">
              Select a file from the list
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
