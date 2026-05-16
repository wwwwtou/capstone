import React, { useState } from "react";
import { recSysService } from "../services/api";
import { Search, Play, UserCircle, Tag, TrendingUp, Terminal } from "lucide-react";

export default function Simulator() {
  const [userId, setUserId] = useState("user_123");
  const [results, setResults] = useState<any>(null);
  const [loading, setLoading] = useState(false);

  const handleFetch = async () => {
    setLoading(true);
    try {
      const res = await recSysService.getRecommendations(userId);
      setResults(res.data);
    } catch (e) {
      console.error(e);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="grid grid-cols-5 gap-8 animate-in fade-in duration-700">
      <div className="col-span-2 space-y-8">
        <div className="bg-slate-900/50 border border-slate-800 p-8 rounded-3xl space-y-6">
          <h3 className="text-xl font-bold flex items-center gap-2">
            <TrendingUp className="w-5 h-5 text-amber-400"/>
            Rec-Engine Simulator
          </h3>
          <p className="text-xs text-slate-500 leading-relaxed">
            Test the live ranking output for specific user profiles. This mirrors the production Go microservice endpoint results.
          </p>
          
          <div className="space-y-4">
             <div className="space-y-2">
               <label className="text-xs font-bold text-slate-400 uppercase tracking-widest pl-1">Target User ID</label>
               <div className="relative">
                 <input 
                   disabled={loading}
                   className="w-full bg-slate-950 border border-slate-800 rounded-2xl pl-12 pr-4 py-4 outline-none focus:ring-2 focus:ring-indigo-500/20 focus:border-indigo-500 transition-all font-mono"
                   value={userId}
                   onChange={(e) => setUserId(e.target.value)}
                   placeholder="Enter UUID or Username..."
                 />
                 <UserCircle className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-600" />
               </div>
             </div>
             
             <button 
               onClick={handleFetch}
               disabled={loading}
               className="w-full bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 h-14 rounded-2xl font-bold flex items-center justify-center gap-3 transition-colors shadow-lg shadow-indigo-600/10"
             >
               {loading ? <div className="w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin" /> : <><Search className="w-5 h-5"/> Fetch Recommendations</>}
             </button>
          </div>
        </div>

        {results && (
          <div className="bg-slate-900/50 border border-slate-800 p-8 rounded-3xl">
             <h4 className="text-xs font-bold text-slate-500 uppercase tracking-widest mb-6 flex items-center gap-2">
               <Terminal className="w-4 h-4" />
               Raw Response (v1)
             </h4>
             <div className="bg-slate-950 rounded-2xl p-6 font-mono text-[10px] text-indigo-300 overflow-auto max-h-80 border border-slate-800">
               <pre>{JSON.stringify(results, null, 2)}</pre>
             </div>
          </div>
        )}
      </div>

      <div className="col-span-3 space-y-6">
        <h3 className="font-bold text-lg text-slate-400 px-2 flex items-center gap-3">
          Ranking Results
          <div className="h-[1px] flex-1 bg-slate-800/50" />
        </h3>

        {!results ? (
           <div className="h-96 flex flex-col items-center justify-center border-2 border-dashed border-slate-800 rounded-3xl text-slate-600 space-y-4">
              <Play className="w-12 h-12 opacity-20" />
              <p className="text-sm font-medium">Trigger simulation to view ranked video feed</p>
           </div>
        ) : (
          <div className="space-y-4">
            {results.data.videos.map((v: any, i: number) => (
              <div key={v.video_id} className="bg-slate-900/40 border border-slate-800/60 p-6 rounded-3xl group hover:border-indigo-500/40 transition-all duration-300 flex justify-between items-center relative overflow-hidden">
                <div className="absolute left-0 top-0 w-1 h-full bg-indigo-600 opacity-0 group-hover:opacity-100 transition-opacity" />
                <div className="flex gap-6 items-center">
                  <div className="w-16 h-20 bg-slate-800 rounded-xl flex items-center justify-center relative overflow-hidden border border-slate-700">
                     <VideoMock index={i} />
                     <div className="absolute top-1 left-1 bg-black/60 rounded px-1.5 py-0.5 text-[8px] font-bold"># {i+1}</div>
                  </div>
                  <div>
                    <h4 className="font-bold text-lg group-hover:text-indigo-400 transition-colors">{v.title}</h4>
                    <div className="flex items-center gap-4 mt-2">
                       <span className="text-xs text-slate-500 flex items-center gap-1.5">
                         <UserCircle className="w-3.5 h-3.5" />
                         {v.author}
                       </span>
                       <span className="text-xs font-bold text-emerald-500/80 bg-emerald-500/5 px-2 py-0.5 rounded-full flex items-center gap-1">
                         <Tag className="w-3 h-3" />
                         {v.reason}
                       </span>
                    </div>
                  </div>
                </div>
                <div className="text-right">
                  <div className="text-xl font-black text-indigo-400">{(v.score * 100).toFixed(0)}</div>
                  <div className="text-[10px] font-bold text-slate-600 uppercase tracking-tighter">Confidence</div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function VideoMock({ index }: { index: number }) {
  const colors = ["bg-indigo-500/20", "bg-amber-500/20", "bg-rose-500/20", "bg-emerald-500/20"];
  return <div className={`w-full h-full ${colors[index % colors.length]} flex items-center justify-center`}><Play className="w-6 h-6 opacity-40 text-white" /></div>;
}
