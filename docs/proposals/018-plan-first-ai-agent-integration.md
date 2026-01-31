---
oep: 018
title: Plan-First AI Agent Integration
description: Using Opal as the "show your work" layer for AI-generated infrastructure operations
status: Draft
type: Tooling
created: 2025-01-31
updated: 2025-01-31
---

# OEP-018: Plan-First AI Agent Integration

## Summary

Use Opal as the transparency and safety layer between AI agents and infrastructure operations. Instead of AI agents executing shell commands directly, they generate Opal scripts that create deterministic plans. Users review the plan (the "show your work" output) before execution, with Opal enforcing capability restrictions and providing audit trails. This addresses the core problem of AI opacity: users currently have no visibility into what an AI agent will do until it does it.

## Motivation

### The Problem

AI agents are increasingly used for infrastructure operations—deployments, configuration changes, troubleshooting. But there's a critical transparency gap:

**Current AI agent workflow:**
```
User: "Deploy my app to production"
AI:  "Okay, running commands..."
     [User waits, unable to see what's happening]
AI:  "Done! (hopefully nothing went wrong)"
```

**Problems with this approach:**

1. **No visibility**: User cannot see what commands will run until they execute
2. **No review**: No opportunity to catch mistakes before they affect production
3. **No verification**: No way to verify the AI's plan matches the intent
4. **No rollback plan**: No structured understanding of what was changed
5. **No audit trail**: No record of what the AI did for compliance/debugging

**Concrete example of the risk:**

```python
# AI agent directly executes shell commands
subprocess.run("kubectl delete -f k8s/", shell=True)
# Oops, wrong namespace! Production is down.
# User had no chance to review before execution.
```

### The Agent Skills Ecosystem

The Agent Skills specification has emerged as a cross-industry standard (adopted by GitHub Copilot, Claude Code, Cursor, OpenAI Codex, and others) for extending AI agent capabilities. This creates both opportunity and risk:

**Opportunity**: AI agents can express complex multi-step workflows as declarative skills rather than imperative code.

**Risk**: Skills executed without verification create a new attack surface. Prompt injection remains a significant vulnerability for LLM applications, particularly when agents execute operations without human review.

### Use Cases

**1. AI-assisted deployment with human review:**

```
User: "Deploy version 1.2.3 to production"

AI generates Opal script:
─────────────────────────────────────────
deploy: {
    var VERSION = "1.2.3"
    var ENV = "production"
    
    echo "Deploying @var.VERSION to @var.ENV"
    kubectl set image deployment/app app=myapp:@var.VERSION
    kubectl rollout status deployment/app
    @http.healthcheck(url="https://api.example.com/health")
}
─────────────────────────────────────────

Opal creates plan:
Plan: deploy
─────────────────────────────────────────
1. @shell("echo \"Deploying 1.2.3 to production\"")
2. @shell("kubectl set image deployment/app app=myapp:1.2.3")
3. @shell("kubectl rollout status deployment/app")
4. @http.healthcheck(url="https://api.example.com/health")

Estimated duration: 2-3 minutes
Network access: kubernetes.local, api.example.com
File access: none
─────────────────────────────────────────

User reviews: "Looks correct"
User executes: opal run --plan deploy.plan
```

**2. Infrastructure troubleshooting with safety:**

```
User: "Debug why the database is slow"

AI generates Opal script:
─────────────────────────────────────────
debug: {
    @postgres.explain(query="SELECT * FROM large_table")
    @postgres.show_indexes(table="large_table")
    @shell("kubectl logs -l app=database --tail=100")
}
─────────────────────────────────────────

Opal creates plan:
─────────────────────────────────────────
Read-only operations detected
No mutations planned
Safe to execute automatically
─────────────────────────────────────────

User executes with confidence
```

**3. CI/CD pipeline with approved plans:**

```yaml
# .github/workflows/deploy.yml
- name: AI Generate Deployment Plan
  run: |
    ai-agent generate-deployment --output deploy.opl
    opal plan deploy.opl --output deploy.plan
    
- name: Human Review (for production)
  if: github.ref == 'refs/heads/main'
  uses: actions/manual-approval@v1
  
- name: Execute Approved Plan
  run: opal run --plan deploy.plan
```

**4. Multi-step operations with checkpoint:**

```
User: "Set up a new production environment"

AI generates complex 15-step Opal script
Opal breaks it into phases with checkpoints:

Phase 1: Infrastructure (VPC, subnets, security groups)
Phase 2: Database (RDS instance, schema, users)
Phase 3: Application (EKS cluster, deployments, ingress)
Phase 4: Validation (health checks, smoke tests)

User reviews each phase separately
User can pause between phases
User can rollback any phase
```

**5. Compliance and audit requirements:**

```
User: "Show me what changes were made last week"

Opal shows from audit log:
─────────────────────────────────────────
2025-01-24 14:32:01 - User: alice - Plan: deploy-v1.2.3
  Executed by: AI agent (Claude Code)
  Approved by: alice (manual review)
  Commands: 4 shell, 2 kubectl, 1 healthcheck
  Duration: 3m 42s
  Exit code: 0
  
  Plan hash: sha256:abc123...
  Full plan: available at ./plans/deploy-v1.2.3.plan
─────────────────────────────────────────
```

