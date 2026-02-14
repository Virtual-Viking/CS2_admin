import { Minus, Square, X } from "lucide-react";
import { cn } from "@/lib/utils";

const isWails =
  typeof window !== "undefined" &&
  typeof (window as any).runtime?.WindowMinimise === "function";

function handleMinimize() {
  (window as any).runtime?.WindowMinimise();
}

function handleMaximize() {
  (window as any).runtime?.WindowToggleMaximise();
}

function handleClose() {
  (window as any).runtime?.Quit();
}

export function TitleBar() {
  return (
    <header
      className={cn(
        "flex h-8 min-h-[32px] shrink-0 items-center justify-between border-b border-border bg-[hsl(222.2,84%,3.5%)] px-3",
        "select-none"
      )}
      style={{ WebkitAppRegion: "drag" } as React.CSSProperties}
    >
      <div className="flex items-center gap-2">
        <div className="h-5 w-5 rounded bg-primary/80" />
        <span className="text-sm font-medium text-foreground">CS2 Admin</span>
      </div>

      {isWails && (
        <div
          className="flex items-center -mr-1"
          style={{ WebkitAppRegion: "no-drag" } as React.CSSProperties}
        >
          <button
            type="button"
            onClick={handleMinimize}
            className="flex h-8 w-12 items-center justify-center text-muted-foreground transition-colors hover:bg-white/10 hover:text-foreground"
          >
            <Minus className="h-4 w-4" strokeWidth={2.5} />
          </button>
          <button
            type="button"
            onClick={handleMaximize}
            className="flex h-8 w-12 items-center justify-center text-muted-foreground transition-colors hover:bg-white/10 hover:text-foreground"
          >
            <Square className="h-3.5 w-3.5" strokeWidth={2.5} />
          </button>
          <button
            type="button"
            onClick={handleClose}
            className="flex h-8 w-12 items-center justify-center text-muted-foreground transition-colors hover:bg-red-500/80 hover:text-white"
          >
            <X className="h-4 w-4" strokeWidth={2.5} />
          </button>
        </div>
      )}
    </header>
  );
}
