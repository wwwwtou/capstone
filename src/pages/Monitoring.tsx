import React, { useState, useEffect, useRef } from "react";
import { Activity, Gauge, ShieldAlert, Database, Server, Timer } from "lucide-react";
import { recSysService } from "../services/api";
import TrafficControls from "../components/TrafficControls";

// Live observability dashboard. Polls the aggregated /api/v1/metrics feed
// (gateway + downstream services, or the BFF mock replica) every 2 seconds and
// renders rolling charts. Counters are cumulative, so rates (QPS, error rate)
// are computed from deltas between consecutive samples — the Prometheus way.

const POLL_MS = 2000;
const MAX_POINTS = 90;

type Sample = {
  t: number;
  total: number;
  errors: number;
  qps: number;
  errRate: number;
  p50: number;
  p99: number;
};

function sumErrors(status: Record<string, number> | undefined): number {
  if (!status) return 0;
  return (status["4xx"] || 0) + (status["5xx"] || 0);
}

export default function Monitoring() {
  const [samples, setSamples] = useState<Sample[]>([]);
  const [latest, setLatest] = useState<any>(null);
  const [down, setDown] = useState(false);
  const prevRef = useRef<{ t: number; total: number; errors: number } | null>(null);

  useEffect(() => {
    let alive = true;
    const poll = async () => {
      try {
        const res = await recSysService.getMetrics();
        if (!alive) return;
        const data = res.data;
        setLatest(data);
        setDown(false);

        const gw = data?.gateway;
        if (!gw) return;
        const t = data.ts || Date.now();
        const total = gw.requests_total || 0;
        const errors = sumErrors(gw.status);
        const prev = prevRef.current;
        prevRef.current = { t, total, errors };
        if (!prev || t <= prev.t) return;

        const dt = (t - prev.t) / 1000;
        const dReq = Math.max(0, total - prev.total);
        const dErr = Math.max(0, errors - prev.errors);
        const sample: Sample = {
          t,
          total,
          errors,
          qps: dReq / dt,
          errRate: dReq > 0 ? (dErr / dReq) * 100 : 0,
          p50: gw.latency_ms?.p50 ?? 0,
          p99: gw.latency_ms?.p99 ?? 0,
        };
        setSamples((prevS) => [...prevS, sample].slice(-MAX_POINTS));
      } catch {
        if (alive) setDown(true);
      }
    };
    poll();
    const timer = setInterval(poll, POLL_MS);
    return () => {
      alive = false;
      clearInterval(timer);
    };
  }, []);

  const gw = latest?.gateway;
  const services: Record<string, any> = latest?.services ?? {};
  const last = samples[samples.length - 1];

  // Breaker states can come from the gateway (proxy mode) and from the
  // recommendation service's own outbound breakers.
  const breakers: Record<string, string> = {};
  for (const [src, snap] of [["gateway", gw], ["recommendation", services.recommendation]] as const) {
    for (const [k, v] of Object.entries(snap?.gauges ?? {})) {
      if (k.startsWith("breaker_")) breakers[`${src}→${k.replace("breaker_", "")}`] = v as string;
    }
  }

  const userCounters = services.user?.counters ?? {};
  const hits = userCounters.cache_hits || 0;
  const misses = userCounters.cache_misses || 0;
  const hitRate = hits + misses > 0 ? (hits / (hits + misses)) * 100 : null;

  return (
    <div className="space-y-8 animate-in fade-in duration-500">
      <TrafficControls />

      {down && (
        <div className="bg-rose-500/10 border border-rose-500/40 text-rose-300 text-xs font-bold px-6 py-4 rounded-2xl">
          Metrics feed unreachable — the gateway may be down.
        </div>
      )}

      {/* Headline stats */}
      <div className="grid grid-cols-4 gap-6">
        <StatCard
          icon={<Gauge className="w-5 h-5 text-amber-400" />}
          label="Throughput (edge)"
          value={last ? `${last.qps.toFixed(1)} req/s` : "—"}
          sub={gw ? `${gw.requests_total} total` : ""}
        />
        <StatCard
          icon={<Timer className="w-5 h-5 text-indigo-400" />}
          label="Latency p99 / p50"
          value={gw ? `${gw.latency_ms?.p99 ?? 0} ms` : "—"}
          sub={gw ? `p50 ${gw.latency_ms?.p50 ?? 0} ms · max ${gw.latency_ms?.max ?? 0} ms` : ""}
        />
        <StatCard
          icon={<ShieldAlert className="w-5 h-5 text-rose-400" />}
          label="Error Rate (window)"
          value={last ? `${last.errRate.toFixed(1)} %` : "—"}
          sub={gw ? `4xx ${gw.status?.["4xx"] || 0} · 5xx ${gw.status?.["5xx"] || 0}` : ""}
        />
        <StatCard
          icon={<Database className="w-5 h-5 text-emerald-400" />}
          label="Profile Cache Hit Rate"
          value={hitRate === null ? "—" : `${hitRate.toFixed(1)} %`}
          sub={hitRate === null ? "no traffic yet" : `${hits} hits · ${misses} misses`}
        />
      </div>

      {/* Charts */}
      <div className="grid grid-cols-2 gap-8">
        <ChartPanel title="Requests per second" subtitle="edge traffic, 2s samples">
          <Sparkline
            series={[{ points: samples.map((s) => s.qps), color: "#818cf8", fill: true }]}
            unit="req/s"
          />
        </ChartPanel>
        <ChartPanel title="Latency percentiles (ms)" subtitle="p99 (violet) vs p50 (emerald)">
          <Sparkline
            series={[
              { points: samples.map((s) => s.p99), color: "#a78bfa", fill: true },
              { points: samples.map((s) => s.p50), color: "#34d399" },
            ]}
            unit="ms"
          />
        </ChartPanel>
      </div>

      {/* Circuit breakers */}
      <div className="bg-slate-900/50 border border-slate-800 p-8 rounded-3xl">
        <h4 className="text-xs font-bold text-slate-500 uppercase tracking-widest mb-5 flex items-center gap-2">
          <Activity className="w-4 h-4" />
          Circuit Breakers
        </h4>
        {Object.keys(breakers).length === 0 ? (
          <p className="text-xs text-slate-600">No breaker telemetry yet.</p>
        ) : (
          <div className="flex flex-wrap gap-3" data-testid="breaker-badges">
            {Object.entries(breakers).map(([name, state]) => (
              <span
                key={name}
                className={`px-4 py-2 rounded-xl text-xs font-bold border ${
                  state === "closed"
                    ? "bg-emerald-500/10 border-emerald-500/40 text-emerald-300"
                    : state === "half-open"
                      ? "bg-amber-500/10 border-amber-500/40 text-amber-300"
                      : "bg-rose-500/10 border-rose-500/40 text-rose-300"
                }`}
              >
                {name}: {state.toUpperCase()}
              </span>
            ))}
          </div>
        )}
      </div>

      {/* Per-service table */}
      <div className="bg-slate-900/50 border border-slate-800 p-8 rounded-3xl">
        <h4 className="text-xs font-bold text-slate-500 uppercase tracking-widest mb-5 flex items-center gap-2">
          <Server className="w-4 h-4" />
          Per-Service Metrics {latest?.mode === "mock" && <span className="text-slate-600">(mock replica)</span>}
        </h4>
        <table className="w-full text-xs" data-testid="service-table">
          <thead>
            <tr className="text-left text-slate-600 uppercase tracking-widest text-[10px]">
              <th className="pb-3">Service</th>
              <th className="pb-3">Requests</th>
              <th className="pb-3">p50</th>
              <th className="pb-3">p99</th>
              <th className="pb-3">5xx</th>
              <th className="pb-3">Uptime</th>
              <th className="pb-3">Status</th>
            </tr>
          </thead>
          <tbody className="font-mono">
            {["user", "content", "recommendation"].map((name) => {
              const s = services[name];
              return (
                <tr key={name} className="border-t border-slate-800/60">
                  <td className="py-3 font-bold text-slate-300">{name}</td>
                  <td className="py-3 text-slate-400">{s ? s.requests_total : "—"}</td>
                  <td className="py-3 text-slate-400">{s ? `${s.latency_ms?.p50 ?? 0} ms` : "—"}</td>
                  <td className="py-3 text-slate-400">{s ? `${s.latency_ms?.p99 ?? 0} ms` : "—"}</td>
                  <td className="py-3 text-slate-400">{s ? s.status?.["5xx"] || 0 : "—"}</td>
                  <td className="py-3 text-slate-400">{s ? `${Math.round(s.uptime_s)}s` : "—"}</td>
                  <td className="py-3">
                    <span
                      className={`px-2 py-0.5 rounded text-[10px] font-black ${
                        s ? "bg-emerald-500/10 text-emerald-400" : "bg-rose-500/10 text-rose-400"
                      }`}
                    >
                      {s ? "UP" : "DOWN"}
                    </span>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function StatCard({ icon, label, value, sub }: { icon: React.ReactNode; label: string; value: string; sub?: string }) {
  return (
    <div className="bg-slate-900/50 border border-slate-800 p-6 rounded-3xl">
      <div className="flex justify-between items-start mb-4">
        <div className="p-2 bg-slate-800 rounded-xl">{icon}</div>
      </div>
      <div className="text-2xl font-black tracking-tight">{value}</div>
      <div className="text-xs text-slate-500 mt-1 font-medium">{label}</div>
      {sub && <div className="text-[10px] text-slate-600 mt-1 font-mono">{sub}</div>}
    </div>
  );
}

function ChartPanel({ title, subtitle, children }: { title: string; subtitle: string; children: React.ReactNode }) {
  return (
    <div className="bg-slate-900/50 border border-slate-800 p-8 rounded-3xl">
      <div className="flex items-baseline justify-between mb-6">
        <h4 className="font-bold text-sm text-slate-300">{title}</h4>
        <span className="text-[10px] text-slate-600 font-bold uppercase tracking-widest">{subtitle}</span>
      </div>
      {children}
    </div>
  );
}

// Dependency-free SVG chart: one or more series scaled to the shared max.
function Sparkline({
  series,
  unit,
}: {
  series: Array<{ points: number[]; color: string; fill?: boolean }>;
  unit: string;
}) {
  const W = 560;
  const H = 140;
  const n = Math.max(...series.map((s) => s.points.length), 0);
  const max = Math.max(1e-6, ...series.flatMap((s) => s.points));

  if (n < 2) {
    return (
      <div className="h-[140px] flex items-center justify-center text-xs text-slate-600 font-bold">
        Collecting samples…
      </div>
    );
  }

  const toPath = (pts: number[], close: boolean) => {
    const coords = pts.map((v, i) => `${((i / (n - 1)) * W).toFixed(1)},${(H - (v / max) * (H - 8)).toFixed(1)}`);
    let d = `M${coords[0]} L${coords.slice(1).join(" L")}`;
    if (close) d += ` L${W},${H} L0,${H} Z`;
    return d;
  };

  return (
    <div className="relative">
      <svg viewBox={`0 0 ${W} ${H}`} className="w-full h-[140px]">
        {[0.25, 0.5, 0.75].map((f) => (
          <line key={f} x1={0} x2={W} y1={H * f} y2={H * f} stroke="#1e293b" strokeWidth={1} />
        ))}
        {series.map((s, i) =>
          s.points.length >= 2 ? (
            <g key={i}>
              {s.fill && <path d={toPath(s.points, true)} fill={s.color} opacity={0.12} />}
              <path d={toPath(s.points, false)} fill="none" stroke={s.color} strokeWidth={2} />
            </g>
          ) : null
        )}
      </svg>
      <span className="absolute top-0 right-0 text-[10px] font-mono text-slate-500">
        max {max >= 10 ? Math.round(max) : max.toFixed(1)} {unit}
      </span>
    </div>
  );
}
