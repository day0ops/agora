import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { TechnicalInspector } from "./TechnicalInspector";

describe("TechnicalInspector", () => {
  it("renders the not-connected banner spanning every column when status is empty", () => {
    render(
      <TechnicalInspector
        result={{ status: "empty", reason: "No live substrate." }}
      />,
    );
    const banner = screen.getByText("No live substrate.");
    expect(banner).toBeInTheDocument();
    expect(banner).not.toHaveClass("error");
    expect(screen.getAllByRole("columnheader")).toHaveLength(5);
  });

  it("renders a real error state with a visibly distinct class from the empty banner", () => {
    render(
      <TechnicalInspector
        result={{ status: "error", message: "execution adapter unreachable" }}
      />,
    );
    const banner = screen.getByText("execution adapter unreachable");
    expect(banner).toBeInTheDocument();
    expect(banner).toHaveClass("error");
  });

  it("renders one row per step when status is real", () => {
    render(
      <TechnicalInspector
        result={{
          status: "real",
          data: {
            steps: [
              {
                actor: "auto-bidder",
                state: "done",
                pool: "agora-workers",
                mcpCall: "placeBid",
                policy: "allow",
              },
            ],
          },
        }}
      />,
    );
    expect(screen.getByText("auto-bidder")).toBeInTheDocument();
    expect(screen.getByText("placeBid")).toBeInTheDocument();
  });
});