## Proposal

### Core Concept

**Opal becomes the "show your work" layer for AI agents:**

```
┌─────────────────────────────────────────────────────────────┐
│  User Intent                                                │
│     ↓                                                       │
│  AI Agent (generates Opal, not shell)                       │
│     ↓                                                       │
│  Opal Parser → Plan (deterministic, reviewable)             │
│     ↓                                                       │
│  User Review (human or automated)                           │
│     ↓                                                       │
│  Opal Executor (with capability restrictions)               │
│     ↓                                                       │
│  Infrastructure + Audit Trail                               │
└─────────────────────────────────────────────────────────────┘
```

**Key insight**: The plan file is the contract between AI and user. It bridges the gap between natural language intent and infrastructure reality.

### How It Works

**Step 1: AI Generates Opal Script**

AI agents output Opal syntax instead of raw shell commands:

```opal
# AI-generated deployment script
deploy: {
    var IMAGE = "myapp:v1.2.3"
    var NAMESPACE = "production"
    
    echo "Deploying @var.IMAGE to @var.NAMESPACE"
    
    kubectl config set-context --current --namespace=@var.NAMESPACE
    kubectl set image deployment/app app=@var.IMAGE
    kubectl rollout status deployment/app --timeout=5m
    
    @retry(attempts=3, delay=10s) {
        curl -f https://api.example.com/health
    }
}
```

**Step 2: Opal Creates Deterministic Plan**

```bash
$ opal plan deploy.opl --output deploy.plan

Plan: deploy
─────────────────────────────────────────
Steps:
  1. @shell("echo \"Deploying myapp:v1.2.3 to production\"")
  2. @shell("kubectl config set-context --current --namespace=production")
  3. @shell("kubectl set image deployment/app app=myapp:v1.2.3")
  4. @shell("kubectl rollout status deployment/app --timeout=5m")
  5. @retry(attempts=3, delay=10s) { ... }
     └─ @shell("curl -f https://api.example.com/health")

Resources:
  Network: kubernetes.local (kubectl), api.example.com (healthcheck)
  Files: none
  Secrets: none

Safety:
  Mutations: Yes (deployment update)
  Rollback plan: kubectl rollout undo deployment/app
  Estimated duration: 5-7 minutes
─────────────────────────────────────────

Plan hash: sha256:a1b2c3d4...
Contract saved to: deploy.plan
```

**Step 3: User Reviews Plan**

The plan is human-readable and shows exactly what will happen:

```
- All commands are visible before execution
- All network access is declared
- Safety information (mutations, rollback, duration)
- No surprises
```

**Step 4: User Approves and Executes**

```bash
# Execute the reviewed plan
opal run --plan deploy.plan

# Or with additional restrictions
opal run --plan deploy.plan --dry-run  # Preview only
opal run --plan deploy.plan --step-by-step  # Pause between steps
opal run --plan deploy.plan --rollback-on-failure  # Auto-rollback
```

### Syntax and Integration

**AI Agent Integration:**

AI agents should be configured to output Opal syntax:

```python
# AI agent configuration
system_prompt = """
When performing infrastructure operations, generate Opal scripts instead of 
executing shell commands directly. Opal syntax:

- Variables: var NAME = "value"
- Variable use: @var.NAME
- Shell commands: kubectl apply -f file.yaml
- Decorators: @retry(attempts=3) { ... }
- Blocks: label: { ... }

Example:
deploy: {
    var IMAGE = "nginx:latest"
    kubectl set image deployment/app app=@var.IMAGE
}

The user will review the generated Opal script before execution.
"""
```

**Opal CLI for AI Workflows:**

```bash
# Generate plan from Opal script
opal plan script.opl --output plan.json

# Review plan (human-readable output)
opal plan script.opl --review

# Execute with confirmation
opal run --plan plan.json --confirm

# Execute specific phase only
opal run --plan plan.json --phase 1

# Compare two plans (what changed?)
opal plan diff plan-v1.json plan-v2.json

# Audit log
opal audit --since 2025-01-01 --format json
```

**Configuration for AI Agent Context:**

```json
// .opal/ai-config.json
{
  "aiIntegration": {
    "defaultMode": "plan-first",
    "capabilities": {
      "allow": ["kubectl", "docker", "terraform"],
      "deny": ["rm -rf", "dd", "mkfs"]
    },
    "restrictions": {
      "requireConfirmation": ["production/*", "*-prod"],
      "autoApprove": ["development/*", "*-dev"],
      "readOnly": ["get", "describe", "logs", "explain"]
    },
    "output": {
      "format": "opal",
      "includeComments": true,
      "includeSafetyInfo": true
    }
  }
}
```

## Core Restrictions

#### Restriction 1: AI Agents Must Generate Opal, Not Execute Directly

AI agents should output Opal scripts for review, not execute commands directly:

