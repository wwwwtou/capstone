# E-commerce Video Recommendation System - Product & Architecture Overview
## Architecture Blueprint & Project Defense Documentation

> **Authoritative diagrams live in [`docs/architecture/`](docs/architecture/) (PlantUML, kept in sync with the code).**
> The Mermaid sketches below are the original blueprint and are retained for
> historical context only — some are aspirational (e.g. the Clean Architecture
> class diagram, the AWS-only deployment) and no longer match the implementation.

### 1. UML Diagrams (Mermaid)

#### 1.1 Use Case Diagram
```mermaid
useCaseDiagram
    actor "Terminal User" as User
    actor "Admin (Operations)" as Admin
    
    package "Video Recommendation System" {
        usecase "Request Recommendations" as UC1
        usecase "Watch Video" as UC2
        usecase "View System Health" as UC3
        usecase "Configure Algo Weights" as UC4
        usecase "Login to Dashboard" as UC5
    }
    
    User --> UC1
    User --> UC2
    Admin --> UC5
    Admin --> UC3
    Admin --> UC4
    UC3 ..> UC5 : include
    UC4 ..> UC5 : include
```

#### 1.2 DDD Context Map
```mermaid
graph TD
    subgraph "Core Domain"
        RecDomain[Recommendation Domain]
    }
    subgraph "Supporting Domains"
        UserDomain[User Profile Domain]
        ContentDomain[Content/Video Domain]
    }
    subgraph "Generic Subdomain"
        MonitorDomain[Monitoring/Dashboard Domain]
    }
    
    UserDomain -- Shared Kernel --> RecDomain
    ContentDomain -- Customer/Supplier --> RecDomain
    RecDomain -- Published Language --> MonitorDomain
```

#### 1.3 Logical Architecture Diagram
```mermaid
graph LR
    subgraph "Frontend Layer"
        ReactApp[React Admin Dashboard]
    end
    
    subgraph "Gateway Layer"
        APIGateway[API Gateway / Nginx]
    end
    
    subgraph "Service Layer (Microservices)"
        RecService[Recommendation Service - Go]
        DashService[Dashboard Service - Go/Node]
    end
    
    subgraph "Infrastructure Layer"
        PG[(PostgreSQL - Meta/Config)]
        Redis[(Redis - Profile/Cache)]
    end
    
    ReactApp --> APIGateway
    APIGateway --> RecService
    APIGateway --> DashService
    RecService --> Redis
    RecService --> PG
    DashService --> PG
```

#### 1.4 Physical/Deployment Diagram (AWS)
```mermaid
graph TD
    Client((Client Browser)) --> Route53[AWS Route 53]
    Route53 --> ALB[AWS Application Load Balancer]
    
    subgraph "VPC / Public Subnet"
        ALB --> Nginx[ECS: Nginx Container]
    end
    
    subgraph "VPC / Private Subnet"
        Nginx --> RecApp[ECS: Go Rec-Service]
        Nginx --> FEApp[ECS: React Admin App]
        
        RecApp --> ElastiCache[(ElastiCache - Redis)]
        RecApp --> RDS[(RDS - PostgreSQL)]
    end
```

#### 1.5 Key Use-Case Sequence Diagram: Get Recommendations
```mermaid
sequenceDiagram
    participant C as Client
    participant G as API Gateway
    participant CTRL as Rec-Controller
    participant RED as Redis (Cache/Profile)
    participant DB as Postgres (Metadata)
    participant RNG as Ranking Engine
    
    C->>G: GET /api/v1/recommendations?user_id=123
    G->>CTRL: Route to Rec-Service
    
    par Async Fetch Data
        CTRL->>RED: Fetch User Tags (User:123:tags)
        RED-->>CTRL: {tags: [tech, sports]}
    and
        CTRL->>RED: Fetch Hot Candidate Videos
        RED-->>CTRL: [v1, v2, v3, ..., v100]
    end
    
    CTRL->>DB: Fetch Algorithm Weights
    DB-->>CTRL: {weight_like: 0.4, weight_finish: 0.6}
    
    CTRL->>RNG: Score & Rank(Candidates, UserTags, Weights)
    Note over RNG: Strategy Pattern applied here
    RNG-->>CTRL: Sorted List [v9, v2, v45...]
    
    CTRL-->>G: 200 OK (JSON)
    G-->>C: Display Video Feed
```

#### 1.6 Class Diagram (Clean Architecture Implementation)
```mermaid
classDiagram
    namespace Application {
        class RecommendationService {
            -videoRepo: VideoRepository
            -userRepo: UserRepository
            +GetRecommendations(ctx, userID)
            -rank(videos, profile, weights)
        }
    }
    
    namespace Domain {
        class VideoRepository <<interface>> {
            +GetCandidates(ctx)
        }
        class UserRepository <<interface>> {
            +GetUserProfile(ctx, userID)
        }
        class Video {
            +ID: string
            +Score: float
        }
    }
    
    namespace Infrastructure {
        class PostgresVideoRepo {
            +GetCandidates(ctx)
        }
        class RedisUserRepo {
            +GetUserProfile(ctx, userID)
        }
    }

    RecommendationService ..|> VideoRepository : uses
    RecommendationService ..|> UserRepository : uses
    PostgresVideoRepo --|> VideoRepository : implements
    RedisUserRepo --|> UserRepository : implements
```

#### 1.7 CI/CD Pipeline Flow
```mermaid
graph LR
    Dev[Developer Push] --> GH[GitHub Actions]
    subgraph "CI Stage"
        GH --> Lint[Linting & Static Analysis]
        Lint --> UT[Unit Testing]
        UT --> Build[Docker Image Build]
    end
    subgraph "CD Stage (Simulation)"
        Build --> Push[Push to Registry]
        Push --> Deploy[Deploy to Cloud Run / Sandbox]
    end
```

---

### 2. Database Design (ER Diagram)
```mermaid
erDiagram
    ALGORITHM_CONFIG ||--o{ RECOMMENDATION : "defines"
    USER_PROFILE ||--o{ INTERACTION : "performs"
    VIDEO_METADATA ||--o{ INTERACTION : "is target of"
    
    USER_PROFILE {
        string user_id PK
        jsonb tags "Interests (e.g. ['tech', 'gaming'])"
        timestamp last_active
    }
    
    VIDEO_METADATA {
        string video_id PK
        string title
        string url
        jsonb metadata "Resolution, duration, etc."
        timestamp created_at
    }
    
    ALGORITHM_CONFIG {
        int id PK
        float weight_like "Weight for likes"
        float weight_finish "Weight for watch-time"
        boolean is_active
        timestamp updated_at
    }
```

---

### 3. Risks & Mitigations

| Category | Concern | Mitigation |
| :--- | :--- | :--- |
| **Management** | Single-person project burnout/delays. | Scope Phase 1 to "Vertical Slicing" (one path from UI to DB) instead of broad features. |
| **Technical** | Golang concurrency handling for high traffic. | Use `goroutines` for async data fetching and `context` for timeout/cancellation. |
| **Technical** | Cold Start problem for new users. | Fallback to "Globally Popular" category when user tags are missing in Redis. |
| **Security** | Dashboard data leakage. | Implement RBAC (Role Based Access Control) and JWT-based authentication for admins. |
| **Security** | API Abuse / DDoS. | Layer API Gateway rate-limiting and enforce Request ID (trace_id) for logging. |
