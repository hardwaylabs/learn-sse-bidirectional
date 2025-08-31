# Use Case: Live Dashboard & Monitoring

Building real-time dashboards using SSE + HTTP POST for monitoring systems, metrics, and interactive controls.

## Architecture Overview

```
┌─────────────┐    SSE     ┌──────────────┐    Data       ┌─────────────┐
│  Dashboard  │◄──────────►│   Dashboard  │◄─────────────►│  Metrics    │
│   Client    │            │   Server     │               │  Sources    │
└─────────────┘            └──────────────┘               └─────────────┘
       │                           ▲
       │ HTTP POST                 │ Commands
       ▼                           ▼
┌─────────────┐                ┌──────────────┐
│   Control   │                │   System     │
│   Actions   │                │  Commands    │
└─────────────┘                └──────────────┘
```

## Why SSE + HTTP POST for Dashboards?

### Real-Time Data Streaming
- **Live Metrics**: CPU, memory, network stats stream continuously
- **Event Notifications**: Alerts, deployments, system events arrive instantly  
- **Multi-Client Updates**: All connected dashboards see the same live data
- **Efficient**: Server pushes only when data changes

### Interactive Controls
- **Bidirectional**: Dashboards can trigger actions (restart service, deploy, etc.)
- **Reliable Commands**: HTTP POST ensures control commands are delivered
- **Immediate Feedback**: Action results stream back via SSE
- **Audit Trail**: All control actions logged with user attribution

### Advantages over Alternatives
| Feature | SSE + HTTP | WebSockets | Polling |
|---------|------------|------------|---------|
| **Real-time Updates** | ✅ Instant | ✅ Instant | ❌ Delayed |
| **Reliable Commands** | ✅ HTTP POST | ⚠️ Connection dependent | ✅ HTTP POST |
| **Network Friendly** | ✅ HTTP only | ❌ Upgrade required | ✅ HTTP only |
| **Debug Friendly** | ✅ Standard tools | ⚠️ Special tools | ✅ Standard tools |
| **Scaling** | ✅ Stateless commands | ⚠️ Stateful connections | ❌ High overhead |

## Implementation Pattern

### Dashboard Client (React)
```javascript
import { useState, useEffect } from 'react';

function SystemDashboard() {
    const [metrics, setMetrics] = useState({});
    const [alerts, setAlerts] = useState([]);
    const [connected, setConnected] = useState(false);
    
    useEffect(() => {
        // Connect to real-time metrics stream
        const eventSource = new EventSource('/dashboard/stream');
        
        eventSource.onopen = () => setConnected(true);
        eventSource.onclose = () => setConnected(false);
        
        eventSource.onmessage = (event) => {
            const data = JSON.parse(event.data);
            
            switch (data.type) {
                case 'metrics':
                    setMetrics(prev => ({ ...prev, ...data.metrics }));
                    break;
                case 'alert':
                    setAlerts(prev => [data.alert, ...prev.slice(0, 49)]);
                    break;
                case 'deployment':
                    showNotification(data.message);
                    break;
            }
        };
        
        return () => eventSource.close();
    }, []);
    
    const executeCommand = async (command, target) => {
        try {
            const response = await fetch('/dashboard/commands', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    command,
                    target,
                    timestamp: Date.now()
                })
            });
            
            const result = await response.json();
            if (result.success) {
                showNotification(`${command} executed successfully`);
            }
        } catch (error) {
            showNotification(`Error: ${error.message}`, 'error');
        }
    };
    
    return (
        <div className="dashboard">
            <ConnectionStatus connected={connected} />
            <MetricsPanel metrics={metrics} />
            <AlertsPanel alerts={alerts} />
            <ControlPanel onCommand={executeCommand} />
        </div>
    );
}
```