```python
# ❌ FORBIDDEN: AI executing directly
subprocess.run("kubectl delete pod app-xyz", shell=True)

# ✅ CORRECT: AI generates Opal script
opal_script = """
deploy: {
    kubectl delete pod app-xyz
}
"""
# User reviews and executes
```

**Why?** Direct execution bypasses all safety mechanisms. The plan-first model requires AI to express intent in a reviewable format.

#### Restriction 2: Plans Must Be Deterministic

Plans generated from the same Opal script must produce identical hashes:

```bash
# ✅ CORRECT: Deterministic plan
opal plan script.opl > plan1.json
opal plan script.opl > plan2.json
diff plan1.json plan2.json  # No difference

# ❌ FORBIDDEN: Non-deterministic elements
# Timestamps in plan body
# Random values without fixed seed
# System-dependent paths
```

**Why?** Determinism enables plan verification. If the plan hash changes, something is different (either the script or the environment).

#### Restriction 3: Capabilities Must Be Declared and Approved

AI-generated plans must declare required capabilities:

```json
// Plan includes capability requirements
{
  "steps": [...],
  "capabilities": {
    "network": ["kubernetes.local", "api.example.com"],
    "filesystem": ["./k8s/*"],
    "commands": ["kubectl", "curl"]
  }
}
```

Execution fails if capabilities exceed approved scope:

```bash
# ❌ FORBIDDEN: Plan requests unapproved capability
opal run --plan plan.json
# Error: Plan requires network:external:* but only network:internal:* approved

# ✅ CORRECT: Approve capabilities or modify plan
opal run --plan plan.json --approve-capabilities network:external:api.example.com
```

**Why?** Capability restrictions limit blast radius if AI generates malicious or mistaken instructions.

#### Restriction 4: High-Risk Operations Require Explicit Confirmation

Plans with high-risk operations (deletions, production changes) require explicit confirmation:

```opal
// ❌ This would require confirmation
deploy: {
    kubectl delete namespace production  # High-risk: deletion
}

// Or flagged by configuration
deploy: {
    kubectl apply -f k8s/production/  # High-risk: production environment
}
```

```bash
opal run --plan risky.plan
# ⚠️  HIGH-RISK OPERATIONS DETECTED:
#    - Delete: namespace/production
#    - Environment: production
#
#    Approve? [y/N]: 
```

**Why?** Prevents accidental execution of destructive operations, even if AI generated them.

#### Restriction 5: Plans Must Include Rollback Information

Plans with mutations must include rollback instructions:

```json
{
  "steps": [
    {
      "type": "mutation",
      "action": "kubectl set image deployment/app",
      "rollback": "kubectl rollout undo deployment/app"
    }
  ]
}
```

**Why?** Enables automatic or manual rollback if something goes wrong.

#### Restriction 6: Audit Trail Is Mandatory

All AI-generated plan executions must be logged:

```json
{
  "timestamp": "2025-01-31T10:30:00Z",
  "aiAgent": "Claude Code",
  "user": "alice",
  "planHash": "sha256:a1b2c3d4...",
  "planPath": "./plans/deploy-v1.2.3.plan",
  "opScriptPath": "./scripts/deploy.opl",
  "result": "success",
  "duration": "3m 42s"
}
```

**Why?** Compliance, debugging, and accountability require complete audit trails.

## Semantics

### Plan Generation

AI-generated Opal scripts are parsed into deterministic plans:

1. **Parse**: Convert Opal syntax to AST
2. **Resolve**: Resolve variables and decorators
3. **Plan**: Generate execution plan with all steps
4. **Hash**: Compute deterministic hash of plan
5. **Store**: Save plan for review and execution

### Capability Declaration

AI-generated plans declare required capabilities based on content analysis:

```python
# Plan analyzer extracts capabilities
def extract_capabilities(plan):
    capabilities = {
        "network": set(),
        "filesystem": set(),
        "commands": set(),
        "mutations": []
    }
    
    for step in plan.steps:
        if step.type == "shell":
            capabilities["commands"].add(step.command)
        if step.type == "kubectl":
            capabilities["network"].add("kubernetes.local")
            if step.verb in ["apply", "delete", "set"]:
                capabilities["mutations"].append(step)
    
    return capabilities
```

### Confirmation Checkpoints

High-risk operations trigger confirmation checkpoints:

```python
def requires_confirmation(plan, user_context):
    for mutation in plan.mutations:
        # Production environment check
        if is_production_environment(mutation):
            return True
        
        # Deletion check
        if mutation.type == "delete":
            return True
        
        # Scope check
        if mutation.scope in user_context.restricted_scopes:
            return True
    
    return False
```

### Rollback Generation

For each mutation, generate rollback instruction:

```python
def generate_rollback(mutation):
    if mutation.type == "kubectl":
        if mutation.verb == "set":
            return f"kubectl rollout undo {mutation.resource}"
        if mutation.verb == "apply":
            return f"kubectl delete -f {mutation.file}"
    
    return None  # No automatic rollback available
```

### Audit Logging

Every plan execution is logged with full context:

