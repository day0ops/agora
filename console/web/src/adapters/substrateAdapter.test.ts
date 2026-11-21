import { describe, it, expect, vi, afterEach } from "vitest";
import { getSubstrateStatus } from "./substrateAdapter";

describe("getSubstrateStatus", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("returns empty when the endpoint reports not configured (501)", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 501,
        json: async () => ({
          error: "substrate status not configured on this deployment",
        }),
      }),
    );
    const result = await getSubstrateStatus();
    expect(result.status).toBe("empty");
  });

  it("returns error (not empty) on a real failure from a configured endpoint", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 502,
        json: async () => ({
          error: "listing actors: kubectl-ate: connection refused",
        }),
      }),
    );
    const result = await getSubstrateStatus();
    expect(result.status).toBe("error");
    if (result.status === "error") {
      expect(result.message).toContain("connection refused");
    }
  });

  it("returns real data on success", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({
          actors: [
            {
              id: "trust-safety-1",
              state: "running",
              pool: "agora-workers",
              lastActivity: "...",
            },
          ],
          workers: [],
        }),
      }),
    );
    const result = await getSubstrateStatus();
    expect(result.status).toBe("real");
    if (result.status === "real") {
      expect(result.data.actors).toHaveLength(1);
    }
  });
});
