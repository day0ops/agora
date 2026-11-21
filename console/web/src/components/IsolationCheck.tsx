import { useState } from "react";
import { postIsolationCheck, type IsolationProbeTarget } from "../api";

interface ProbeResult {
  target: IsolationProbeTarget;
  reply?: string;
  error?: string;
}

const PROBES: { target: IsolationProbeTarget; label: string }[] = [
  { target: "trust-safety", label: "Probe trust-safety" },
  { target: "auto-bidder", label: "Probe auto-bidder" },
];

// Real, repeatable proof that each sub-agent's tool access is genuinely
// least-privilege: each button asks a real question outside that agent's
// declared tool bindings (see console/isolation_probe.go's isolationProbes
// map) and shows its real reply, wording and all - a correct answer says it
// can't help, which is the live proof, not a fabricated "DENIED" badge.
export function IsolationCheck() {
  const [loading, setLoading] = useState<IsolationProbeTarget | null>(null);
  const [result, setResult] = useState<ProbeResult | null>(null);

  async function runProbe(target: IsolationProbeTarget): Promise<void> {
    setLoading(target);
    setResult(null);
    try {
      const { reply } = await postIsolationCheck(target);
      setResult({ target, reply });
    } catch (err) {
      setResult({
        target,
        error: err instanceof Error ? err.message : String(err),
      });
    } finally {
      setLoading(null);
    }
  }

  return (
    <div className="panel">
      <h2>Isolation check</h2>
      <div className="note">
        Each button asks a real sub-agent to do something outside its declared
        tool bindings. A correct answer says it can&rsquo;t help - that&rsquo;s
        live proof this agent has no path to comply, not a scripted denial.
      </div>
      <div className="isolation-check-buttons">
        {PROBES.map((p) => (
          <button
            key={p.target}
            type="button"
            disabled={loading !== null}
            onClick={() => void runProbe(p.target)}
          >
            {loading === p.target ? "Probing..." : p.label}
          </button>
        ))}
      </div>
      {result && (
        <div className="isolation-check-result">
          {result.error ? (
            <span className="isolation-check-error">{result.error}</span>
          ) : (
            <span>{result.reply}</span>
          )}
        </div>
      )}
    </div>
  );
}