```python
audit_entry = {
    "timestamp": datetime.utcnow().isoformat(),
    "ai_agent": detect_ai_agent(),  # From environment
    "user": get_current_user(),
    "plan": {
        "hash": compute_plan_hash(plan),
        "path": plan_path,
        "capabilities": plan.capabilities
    },
    "execution": {
        "start_time": start_time,
        "end_time": end_time,
        "duration": duration,
        "exit_code": exit_code,
        "steps_run": steps_run
    },
    "approval": {
        "method": approval_method,  # "manual", "auto", "policy"
        "approver": approver,
        "timestamp": approval_timestamp
    }
}
```

## Examples

### Example 1: Simple Deployment with Review

```
User: "Deploy myapp v1.2.3 to production"

AI generates:
─────────────────────────────────────────
// deploy.opl - Generated by Claude Code
deploy: {
    var VERSION = "1.2.3"
    var NAMESPACE = "production"
    
    echo "Deploying myapp:@var.VERSION to @var.NAMESPACE"
    kubectl set image deployment/myapp app=myapp:@var.VERSION -n @var.NAMESPACE
    kubectl rollout status deployment/myapp -n @var.NAMESPACE --timeout=5m
}
─────────────────────────────────────────

Opal plan output:
─────────────────────────────────────────
Plan: deploy
Safety Level: High-Risk (production environment)

Steps:
  1. echo "Deploying myapp:1.2.3 to production"
  2. kubectl set image deployment/myapp app=myapp:1.2.3 -n production
  3. kubectl rollout status deployment/myapp -n production --timeout=5m

Capabilities Required:
  - network: kubernetes.local
  - commands: kubectl

Mutations:
  - Deployment image update (rollback: kubectl rollout undo)

Estimated Duration: 5 minutes
─────────────────────────────────────────

User reviews: "Version 1.2.3 is correct, production namespace is correct"
User executes: opal run --plan deploy.plan

Result: Success, deployment rolled out successfully
Audit: Logged with plan hash and user approval
```

### Example 2: Debugging with Read-Only Safety

```
User: "Debug why the API is returning 500 errors"

AI generates:
─────────────────────────────────────────
debug: {
    echo "Checking API pod logs..."
    kubectl logs -l app=api --tail=50
    
    echo "Checking recent errors..."
    kubectl logs -l app=api | grep ERROR | tail -20
    
    echo "Checking pod status..."
    kubectl get pods -l app=api -o wide
}
─────────────────────────────────────────

Opal plan output:
─────────────────────────────────────────
Plan: debug
Safety Level: Read-Only (safe to auto-execute)

Steps:
  1. kubectl logs -l app=api --tail=50
  2. kubectl logs -l app=api | grep ERROR | tail -20
  3. kubectl get pods -l app=api -o wide

Mutations: None (all read-only operations)
─────────────────────────────────────────

Opal detects read-only: Auto-executes with user notification
User sees: Output immediately, no manual approval needed
```

### Example 3: Multi-Phase Infrastructure Setup

```
User: "Set up a new staging environment"

AI generates 15-step Opal script across 4 phases

Opal breaks into phases:
─────────────────────────────────────────
Phase 1: Infrastructure (VPC, subnets, security groups)
  - 4 steps
  - Duration: ~10 minutes
  - Risk: Low (creation only)

Phase 2: Database (RDS instance, schema, users)
  - 3 steps
  - Duration: ~15 minutes
  - Risk: Medium (contains secrets)

Phase 3: Application (EKS cluster, deployments, ingress)
  - 6 steps
  - Duration: ~20 minutes
  - Risk: High (exposes to internet)

Phase 4: Validation (health checks, smoke tests)
  - 2 steps
  - Duration: ~5 minutes
  - Risk: None (read-only)
─────────────────────────────────────────

User executes:
  opal run --plan setup.plan --phase 1  # Review, execute phase 1
  opal run --plan setup.plan --phase 2  # Review, execute phase 2
  opal run --plan setup.plan --phase 3  # Review carefully, execute phase 3
  opal run --plan setup.plan --phase 4  # Auto-execute validation
```

### Example 4: CI/CD with Contract Verification

```yaml
# .github/workflows/ai-deployment.yml
name: AI-Assisted Deployment

on:
  push:
    branches: [main]

jobs:
  plan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: AI Generate Deployment Plan
        run: |
          ai-agent generate-deployment \
            --version ${{ github.sha }} \
            --environment production \
            --output deploy.opl
          
          opal plan deploy.opl --output deploy.plan
          
      - name: Upload Plan
        uses: actions/upload-artifact@v4
        with:
          name: deployment-plan
          path: deploy.plan

  review:
    needs: plan
    runs-on: ubuntu-latest
    environment: production  # Requires manual approval
    steps:
      - name: Download Plan
        uses: actions/download-artifact@v4
        with:
          name: deployment-plan
          
      - name: Display Plan
        run: opal plan --review deploy.plan
        
      - name: Execute Approved Plan
        run: opal run --plan deploy.plan
```

### Example 5: Audit and Compliance Report

