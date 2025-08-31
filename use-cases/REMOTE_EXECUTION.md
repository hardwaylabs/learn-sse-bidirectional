# Use Case: Remote Code Execution

Building secure remote execution systems using SSE + HTTP POST for real-time command execution, output streaming, and interactive control.

## Architecture Overview

```
┌─────────────┐    Commands    ┌──────────────┐    Execute     ┌─────────────┐
│   Client    │───────────────►│   Execution  │───────────────►│   Target    │
│   (IDE/CLI) │                │   Server     │                │   System    │
└─────────────┘                └──────────────┘                └─────────────┘
       ▲                               │                               │
       │ SSE (Output Stream)          │                               │
       └──────────────────────────────┘                               │
                                       │◄──────────────────────────────┘
                                    Process Output
```

## Why SSE + HTTP POST for Remote Execution?

### Real-Time Output Streaming
- **Live Feedback**: See command output as it happens, not after completion
- **Interactive Debugging**: Watch build processes, test runs, deployments in real-time
- **Progress Monitoring**: Track long-running operations with immediate visual feedback
- **Multiple Clients**: Share execution sessions across team members

### Reliable Command Delivery
- **HTTP POST**: Ensures commands are delivered and acknowledged
- **Error Handling**: Clear success/failure responses for command submission
- **Authentication**: Standard HTTP auth for secure command execution
- **Audit Trail**: Complete log of who executed what commands when

### Advantages over Alternatives
| Feature | SSE + HTTP | SSH | WebSockets | gRPC Streaming |
|---------|------------|-----|------------|----------------|
| **Real-time Output** | ✅ Instant | ✅ Terminal | ✅ Instant | ✅ Instant |
| **Web Integration** | ✅ Native | ❌ Complex | ⚠️ Upgrade needed | ❌ Binary protocol |
| **Firewall Friendly** | ✅ HTTP only | ⚠️ Port 22 | ⚠️ Upgrade blocked | ⚠️ Port config |
| **Multi-client** | ✅ Built-in | ❌ Session per client | ⚠️ Complex | ⚠️ Complex |
| **Debug Tools** | ✅ Browser DevTools | ❌ SSH specific | ⚠️ Special tools | ❌ Proto tools |

## Implementation Pattern

