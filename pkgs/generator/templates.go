package generator

// Updated template component definitions with proper shell command escaping

const packageTemplate = `{{define "package"}}package {{.PackageName}}{{end}}`

const importsTemplate = `{{define "imports"}}
import (
{{- range .Imports}}
	"{{.}}"
{{- end}}
)
{{end}}`

const processTypesTemplate = `{{define "process-types"}}{{if .HasProcessMgmt}}
// ProcessInfo represents a managed background process
type ProcessInfo struct {
	Name      string    ` + "`json:\"name\"`" + `
	PID       int       ` + "`json:\"pid\"`" + `
	Command   string    ` + "`json:\"command\"`" + `
	StartTime time.Time ` + "`json:\"start_time\"`" + `
	LogFile   string    ` + "`json:\"log_file\"`" + `
	Status    string    ` + "`json:\"status\"`" + `
}

// ProcessRegistry manages background processes
type ProcessRegistry struct {
	dir       string
	processes map[string]*ProcessInfo
}
{{- end}}{{end}}`

const processRegistryTemplate = `{{define "process-registry"}}{{if .HasProcessMgmt}}
// NewProcessRegistry creates a new process registry
func NewProcessRegistry() *ProcessRegistry {
	dir := ".devcmd"
	os.MkdirAll(dir, 0755)

	pr := &ProcessRegistry{
		dir:       dir,
		processes: make(map[string]*ProcessInfo),
	}
	pr.loadProcesses()
	return pr
}

// loadProcesses loads existing processes from registry file
func (pr *ProcessRegistry) loadProcesses() {
	registryFile := filepath.Join(pr.dir, "registry.json")
	data, err := os.ReadFile(registryFile)
	if err != nil {
		return // File doesn't exist or can't be read
	}

	var processes map[string]*ProcessInfo
	if err := json.Unmarshal(data, &processes); err != nil {
		return
	}

	// Verify processes are still running
	for name, proc := range processes {
		if pr.isProcessRunning(proc.PID) {
			proc.Status = "running"
			pr.processes[name] = proc
		}
	}
	pr.saveProcesses()
}

// saveProcesses saves current processes to registry file
func (pr *ProcessRegistry) saveProcesses() {
	registryFile := filepath.Join(pr.dir, "registry.json")
	data, err := json.MarshalIndent(pr.processes, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(registryFile, data, 0644)
}

// isProcessRunning checks if a process is still running
func (pr *ProcessRegistry) isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// addProcess adds a process to the registry
func (pr *ProcessRegistry) addProcess(name string, pid int, command string, logFile string) {
	pr.processes[name] = &ProcessInfo{
		Name:      name,
		PID:       pid,
		Command:   command,
		StartTime: time.Now(),
		LogFile:   logFile,
		Status:    "running",
	}
	pr.saveProcesses()
}

// removeProcess removes a process from the registry
func (pr *ProcessRegistry) removeProcess(name string) {
	delete(pr.processes, name)
	pr.saveProcesses()
}

// getProcess gets a process by name
func (pr *ProcessRegistry) getProcess(name string) (*ProcessInfo, bool) {
	proc, exists := pr.processes[name]
	return proc, exists
}

// listProcesses returns all processes
func (pr *ProcessRegistry) listProcesses() []*ProcessInfo {
	var procs []*ProcessInfo
	for _, proc := range pr.processes {
		procs = append(procs, proc)
	}
	return procs
}

// gracefulStop attempts to stop a process gracefully
func (pr *ProcessRegistry) gracefulStop(name string) error {
	proc, exists := pr.getProcess(name)
	if !exists {
		return fmt.Errorf("no process named '%s' found", name)
	}

	// Try to terminate gracefully
	process, err := os.FindProcess(proc.PID)
	if err != nil {
		pr.removeProcess(name)
		return fmt.Errorf("process not found: %v", err)
	}

	fmt.Printf("Stopping process %s (PID: %d)...\n", name, proc.PID)

	// Send SIGTERM
	process.Signal(syscall.SIGTERM)

	// Wait up to 5 seconds for graceful shutdown
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			// Force kill
			fmt.Printf("Force killing process %s...\n", name)
			process.Signal(syscall.SIGKILL)
			pr.removeProcess(name)
			return nil
		case <-ticker.C:
			if !pr.isProcessRunning(proc.PID) {
				fmt.Printf("Process %s stopped successfully\n", name)
				pr.removeProcess(name)
				return nil
			}
		}
	}
}
{{- end}}{{end}}`

