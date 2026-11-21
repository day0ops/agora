import type { ReactNode } from "react";
import { NotConnectedBanner } from "./NotConnectedBanner";
import type { AdapterResult } from "../adapters/types";
import type { SubstrateStatus } from "../adapters/substrateAdapter";

const ACTOR_COLUMNS = ["Actor", "State", "Pool", "Last activity"] as const;
const WORKER_COLUMNS = ["Worker pod", "Pool", "State", "Occupancy"] as const;

function actorTable(body: ReactNode) {
  const header = (
    <tr>
      {ACTOR_COLUMNS.map((c) => (
        <th key={c}>{c}</th>
      ))}
    </tr>
  );
  return (
    <div className="inspector-table-wrap">
      <table className="inspector-table">
        <thead>{header}</thead>
        <tbody>{body}</tbody>
      </table>
    </div>
  );
}

// Three explicit branches, one per AdapterResult variant - "empty" (no live
// substrate configured) and "error" (a configured call that genuinely
// failed) render different messages and must never be conflated; only
// "real" ever reads result.data.
export function SubstrateTopology({
  result,
}: {
  result: AdapterResult<SubstrateStatus>;
}) {
  if (result.status === "empty") {
    return actorTable(
      <tr>
        <td colSpan={ACTOR_COLUMNS.length}>
          <NotConnectedBanner reason={result.reason} />
        </td>
      </tr>,
    );
  }
  if (result.status === "error") {
    // Deliberately NOT NotConnectedBanner (that's visually identical to the
    // "empty" case otherwise - final review finding #6): a configured
    // source that genuinely failed is a different claim from "not
    // connected", and must look different, not just read different.
    return actorTable(
      <tr>
        <td colSpan={ACTOR_COLUMNS.length}>
          <div className="note error">{result.message}</div>
        </td>
      </tr>,
    );
  }
  return (
    <>
      {actorTable(
        result.data.actors.map((a) => (
          <tr key={a.id}>
            <td>{a.template}</td>
            <td>{a.state}</td>
            <td>{a.pool}</td>
            <td>{a.lastActivity}</td>
          </tr>
        )),
      )}
      {result.data.workers.length > 0 && (
        <div className="inspector-table-wrap">
          <table className="inspector-table">
            <thead>
              <tr>
                {WORKER_COLUMNS.map((c) => (
                  <th key={c}>{c}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {result.data.workers.map((w) => (
                <tr key={w.name}>
                  <td>{w.name}</td>
                  <td>{w.pool}</td>
                  <td>{w.state}</td>
                  <td>{w.occupancy}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </>
  );
}
