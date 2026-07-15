import React, { useState, useEffect, useCallback, useRef } from "react";
import { motion, AnimatePresence } from "motion/react";
import {
  Heart,
  Share2,
  MessageCircle,
  ChevronUp,
  ChevronDown,
  RefreshCw,
  UserCircle,
  Sparkles,
  Tag,
  Play,
  Radio,
} from "lucide-react";
import { recSysService } from "../services/api";

// Category-driven cover art: no real video files, each card renders a gradient
// "poster" so the ranked order and reasons stay the visual focus.
const CATEGORY_STYLES: Record<string, { gradient: string; emoji: string }> = {
  fashion: { gradient: "from-rose-500/80 via-pink-600/60 to-fuchsia-900", emoji: "👗" },
  electronics: { gradient: "from-indigo-500/80 via-blue-600/60 to-slate-900", emoji: "🎧" },
  home: { gradient: "from-amber-500/80 via-orange-600/60 to-stone-900", emoji: "🏺" },
  food: { gradient: "from-orange-500/80 via-red-600/60 to-rose-900", emoji: "🍜" },
  tech: { gradient: "from-violet-500/80 via-purple-600/60 to-indigo-950", emoji: "🤖" },
  fitness: { gradient: "from-emerald-500/80 via-teal-600/60 to-slate-900", emoji: "💪" },
  travel: { gradient: "from-sky-500/80 via-cyan-600/60 to-blue-950", emoji: "🏝️" },
};
const FALLBACK_STYLE = { gradient: "from-slate-600/80 via-slate-700/60 to-slate-950", emoji: "🎬" };

const PERSONAS = ["user_123", "user_fashion", "user_foodie", "user_new"];

type FeedVideo = {
  video_id: string;
  title: string;
  author: string;
  category?: string;
  score: number;
  reason: string;
};

type EventEntry = { ts: string; text: string };