const cliStructTemplate = `{{define "cli-struct"}}
// CLI represents the command line interface
type CLI struct {
{{- if .HasProcessMgmt}}
	registry *ProcessRegistry
{{- end}}
}

// NewCLI creates a new CLI instance
func NewCLI() *CLI {
	return &CLI{
{{- if .HasProcessMgmt}}
		registry: NewProcessRegistry(),
{{- end}}
	}
}
{{end}}`

const commandSwitchTemplate = `{{define "command-switch"}}
// Execute runs the CLI with given arguments
func (c *CLI) Execute() {
	if len(os.Args) < 2 {
		{{if not .HasUserDefinedHelp}}c.showHelp(){{else}}fmt.Fprintf(os.Stderr, "Usage: %s <command>\nRun '%s help' for available commands.\n", os.Args[0], os.Args[0]){{end}}
		return
	}

	command := os.Args[1]
{{- if .Commands}}
	args := os.Args[2:]

	switch command {
{{- if .HasProcessMgmt}}
	case "status":
		c.showStatus()
{{- end}}
{{- range .Commands}}
	case "{{.GoCase}}":
		c.{{.FunctionName}}(args)
{{- end}}
{{- if not .HasUserDefinedHelp}}
	case "help", "--help", "-h":
		c.showHelp()
{{- end}}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		{{if not .HasUserDefinedHelp}}c.showHelp(){{else}}fmt.Fprintf(os.Stderr, "Run '%s help' for available commands.\n", os.Args[0]){{end}}
		os.Exit(1)
	}
{{- else}}
	switch command {
{{- if not .HasUserDefinedHelp}}
	case "help", "--help", "-h":
		c.showHelp()
{{- end}}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		{{if not .HasUserDefinedHelp}}c.showHelp(){{else}}fmt.Fprintf(os.Stderr, "Run '%s help' for available commands.\n", os.Args[0]){{end}}
		os.Exit(1)
	}
{{- end}}
}
{{end}}`

const helpFunctionTemplate = `{{define "help-function"}}{{if not .HasUserDefinedHelp}}
// showHelp displays available commands
func (c *CLI) showHelp() {
	fmt.Println("Available commands:")
{{- if .HasProcessMgmt}}
	fmt.Println("  status              - Show running background processes")
{{- end}}
{{- range .Commands}}
	fmt.Println("  {{.HelpDescription}}")
{{- end}}
}
{{- end}}{{end}}`

const statusFunctionTemplate = `{{define "status-function"}}{{if .HasProcessMgmt}}
// showStatus displays running processes
func (c *CLI) showStatus() {
	processes := c.registry.listProcesses()
	if len(processes) == 0 {
		fmt.Println("No background processes running")
		return
	}

	fmt.Printf("%-15s %-8s %-10s %-20s %s\n", "NAME", "PID", "STATUS", "STARTED", "COMMAND")
	fmt.Println(strings.Repeat("-", 80))

	for _, proc := range processes {
		// Verify process is still running
		if !c.registry.isProcessRunning(proc.PID) {
			proc.Status = "stopped"
		}

		startTime := proc.StartTime.Format("15:04:05")
		command := proc.Command
		if len(command) > 30 {
			command = command[:27] + "..."
		}

		fmt.Printf("%-15s %-8d %-10s %-20s %s\n",
			proc.Name, proc.PID, proc.Status, startTime, command)
	}
}
{{- end}}{{end}}`

