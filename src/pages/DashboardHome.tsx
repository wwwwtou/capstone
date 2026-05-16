import React, { useState, useEffect } from "react";
import { recSysService } from "../services/api";
import { Activity, Zap, Server, Database, Layers, Globe } from "lucide-react";

export default function DashboardHome() {
  const [health, setHealth] = useState<any>(null);

  useEffect(() => {
    recSysService.getHealth().then(res => setHealth(res.data));
  }, []);

  return (
    <div className="space-y-8 animate-in fade-in duration-500">
      <div className="grid grid-cols-4 gap-6">
        <MetricCard 
          label="Global Throughput" 
          value={health?.metrics.throughput_rps + " RPS"} 
          icon={<Zap className="text-amber-400 w-5 h-5"/>} 
          status="Nominal"
        />
        <MetricCard 
          label="P99 Latency" 
          value={health?.metrics.avg_p99_latency_ms + " ms"} 
          icon={<Activity className="text-indigo-400 w-5 h-5"/>} 
          status="Optimized"
        />
        <MetricCard 
          label="Redis Shards" 
          value={health?.instances.redis_shards} 
          icon={<Layers className="text-emerald-400 w-5 h-5"/>} 
          status="Clustered"
        />
        <MetricCard 
          label="Uptime" 
          value="99.99%" 
          icon={<Globe className="text-sky-400 w-5 h-5"/>} 
          status="High Availability"
        />
      </div>

      <div className="grid grid-cols-3 gap-8">
        <div className="col-span-2 bg-slate-900/50 border border-slate-800 rounded-3xl p-8">
          <h3 className="font-bold text-lg mb-6 flex items-center gap-2">
            <Server className="w-5 h-5 text-slate-400" />
            Microservice Topology
          </h3>
          <div className="space-y-4">
             <InstanceRow name="Recommendation Core (Go)" status={health?.instances.rec_service_go} load="24%" />
             <InstanceRow name="Management Dashboard (React)" status={health?.instances.dashboard_fe} load="12%" />
             <InstanceRow name="PostgreSQL Cluster" status={health?.instances.postgres_primary} load="38%" />
          </div>
        </div>

        <div className="bg-gradient-to-br from-indigo-600 to-indigo-900 rounded-3xl p-8 text-white shadow-xl shadow-indigo-500/20">
           <h3 className="font-bold text-lg mb-4 flex items-center gap-2">
             <Database className="w-5 h-5" />
             Data Storage
           </h3>
           <p className="text-indigo-100 text-sm leading-relaxed mb-6">
             The system utilizes **PostgreSQL** for persistent metadata and **Redis** for sub-millisecond user profile retrieval.
           </p>
           <div className="space-y-3">
              <div className="flex justify-between text-xs font-bold">
                 <span>Cache Hit Rate</span>
                 <span>94.2%</span>
              </div>
              <div className="w-full h-2 bg-white/20 rounded-full overflow-hidden">
                 <div className="h-full bg-white w-[94.2%]" />
              </div>
           </div>
        </div>
      </div>
    </div>
  );
}

function MetricCard({ label, value, icon, status }: any) {
  return (
    <div className="bg-slate-900/50 border border-slate-800 p-6 rounded-3xl hover:bg-slate-800/50 transition-all group">
      <div className="flex justify-between items-start mb-4">
        <div className="p-2 bg-slate-800 rounded-xl">{icon}</div>
        <span className="text-[10px] font-bold text-slate-500 uppercase tracking-widest">{status}</span>
      </div>
      <div className="text-2xl font-black tracking-tight">{value}</div>
      <div className="text-xs text-slate-500 mt-1 font-medium italic">{label}</div>
    </div>
  );
}

function InstanceRow({ name, status, load }: any) {
  return (
    <div className="flex items-center justify-between p-4 bg-slate-950/50 border border-slate-800 rounded-2xl">
      <div className="flex items-center gap-4">
        <div className={`w-2.5 h-2.5 rounded-full ${status === 'UP' || status === 'ACTIVE' ? 'bg-emerald-500' : 'bg-slate-600'} shadow-[0_0_8px_rgba(16,185,129,0.5)]`} />
        <span className="font-bold text-sm">{name}</span>
      </div>
      <div className="flex gap-8 items-center">
         <span className="text-xs text-slate-500 font-mono">CPU: {load}</span>
         <span className="text-[10px] font-black text-slate-400 bg-slate-800 px-2 py-0.5 rounded uppercase">{status}</span>
      </div>
    </div>
  );
}