export default function Feed() {
  const [userId, setUserId] = useState("user_123");
  const [videos, setVideos] = useState<FeedVideo[]>([]);
  const [index, setIndex] = useState(0);
  const [direction, setDirection] = useState(1);
  const [strategy, setStrategy] = useState("");
  const [profileTags, setProfileTags] = useState<Record<string, number>>({});
  const [liked, setLiked] = useState<Set<string>>(new Set());
  const [events, setEvents] = useState<EventEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const viewedRef = useRef<Set<string>>(new Set());
  const wheelLockRef = useRef(0);

  const logEvent = useCallback((text: string) => {
    setEvents((prev) => [{ ts: new Date().toLocaleTimeString(), text }, ...prev].slice(0, 8));
  }, []);

  const refreshProfile = useCallback(async (uid: string) => {
    try {
      const res = await recSysService.getProfile(uid);
      setProfileTags(res.data?.tags ?? {});
    } catch {
      setProfileTags({});
    }
  }, []);

  const loadFeed = useCallback(
    async (uid: string, announce = false) => {
      setLoading(true);
      try {
        const res = await recSysService.getRecommendations(uid);
        setVideos(res.data?.data?.videos ?? []);
        setStrategy(res.data?.data?.strategy ?? "");
        setIndex(0);
        setDirection(1);
        viewedRef.current = new Set();
        if (announce) logEvent(`feed re-ranked for ${uid}`);
      } catch (e) {
        console.error("feed load failed", e);
      } finally {
        setLoading(false);
      }
    },
    [logEvent]
  );

  useEffect(() => {
    loadFeed(userId);
    refreshProfile(userId);
    setLiked(new Set());
  }, [userId, loadFeed, refreshProfile]);

  const current = videos[index];

  // Auto-log a "view" once the card has been on screen for a moment — this is
  // the implicit-feedback half of the interaction -> profile -> ranking loop.
  useEffect(() => {
    if (!current || viewedRef.current.has(current.video_id)) return;
    const timer = setTimeout(async () => {
      viewedRef.current.add(current.video_id);
      try {
        await recSysService.postInteraction(userId, {
          event_type: "view",
          metadata: { category: current.category ?? "", video_id: current.video_id },
        });
        logEvent(`view ${current.video_id} (${current.category})`);
        refreshProfile(userId);
      } catch {
        /* interaction endpoint unavailable */
      }
    }, 1200);
    return () => clearTimeout(timer);
  }, [current, userId, logEvent, refreshProfile]);

  const step = useCallback(
    (dir: 1 | -1) => {
      setDirection(dir);
      setIndex((i) => Math.max(0, Math.min(videos.length - 1, i + dir)));
    },
    [videos.length]
  );

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "ArrowDown") step(1);
      if (e.key === "ArrowUp") step(-1);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [step]);

  const handleWheel = (e: React.WheelEvent) => {
    const now = Date.now();
    if (now - wheelLockRef.current < 400) return;
    wheelLockRef.current = now;
    step(e.deltaY > 0 ? 1 : -1);
  };

  const handleLike = async () => {
    if (!current) return;
    const already = liked.has(current.video_id);
    setLiked((prev) => {
      const next = new Set(prev);
      if (already) next.delete(current.video_id);
      else next.add(current.video_id);
      return next;
    });
    if (already) return; // unlike is UI-only; profile keeps the signal
    try {
      await recSysService.postInteraction(userId, {
        event_type: "like",
        metadata: { category: current.category ?? "", video_id: current.video_id },
      });
      logEvent(`like ${current.video_id} (${current.category})`);
      refreshProfile(userId);
    } catch {
      /* interaction endpoint unavailable */
    }
  };

  const style = CATEGORY_STYLES[current?.category ?? ""] ?? FALLBACK_STYLE;
  const maxTag = Math.max(1, ...(Object.values(profileTags) as number[]));

  return (
    <div className="grid grid-cols-5 gap-8 animate-in fade-in duration-700">
      {/* Phone-frame vertical feed */}
      <div className="col-span-2 flex justify-center">
        <div className="relative w-[340px]">
          <div
            data-testid="feed-frame"
            onWheel={handleWheel}
            className="relative h-[600px] rounded-[2.5rem] border-4 border-slate-800 bg-slate-950 overflow-hidden shadow-2xl shadow-indigo-500/10"
          >
            <AnimatePresence mode="popLayout" custom={direction}>
              {current ? (
                <motion.div
                  key={current.video_id}
                  custom={direction}
                  initial={{ y: direction > 0 ? 600 : -600, opacity: 0.6 }}
                  animate={{ y: 0, opacity: 1 }}
                  exit={{ y: direction > 0 ? -600 : 600, opacity: 0.4 }}
                  transition={{ type: "spring", stiffness: 300, damping: 32 }}
                  className={`absolute inset-0 bg-gradient-to-b ${style.gradient} flex flex-col justify-between p-6`}
                >
                  {/* Top: rank + reason badge */}
                  <div className="flex justify-between items-start pt-2">
                    <span className="bg-black/40 backdrop-blur px-3 py-1 rounded-full text-[10px] font-black tracking-widest uppercase">
                      Rank #{index + 1} / {videos.length}
                    </span>
                    <span className="bg-emerald-500/20 border border-emerald-400/40 text-emerald-200 px-3 py-1 rounded-full text-[10px] font-bold flex items-center gap-1">
                      <Tag className="w-3 h-3" />
                      {current.reason}
                    </span>
                  </div>

                  {/* Center: poster emoji */}
                  <div className="flex-1 flex items-center justify-center">
                    <div className="text-7xl drop-shadow-2xl select-none">{style.emoji}</div>
                  </div>

                  {/* Bottom: metadata */}
                  <div className="space-y-2 pb-2 pr-14">
                    <div className="flex items-center gap-2 text-xs font-bold text-white/80">
                      <UserCircle className="w-4 h-4" />@{current.author}
                      <span className="bg-white/10 px-2 py-0.5 rounded-full text-[10px] uppercase tracking-wide">
                        {current.category}
                      </span>
                    </div>
                    <h3 className="text-xl font-black leading-tight drop-shadow-lg">{current.title}</h3>
                    <div className="text-[10px] font-bold text-white/60 uppercase tracking-widest">
                      Model confidence {(current.score * 100).toFixed(0)}%
                    </div>
                  </div>
                </motion.div>
              ) : (
                <div className="absolute inset-0 flex flex-col items-center justify-center text-slate-600 gap-3">
                  <Play className="w-10 h-10 opacity-30" />
                  <span className="text-xs font-bold">{loading ? "Loading feed..." : "No videos"}</span>
                </div>
              )}
            </AnimatePresence>

            {/* Action rail */}
            {current && (
              <div className="absolute right-3 bottom-24 flex flex-col items-center gap-5 z-10">
                <button
                  data-testid="like-button"
                  onClick={handleLike}
                  className="flex flex-col items-center gap-1 group"
                >
                  <span
                    className={`p-3 rounded-full backdrop-blur transition-all active:scale-90 ${
                      liked.has(current.video_id)
                        ? "bg-rose-500 text-white shadow-lg shadow-rose-500/40"
                        : "bg-black/40 text-white group-hover:bg-black/60"
                    }`}
                  >
                    <Heart className={`w-5 h-5 ${liked.has(current.video_id) ? "fill-white" : ""}`} />
                  </span>
                  <span className="text-[9px] font-black text-white/80">LIKE</span>
                </button>
                <span className="p-3 rounded-full bg-black/40 backdrop-blur text-white/70">
                  <MessageCircle className="w-5 h-5" />
                </span>
                <span className="p-3 rounded-full bg-black/40 backdrop-blur text-white/70">
                  <Share2 className="w-5 h-5" />
                </span>
              </div>
            )}
          </div>

          {/* Up/down nav */}
          <div className="absolute -right-14 top-1/2 -translate-y-1/2 flex flex-col gap-3">
            <button
              onClick={() => step(-1)}
              disabled={index === 0}
              className="p-3 bg-slate-900 border border-slate-800 rounded-2xl hover:bg-slate-800 disabled:opacity-30 transition-all"
              aria-label="Previous video"
            >
              <ChevronUp className="w-5 h-5" />
            </button>
            <button
              onClick={() => step(1)}
              disabled={index >= videos.length - 1}
              className="p-3 bg-slate-900 border border-slate-800 rounded-2xl hover:bg-slate-800 disabled:opacity-30 transition-all"
              aria-label="Next video"
            >
              <ChevronDown className="w-5 h-5" />
            </button>
          </div>
        </div>
      </div>

      {/* Control / insight panel */}
      <div className="col-span-3 space-y-6">
        <div className="bg-slate-900/50 border border-slate-800 p-8 rounded-3xl space-y-6">
          <div className="flex items-center justify-between">
            <h3 className="text-xl font-bold flex items-center gap-2">
              <Sparkles className="w-5 h-5 text-indigo-400" />
              For You Feed
            </h3>
            {strategy && (
              <span className="text-[10px] font-black uppercase tracking-widest bg-indigo-500/10 text-indigo-300 border border-indigo-500/30 px-3 py-1 rounded-full">
                strategy: {strategy}
              </span>
            )}
          </div>
          <p className="text-xs text-slate-500 leading-relaxed">
            Consumer-side view of the recommendation loop: every view/like is posted to the User
            Profile service, updates the interest tags, and changes the ranking the next time the
            feed is refreshed.
          </p>

          <div className="flex flex-wrap gap-2">
            {PERSONAS.map((p) => (
              <button
                key={p}
                onClick={() => setUserId(p)}
                className={`px-4 py-2 rounded-xl text-xs font-bold border transition-all ${
                  userId === p
                    ? "bg-indigo-600 border-indigo-500 text-white"
                    : "bg-slate-950 border-slate-800 text-slate-400 hover:border-slate-600"
                }`}
              >
                {p}
              </button>
            ))}
            <button
              data-testid="rerank-button"
              onClick={() => loadFeed(userId, true)}
              disabled={loading}
              className="ml-auto flex items-center gap-2 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 px-5 py-2 rounded-xl text-xs font-bold transition-all active:scale-95"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? "animate-spin" : ""}`} />
              Re-rank Feed
            </button>
          </div>
        </div>

        {/* Live profile */}
        <div className="bg-slate-900/50 border border-slate-800 p-8 rounded-3xl">
          <h4 className="text-xs font-bold text-slate-500 uppercase tracking-widest mb-5 flex items-center gap-2">
            <UserCircle className="w-4 h-4" />
            Live Interest Profile — {userId}
          </h4>
          {Object.keys(profileTags).length === 0 ? (
            <p className="text-xs text-slate-600 font-medium">
              Cold start: no interest signals yet. Like or watch a few videos.
            </p>
          ) : (
            <div className="space-y-3" data-testid="profile-tags">
              {(Object.entries(profileTags) as [string, number][])
                .sort((a, b) => b[1] - a[1])
                .map(([tag, count]) => (
                  <div key={tag} className="flex items-center gap-3">
                    <span className="w-24 text-xs font-bold text-slate-300">{tag}</span>
                    <div className="flex-1 h-2 bg-slate-800 rounded-full overflow-hidden">
                      <motion.div
                        layout
                        className="h-full bg-gradient-to-r from-indigo-500 to-fuchsia-500"
                        animate={{ width: `${(count / maxTag) * 100}%` }}
                        transition={{ duration: 0.5 }}
                      />
                    </div>
                    <span className="w-8 text-right text-xs font-black text-indigo-300">{count}</span>
                  </div>
                ))}
            </div>
          )}
        </div>

        {/* Event log */}
        <div className="bg-slate-900/50 border border-slate-800 p-8 rounded-3xl">
          <h4 className="text-xs font-bold text-slate-500 uppercase tracking-widest mb-5 flex items-center gap-2">
            <Radio className="w-4 h-4" />
            Interaction Events (POST /users/:id/interactions)
          </h4>
          {events.length === 0 ? (
            <p className="text-xs text-slate-600 font-medium">No events sent yet this session.</p>
          ) : (
            <div className="space-y-2 font-mono text-[11px]" data-testid="event-log">
              {events.map((e, i) => (
                <div key={`${e.ts}-${i}`} className="flex gap-3 text-slate-400">
                  <span className="text-slate-600">{e.ts}</span>
                  <span className="text-emerald-400/80">{e.text}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
