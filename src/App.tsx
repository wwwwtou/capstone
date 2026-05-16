import React, { useState } from "react";
import { 
  Activity, 
  BarChart3, 
  Zap, 
  LayoutDashboard,
  Cpu,
  LogIn,
  CheckCircle2
} from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import DashboardHome from "./pages/DashboardHome";
import AlgoConfig from "./pages/AlgoConfig";
import Simulator from "./pages/Simulator";
import { recSysService } from "./services/api";

type TabID = "home" | "config" | "simulator";

export default function App() {
  const [activeTab, setActiveTab ] = useState<TabID>("home");
  const [isLoggedIn, setIsLoggedIn] = useState(!!localStorage.getItem("admin_token"));
  const [showToast, setShowToast] = useState(false);

  const handleLogin = async () => {
    try {
      const res = await recSysService.login();
      localStorage.setItem("admin_token", res.data.data.token);
      setIsLoggedIn(true);
      setShowToast(true);
      setTimeout(() => setShowToast(false), 3000);
    } catch (e) {
      console.error("Login failed", e);
    }
  };

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 font-sans selection:bg-indigo-500/30 overflow-x-hidden">
      {/* Sidebar */}
      <aside className="fixed left-0 top-0 h-full w-64 bg-slate-900 border-r border-slate-800 p-6 z-20">
        <div className="flex items-center gap-3 mb-10 px-2">
          <div className="w-10 h-10 bg-indigo-600 rounded-lg flex items-center justify-center shadow-lg shadow-indigo-500/20">
            <Zap className="text-white w-6 h-6 fill-white" />
          </div>
          <h1 className="font-bold text-lg tracking-tight italic">TikTok <span className="text-indigo-400">Global</span></h1>
        </div>

        <nav className="space-y-1">
          <NavButton id="home" label="Dashboard Home" icon={<LayoutDashboard />} active={activeTab} onClick={setActiveTab} />
          <NavButton id="config" label="Algorithm Config" icon={<BarChart3 />} active={activeTab} onClick={setActiveTab} />
          <NavButton id="simulator" label="Demo Simulator" icon={<Cpu />} active={activeTab} onClick={setActiveTab} />
          
          <div className="pt-10 pb-4 px-4 text-[10px] font-black text-slate-600 uppercase tracking-[0.2em]">Deployment Info</div>
          <div className="px-4 py-3 bg-slate-800/30 rounded-2xl border border-white/5 space-y-2">
             <div className="flex justify-between items-center text-[10px]">
               <span className="text-slate-500">Region</span>
               <span className="font-bold text-indigo-400">AWS-SEA-1</span>
             </div>
             <div className="flex justify-between items-center text-[10px]">
               <span className="text-slate-500">Env</span>
               <span className="font-bold text-emerald-400">PROD-SHARD-A</span>
             </div>
          </div>
        </nav>
      </aside>

      {/* Main Content */}
      <main className="ml-64 p-10 max-w-7xl min-h-screen flex flex-col">
        <header className="flex justify-between items-center mb-12">
          <div>
            <h2 className="text-3xl font-black tracking-tighter mb-1 uppercase">
              {activeTab === "home" && "Operational Pulse"}
              {activeTab === "config" && "Ranking Architecture"}
              {activeTab === "simulator" && "Endpoint Simulator"}
            </h2>
            <p className="text-slate-500 text-sm font-medium">
              TikTok Global Ecommerce Recommendation Infrastructure
            </p>
          </div>
          <div className="flex items-center gap-4">
            <button 
              onClick={handleLogin}
              className={`flex items-center gap-2 border px-4 py-2 rounded-xl text-xs font-bold transition-all active:scale-95 ${isLoggedIn ? 'bg-emerald-500/10 border-emerald-500/50 text-emerald-400' : 'bg-slate-900 border-slate-800 hover:bg-slate-800 text-slate-100'}`}
            >
              {isLoggedIn ? <CheckCircle2 className="w-4 h-4" /> : <LogIn className="w-4 h-4" />}
              {isLoggedIn ? "Admin Logged In" : "Admin Access"}
            </button>
          </div>
        </header>

        <AnimatePresence>
          {showToast && (
            <motion.div 
              initial={{ opacity: 0, y: 50, x: "-50%" }}
              animate={{ opacity: 1, y: 0, x: "-50%" }}
              exit={{ opacity: 0, y: 20, x: "-50%" }}
              className="fixed bottom-10 left-1/2 -translate-x-1/2 z-50 bg-emerald-600 text-white px-6 py-3 rounded-2xl shadow-xl flex items-center gap-3 font-bold text-sm"
            >
              <CheckCircle2 className="w-5 h-5" />
              Authenticated with TikTok Global IAM
            </motion.div>
          )}
        </AnimatePresence>

        <section className="flex-1">
          <AnimatePresence mode="wait">
            <motion.div
              key={activeTab}
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -10 }}
              transition={{ duration: 0.3 }}
            >
              {activeTab === "home" && <DashboardHome />}
              {activeTab === "config" && <AlgoConfig />}
              {activeTab === "simulator" && <Simulator />}
            </motion.div>
          </AnimatePresence>
        </section>

        <footer className="mt-20 pt-8 border-t border-slate-900 text-center">
            <p className="text-[10px] text-slate-600 font-bold tracking-widest uppercase">
              MTech Software Engineering Defense • Project Proxy MVP • 2026
            </p>
        </footer>
      </main>
    </div>
  );
}

function NavButton({ id, label, icon, active, onClick }: { id: TabID, label: string, icon: React.ReactNode, active: TabID, onClick: (id: TabID) => void }) {
  const isActive = id === active;
  return (
    <button 
      onClick={() => onClick(id)}
      className={`w-full flex items-center gap-3 px-4 py-3.5 rounded-2xl transition-all duration-300 group ${isActive ? "bg-indigo-600/10 text-indigo-400 font-bold shadow-inner" : "text-slate-400 hover:bg-slate-800 hover:text-slate-100"}`}
    >
      <span className={`w-5 h-5 transition-transform duration-300 ${isActive ? "scale-110" : "group-hover:scale-110"}`}>{icon}</span>
      <span className="text-sm tracking-tight">{label}</span>
      {isActive && <div className="ml-auto w-1.5 h-1.5 rounded-full bg-indigo-500 shadow-[0_0_8px_rgba(99,102,241,0.6)]" />}
    </button>
  );
}