```bash
# Generate compliance report for auditors
opal audit --since 2025-01-01 --format html > audit-report.html

Report shows:
─────────────────────────────────────────
AI-Assisted Operations - January 2025

Total Operations: 47
- Successful: 45
- Failed: 2
- Rolled Back: 1

By AI Agent:
- Claude Code: 32 operations
- Copilot: 12 operations
- Custom Agent: 3 operations

By Risk Level:
- High-Risk (production): 15 operations
  - All had manual approval
  - Average approval time: 3.2 minutes
- Medium-Risk: 18 operations
- Low-Risk/Read-Only: 14 operations

By User:
- alice@company.com: 28 operations
- bob@company.com: 19 operations

All operations have:
✓ Plan hash recorded
✓ User approval recorded
✓ Execution result recorded
✓ Rollback plan available
─────────────────────────────────────────
```

## AI Tools Perspective

This section describes what AI tool vendors need to implement to support Opal's plan-first execution model. The goal is to make integration straightforward while enabling powerful safety features.

### What AI Tools Need to Add

**1. Opal Output Mode**

AI tools need a configuration option to output Opal syntax instead of raw shell commands:

```python
# Example: Claude Code configuration
class OpalMode:
    """Generate Opal scripts instead of executing directly."""
    
    system_prompt = """
    You are an infrastructure assistant. When performing operations:
    
    1. Generate Opal syntax, not shell commands
    2. Use explicit variables for all user-provided values
    3. Group related operations into labeled blocks
    4. Include comments explaining the intent
    
    Opal syntax reference:
    - Variables: var NAME = "value"
    - Variable use: @var.NAME
    - Blocks: label: { ... }
    - Commands: Write shell commands directly
    - Decorators: @retry(attempts=3) { ... }
    
    Example:
    deploy: {
        var IMAGE = "myapp:v1.2.3"
        echo "Deploying @var.IMAGE"
        kubectl set image deployment/app app=@var.IMAGE
    }
    """
    
    def execute(self, user_intent):
        # Generate Opal script
        opal_script = self.generate_opal(user_intent)
        
        # Call opal plan
        plan_result = subprocess.run(
            ["opal", "plan", "--format", "json"],
            input=opal_script,
            capture_output=True,
            text=True
        )
        
        plan = json.loads(plan_result.stdout)
        
        # Check risk level
        risk_level = self.assess_risk(plan)
        
        # Route to appropriate approval workflow
        return self.route_for_approval(plan, risk_level)
```

**2. Risk Assessment**

AI tools should classify plans into risk levels for appropriate routing:

```python
def assess_risk(plan):
    """
    Classify plan into risk level based on operations.
    Returns: "safe" | "moderate" | "sensitive" | "dangerous"
    """
    has_mutations = any(step.get("mutates", False) for step in plan["steps"])
    has_deletions = any("delete" in step.get("command", "") for step in plan["steps"])
    touches_production = any("production" in str(step) for step in plan["steps"])
    touches_secrets = "vault" in str(plan.get("capabilities", {}))
    
    if has_deletions or (touches_production and has_mutations):
        return "dangerous"
    elif touches_secrets or touches_production:
        return "sensitive"
    elif has_mutations:
        return "moderate"
    else:
        return "safe"
```

### Approval Spectrum

The approval spectrum defines how different risk levels are handled:

| Risk Level | Examples | Approval Mode | User Experience |
|------------|----------|---------------|-----------------|
| **Safe** | `ls`, `cat`, `kubectl get`, read-only queries | Auto-approve | Execute immediately, notify after |
| **Moderate** | File writes, git operations, `kubectl apply` to dev | Notify + proceed | Show plan briefly, proceed unless interrupted |
| **Sensitive** | Network calls, secret access, staging deploys | Require review | Pause, show plan, wait for explicit approval |
| **Dangerous** | `rm -rf`, production deploys, DB mutations | Explicit approval + confirmation | Full review, confirmation dialog, possible 2FA |

**Configuration Example:**

```json
{
  "approvalSpectrum": {
    "safe": {
      "autoExecute": true,
      "notifyAfter": true,
      "timeout": 0
    },
    "moderate": {
      "autoExecute": true,
      "showPlan": true,
      "timeout": 10,
      "timeoutAction": "proceed"
    },
    "sensitive": {
      "autoExecute": false,
      "requireApproval": true,
      "timeout": 300,
      "timeoutAction": "cancel"
    },
    "dangerous": {
      "autoExecute": false,
      "requireApproval": true,
      "requireConfirmation": true,
      "timeout": null,
      "timeoutAction": "cancel"
    }
  }
}
```

### User-in-the-Loop Protocol

For long-running operations, AI tools can use a "parallel execution with review gates" model:

```
AI generates 15-step plan
    │
    ▼
Phase 1-3 (safe operations) ──→ Auto-execute in parallel
    │                           (file reads, queries)
    │
    ▼
Phase 4 (sensitive) ──→ Pause, notify user
    │                   (push notification: "Ready to create database")
    │
    ▼
User approves via mobile app
    │
    ▼
Phase 4 executes
    │
    ▼
Phase 5 (dangerous) ──→ Pause, require explicit confirmation
    │                   (production deployment)
    │
    ▼
User reviews on desktop, confirms
    │
    ▼
Phase 5 executes
```

