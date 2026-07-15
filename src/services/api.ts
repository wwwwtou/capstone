import axios from "axios";

const api = axios.create({
  baseURL: "/api/v1",
});

// Automatically inject JWT if present
api.interceptors.request.use((config) => {
  const token = localStorage.getItem("admin_token");
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

export const recSysService = {
  login: () => api.post("/login"),

  getRecommendations: (userId: string) =>
    api.get(`/recommendations?user_id=${userId}`),

  getConfigs: () =>
    api.get("/configs"),

  getConfigHistory: () =>
    api.get("/configs/history"),

  updateConfigs: (data: { strategy_name: string; weight: number }) =>
    api.put("/configs", data),

  getHealth: () =>
    api.get("/health"),

  // --- Consumer feed: interaction events + live profile ---
  postInteraction: (userId: string, data: { event_type: string; metadata: Record<string, any> }) =>
    api.post(`/users/${encodeURIComponent(userId)}/interactions`, data),

  getProfile: (userId: string) =>
    api.get(`/users/${encodeURIComponent(userId)}/profile`),

  // --- Observability: aggregated metrics (gateway + all services) ---
  getMetrics: () =>
    api.get("/metrics"),

  // --- Demo traffic generator (BFF-hosted) ---
  getTrafficStatus: () =>
    api.get("/simulator/traffic"),

  setTraffic: (data: { enabled: boolean; rps?: number }) =>
    api.post("/simulator/traffic", data),

  runBurst: (data?: { count?: number; concurrency?: number }) =>
    api.post("/simulator/burst", data ?? {}),
};