### Client Side (IDE Integration)
```typescript
// RemoteExecutor.ts
export class RemoteExecutor {
    private eventSource: EventSource | null = null;
    private sessionId: string;
    
    constructor(private serverUrl: string, private authToken: string) {
        this.sessionId = this.generateSessionId();
    }
    
    async executeCommand(command: string, workingDir: string = '.'): Promise<ExecutionSession> {
        // Start execution via HTTP POST
        const response = await fetch(`${this.serverUrl}/execute`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${this.authToken}`,
            },
            body: JSON.stringify({
                command,
                working_directory: workingDir,
                session_id: this.sessionId,
                environment: this.getEnvironmentVars(),
            }),
        });
        
        if (!response.ok) {
            throw new Error(`Command failed: ${response.statusText}`);
        }
        
        const { execution_id } = await response.json();
        
        // Connect to output stream
        return this.streamOutput(execution_id);
    }
    
    private streamOutput(executionId: string): ExecutionSession {
        const session = new ExecutionSession(executionId);
        
        // Connect to SSE stream for real-time output
        this.eventSource = new EventSource(
            `${this.serverUrl}/execution/${executionId}/stream`
        );
        
        this.eventSource.onmessage = (event) => {
            const data = JSON.parse(event.data);
            
            switch (data.type) {
                case 'stdout':
                    session.onStdout(data.content);
                    break;
                case 'stderr':
                    session.onStderr(data.content);
                    break;
                case 'exit':
                    session.onExit(data.code);
                    this.eventSource?.close();
                    break;
                case 'error':
                    session.onError(data.message);
                    break;
            }
        };
        
        return session;
    }
    
    async sendInput(executionId: string, input: string): Promise<void> {
        await fetch(`${this.serverUrl}/execution/${executionId}/input`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${this.authToken}`,
            },
            body: JSON.stringify({ input }),
        });
    }
    
    async terminateExecution(executionId: string): Promise<void> {
        await fetch(`${this.serverUrl}/execution/${executionId}/terminate`, {
            method: 'POST',
            headers: { 'Authorization': `Bearer ${this.authToken}` },
        });
    }
}

class ExecutionSession {
    constructor(public readonly id: string) {}
    
    onStdout(content: string) {
        this.appendOutput(content, 'stdout');
    }
    
    onStderr(content: string) {
        this.appendOutput(content, 'stderr');
    }
    
    onExit(code: number) {
        this.showExitStatus(code);
    }
    
    onError(message: string) {
        this.showError(message);
    }
    
    private appendOutput(content: string, type: 'stdout' | 'stderr') {
        // Update IDE terminal/output panel
        const outputPanel = this.getOutputPanel();
        outputPanel.append(content, type);
    }
}
```

### Server Side (Go)
```go
type ExecutionServer struct {
    sessions    map[string]*ExecutionSession
    mutex       sync.RWMutex
    authManager *AuthManager
}

type ExecutionSession struct {
    ID          string
    Command     string
    WorkingDir  string
    Process     *exec.Cmd
    Clients     map[string]chan OutputEvent
    StartTime   time.Time
    Status      string
    ExitCode    *int
    mutex       sync.RWMutex
}

type OutputEvent struct {
    Type      string `json:"type"`      // stdout, stderr, exit, error
    Content   string `json:"content,omitempty"`
    Code      *int   `json:"code,omitempty"`
    Message   string `json:"message,omitempty"`
    Timestamp int64  `json:"timestamp"`
}

// HTTP POST endpoint for command execution
func (es *ExecutionServer) HandleExecute(w http.ResponseWriter, r *http.Request) {
    // Parse and validate request
    var req ExecuteRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // Check authentication and permissions
    user := es.authManager.GetUser(r)
    if !user.CanExecute(req.Command) {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }
    
    // Create execution session
    session := &ExecutionSession{
        ID:         generateExecutionID(),
        Command:    req.Command,
        WorkingDir: req.WorkingDirectory,
        Clients:    make(map[string]chan OutputEvent),
        StartTime:  time.Now(),
        Status:     "starting",
    }
    
    es.addSession(session)
    
    // Start command execution
    go es.executeCommand(session)
    
    // Return execution ID
    json.NewEncoder(w).Encode(map[string]string{
        "execution_id": session.ID,
        "status":       "started",
    })
}

// SSE endpoint for streaming output
func (es *ExecutionServer) HandleStream(w http.ResponseWriter, r *http.Request) {
    executionID := mux.Vars(r)["executionId"]
    clientID := generateClientID()
    
    session := es.getSession(executionID)
    if session == nil {
        http.Error(w, "Execution not found", http.StatusNotFound)
        return
    }
    
    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    // Create client channel
    clientChan := make(chan OutputEvent, 100)
    session.addClient(clientID, clientChan)
    defer session.removeClient(clientID)
    
    // Send historical output if available
    es.sendHistoricalOutput(session, clientChan)
    
    // Stream live output
    for event := range clientChan {
        eventJSON, _ := json.Marshal(event)
        fmt.Fprintf(w, "data: %s\n\n", eventJSON)
        
        if flusher, ok := w.(http.Flusher); ok {
            flusher.Flush()
        }
    }
}

func (es *ExecutionServer) executeCommand(session *ExecutionSession) {
    // Prepare command
    cmd := exec.Command("bash", "-c", session.Command)
    cmd.Dir = session.WorkingDir
    
    // Create pipes for stdout/stderr
    stdoutPipe, _ := cmd.StdoutPipe()
    stderrPipe, _ := cmd.StderrPipe()
    
    // Start command
    if err := cmd.Start(); err != nil {
        session.broadcastEvent(OutputEvent{
            Type:      "error",
            Message:   fmt.Sprintf("Failed to start command: %v", err),
            Timestamp: time.Now().Unix(),
        })
        return
    }
    
    session.Process = cmd
    session.Status = "running"
    
    // Stream stdout
    go es.streamPipe(stdoutPipe, "stdout", session)
    
    // Stream stderr
    go es.streamPipe(stderrPipe, "stderr", session)
    
    // Wait for completion
    err := cmd.Wait()
    
    exitCode := 0
    if err != nil {
        if exitError, ok := err.(*exec.ExitError); ok {
            exitCode = exitError.ExitCode()
        } else {
            exitCode = 1
        }
    }
    
    session.ExitCode = &exitCode
    session.Status = "completed"
    
    // Send exit event
    session.broadcastEvent(OutputEvent{
        Type:      "exit",
        Code:      &exitCode,
        Timestamp: time.Now().Unix(),
    })
}

func (es *ExecutionServer) streamPipe(pipe io.ReadCloser, outputType string, session *ExecutionSession) {
    defer pipe.Close()
    
    scanner := bufio.NewScanner(pipe)
    for scanner.Scan() {
        line := scanner.Text()
        
        event := OutputEvent{
            Type:      outputType,
            Content:   line + "\n",
            Timestamp: time.Now().Unix(),
        }
        
        session.broadcastEvent(event)
    }
}
```

## Interactive Features

### Input Handling
```go
// HTTP POST endpoint for sending input to running process
func (es *ExecutionServer) HandleInput(w http.ResponseWriter, r *http.Request) {
    executionID := mux.Vars(r)["executionId"]
    
    var req InputRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    session := es.getSession(executionID)
    if session == nil {
        http.Error(w, "Execution not found", http.StatusNotFound)
        return
    }
    
    // Send input to process stdin
    if session.Process != nil && session.Status == "running" {
        if stdin := session.Process.Stdin; stdin != nil {
            _, err := stdin.Write([]byte(req.Input + "\n"))
            if err != nil {
                http.Error(w, "Failed to send input", http.StatusInternalServerError)
                return
            }
        }
    }
    
    w.WriteHeader(http.StatusOK)
}
```

### Process Control
```go
func (es *ExecutionServer) HandleTerminate(w http.ResponseWriter, r *http.Request) {
    executionID := mux.Vars(r)["executionId"]
    
    session := es.getSession(executionID)
    if session == nil {
        http.Error(w, "Execution not found", http.StatusNotFound)
        return
    }
    
    if session.Process != nil && session.Status == "running" {
        // Send SIGTERM first
        session.Process.Process.Signal(os.Interrupt)
        
        // Wait a moment, then SIGKILL if necessary
        go func() {
            time.Sleep(5 * time.Second)
            if session.Status == "running" {
                session.Process.Process.Kill()
            }
        }()
        
        session.broadcastEvent(OutputEvent{
            Type:      "terminated",
            Message:   "Process terminated by user",
            Timestamp: time.Now().Unix(),
        })
    }
    
    w.WriteHeader(http.StatusOK)
}
```

## Security & Sandboxing

### Command Validation
```go
type CommandValidator struct {
    allowedCommands []string
    blockedPatterns []string
    pathRestrictions []string
}

func (cv *CommandValidator) Validate(cmd string, workingDir string) error {
    // Check for dangerous patterns
    for _, pattern := range cv.blockedPatterns {
        if matched, _ := regexp.MatchString(pattern, cmd); matched {
            return fmt.Errorf("command contains blocked pattern: %s", pattern)
        }
    }
    
    // Validate working directory
    absPath, err := filepath.Abs(workingDir)
    if err != nil {
        return fmt.Errorf("invalid working directory: %v", err)
    }
    
    allowed := false
    for _, allowedPath := range cv.pathRestrictions {
        if strings.HasPrefix(absPath, allowedPath) {
            allowed = true
            break
        }
    }
    
    if !allowed {
        return fmt.Errorf("working directory not allowed: %s", absPath)
    }
    
    return nil
}
```

### Resource Limits
```go
func (es *ExecutionServer) executeCommandWithLimits(session *ExecutionSession) {
    cmd := exec.Command("bash", "-c", session.Command)
    cmd.Dir = session.WorkingDir
    
    // Set resource limits
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setpgid: true, // Create new process group for easier cleanup
    }
    
    // Memory limit (using cgroups on Linux)
    if runtime.GOOS == "linux" {
        es.setCgroupLimits(cmd, session.ID)
    }
    
    // Timeout handling
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
    defer cancel()
    
    cmd = exec.CommandContext(ctx, "bash", "-c", session.Command)
    
    // Execute with monitoring
    es.executeWithMonitoring(cmd, session)
}

func (es *ExecutionServer) setCgroupLimits(cmd *exec.Cmd, sessionID string) {
    // Create cgroup for this execution
    cgroupPath := fmt.Sprintf("/sys/fs/cgroup/memory/remote-exec/%s", sessionID)
    os.MkdirAll(cgroupPath, 0755)
    
    // Set memory limit (512MB)
    ioutil.WriteFile(
        filepath.Join(cgroupPath, "memory.limit_in_bytes"),
        []byte("536870912"),
        0644,
    )
    
    // Add process to cgroup after start
    go func() {
        if cmd.Process != nil {
            ioutil.WriteFile(
                filepath.Join(cgroupPath, "cgroup.procs"),
                []byte(fmt.Sprintf("%d", cmd.Process.Pid)),
                0644,
            )
        }
    }()
}
```

### Container Isolation
```go
type DockerExecutor struct {
    client *client.Client
}

func (de *DockerExecutor) ExecuteInContainer(session *ExecutionSession) {
    ctx := context.Background()
    
    // Create container
    resp, err := de.client.ContainerCreate(ctx, &container.Config{
        Image:        "ubuntu:latest",
        Cmd:          []string{"bash", "-c", session.Command},
        WorkingDir:   "/workspace",
        AttachStdout: true,
        AttachStderr: true,
        OpenStdin:    true,
    }, &container.HostConfig{
        AutoRemove: true,
        Memory:     512 * 1024 * 1024, // 512MB limit
        CPUQuota:   50000,             // 50% CPU limit
        Binds:      []string{"/tmp/workspace:/workspace:rw"},
    }, nil, nil, "")
    
    if err != nil {
        session.broadcastEvent(OutputEvent{
            Type:    "error",
            Message: fmt.Sprintf("Failed to create container: %v", err),
        })
        return
    }
    
    containerID := resp.ID
    
    // Start container
    if err := de.client.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
        session.broadcastEvent(OutputEvent{
            Type:    "error",
            Message: fmt.Sprintf("Failed to start container: %v", err),
        })
        return
    }
    
    // Attach to container output
    de.streamContainerOutput(ctx, containerID, session)
}
```

## IDE Integration Examples

### VS Code Extension
```typescript
// extension.ts
import * as vscode from 'vscode';

export function activate(context: vscode.ExtensionContext) {
    const remoteExecutor = new RemoteExecutor(
        'https://remote-exec.example.com',
        getAuthToken()
    );
    
    // Command to execute current file
    const executeFileCommand = vscode.commands.registerCommand(
        'remote-exec.executeFile',
        async () => {
            const editor = vscode.window.activeTextEditor;
            if (!editor) return;
            
            const filePath = editor.document.fileName;
            const workspaceFolder = vscode.workspace.getWorkspaceFolder(editor.document.uri);
            
            // Create output channel
            const outputChannel = vscode.window.createOutputChannel('Remote Execution');
            outputChannel.show();
            
            try {
                const session = await remoteExecutor.executeCommand(
                    `python "${filePath}"`,
                    workspaceFolder?.uri.fsPath
                );
                
                // Stream output to VS Code terminal
                session.onStdout((content) => outputChannel.append(content));
                session.onStderr((content) => outputChannel.append(content));
                session.onExit((code) => {
                    outputChannel.appendLine(`Process exited with code ${code}`);
                });
                
            } catch (error) {
                vscode.window.showErrorMessage(`Execution failed: ${error.message}`);
            }
        }
    );
    
    context.subscriptions.push(executeFileCommand);
}
```

### JetBrains Plugin
```kotlin
// RemoteExecutionAction.kt
class RemoteExecutionAction : AnAction() {
    override fun actionPerformed(e: AnActionEvent) {
        val project = e.project ?: return
        val file = e.getData(CommonDataKeys.VIRTUAL_FILE) ?: return
        
        val executor = RemoteExecutor(
            "https://remote-exec.example.com",
            getAuthToken()
        )
        
        // Create tool window for output
        val toolWindow = ToolWindowManager.getInstance(project)
            .getToolWindow("Remote Execution")
        
        val outputPanel = OutputPanel()
        toolWindow.contentManager.addContent(
            ContentFactory.SERVICE.getInstance().createContent(
                outputPanel,
                file.name,
                false
            )
        )
        
        // Execute file
        executor.executeCommand("python ${file.path}") { session ->
            session.onStdout { content ->
                ApplicationManager.getApplication().invokeLater {
                    outputPanel.appendOutput(content, OutputType.STDOUT)
                }
            }
            
            session.onStderr { content ->
                ApplicationManager.getApplication().invokeLater {
                    outputPanel.appendOutput(content, OutputType.STDERR)
                }
            }
        }
    }
}
```

## Advanced Use Cases

### Distributed Build System
```go
type BuildCoordinator struct {
    workers map[string]*WorkerNode
    queue   chan BuildJob
}

type BuildJob struct {
    ID          string
    Repository  string
    Branch      string
    Commands    []string
    Artifacts   []string
}

func (bc *BuildCoordinator) DistributeBuild(job BuildJob) {
    // Find available worker
    worker := bc.selectWorker()
    
    // Execute build commands sequentially
    for i, command := range job.Commands {
        session := worker.ExecuteCommand(command, job.Repository)
        
        // Stream build output with job context
        session.OnOutput(func(output string) {
            bc.broadcastBuildOutput(job.ID, i, output)
        })
        
        // Wait for completion
        exitCode := session.Wait()
        if exitCode != 0 {
            bc.broadcastBuildFailure(job.ID, i, exitCode)
            return
        }
    }
    
    bc.broadcastBuildSuccess(job.ID)
}
```

### Interactive Debugging
```go
type DebugSession struct {
    ExecutionSession
    breakpoints []Breakpoint
    variables   map[string]interface{}
}

func (ds *DebugSession) SetBreakpoint(file string, line int) {
    // Send debugger command
    ds.sendInput(fmt.Sprintf("b %s:%d", file, line))
    
    // Add to breakpoints list
    ds.breakpoints = append(ds.breakpoints, Breakpoint{
        File: file,
        Line: line,
    })
}

func (ds *DebugSession) StepOver() {
    ds.sendInput("n")
}

func (ds *DebugSession) Continue() {
    ds.sendInput("c")
}
```

### Collaborative Execution
```go
type SharedSession struct {
    *ExecutionSession
    participants map[string]User
    permissions  map[string][]Permission
}

func (ss *SharedSession) AddParticipant(user User, role string) {
    ss.participants[user.ID] = user
    ss.permissions[user.ID] = getRolePermissions(role)
    
    // Notify all participants
    ss.broadcastEvent(OutputEvent{
        Type:    "participant_joined",
        Message: fmt.Sprintf("%s joined the session", user.Name),
    })
}

func (ss *SharedSession) HandleCommand(userID, command string) {
    if !ss.canExecute(userID) {
        ss.sendToUser(userID, OutputEvent{
            Type:    "error",
            Message: "You don't have permission to execute commands",
        })
        return
    }
    
    // Broadcast command to all participants
    ss.broadcastEvent(OutputEvent{
        Type:    "command",
        Message: fmt.Sprintf("%s executed: %s", ss.participants[userID].Name, command),
    })
    
    // Execute command normally
    ss.ExecutionSession.executeCommand(command)
}
```

## Performance & Monitoring

### Resource Monitoring
```go
type ResourceMonitor struct {
    sessions map[string]*ExecutionSession
}

func (rm *ResourceMonitor) MonitorSession(session *ExecutionSession) {
    ticker := time.NewTicker(1 * time.Second)
    
    go func() {
        for range ticker.C {
            if session.Process == nil {
                return
            }
            
            // Get process stats
            pid := session.Process.Process.Pid
            stats := rm.getProcessStats(pid)
            
            // Broadcast resource usage
            session.broadcastEvent(OutputEvent{
                Type: "resource_usage",
                Data: map[string]interface{}{
                    "cpu":    stats.CPUPercent,
                    "memory": stats.MemoryMB,
                    "pid":    pid,
                },
                Timestamp: time.Now().Unix(),
            })
            
            // Check limits
            if stats.MemoryMB > 1000 { // 1GB limit
                session.Process.Process.Kill()
                session.broadcastEvent(OutputEvent{
                    Type:    "killed",
                    Message: "Process killed due to excessive memory usage",
                })
                return
            }
        }
    }()
}
```

### Load Balancing
```go
type LoadBalancer struct {
    servers []ExecutionServer
    metrics map[string]ServerMetrics
}

func (lb *LoadBalancer) SelectServer() *ExecutionServer {
    var bestServer *ExecutionServer
    var lowestLoad float64 = math.MaxFloat64
    
    for _, server := range lb.servers {
        metrics := lb.metrics[server.ID]
        load := metrics.CalculateLoad()
        
        if load < lowestLoad {
            lowestLoad = load
            bestServer = &server
        }
    }
    
    return bestServer
}
```

---

**Remote Execution Made Secure** ⚡

*Build powerful remote development environments with real-time feedback and secure command execution using SSE + HTTP POST patterns.*