**Implementation Example:**

```python
class ParallelExecutionWithGates:
    """Execute safe phases in parallel, pause at review gates."""
    
    def execute_plan(self, plan):
        phases = self.group_into_phases(plan)
        
        for phase in phases:
            risk = self.assess_phase_risk(phase)
            
            if risk in ["safe", "moderate"]:
                # Execute immediately
                self.execute_phase(phase)
                
            elif risk == "sensitive":
                # Notify user, short timeout
                approval = self.request_approval(
                    phase,
                    timeout=300,
                    urgency="normal",
                    channels=["desktop", "mobile"]
                )
                if approval.received:
                    self.execute_phase(phase)
                else:
                    self.pause_and_wait(phase)
                    
            elif risk == "dangerous":
                # Full review required
                approval = self.request_approval(
                    phase,
                    timeout=None,  # No timeout
                    urgency="high",
                    channels=["desktop"],
                    requireConfirmation=True
                )
                if approval.confirmed:
                    self.execute_phase(phase)
                else:
                    raise ExecutionAborted("User rejected dangerous operation")
```

### Push Notification Integration

AI tools can integrate with push notification systems for mobile approval:

```python
class PushNotificationApprover:
    """Send push notifications for plan approval."""
    
    def request_approval(self, plan, context):
        """Send push notification and wait for response."""
        
        # Generate plan summary
        summary = self.summarize_plan(plan)
        
        # Send push notification
        notification = {
            "title": f"Approval Required: {context['operation']}",
            "body": summary,
            "data": {
                "planId": plan["id"],
                "riskLevel": context["risk"],
                "actions": ["approve", "reject", "review"]
            }
        }
        
        # Send to user's devices
        self.push_service.send(
            user=context["user"],
            notification=notification
        )
        
        # Wait for response (with timeout)
        response = self.wait_for_response(
            plan_id=plan["id"],
            timeout=context.get("timeout", 300)
        )
        
        return response
    
    def summarize_plan(self, plan):
        """Create human-readable summary for mobile."""
        steps = len(plan["steps"])
        mutations = sum(1 for s in plan["steps"] if s.get("mutates"))
        
        if mutations == 0:
            return f"{steps} read-only operations"
        elif mutations <= 3:
            return f"{steps} operations ({mutations} changes)"
        else:
            return f"{steps} operations ({mutations} changes) - Review recommended"
```

### Mobile App Integration

Example mobile app flow for approving AI operations:

```
┌─────────────────────────────────────────┐
│  🔔 Opal Approval Request               │
│                                         │
│  Deploy v1.2.3 to production            │
│                                         │
│  5 operations (2 changes)               │
│  Estimated: 3 minutes                   │
│                                         │
│  [Review]    [Approve]    [Reject]      │
└─────────────────────────────────────────┘

Tapping [Review] shows:
┌─────────────────────────────────────────┐
│  Plan Details                           │
│                                         │
│  1. kubectl set image... ✏️             │
│  2. kubectl rollout status...           │
│  3. curl health check ✅                │
│                                         │
│  [Approve]              [Reject]        │
└─────────────────────────────────────────┘
```

### Capability Scoping

AI tools should scope what they can generate based on user configuration:

```json
{
  "aiCapabilities": {
    "allowed": {
      "commands": ["kubectl", "helm", "docker"],
      "verbs": ["get", "describe", "logs", "apply", "set"],
      "environments": ["development", "staging"],
      "resources": ["deployment/*", "service/*"]
    },
    "denied": {
      "commands": ["rm", "dd", "mkfs"],
      "verbs": ["delete"],
      "resources": ["namespace/production", "secret/*"]
    },
    "requireApproval": {
      "environments": ["production"],
      "verbs": ["apply", "set"]
    }
  }
}
```

**Implementation:**

```python
class CapabilityEnforcer:
    """Enforce capability restrictions on AI-generated plans."""
    
    def validate_plan(self, plan, capabilities):
        """Check if plan exceeds allowed capabilities."""
        violations = []
        
        for step in plan["steps"]:
            # Check command
            if step["command"] not in capabilities["allowed"]["commands"]:
                violations.append(f"Command '{step['command']}' not allowed")
            
            # Check environment
            if step.get("environment") in capabilities["denied"]["environments"]:
                violations.append(f"Environment '{step['environment']}' denied")
            
            # Check if requires approval
            if self.requires_approval(step, capabilities):
                step["requiresApproval"] = True
        
        if violations:
            raise CapabilityViolation(violations)
        
        return plan
```

### Example: Claude Code Integration

