import { NavLink, useNavigate } from "react-router-dom";
import {
  Server,
  Plus,
  Settings,
  Info,
  ChevronLeft,
  ChevronRight,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app-store";
import type { ServerInstance } from "@/types";

function getStatusColor(status: ServerInstance["status"]) {
  switch (status) {
    case "running":
      return "bg-emerald-500";
    case "stopped":
    case "crashed":
      return "bg-red-500";
    case "starting":
    case "stopping":
      return "bg-amber-500";
    case "installing":
    case "updating":
      return "bg-blue-500";
    default:
      return "bg-zinc-500";
  }
}

export function Sidebar() {
  const instances = useAppStore((s) => s.instances);
  const sidebarCollapsed = useAppStore((s) => s.sidebarCollapsed);
  const toggleSidebar = useAppStore((s) => s.toggleSidebar);
  const setSelectedInstanceId = useAppStore((s) => s.setSelectedInstanceId);
  const navigate = useNavigate();

  const handleInstanceClick = (id: string) => {
    setSelectedInstanceId(id);
    navigate(`/instances/${id}`);
  };

  const handleNewInstance = () => {
    // TODO: Open new instance wizard
    navigate("/");
  };

  return (
    <aside
      className={cn(
        "flex flex-col border-r border-border bg-sidebar text-sidebar-foreground transition-all duration-200",
        sidebarCollapsed ? "w-[60px]" : "w-60"
      )}
    >
      {/* Logo / App name */}
      <div
        className={cn(
          "flex h-14 shrink-0 items-center border-b border-border px-3",
          sidebarCollapsed && "justify-center"
        )}
      >
        {!sidebarCollapsed ? (
          <div className="flex items-center gap-2">
            <Server className="h-6 w-6 shrink-0 text-primary" />
            <span className="font-semibold tracking-tight">CS2 Admin</span>
          </div>
        ) : (
          <Server className="h-6 w-6 shrink-0 text-primary" />
        )}
      </div>

      {/* Instance list */}
      <div className="flex-1 overflow-y-auto py-2">
        <div
          className={cn(
            "flex items-center justify-between px-2 pb-2",
            sidebarCollapsed && "flex-col gap-1"
          )}
        >
          {!sidebarCollapsed && (
            <span className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
              Instances
            </span>
          )}
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8 shrink-0"
            onClick={handleNewInstance}
            title="Add new instance"
          >
            <Plus className="h-4 w-4" />
          </Button>
        </div>

        <div className="space-y-0.5 px-2">
          {instances.map((inst) => (
            <button
              key={inst.id}
              type="button"
              onClick={() => handleInstanceClick(inst.id)}
              className={cn(
                "flex w-full items-center gap-2 rounded-md px-2 py-2 text-left transition-colors hover:bg-accent hover:text-accent-foreground",
                sidebarCollapsed && "justify-center px-0"
              )}
            >
              <span
                className={cn(
                  "h-2 w-2 shrink-0 rounded-full",
                  getStatusColor(inst.status)
                )}
                title={inst.status}
              />
              {!sidebarCollapsed && (
                <span className="truncate text-sm">{inst.name}</span>
              )}
            </button>
          ))}
        </div>
      </div>

      {/* Bottom section */}
      <div
        className={cn(
          "border-t border-border p-2",
          sidebarCollapsed ? "flex flex-col items-center gap-1" : "space-y-0.5"
        )}
      >
        <NavLink
          to="/settings"
          className={({ isActive }) =>
            cn(
              "flex items-center gap-2 rounded-md px-2 py-2 text-sm transition-colors",
              isActive
                ? "bg-accent text-accent-foreground"
                : "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
              sidebarCollapsed && "justify-center px-0"
            )
          }
        >
          <Settings className="h-4 w-4 shrink-0" />
          {!sidebarCollapsed && <span>Settings</span>}
        </NavLink>
        <div
          className={cn(
            "flex items-center gap-2 rounded-md px-2 py-2 text-sm text-muted-foreground",
            sidebarCollapsed && "justify-center px-0"
          )}
        >
          <Info className="h-4 w-4 shrink-0" />
          {!sidebarCollapsed && <span>About</span>}
        </div>

        <Button
          variant="ghost"
          size="icon"
          className="mt-2 h-8 w-8 shrink-0"
          onClick={toggleSidebar}
          title={sidebarCollapsed ? "Expand sidebar" : "Collapse sidebar"}
        >
          {sidebarCollapsed ? (
            <ChevronRight className="h-4 w-4" />
          ) : (
            <ChevronLeft className="h-4 w-4" />
          )}
        </Button>
      </div>
    </aside>
  );
}