const processMgmtFunctionsTemplate = `{{define "process-mgmt-functions"}}{{if .HasProcessMgmt}}
// showLogsFor displays logs for a specific process
func (c *CLI) showLogsFor(name string) {
	proc, exists := c.registry.getProcess(name)
	if !exists {
		fmt.Fprintf(os.Stderr, "No process named '%s' is currently running\n", name)
		os.Exit(1)
	}

	if _, err := os.Stat(proc.LogFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Log file not found: %s\n", proc.LogFile)
		os.Exit(1)
	}

	// Stream log file
	file, err := os.Open(proc.LogFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening log file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	io.Copy(os.Stdout, file)
}

// runInBackground starts a command in background with logging
func (c *CLI) runInBackground(name, command string) error {
	logFile := filepath.Join(c.registry.dir, name+".log")

	// Create or truncate log file
	logFileHandle, err := os.Create(logFile)
	if err != nil {
		return fmt.Errorf("failed to create log file: %v", err)
	}

	// Start command
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = logFileHandle
	cmd.Stderr = logFileHandle

	if err := cmd.Start(); err != nil {
		logFileHandle.Close()
		return fmt.Errorf("failed to start command: %v", err)
	}

	// Register process
	c.registry.addProcess(name, cmd.Process.Pid, command, logFile)

	fmt.Printf("Started %s in background (PID: %d)\n", name, cmd.Process.Pid)
	fmt.Printf("To see logs, run: %s %s logs\n", os.Args[0], name)

	// Monitor process in goroutine
	go func() {
		defer logFileHandle.Close()
		cmd.Wait()
		c.registry.removeProcess(name)
	}()

	return nil
}
{{- end}}{{end}}`

const commandFunctionsTemplate = `{{define "command-functions"}}
// Command implementations
{{- range .Commands}}

func (c *CLI) {{.FunctionName}}(args []string) {
{{- template "command-impl" .}}
}
{{- end}}
{{end}}`

const regularCommandTemplate = `{{define "regular-command"}}
	// Regular command
	cmd := exec.Command("sh", "-c", {{printf "%q" .ShellCommand}})
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Command failed: %v\n", err)
		os.Exit(1)
	}
{{- end}}`

const watchStopCommandTemplate = `{{define "watch-stop-command"}}
	// Watch/stop command pair
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: %s {{.Name}} <start|stop|logs>\n", os.Args[0])
		os.Exit(1)
	}

	subcommand := args[0]
	switch subcommand {
	case "start":
		command := {{printf "%q" .WatchCommand}}
		if err := c.runInBackground("{{.Name}}", command); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting {{.Name}}: %v\n", err)
			os.Exit(1)
		}
	case "stop":
		// Run custom stop command
		cmd := exec.Command("sh", "-c", {{printf "%q" .StopCommand}})
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Stop command failed: %v\n", err)
		}
		// Also stop via process registry
		if err := c.registry.gracefulStop("{{.Name}}"); err != nil {
			fmt.Fprintf(os.Stderr, "Registry stop failed: %v\n", err)
		}
	case "logs":
		c.showLogsFor("{{.Name}}")
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s. Use 'start', 'stop', or 'logs'\n", subcommand)
		os.Exit(1)
	}
{{- end}}`

const watchOnlyCommandTemplate = `{{define "watch-only-command"}}
	// Watch-only command (no custom stop)
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: %s {{.Name}} <start|stop|logs>\n", os.Args[0])
		os.Exit(1)
	}

	subcommand := args[0]
	switch subcommand {
	case "start":
		command := {{printf "%q" .WatchCommand}}
		if err := c.runInBackground("{{.Name}}", command); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting {{.Name}}: %v\n", err)
			os.Exit(1)
		}
	case "stop":
		// Use generic process management for stopping
		if err := c.registry.gracefulStop("{{.Name}}"); err != nil {
			fmt.Fprintf(os.Stderr, "Error stopping {{.Name}}: %v\n", err)
			os.Exit(1)
		}
	case "logs":
		c.showLogsFor("{{.Name}}")
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s. Use 'start', 'stop', or 'logs'\n", subcommand)
		os.Exit(1)
	}
{{- end}}`

