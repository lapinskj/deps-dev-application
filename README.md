# deps.dev Dependency Viewer

This project is a full-stack application that visualizes the dependencies and OpenSSF security scores of the React package (npm ecosystem, version 18.2.0), using the deps.dev API.

It consists of:
- A frontend for browsing dependencies and visualizing scores
- A backend API in Go
- An SQLite database for storage
- Docker Compose for easy deployment

Features
- Fetches dependencies for the package npm/react/18.2.0 via the deps.dev API
- Stores dependencies and metadata (OpenSSF score, relation, source repo) in SQLite
- Offers a REST API for querying and managing dependencies, supporting full CRUD functionality
- Includes background dependencies refresh (daily or on demand)
- Filters dependencies by name and minimum score
- Simple and dynamic frontend
- Fully containerized with Docker Compose

## Assumptions
- The original task specified `https://github.com/cli/cli`, but it had no dependencies listed via the deps.dev API.
  - For demonstration, the application uses `npm/react/18.2.0` which includes meaningful dependency data.

- Each dependency is uniquely identified by the combination of:
  - `system` (ecosystem, e.g., `npm`, `go`)
  - `name`
  - `version`

- Only the following fields are editable by users:
  - `relation` (`SELF`, `DIRECT`, `INDIRECT`)
  - `source_repo` (derived from deps.dev's project ID)
  - `openssf_score` (OpenSSF scorecard score, if available)

- When dependencies are refreshed (manually or via scheduled cron):
  - Existing fields are only **overwritten if the API provides non-empty values**
  - Manually filled fields **are preserved** if the API returns empty values


## Setup and Running
### 1. Clone the Repository

```bash
git clone https://github.com/lapinskj/deps-dev-application.git
cd deps-dev-application
```

### 2. Run with Docker Compose

```bash
docker-compose up --build -d
```

This will launch:
- A Go backend server
- A React frontend
- An SQLite DB (embedded)

### 3. Stop the Application

```bash
docker-compose down
```

### 4. Environment Variables

You can configure the application by setting environment variables in `.env` or as environment flags:

```env
# Server ports
BACKEND_PORT=8080           # Backend runs at http://localhost:8080
FRONTEND_PORT=5173          # Frontend runs at http://localhost:5173

# SQLite database path
SQLITE_PATH=./data/app.db   # Local DB file path

# API base (used by frontend to call backend)
API_BASE_URL=http://localhost:8080

# Data refresh options
WITH_INITIAL_DATA_REFRESH=true  # Run an initial fetch from deps.dev at startup
WITH_DAILY_DATA_REFRESH=true    # Schedule automatic daily refresh (via cron)
```

## API Documentation

### `GET /dependencies`

Retrieve all stored dependencies, optionally filtered.

**Query Parameters:**

- `name`: filter by dependency name
- `min_score`: filter by OpenSSF score (e.g. `min_score=7.0`)

**Example:**

```bash
GET /dependencies?name=react&min_score=7
```

---

### `POST /dependencies`

Create a new dependency entry.

**Request Body:**

```json
{
  "system": "npm",
  "name": "lodash",
  "version": "4.17.21",
  "relation": "direct",
  "source_repo": "github.com/lodash/lodash",
  "openssf_score": 8.2
}
```

- `system`, `name`, `version` are **required**
- Will return `409 Conflict` if it already exists

---

### `GET /dependencies/{system}/{name}/{version}`

Get a single dependency by system + name + version.

**Example:**

```bash
GET /dependencies/npm/react/18.2.0
```

---

### `PUT /dependencies/{system}/{name}/{version}`

Update an existing dependency. Only `relation`, `source_repo`, and `openssf_score` can be updated.

**Request Body:**

```json
{
  "relation": "DIRECT",
  "openssf_score": 9.1
}
```

---

### `DELETE /dependencies/{system}/{name}/{version}`

Delete a dependency.

---

### `POST /dependencies/refresh`

Trigger a manual refresh for `npm/react@18.2.0`, pulling updated data from deps.dev and updating the DB.

## SQLite Schema

The application uses a single SQLite database table named `dependencies` to store dependency information.

### Table: `dependencies`

| Column          | Type     | Description                                                                 |
|------------------|----------|-----------------------------------------------------------------------------|
| `system`         | TEXT     | Package ecosystem (e.g. `npm`, `go`, etc.)                                 |
| `name`           | TEXT     | Package name                                                                |
| `version`        | TEXT     | Specific version of the dependency                                          |
| `relation`       | TEXT     | How the package is related to the root project: one of `SELF`, `DIRECT`, or `INDIRECT` |
| `source_repo`    | TEXT     | Source repository identifier, derived from the `projectKey.ID` field in the deps.dev API (e.g. `github.com/user/repo`) |
| `openssf_score`  | REAL     | OpenSSF Scorecard score (float between 0–10), if available                  |

### Constraints

- **Primary Key**: Composite of `system`, `name`, and `version`

This schema is initialized automatically on first run if it doesn't exist.



## Data refresh

### Initial data import
If `WITH_INITIAL_DATA_REFRESH=true` is set, the app will run data import at the startup

### Cron Job (Daily Refresh)
If `WITH_DAILY_DATA_REFRESH=true` is set, the app will run a background cron job to:
- Re-fetch dependency data every 24h
- Only overwrite `relation`, `source_repo`, or `openssf_score` **if the new value is not empty**

## Testing

Unit tests cover the core components:
- API client (Deps.dev integration)
- Handlers (HTTP API endpoints)
- Data manager (dependency refresh logic)
- Storage layer (SQLite interactions)

### Run All Tests (from backend folder)

```bash
cd deps-dev-backend
go test ./...
```

## Frontend Features
- List of all dependencies
- View single dependency
- Create, edit or delete dependency
- Search by name or score
- Chart of OpenSSF scores
- Trigger data refresh