### Dashboard Server (Go)
```go
type DashboardServer struct {
    clients      map[string]chan DashboardEvent
    metricsCache map[string]interface{}
    mutex        sync.RWMutex
    commandQueue chan Command
}

type DashboardEvent struct {
    Type      string      `json:"type"`
    Metrics   interface{} `json:"metrics,omitempty"`
    Alert     *Alert      `json:"alert,omitempty"`
    Message   string      `json:"message,omitempty"`
    Timestamp int64       `json:"timestamp"`
}

// SSE endpoint for streaming dashboard data
func (ds *DashboardServer) HandleStream(w http.ResponseWriter, r *http.Request) {
    clientID := generateClientID()
    
    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    // Create client channel
    clientChan := make(chan DashboardEvent, 100)
    ds.addClient(clientID, clientChan)
    defer ds.removeClient(clientID)
    
    // Send current state immediately
    ds.sendCurrentState(clientChan)
    
    // Stream events
    for event := range clientChan {
        eventJSON, _ := json.Marshal(event)
        fmt.Fprintf(w, "data: %s\n\n", eventJSON)
        
        if flusher, ok := w.(http.Flusher); ok {
            flusher.Flush()
        }
    }
}

// HTTP POST endpoint for dashboard commands
func (ds *DashboardServer) HandleCommand(w http.ResponseWriter, r *http.Request) {
    var cmd Command
    if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Validate command
    if err := ds.validateCommand(cmd); err != nil {
        http.Error(w, err.Error(), http.StatusForbidden)
        return
    }
    
    // Execute command asynchronously
    go func() {
        result := ds.executeCommand(cmd)
        
        // Broadcast result to all clients
        event := DashboardEvent{
            Type:      "command_result",
            Message:   result.Message,
            Timestamp: time.Now().Unix(),
        }
        ds.broadcastEvent(event)
    }()
    
    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}
```

## Data Collection & Streaming

### System Metrics Collection
```go
type MetricsCollector struct {
    dashboard *DashboardServer
    interval  time.Duration
    sources   []MetricSource
}

func (mc *MetricsCollector) Start() {
    ticker := time.NewTicker(mc.interval)
    
    go func() {
        for range ticker.C {
            metrics := mc.collectAllMetrics()
            
            event := DashboardEvent{
                Type:      "metrics",
                Metrics:   metrics,
                Timestamp: time.Now().Unix(),
            }
            
            mc.dashboard.broadcastEvent(event)
        }
    }()
}

func (mc *MetricsCollector) collectAllMetrics() map[string]interface{} {
    metrics := make(map[string]interface{})
    
    // CPU usage
    cpuPercent, _ := cpu.Percent(0, false)
    metrics["cpu"] = cpuPercent[0]
    
    // Memory usage
    memInfo, _ := mem.VirtualMemory()
    metrics["memory"] = map[string]interface{}{
        "used":    memInfo.Used,
        "total":   memInfo.Total,
        "percent": memInfo.UsedPercent,
    }
    
    // Disk usage
    diskInfo, _ := disk.Usage("/")
    metrics["disk"] = map[string]interface{}{
        "used":    diskInfo.Used,
        "total":   diskInfo.Total,
        "percent": diskInfo.UsedPercent,
    }
    
    // Application metrics
    metrics["services"] = mc.getServiceStatus()
    metrics["requests_per_second"] = mc.getRequestRate()
    metrics["active_connections"] = mc.getActiveConnections()
    
    return metrics
}
```

### Alert Processing
```go
type AlertManager struct {
    dashboard *DashboardServer
    rules     []AlertRule
    history   []Alert
}

type Alert struct {
    ID       string    `json:"id"`
    Level    string    `json:"level"`    // info, warning, error, critical
    Service  string    `json:"service"`
    Message  string    `json:"message"`
    Time     time.Time `json:"time"`
    Resolved bool      `json:"resolved"`
}

func (am *AlertManager) ProcessMetrics(metrics map[string]interface{}) {
    for _, rule := range am.rules {
        if alert := rule.Evaluate(metrics); alert != nil {
            am.sendAlert(*alert)
        }
    }
}

func (am *AlertManager) sendAlert(alert Alert) {
    // Store in history
    am.history = append(am.history, alert)
    
    // Broadcast to dashboard clients
    event := DashboardEvent{
        Type:      "alert",
        Alert:     &alert,
        Timestamp: time.Now().Unix(),
    }
    
    am.dashboard.broadcastEvent(event)
    
    // Send to external systems (Slack, email, etc.)
    am.sendExternalNotification(alert)
}
```

