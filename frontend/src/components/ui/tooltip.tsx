import * as React from "react";
import { cn } from "@/lib/utils";

export interface TooltipProps
  extends Omit<React.HTMLAttributes<HTMLDivElement>, "content"> {
  content: React.ReactNode;
  side?: "top" | "bottom" | "left" | "right";
}

const Tooltip = React.forwardRef<HTMLDivElement, TooltipProps>(
  ({ className, content, side = "top", children, ...props }, ref) => {
    const [isVisible, setIsVisible] = React.useState(false);
    const containerRef = React.useRef<HTMLDivElement>(null);

    const positionClasses = {
      top: "bottom-full left-1/2 -translate-x-1/2 -translate-y-2 mb-1",
      bottom: "top-full left-1/2 -translate-x-1/2 translate-y-2 mt-1",
      left: "right-full top-1/2 -translate-y-1/2 -translate-x-2 mr-1",
      right: "left-full top-1/2 -translate-y-1/2 translate-x-2 ml-1",
    };

    return (
      <div
        ref={(el) => {
          (containerRef as React.MutableRefObject<HTMLDivElement | null>).current = el;
          if (typeof ref === "function") ref(el);
          else if (ref) ref.current = el;
        }}
        className={cn("relative inline-block", className)}
        onMouseEnter={() => setIsVisible(true)}
        onMouseLeave={() => setIsVisible(false)}
        {...props}
      >
        {children}
        {isVisible && (
          <div
            className={cn(
              "absolute z-50 whitespace-nowrap rounded-md border border-border bg-popover px-3 py-1.5 text-sm text-popover-foreground shadow-md",
              positionClasses[side]
            )}
          >
            {content}
          </div>
        )}
      </div>
    );
  }
);
Tooltip.displayName = "Tooltip";

export { Tooltip };
