import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { SubstrateTopology } from "./SubstrateTopology";

describe("SubstrateTopology", () => {
  it("renders the not-connected banner when status is empty", () => {
    render(
      <SubstrateTopology
        result={{
          status: "empty",
          reason: "Not connected to a live substrate.",
        }}
      />,
    );
    expect(
      screen.getByText("Not connected to a live substrate."),
    ).toBeInTheDocument();
  });

  it("renders one row per actor when status is real", () => {
    render(
      <SubstrateTopology
        result={{
          status: "real",
          data: {
            actors: [
              {
                id: "agora-workers-0",
                state: "running",
                pool: "agora-workers",
                lastActivity: "2s ago",
              },
            ],
            workers: [],
          },
        }}
      />,
    );
    expect(screen.getByText("agora-workers-0")).toBeInTheDocument();
  });

  it("renders a real error state distinctly from empty", () => {
    render(
      <SubstrateTopology
        result={{ status: "error", message: "ate-api unreachable" }}
      />,
    );
    expect(screen.getByText(/ate-api unreachable/)).toBeInTheDocument();
  });

  it("renders worker occupancy alongside actors", () => {
    render(
      <SubstrateTopology
        result={{
          status: "real",
          data: {
            actors: [
              {
                id: "trust-safety-1",
                state: "running",
                pool: "agora-workers",
                lastActivity: "2026-11-22T10:00:00Z",
              },
            ],
            workers: [
              {
                name: "agora-workers-0",
                pool: "agora-workers",
                state: "active",
                occupancy: "3/10",
              },
            ],
          },
        }}
      />,
    );
    expect(screen.getByText("3/10")).toBeInTheDocument();
  });
});
