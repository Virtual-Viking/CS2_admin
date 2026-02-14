import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useAppStore } from "@/stores/app-store";

const RCON_FOCUS_EVENT = "cs2admin:focus-rcon";

export function useKeyboardShortcuts() {
  const navigate = useNavigate();

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      // Ctrl+1 through Ctrl+9: switch instances
      if (e.ctrlKey && e.key >= "1" && e.key <= "9") {
        e.preventDefault();
        const idx = parseInt(e.key) - 1;
        const instances = useAppStore.getState().instances;
        if (instances[idx]) {
          navigate(`/instances/${instances[idx].id}`);
        }
      }

      // Ctrl+N: navigate to dashboard (new instance)
      if (e.ctrlKey && e.key.toLowerCase() === "n") {
        e.preventDefault();
        navigate("/");
      }

      // Ctrl+R: focus RCON input (dispatch custom event)
      if (e.ctrlKey && e.key.toLowerCase() === "r") {
        e.preventDefault();
        window.dispatchEvent(new CustomEvent(RCON_FOCUS_EVENT));
      }

      // F5: refresh
      if (e.key === "F5") {
        // Allow default (page refresh) - or prevent for SPA reload
        // For desktop Wails app, F5 typically reloads the webview
        // Leaving default behavior
      }
    };

    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [navigate]);
}

/** Event name dispatched when Ctrl+R is pressed to focus RCON input */
export const RCON_FOCUS_EVENT_NAME = RCON_FOCUS_EVENT;
