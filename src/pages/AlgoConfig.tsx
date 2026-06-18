import React, { useState, useEffect } from "react";
import { recSysService } from "../services/api";
import { Save, RefreshCcw, CheckCircle2 } from "lucide-react";

export default function AlgoConfig() {
  const [strategy, setStrategy] = useState("engagement");
  const [weight, setWeight] = useState(0.5);
  const [loading, setLoading] = useState(false);
  const [history, setHistory] = useState<any[]>([]);
  const [msg, setMsg] = useState("");

  useEffect(() => {
    fetchCurrent();
    fetchHistory();
  }, []);

  const fetchCurrent = async () => {
    try {
      const res = await recSysService.getConfigs();
      const { strategy_name, weight } = res.data.data;
      setStrategy(strategy_name);
      setWeight(weight);
    } catch (e) {
      console.error(e);
    }
  };

  const fetchHistory = async () => {
    try {
      const res = await recSysService.getConfigHistory();
      setHistory(res.data.data || []);
    } catch (e) {
      console.error(e);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      // Mock login first if token missing for demo
      if (!localStorage.getItem("admin_token")) {
        const loginRes = await recSysService.login();
        localStorage.setItem("admin_token", loginRes.data.data.token);
      }

      const res = await recSysService.updateConfigs({ strategy_name: strategy, weight });
      setMsg(res.data.message);
      // Reload the persisted deployment log from the DB so it survives navigation.
      await fetchHistory();
      setTimeout(() => setMsg(""), 3000);
    } catch (e: any) {
      setMsg("Deployment failed: " + (e.response?.data?.message || e.message));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
      <div className="bg-slate-900/50 border border-slate-800 p-8 rounded-3xl">
        <h3 className="text-xl font-bold mb-6 flex items-center gap-2">
            <RefreshCcw className="w-5 h-5 text-indigo-400"/>
            Active Ranking Strategy
        </h3>
        
        <form onSubmit={handleSubmit} className="space-y-8 max-w-2xl">
          <div className="grid grid-cols-2 gap-8">
            <div className="space-y-3">
              <label className="text-sm font-semibold text-slate-400">Strategy Model</label>
              <select 
                value={strategy}
                onChange={(e) => setStrategy(e.target.value)}
                className="w-full bg-slate-950 border border-slate-800 rounded-xl px-4 py-3 outline-none focus:border-indigo-500 transition-colors"
              >
                <option value="engagement">Engagement Strategy (ML)</option>
                <option value="chronological">Chronological (Recency)</option>
                <option value="diversity">Diversity Maximizer (Experimental)</option>
              </select>
            </div>

            <div className="space-y-3">
              <label className="text-sm font-semibold text-slate-400">Global Threshold Weight</label>
              <div className="flex items-center gap-4">
                <input 
                  type="range" 
                  min="0" max="1" step="0.05"
                  value={weight}
                  onChange={(e) => setWeight(parseFloat(e.target.value))}
                  className="flex-1 h-2 bg-slate-800 rounded-lg appearance-none cursor-pointer accent-indigo-500"
                />
                <span className="font-mono font-bold text-indigo-400">{(weight * 100).toFixed(0)}%</span>
              </div>
            </div>
          </div>

          <div className="pt-4 flex items-center gap-4">
            <button 
              disabled={loading}
              className="bg-indigo-600 hover:bg-indigo-500 px-8 py-3 rounded-2xl font-bold text-white flex items-center gap-2 transition-all active:scale-95 shadow-lg shadow-indigo-600/20 disabled:opacity-50"
            >
              {loading ? "Deploying..." : <><Save className="w-4 h-4"/> Deploy to Production</>}
            </button>
            {msg && (
              <span className={`text-xs font-bold px-3 py-1 rounded-full ${msg.includes("failed") ? "bg-rose-500/10 text-rose-400" : "bg-emerald-500/10 text-emerald-400"}`}>
                {msg}
              </span>
            )}
          </div>
        </form>
      </div>

      <div className="bg-slate-900/50 border border-slate-800 rounded-3xl overflow-hidden">
        <div className="px-8 py-4 bg-slate-800/30 border-b border-slate-800 font-bold text-sm text-slate-400">Deployment Logs</div>
        <div className="p-8 space-y-4">
           {history.length === 0 ? (
             <div className="text-slate-600 italic text-sm">No recent deployments recorded in this session.</div>
           ) : (
             history.map((h, i) => (
               <div key={i} className="flex items-center justify-between text-xs border-b border-slate-800/50 pb-3 last:border-0 last:pb-0">
                 <div className="flex items-center gap-3">
                   <div className="w-1.5 h-1.5 rounded-full bg-emerald-500 shadow-[0_0_5px_rgba(16,185,129,0.5)]"/>
                   <span className="font-mono text-slate-300">[{h.updated_at}]</span>
                   <span className="font-bold text-indigo-400">STRATEGY_SET: {h.strategy_name.toUpperCase()}</span>
                 </div>
                 <div className="text-slate-500">Weight: {(h.weight * 100).toFixed(0)}% • Trace: {Math.random().toString(36).substring(7)}</div>
               </div>
             ))
           )}
        </div>
      </div>
    </div>
  );
}
