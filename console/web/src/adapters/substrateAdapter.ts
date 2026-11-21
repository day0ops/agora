import type { AdapterResult } from "./types";
import { ApiError, getSubstrateStatusRaw } from "../api";

export interface SubstrateActor {
  id: string;
  state: string;
  pool: string;
  lastActivity: string;
}

export interface SubstrateWorker {
  name: string;
  pool: string;
  state: string;
  occupancy: string;
}

export interface SubstrateStatus {
  actors: SubstrateActor[];
  workers: SubstrateWorker[];
}

// Real when a live substrate is configured on this deployment (backend
// returns actors/workers, or a 501 "not configured" which maps to the
// honest-empty state below); a genuine request failure against a
// *configured* endpoint (any other non-2xx, or a network error) is reported
// as "error", never silently collapsed into "empty" - those are different
// claims (section 2.1's honesty rule).
export async function getSubstrateStatus(): Promise<
  AdapterResult<SubstrateStatus>
> {
  try {
    const data = await getSubstrateStatusRaw();
    return { status: "real", data };
  } catch (err) {
    if (err instanceof ApiError) {
      if (err.status === 501) {
        return { status: "empty", reason: err.message };
      }
      return { status: "error", message: err.message };
    }
    return {
      status: "error",
      message: err instanceof Error ? err.message : String(err),
    };
  }
}
