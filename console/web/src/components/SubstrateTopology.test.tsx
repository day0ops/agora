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
    const banner = screen.getByText("Not connected to a live substrate.");
    expect(banner).toBeInTheDocument();
    expect(banner).not.toHaveClass("error");
  });

  it("renders one row per actor when status is real, labelled by template not id", () => {
    render(
      <SubstrateTopology
        result={{
          status: "real",
          data: {
            actors: [
              {
                id: "uuid-1",
                template: "trust-safety-agora-swarm-1",
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
    expect(screen.getByText("trust-safety-agora-swarm-1")).toBeInTheDocument();
    expect(screen.queryByText("uuid-1")).not.toBeInTheDocument();
  });

  it("renders a distinct row per actor even when several share the same template (the density case)", () => {
    render(
      <SubstrateTopology
        result={{
          status: "real",
          data: {
            actors: [
              {
                id: "uuid-1",
                template: "trust-safety-agora-swarm-1",
                state: "running",
                pool: "agora-workers",
                lastActivity: "2s ago",
              },
              {
                id: "uuid-2",
                template: "trust-safety-agora-swarm-1",
                state: "suspended",
                pool: "unassigned",
                lastActivity: "1m ago",
              },
            ],
            workers: [],
          },
        }}
      />,
    );
    expect(screen.getAllByText("trust-safety-agora-swarm-1")).toHaveLength(2);
  });

  it("renders a real error state with a visibly distinct class from the empty banner", () => {
    render(
      <SubstrateTopology
        result={{ status: "error", message: "ate-api unreachable" }}
      />,
    );
    const banner = screen.getByText(/ate-api unreachable/);
    expect(banner).toBeInTheDocument();
    expect(banner).toHaveClass("error");
  });

  it("renders worker occupancy alongside actors", () => {
    render(
      <SubstrateTopology
        result={{
          status: "real",
          data: {
            actors: [
              {
                id: "uuid-1",
                template: "trust-safety-1",
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