## Interactive Controls

### Command System
```go
type Command struct {
    ID        string            `json:"id"`
    Type      string            `json:"type"`      // restart, deploy, scale, etc.
    Target    string            `json:"target"`    // service name, server, etc.
    Params    map[string]string `json:"params"`
    UserID    string            `json:"user_id"`
    Timestamp int64             `json:"timestamp"`
}

type CommandExecutor struct {
    handlers map[string]CommandHandler
}

type CommandHandler interface {
    Execute(cmd Command) CommandResult
    Validate(cmd Command) error
}

// Service restart handler
type RestartHandler struct {
    services map[string]Service
}

func (rh *RestartHandler) Execute(cmd Command) CommandResult {
    service := rh.services[cmd.Target]
    
    // Stop service
    if err := service.Stop(); err != nil {
        return CommandResult{
            Success: false,
            Message: fmt.Sprintf("Failed to stop %s: %v", cmd.Target, err),
        }
    }
    
    // Start service
    if err := service.Start(); err != nil {
        return CommandResult{
            Success: false,
            Message: fmt.Sprintf("Failed to start %s: %v", cmd.Target, err),
        }
    }
    
    return CommandResult{
        Success: true,
        Message: fmt.Sprintf("Successfully restarted %s", cmd.Target),
    }
}

// Deployment handler
type DeployHandler struct {
    deploymentService DeploymentService
}

func (dh *DeployHandler) Execute(cmd Command) CommandResult {
    version := cmd.Params["version"]
    environment := cmd.Params["environment"]
    
    deploymentID, err := dh.deploymentService.Deploy(cmd.Target, version, environment)
    if err != nil {
        return CommandResult{
            Success: false,
            Message: fmt.Sprintf("Deployment failed: %v", err),
        }
    }
    
    return CommandResult{
        Success: true,
        Message: fmt.Sprintf("Deployment %s started for %s v%s", deploymentID, cmd.Target, version),
        Data:    map[string]interface{}{"deployment_id": deploymentID},
    }
}
```

### Real-Time Command Feedback
```go
func (ds *DashboardServer) executeCommand(cmd Command) CommandResult {
    // Send "command started" event
    ds.broadcastEvent(DashboardEvent{
        Type:      "command_started",
        Message:   fmt.Sprintf("Executing %s on %s", cmd.Type, cmd.Target),
        Timestamp: time.Now().Unix(),
    })
    
    // Execute command
    handler := ds.commandExecutor.GetHandler(cmd.Type)
    result := handler.Execute(cmd)
    
    // Send completion event
    eventType := "command_completed"
    if !result.Success {
        eventType = "command_failed"
    }
    
    ds.broadcastEvent(DashboardEvent{
        Type:      eventType,
        Message:   result.Message,
        Timestamp: time.Now().Unix(),
    })
    
    return result
}
```

## Dashboard Components

### Metrics Visualization
```javascript
// MetricsPanel.jsx
function MetricsPanel({ metrics }) {
    return (
        <div className="metrics-grid">
            <MetricCard
                title="CPU Usage"
                value={metrics.cpu}
                unit="%"
                threshold={80}
                format="percentage"
            />
            <MetricCard
                title="Memory Usage"
                value={metrics.memory?.percent}
                unit="%"
                threshold={85}
                format="percentage"
            />
            <MetricCard
                title="Active Connections"
                value={metrics.active_connections}
                unit=""
                threshold={1000}
                format="number"
            />
            <ServiceStatus services={metrics.services} />
        </div>
    );
}

function MetricCard({ title, value, unit, threshold, format }) {
    const getStatus = () => {
        if (value > threshold) return 'critical';
        if (value > threshold * 0.8) return 'warning';
        return 'normal';
    };
    
    const formatValue = (val) => {
        if (format === 'percentage') return `${val?.toFixed(1) || 0}%`;
        if (format === 'number') return val?.toLocaleString() || '0';
        return val || 'N/A';
    };
    
    return (
        <div className={`metric-card ${getStatus()}`}>
            <h3>{title}</h3>
            <div className="value">{formatValue(value)}</div>
            <div className="threshold">Threshold: {threshold}{unit}</div>
        </div>
    );
}
```