const stopOnlyCommandTemplate = `{{define "stop-only-command"}}
	// Stop-only command (unusual case)
	// Run stop command
	cmd := exec.Command("sh", "-c", {{printf "%q" .StopCommand}})
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Stop command failed: %v\n", err)
	}
{{- end}}`

// Updated parallel command template using standard library with proper command escaping
const parallelCommandTemplate = `{{define "parallel-command"}}
	// Parallel command execution using standard library goroutines
	{
		var wg sync.WaitGroup
		errChan := make(chan error, {{len .ParallelCommands}})

		{{range .ParallelCommands}}
		wg.Add(1)
		go func() {
			defer wg.Done()
			cmd := exec.Command("sh", "-c", {{printf "%q" .}})
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				errChan <- fmt.Errorf("command failed: %v", err)
				return
			}
		}()
		{{end}}

		// Wait for all goroutines to complete
		go func() {
			wg.Wait()
			close(errChan)
		}()

		// Check for errors
		for err := range errChan {
			if err != nil {
				fmt.Fprintf(os.Stderr, "Parallel execution failed: %v\n", err)
				os.Exit(1)
			}
		}
	}
{{- end}}`

// Updated mixed command template with proper command escaping for multiple parallel segments
const mixedCommandTemplate = `{{define "mixed-command"}}
	// Mixed command with both parallel and sequential execution
	{{range $index, $segment := .CommandSegments}}
	{{if .IsParallel}}
	// Parallel segment using standard library
	{
		var wg sync.WaitGroup
		errChan := make(chan error, {{len .Commands}})

		{{range .Commands}}
		wg.Add(1)
		go func() {
			defer wg.Done()
			cmd := exec.Command("sh", "-c", {{printf "%q" .}})
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				errChan <- fmt.Errorf("command failed: %v", err)
				return
			}
		}()
		{{end}}

		go func() {
			wg.Wait()
			close(errChan)
		}()

		for err := range errChan {
			if err != nil {
				fmt.Fprintf(os.Stderr, "Parallel execution failed: %v\n", err)
				os.Exit(1)
			}
		}
	}
	{{else}}
	// Sequential command
	{
		cmd := exec.Command("sh", "-c", {{printf "%q" .Command}})
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Run(); err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				os.Exit(exitError.ExitCode())
			}
			fmt.Fprintf(os.Stderr, "Command failed: %v\n", err)
			os.Exit(1)
		}
	}
	{{end}}
	{{end}}
{{- end}}`

const mainFunctionTemplate = `{{define "main-function"}}
func main() {
	cli := NewCLI()
{{- if .HasProcessMgmt}}

	// Handle interrupt signals gracefully
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nShutting down...")
		os.Exit(0)
	}()
{{- end}}

	cli.Execute()
}
{{end}}`

// Master template that composes all components with improved formatting
const masterTemplate = `{{define "main"}}{{template "package" .}}

{{template "imports" .}}
{{template "process-types" .}}
{{template "process-registry" .}}
{{template "cli-struct" .}}
{{template "command-switch" .}}
{{template "help-function" .}}
{{template "status-function" .}}
{{template "process-mgmt-functions" .}}
{{template "command-functions" .}}
{{template "main-function" .}}{{end}}

	{{define "command-impl"}}{{if eq .Type "regular"}}{{template "regular-command" .}}{{else if eq .Type "watch-stop"}}{{template "watch-stop-command" .}}{{else if eq .Type "watch-only"}}{{template "watch-only-command" .}}{{else if eq .Type "stop-only"}}{{template "stop-only-command" .}}{{else if eq .Type "parallel"}}{{template "parallel-command" .}}{{else if eq .Type "mixed"}}{{template "mixed-command" .}}{{end}}{{end}}`