```typescript
// Claude Code Opal integration
interface OpalConfig {
  mode: 'plan-first' | 'execute-direct';
  autoApprove: ('safe' | 'moderate')[];
  notificationChannels: ('desktop' | 'mobile' | 'slack')[];
}

class OpalIntegration {
  async execute(userIntent: string): Promise<Result> {
    // Generate Opal script
    const opalScript = await this.claude.generate({
      prompt: userIntent,
      system: OPAL_SYSTEM_PROMPT,
      outputFormat: 'opal'
    });
    
    // Create plan
    const plan = await this.opal.plan(opalScript);
    
    // Assess risk
    const risk = this.assessRisk(plan);
    
    // Route appropriately
    switch (risk) {
      case 'safe':
        return this.executeImmediately(plan);
      case 'moderate':
        return this.executeWithNotification(plan);
      case 'sensitive':
        return this.requestApproval(plan, { timeout: 300 });
      case 'dangerous':
        return this.requestExplicitConfirmation(plan);
    }
  }
}
```

## Rationale

### Why Plan-First for AI Agents?

The plan-first model addresses the fundamental opacity problem of AI agents:

| Problem | Plan-First Solution |
|---------|---------------------|
| Can't see what AI will do | Plan shows all commands before execution |
| No chance to catch mistakes | Human review step catches errors |
| No record of what happened | Audit trail with plan hashes |
| Can't verify AI's work | Plan hash proves what was approved |
| No rollback capability | Automatic rollback generation |

### Why Opal Instead of Raw Shell?

Raw shell scripts from AI agents have problems:

```bash
# AI-generated shell script - hard to review
#!/bin/bash
kubectl set image deployment/app app=myapp:v1.2.3
kubectl rollout status deployment/app
# Is this safe? What namespace? What if it fails?
```

Opal provides structure:

```opal
# AI-generated Opal - structured and reviewable
deploy: {
    var IMAGE = "myapp:v1.2.3"
    var NAMESPACE = "production"
    
    kubectl set image deployment/app app=@var.IMAGE -n @var.NAMESPACE
    kubectl rollout status deployment/app -n @var.NAMESPACE --timeout=5m
}
```

Benefits:
- **Variables are explicit**: No hidden values
- **Structure is clear**: Block scoping shows intent
- **Safety is built-in**: Rollback, timeouts, retries
- **Deterministic**: Same script → same plan → same execution

### Why Not Just Trust the AI?

Even trustworthy AI makes mistakes:

- **Hallucination**: AI might reference non-existent resources
- **Context errors**: AI might use wrong namespace/environment
- **Drift**: AI trained on old documentation might use deprecated APIs
- **Prompt injection**: Malicious input could manipulate AI output

The plan-first model doesn't distrust the AI—it adds a verification layer that catches these errors before they affect production.

### Why Capability Restrictions?

Capability restrictions limit the blast radius of AI mistakes:

```json
{
  "capabilities": {
    "network": ["kubernetes.local"],  // AI can't reach external APIs
    "commands": ["kubectl", "helm"],  // AI can't run arbitrary shell
    "environments": ["staging", "dev"] // AI can't touch production
  }
}
```

Even if AI generates malicious instructions, it can't exceed approved capabilities.

### Why Deterministic Plans?

Determinism enables:

- **Verification**: Plan hash proves what was reviewed
- **Reproducibility**: Same plan → same execution
- **Caching**: Plans can be cached and reused
- **Comparison**: Diff plans to see what changed

### Why Mandatory Audit Logging?

Compliance and debugging require complete records:

- **SOC2**: "Who did what and when?"
- **Incident Response**: "What was the state before the change?"
- **Post-Mortems**: "What was the AI thinking?"
- **Compliance**: "Prove you have human oversight"

## Alternatives Considered

### Alternative 1: AI Executes Directly with Logging

**Approach**: AI executes shell commands directly, logs everything for audit.

**Rejected**:
- Too late to catch mistakes (damage already done)
- Logs don't show intent, only execution
- No review opportunity
- Reactive rather than proactive

**Tradeoff**: We accept the overhead of plan generation and review for the safety benefits.

### Alternative 2: Human Reviews Every AI Action

**Approach**: AI suggests action, human approves each step individually.

**Rejected**:
- Too slow for complex operations (15 approvals for 15 steps)
- Human fatigue leads to rubber-stamp approvals
- Breaks flow of AI-assisted development

**Tradeoff**: We batch operations into reviewable plans rather than individual approvals, accepting that batching requires careful scope definition.

### Alternative 3: Sandboxed AI Execution

**Approach**: Run AI-generated code in containers or sandboxes.

**Rejected**:
- Doesn't address the "what will it do?" problem
- Overhead of containerization
- Still need to review before execution
- Limited by sandbox capabilities

**Tradeoff**: We use capability restrictions within Opal rather than OS-level sandboxing, accepting less isolation but better visibility.

### Alternative 4: Pre-Approved Skill Library

**Approach**: Create library of pre-approved skills, AI only uses approved skills.

**Rejected**:
- Too rigid—can't handle novel situations
- Requires maintaining large skill library
- Doesn't leverage AI's flexibility

**Tradeoff**: We allow AI to generate arbitrary Opal scripts with review, accepting the risk of novel (untested) operations.

### Alternative 5: AI Explains in Natural Language