### Real-Time Charts
```javascript
// ChartPanel.jsx
import { Line } from 'react-chartjs-2';

function ChartPanel({ metrics }) {
    const [history, setHistory] = useState([]);
    
    useEffect(() => {
        if (metrics.cpu !== undefined) {
            setHistory(prev => [...prev.slice(-29), {
                time: new Date().toLocaleTimeString(),
                cpu: metrics.cpu,
                memory: metrics.memory?.percent || 0,
                timestamp: Date.now()
            }]);
        }
    }, [metrics]);
    
    const chartData = {
        labels: history.map(h => h.time),
        datasets: [
            {
                label: 'CPU %',
                data: history.map(h => h.cpu),
                borderColor: '#ff6b6b',
                backgroundColor: 'rgba(255, 107, 107, 0.1)',
            },
            {
                label: 'Memory %',
                data: history.map(h => h.memory),
                borderColor: '#4ecdc4',
                backgroundColor: 'rgba(78, 205, 196, 0.1)',
            }
        ]
    };
    
    const options = {
        responsive: true,
        scales: {
            y: {
                beginAtZero: true,
                max: 100
            }
        },
        animation: false, // Disable animation for real-time updates
    };
    
    return (
        <div className="chart-panel">
            <h3>System Performance</h3>
            <Line data={chartData} options={options} />
        </div>
    );
}
```

### Control Panel
```javascript
// ControlPanel.jsx
function ControlPanel({ onCommand }) {
    const [loading, setLoading] = useState({});
    
    const executeCommand = async (type, target, params = {}) => {
        const key = `${type}-${target}`;
        setLoading(prev => ({ ...prev, [key]: true }));
        
        try {
            await onCommand(type, target, params);
        } finally {
            setLoading(prev => ({ ...prev, [key]: false }));
        }
    };
    
    return (
        <div className="control-panel">
            <h3>System Controls</h3>
            
            <div className="service-controls">
                <h4>Services</h4>
                {['web-server', 'api-server', 'worker'].map(service => (
                    <div key={service} className="service-control">
                        <span>{service}</span>
                        <button
                            onClick={() => executeCommand('restart', service)}
                            disabled={loading[`restart-${service}`]}
                        >
                            {loading[`restart-${service}`] ? 'Restarting...' : 'Restart'}
                        </button>
                        <button
                            onClick={() => executeCommand('logs', service)}
                            disabled={loading[`logs-${service}`]}
                        >
                            View Logs
                        </button>
                    </div>
                ))}
            </div>
            
            <div className="deployment-controls">
                <h4>Deployments</h4>
                <DeploymentForm onDeploy={executeCommand} />
            </div>
        </div>
    );
}
```

## Advanced Features

### Multi-Environment Support
```go
type EnvironmentManager struct {
    environments map[string]*Environment
    dashboards   map[string]*DashboardServer
}

type Environment struct {
    Name     string
    Servers  []Server
    Services []Service
    Metrics  MetricsCollector
}

func (em *EnvironmentManager) GetEnvironmentStream(envName string) *DashboardServer {
    if dashboard, exists := em.dashboards[envName]; exists {
        return dashboard
    }
    
    // Create new dashboard for environment
    env := em.environments[envName]
    dashboard := NewDashboardServer()
    
    // Connect environment metrics to dashboard
    env.Metrics.SetCallback(func(metrics map[string]interface{}) {
        dashboard.broadcastEvent(DashboardEvent{
            Type:      "metrics",
            Metrics:   metrics,
            Timestamp: time.Now().Unix(),
        })
    })
    
    em.dashboards[envName] = dashboard
    return dashboard
}
```

### User-Specific Views
```go
func (ds *DashboardServer) HandleUserStream(w http.ResponseWriter, r *http.Request) {
    userID := getUserID(r)
    permissions := getUserPermissions(userID)
    
    clientChan := make(chan DashboardEvent, 100)
    
    // Filter events based on user permissions
    go func() {
        for event := range ds.globalEvents {
            if permissions.CanView(event) {
                clientChan <- event
            }
        }
    }()
    
    // Stream filtered events
    ds.streamToClient(w, clientChan)
}
```

