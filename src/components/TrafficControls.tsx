import React, { useState, useEffect, useRef } from "react";
import { Play, Square, Flame, Loader2 } from "lucide-react";
import { recSysService } from "../services/api";

// One-click demo traffic: a continuous synthetic-load toggle plus a burst
// button. The load is generated server-side (BFF), so it keeps running while
// you navigate between dashboard pages.
export default function TrafficControls() {
  const [enabled, setEnabled] = useState(false);
  const [rps, setRps] = useState(8);
  const [sent, setSent] = useState(0);
  const [errors, setErrors] = useState(0);
  const [bursting, setBursting] = useState(false);
  const [burstResult, setBurstResult] = useState<any>(null);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const refreshStatus = async () => {
    try {
      const res = await recSysService.getTrafficStatus();
      const d = res.data?.data;
      if (d) {
        setEnabled(d.enabled);
        setSent(d.sent);
        setErrors(d.errors);
        if (d.enabled) setRps(d.rps);
      }
    } catch {
      /* status endpoint unavailable */
    }
  };

  useEffect(() => {
    refreshStatus();
    pollRef.current = setInterval(refreshStatus, 2000);
    return () => {
      if (pollRef.current) clearInterval(pollRef.current);
    };
  }, []);

  const toggle = async () => {
    const next = !enabled;
    setEnabled(next);
    try {
      await recSysService.setTraffic({ enabled: next, rps });
    } catch {
      setEnabled(!next);
    }
  };

  const burst = async () => {
    setBursting(true);
    setBurstResult(null);
    try {
      const res = await recSysService.runBurst({ count: 300, concurrency: 25 });
      setBurstResult(res.data?.data);
    } catch {
      /* burst failed */
    } finally {
      setBursting(false);
    }
  };

  return (
    <div className="bg-slate-900/50 border border-slate-800 p-6 rounded-3xl space-y-4">
      <div className="flex items-center justify-between gap-4 flex-wrap">
        <div>
          <h4 className="text-xs font-bold text-slate-400 uppercase tracking-widest">Demo Traffic Generator</h4>
          <p className="text-[11px] text-slate-600 mt-1">
            Synthetic load fired server-side through the full request chain.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <button
            data-testid="traffic-toggle"
            onClick={toggle}
            className={`flex items-center gap-2 px-5 py-2.5 rounded-xl text-xs font-bold transition-all active:scale-95 border ${
              enabled
                ? "bg-rose-600/90 border-rose-500 hover:bg-rose-500 text-white"
                : "bg-emerald-600/90 border-emerald-500 hover:bg-emerald-500 text-white"
            }`}
          >
            {enabled ? <Square className="w-4 h-4" /> : <Play className="w-4 h-4" />}
            {enabled ? "Stop Traffic" : "Start Traffic"}
          </button>
          <button
            data-testid="burst-button"
            onClick={burst}
            disabled={bursting}
            className="flex items-center gap-2 px-5 py-2.5 rounded-xl text-xs font-bold bg-amber-600/90 border border-amber-500 hover:bg-amber-500 disabled:opacity-50 text-white transition-all active:scale-95"
          >
            {bursting ? <Loader2 className="w-4 h-4 animate-spin" /> : <Flame className="w-4 h-4" />}
            Burst 300
          </button>
        </div>
      </div>

      <div className="flex items-center gap-6 flex-wrap">
        <label className="flex items-center gap-3 text-xs text-slate-400 font-bold">
          Rate
          <input
            type="range"
            min={1}
            max={30}
            value={rps}
            disabled={enabled}
            onChange={(e) => setRps(Number(e.target.value))}
            className="w-36 accent-indigo-500"
          />
          <span className="text-indigo-300 font-black w-14">{rps} rps</span>
        </label>
        <span className="text-xs font-mono text-slate-500" data-testid="traffic-sent">
          sent: <span className="text-emerald-400 font-bold">{sent}</span>
          {errors > 0 && (
            <>
              {" "}errors: <span className="text-rose-400 font-bold">{errors}</span>
            </>
          )}
        </span>
        {burstResult && (
          <span className="text-xs font-mono text-slate-500" data-testid="burst-result">
            burst: <span className="text-amber-300 font-bold">{burstResult.achieved_rps} rps</span>, p99{" "}
            <span className="text-amber-300 font-bold">{burstResult.latency_ms?.p99} ms</span>,{" "}
            {burstResult.errors} errors
          </span>
        )}
      </div>
    </div>
  );
}
