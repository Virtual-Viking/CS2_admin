import { useRef, useEffect, useState, useCallback } from "react";
import { Copy, Terminal } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useAppStore } from "@/stores/app-store";
import { cn } from "@/lib/utils";
import { RCON_FOCUS_EVENT_NAME } from "@/hooks/useKeyboardShortcuts";

export interface ConsoleTabProps {
  instanceId: string;
}

export function ConsoleTab({ instanceId }: ConsoleTabProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const [input, setInput] = useState("");
  const [history, setHistory] = useState<string[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);

  const consoleLogs = useAppStore((s) => s.consoleLogs);
  const appendConsoleLog = useAppStore((s) => s.appendConsoleLog);
  const clearConsoleLogs = useAppStore((s) => s.clearConsoleLogs);

  const lines = consoleLogs[instanceId] ?? [];

  // Focus RCON input on Ctrl+R (global shortcut)
  useEffect(() => {
    const handler = () => inputRef.current?.focus();
    window.addEventListener(RCON_FOCUS_EVENT_NAME, handler);
    return () => window.removeEventListener(RCON_FOCUS_EVENT_NAME, handler);
  }, []);

  // Subscribe to console events
  useEffect(() => {
    if (!instanceId) return;
    const eventName = `console:${instanceId}`;
    const cb = (line: unknown) => {
      appendConsoleLog(instanceId, String(line ?? ""));
    };
    try {
      (window as any).runtime?.EventsOn?.(eventName, cb);
    } catch {
      // no-op
    }
    return () => {
      try {
        (window as any).runtime?.EventsOff?.(eventName);
      } catch {
        // no-op
      }
    };
  }, [instanceId, appendConsoleLog]);

  // Auto-scroll to bottom on new output
  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    el.scrollTop = el.scrollHeight;
  }, [lines.length]);

  const sendCommand = useCallback(() => {
    const cmd = input.trim();
    if (!cmd) return;
    setInput("");
    setHistory((h) => [...h, cmd].slice(-100));
    setHistoryIndex(-1);
    try {
      (window as any).go?.main?.App?.SendRCON?.(instanceId, cmd);
    } catch {
      appendConsoleLog(instanceId, `> ${cmd}`);
      appendConsoleLog(instanceId, "Error: RCON not available");
    }
  }, [instanceId, input, appendConsoleLog]);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault();
      sendCommand();
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      if (history.length === 0) return;
      const next = historyIndex < 0 ? history.length - 1 : Math.max(0, historyIndex - 1);
      setHistoryIndex(next);
      setInput(history[next]);
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      if (historyIndex < 0) return;
      const next = historyIndex + 1;
      if (next >= history.length) {
        setHistoryIndex(-1);
        setInput("");
      } else {
        setHistoryIndex(next);
        setInput(history[next]);
      }
    }
  };

  const handleCopy = async () => {
    const text = lines.join("\n");
    if (!text) return;
    try {
      await navigator.clipboard.writeText(text);
    } catch {
      // fallback
    }
  };

  const handleClear = () => {
    clearConsoleLogs(instanceId);
  };

  return (
    <div className="flex h-full min-h-[400px] flex-col rounded-lg border border-border bg-zinc-950">
      {/* Toolbar */}
      <div className="flex items-center justify-between border-b border-border bg-zinc-900/80 px-3 py-2">
        <div className="flex items-center gap-2">
          <Terminal className="h-4 w-4 text-emerald-500" />
          <span className="text-sm font-medium">RCON Console</span>
        </div>
        <div className="flex gap-1">
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8 text-muted-foreground hover:text-foreground"
            onClick={handleCopy}
          >
            <Copy className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="h-8 text-muted-foreground hover:text-foreground"
            onClick={handleClear}
          >
            Clear
          </Button>
        </div>
      </div>

      {/* Output area */}
      <div
        ref={scrollRef}
        className={cn(
          "flex-1 overflow-auto p-3 font-mono text-[13px] leading-relaxed",
          "min-h-[300px] bg-black/40",
          "[&::-webkit-scrollbar]:w-2 [&::-webkit-scrollbar-track]:bg-zinc-900",
          "[&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-thumb]:bg-zinc-700"
        )}
      >
        {lines.length === 0 ? (
          <p className="text-zinc-500">No output yet. Start the server to see console logs.</p>
        ) : (
          lines.map((line, i) => (
            <div
              key={`${i}-${line.slice(0, 20)}`}
              className="whitespace-pre-wrap break-all text-zinc-300"
            >
              {line}
            </div>
          ))
        )}
      </div>

      {/* Input area */}
      <div className="border-t border-border bg-zinc-900/80 p-2">
        <div className="flex gap-2">
          <span className="flex items-center text-emerald-500">$</span>
          <input
            ref={inputRef}
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Enter RCON command..."
            className="flex-1 bg-transparent font-mono text-sm text-foreground placeholder:text-zinc-500 focus:outline-none"
          />
        </div>
        <p className="mt-1.5 text-xs text-zinc-500">
          Press Enter to send • Up/Down for command history
        </p>
      </div>
    </div>
  );
}