### Historical Data & Analytics
```go
type HistoricalData struct {
    storage TimeSeriesStorage
}

func (hd *HistoricalData) HandleHistoryRequest(w http.ResponseWriter, r *http.Request) {
    metric := r.URL.Query().Get("metric")
    duration := r.URL.Query().Get("duration") // 1h, 24h, 7d
    
    data := hd.storage.GetMetricHistory(metric, duration)
    
    json.NewEncoder(w).Encode(map[string]interface{}{
        "metric": metric,
        "data":   data,
    })
}
```

## Performance & Scaling

### Connection Management
```go
type ConnectionPool struct {
    connections map[string]*Connection
    maxPerUser  int
    cleanup     *time.Ticker
}

func (cp *ConnectionPool) AddConnection(userID string, conn *Connection) error {
    userConns := cp.getUserConnections(userID)
    
    if len(userConns) >= cp.maxPerUser {
        // Close oldest connection
        oldestConn := userConns[0]
        oldestConn.Close()
    }
    
    cp.connections[conn.ID] = conn
    return nil
}
```

### Data Aggregation
```go
type MetricsAggregator struct {
    buffer []MetricPoint
    window time.Duration
    mutex  sync.Mutex
}

func (ma *MetricsAggregator) Aggregate() map[string]interface{} {
    ma.mutex.Lock()
    defer ma.mutex.Unlock()
    
    if len(ma.buffer) == 0 {
        return nil
    }
    
    // Calculate aggregated values
    result := map[string]interface{}{
        "avg_cpu":    ma.calculateAverage("cpu"),
        "max_memory": ma.calculateMax("memory"),
        "min_disk":   ma.calculateMin("disk"),
        "count":      len(ma.buffer),
    }
    
    // Clear buffer
    ma.buffer = ma.buffer[:0]
    
    return result
}
```

## Security & Access Control

### Authentication & Authorization
```go
type DashboardAuth struct {
    jwtSecret []byte
    roles     map[string][]Permission
}

func (da *DashboardAuth) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        
        claims, err := da.validateJWT(token)
        if err != nil {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        // Add user context
        ctx := context.WithValue(r.Context(), "user", claims)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func (da *DashboardAuth) CanExecuteCommand(userID, command, target string) bool {
    userRoles := da.getUserRoles(userID)
    
    for _, role := range userRoles {
        permissions := da.roles[role]
        for _, perm := range permissions {
            if perm.Allows(command, target) {
                return true
            }
        }
    }
    
    return false
}
```

### Audit Logging
```go
type AuditLogger struct {
    storage AuditStorage
}

func (al *AuditLogger) LogCommand(cmd Command, result CommandResult) {
    entry := AuditEntry{
        UserID:    cmd.UserID,
        Command:   cmd.Type,
        Target:    cmd.Target,
        Success:   result.Success,
        Message:   result.Message,
        Timestamp: time.Now(),
        IP:        cmd.SourceIP,
    }
    
    al.storage.Store(entry)
}
```

## Testing & Monitoring

### Load Testing
```bash
# Test SSE connection capacity
artillery run load-test.yml

# load-test.yml
config:
  target: 'http://localhost:8080'
  phases:
    - duration: 60
      arrivalRate: 10
scenarios:
  - name: Dashboard SSE
    flow:
      - get:
          url: "/dashboard/stream"
          headers:
            Accept: "text/event-stream"
```

### Health Checks
```go
func (ds *DashboardServer) HealthCheck(w http.ResponseWriter, r *http.Request) {
    status := map[string]interface{}{
        "status":            "healthy",
        "active_clients":    len(ds.clients),
        "uptime":           time.Since(ds.startTime),
        "memory_usage":     getCurrentMemoryUsage(),
        "last_metric_time": ds.lastMetricTime,
    }
    
    json.NewEncoder(w).Encode(status)
}
```

---

**Real-Time Monitoring Made Simple** 📊

*Build powerful dashboards with instant updates and reliable controls using SSE + HTTP POST patterns.*