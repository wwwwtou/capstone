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
    
  updateConfigs: (data: { strategy_name: string; weight: number }) => 
    api.put("/configs", data),
    
  getHealth: () => 
    api.get("/health"),
};
