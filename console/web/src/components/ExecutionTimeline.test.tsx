import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { ExecutionTimeline } from "./ExecutionTimeline";

describe("ExecutionTimeline", () => {
  it("renders the not-connected banner when status is empty", () => {
    render(
      <ExecutionTimeline
        result={{ status: "empty", reason: "No live substrate." }}
      />,
    );
    const banner = screen.getByText("No live substrate.");
    expect(banner).toBeInTheDocument();
    expect(banner).not.toHaveClass("error");
  });

  it("renders a real error state with a visibly distinct class from the empty banner", () => {
    render(
      <ExecutionTimeline
        result={{ status: "error", message: "execution adapter unreachable" }}
      />,
    );
    const banner = screen.getByText("execution adapter unreachable");
    expect(banner).toBeInTheDocument();
    expect(banner).toHaveClass("error");
  });

  it("renders each step when status is real", () => {
    render(
      <ExecutionTimeline
        result={{
          status: "real",
          data: {
            steps: [
              {
                actor: "auto-bidder",
                state: "x",
                pool: "p",
                mcpCall: "c",
                policy: "pol",
              },
            ],
          },
        }}
      />,
    );
    expect(screen.getByText("auto-bidder")).toBeInTheDocument();
  });
});