**Approach**: AI explains what it will do in English, user approves.

**Rejected**:
- Natural language is ambiguous
- "Update the deployment" vs "kubectl set image deployment/app"
- Hard to verify against actual execution
- Subject to interpretation

**Tradeoff**: We require structured Opal output rather than natural language, accepting that it's more verbose but unambiguous.

### Alternative 6: Static Analysis of AI Output

**Approach**: Use static analysis tools to catch dangerous patterns in AI output.

**Rejected**:
- Brittle—easy to bypass with creative code
- False positives block legitimate operations
- Doesn't show what will actually happen
- Reactive rather than proactive

**Tradeoff**: We use plan review rather than static analysis, accepting that humans must be in the loop for high-risk operations.

## Implementation

### Phase 1: AI Output Parser

- Create parser for AI-generated Opal scripts
- Handle common AI output patterns (markdown code blocks, comments)
- Validate Opal syntax
- Generate helpful error messages for AI-generated code
- **Deliverable**: AI → Opal conversion pipeline

### Phase 2: Plan Review Interface

- Human-readable plan output format
- Structured JSON plan format for programmatic review
- Diff capabilities between plans
- Integration with popular AI agent frameworks
- **Deliverable**: Plan review CLI and library

### Phase 3: Capability System

- Capability declaration in plans
- Capability approval workflow
- Policy engine for automatic approval/rejection
- Integration with Opal's existing Vault security model
- **Deliverable**: Capability-restricted execution

### Phase 4: Safety Features

- High-risk operation detection
- Rollback plan generation
- Step-by-step execution mode
- Dry-run and preview modes
- **Deliverable**: Comprehensive safety features

### Phase 5: Audit and Compliance

- Audit log generation
- Audit log querying and reporting
- Compliance report generation (SOC2, etc.)
- Integration with external SIEM tools
- **Deliverable**: Complete audit trail

### Phase 6: AI Agent Integrations

- Claude Code integration
- GitHub Copilot integration
- Cursor integration
- OpenAI Codex integration
- Custom agent SDK
- **Deliverable**: Broad AI agent ecosystem support

### Phase 7: Enterprise Features

- Multi-user approval workflows
- Policy-as-code for automatic approval
- Integration with enterprise identity systems
- Advanced audit analytics
- **Deliverable**: Enterprise-ready solution

## Compatibility

**Breaking changes:** None. This is a new layer on top of existing Opal functionality.

**Migration path:**

1. Existing Opal scripts work unchanged
2. AI agents can opt-in to plan-first output
3. Users can gradually adopt AI-generated plans
4. No changes required to existing workflows

## Open Questions

1. **Plan format**: Should plans be JSON, YAML, or a custom format? JSON is machine-friendly; YAML is human-friendly; custom could be optimized for diffing.

2. **AI detection**: How do we detect which AI agent generated a plan for audit purposes? Environment variables? Explicit declaration?

3. **Plan storage**: Where should plans be stored? Version control? Plan registry? Distributed with the Opal script?

4. **Approval UX**: What's the ideal user experience for plan approval? CLI interactive? Web UI? IDE integration?

5. **False positive rate**: How do we minimize false positives in high-risk detection without missing real risks?

6. **AI training**: Should we train AI models specifically on Opal syntax, or rely on few-shot prompting?

7. **Custom safety policies**: How do organizations define custom safety policies? Policy-as-code? Configuration files?

8. **Multi-step plan execution**: How do we handle partial execution (some phases succeed, later phases fail)?

9. **Plan versioning**: How do we version plans and track changes over time?

10. **Integration with existing tools**: How does this integrate with existing infrastructure tools (Terraform, Ansible, etc.)?

## References

- **Related OEPs:**
  - OEP-001: Runtime Variable Binding with `let` (variables in AI-generated plans)
  - OEP-004: Plan Verification (contract verification foundation)
  - OEP-012: Module Composition (reusable AI-generated modules)
  - OEP-013: Observability & Telemetry Hooks (audit logging)

- **External Inspiration:**
  - [Claude Code](https://claude.ai/code): AI agent with approval workflows
  - [GitHub Copilot](https://github.com/features/copilot): AI pair programming
  - [OpenAI Codex](https://openai.com/codex): AI software engineering agent
  - [Cursor](https://cursor.com): AI-first code editor
  - [OWASP LLM Top 10](https://genai.owasp.org/): LLM security vulnerabilities
  - [Agent Skills Specification](https://agentskills.io/specification): Industry-standard skill format

- **Security Research:**
  - "Prompt Injection Attacks on Agentic Coding Assistants" (arXiv, 2025)
  - "Mitigating the risk of prompt injections in browser use" (Anthropic, 2025)
  - "Understanding prompt injections" (OpenAI, 2025)
  - "The Alignment Problem" (Christian, 2020)

- **Academic References:**
  - "Capability-Based Security" (Dennis and Van Horn, 1966)
  - "Reflections on Trusting Trust" (Thompson, 1984)
  - "Human-Computer Interaction" (Card, Moran, and Newell, 1983) - review cycles
