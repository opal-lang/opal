---
title: "AST Design for Opal"
audience: "Tooling Developers & LSP Implementers"
summary: "Event-based parser implementation for execution and tooling"
---

# AST Design for Opal

**Goal**: Fast, resilient parser that supports LSP tooling and development tools.

**Audience**: Tooling developers, LSP implementers, and contributors working on the parser or static analysis.

**See also**:
- [GRAMMAR.md](GRAMMAR.md) - Formal EBNF syntax specification
- [ARCHITECTURE.md](ARCHITECTURE.md#dual-path-architecture-execution-vs-tooling) - Dual-path design rationale

**Important**: The AST is **optional** and only built for tooling (LSP, formatters, linters). For execution, Opal generates plans directly from parser events without constructing an AST.

## Dual-Path Pipeline

```
Source Code
    ↓
Lexer → Tokens
    ↓
Parser → Events
    ↓    ↓
    ↓    └──→ AST (tooling: LSP, linters, formatters)
    ↓
    └──────→ Plan (execution: runtime)
```

**Key insight**: Events are universal. AST construction is deferred until needed.

## Design Constraints

**See also**: 
- [GRAMMAR.md](GRAMMAR.md) - Formal EBNF syntax specification
- [ARCHITECTURE.md](ARCHITECTURE.md#dual-path-architecture-execution-vs-tooling) - Why we chose this design

### Performance Requirements
- **Lexer**: Zero allocations for hot paths
- **Parser**: Minimal allocations, event-based
- **AST**: Lightweight, cache-friendly representation

### Token Model
- **Comments preserved** as `COMMENT` tokens
- **Whitespace**: Discarded, but `HasSpaceBefore` flag preserved
- **Position**: Full Line/Column/Offset tracking
- **Text**: `[]byte` slices into source (zero-copy)

### Existing Lexer Design
```go
type Token struct {
    Type           TokenType
    Text           []byte      // Zero-copy slice into source
    Position       Position
    HasSpaceBefore bool        // Lossy but meaningful whitespace info
}

type Position struct {
    Line   int  // 1-based
    Column int  // 1-based
    Offset int  // 0-based byte offset
}
```

## AST Architecture

### Two-Path Design

Opal's parser produces events that can be consumed in two ways:

**Path 1: Events → Plan (Execution)**
- Direct event consumption by interpreter
- No AST construction
- Sub-millisecond plan generation
- Used by: CLI execution, runtime

**Path 2: Events → AST (Tooling)**
- Lazy AST construction from events
- Strongly typed node access
- Full semantic analysis
- Used by: LSP, formatters, linters, analysis tools

### Parse Tree (Event-Based)

The parser always produces a flat event stream (inspired by rust-analyzer):
- Resilient: errors don't stop parsing
- Fast: minimal allocations
- Universal: consumed by both execution and tooling paths

### Typed AST (Tooling Only)

When tooling needs semantic analysis, events are materialized into a typed AST:
- Lazy construction from parse tree
- Strongly typed accessors
- Parent/child relationships
- Symbol tables

### Parse Tree (Event-Based)

Inspired by rust-analyzer and matklad's resilient LL parsing tutorial.

```go
// Event represents a parse tree construction event
type EventKind uint8

const (
    EventOpen   EventKind = iota  // Open syntax node
    EventClose                     // Close syntax node
    EventToken                     // Consume token
    EventError                     // Error recovery marker
)

type Event struct {
    Kind EventKind
    Data uint32  // NodeKind for Open, token index for Token, error index for Error
}

type ParseTree struct {
    source  []byte        // Original source text
    tokens  []Token       // From lexer (already efficient)
    events  []Event       // Compact event stream
    errors  []ParseError  // Separate error list
}

type ParseError struct {
    Span    Span
    Message string
    Kind    ErrorKind
}

type ErrorKind uint8

const (
    ErrorUnexpectedToken ErrorKind = iota
    ErrorMissingToken
    ErrorInvalidSyntax
)

type Span struct {
    Start Position
    End   Position
}
```

**Why event-based?**
- Matches matklad's proven resilient approach
- Complements fast lexer (no extra allocations)
- LSP can build views lazily
- IR generation can skip unused branches
- Natural representation for incomplete code

**Example event stream:**
```
Input: "fun greet(name) { }"

Events:
  Open(NodeFile)
    Open(NodeFnDecl)
      Token(FUN)           // "fun"
      Token(IDENTIFIER)    // "greet"
      Open(NodeParamList)
        Token(LPAREN)      // "("
        Open(NodeParam)
          Token(IDENTIFIER) // "name"
        Close
        Token(RPAREN)      // ")"
      Close
      Open(NodeBlock)
        Token(LBRACE)      // "{"
        Token(RBRACE)      // "}"
      Close
    Close
  Close
```

### Error Recovery Strategy

Following matklad's resilient LL parsing principles:

**FIRST sets** - Check if token can start construct
**RECOVERY sets** - Ancestors' FOLLOW sets
**Never panic** - Always produce partial tree

```go
// Parser maintains FIRST and RECOVERY sets
const (
    EXPR_FIRST = [INTEGER, FLOAT, STRING, IDENTIFIER, LPAREN, AT]
    STMT_FIRST = [VAR, FOR, IF, WHEN, TRY, IDENTIFIER, AT]
    STMT_RECOVERY = [FUN, RBRACE, EOF]
    PARAM_RECOVERY = [ARROW, LBRACE, FUN]
)

// Example: parsing a block with error recovery
func (p *Parser) block() {
    p.open(NodeBlock)
    p.expect(LBRACE)
    
    for !p.at(RBRACE) && !p.eof() {
        if p.atAny(STMT_FIRST) {
            p.stmt()
        } else if p.atAny(STMT_RECOVERY) {
            break  // Let outer context handle
        } else {
            p.advanceWithError("expected statement")
        }
    }
    
    p.expect(RBRACE)
    p.close(NodeBlock)
}
```

**Recovery example:**
```
Input: "fun f(x: i32, fn g() {}"
                    ^^
                    error here

Parse tree:
  File
    Fn
      "fun" "f"
      ParamList
        "(" 
        Param("x" ":" "i32" ",")
        Error("fn")  // Unexpected token, but we recover
      // Break out of param list, continue with next function
    Fn
      "fun" "g"
      ParamList("(" ")")
      Block("{" "}")
```

### Node Types

```go
type NodeKind uint16

const (
    // Error recovery
    NodeError NodeKind = iota
    
    // Top level
    NodeFile
    NodeFnDecl
    
    // Parameters and types
    NodeParamList
    NodeParam
    NodeTypeExpr
    
    // Statements
    NodeBlock
    NodeLetStmt
    NodeForStmt
    NodeIfStmt
    NodeWhenStmt
    NodeWhenArm
    NodeTryStmt
    NodeCatchClause
    NodeFinallyClause
    NodeExprStmt
    
    // Expressions
    NodeBinaryExpr
    NodeCallExpr
    NodeNameExpr
    NodeLiteralExpr
    NodeParenExpr
    
    // Decorators (unified - both value & execution)
    NodeDecorator
    NodeDecoratorPath   // @var.NAME or @aws.secret.key
    NodeArgList
    NodeArg
)
```

### Typed API (Zero-Cost Wrappers)

Lazy accessors over the parse tree - no allocation until needed.

```go
// NodeRef is an index into the event stream
type NodeRef uint32

// File is a typed view over the parse tree
type File struct {
    tree *ParseTree
    node NodeRef
}

func (f File) Functions() []Function {
    // Walk events, collect NodeFnDecl
    var fns []Function
    // ... implementation
    return fns
}

type Function struct {
    tree *ParseTree
    node NodeRef
}

func (fn Function) Name() Option[Token] {
    // Walk events to find IDENTIFIER after FUN
    // Return None if missing (error recovery case)
}

func (fn Function) Params() Option[ParamList] {
    // Find NodeParamList child
}

func (fn Function) Body() Option[Block] {
    // Find NodeBlock child
}

func (fn Function) Span() Span {
    // Compute from first/last token in subtree
}

type Block struct {
    tree *ParseTree
    node NodeRef
}

func (b Block) Statements() []Stmt {
    // Collect all statement nodes
}

// Stmt is a discriminated union
type Stmt struct {
    tree *ParseTree
    node NodeRef
    kind NodeKind  // NodeLetStmt, NodeForStmt, etc.
}

func (s Stmt) AsLet() Option[LetStmt] {
    if s.kind == NodeLetStmt {
        return Some(LetStmt{s.tree, s.node})
    }
    return None
}

// Similar for expressions, decorators, etc.
```

### LSP Integration Points

```go
// Position-based query (for hover, go-to-def)
func (t *ParseTree) NodeAt(offset int) Option[NodeRef] {
    // Binary search events by token positions
    // Return innermost node containing offset
}

// Symbol extraction (for document symbols, outline)
func (f File) Symbols() []Symbol {
    for fn := range f.Functions() {
        if name := fn.Name(); name.IsSome() {
            symbols = append(symbols, Symbol{
                Name: name.Unwrap().String(),
                Kind: SymbolFunction,
                Span: fn.Span(),
            })
        }
    }
    return symbols
}

// Incremental re-parse (future optimization)
func (t *ParseTree) Edit(edit TextEdit) *ParseTree {
    // Re-lex + re-parse affected region
    // For MVP: full re-parse is fast enough
    // Future: Tree-sitter-style incremental parsing
}

// Comment extraction (for hover documentation)
func (t *ParseTree) CommentsFor(node NodeRef) []Token {
    // Find COMMENT tokens near node
    // Use for hover hints, documentation
}
```

## Parser Implementation

### Parser Structure

```go
type Parser struct {
    tokens []Token
    pos    int
    events []Event
    errors []ParseError
    fuel   int  // Prevent infinite loops
}

func (p *Parser) open(kind NodeKind) MarkOpened {
    mark := MarkOpened{index: len(p.events)}
    p.events = append(p.events, Event{
        Kind: EventOpen,
        Data: uint32(NodeError),  // Placeholder
    })
    return mark
}

func (p *Parser) close(m MarkOpened, kind NodeKind) MarkClosed {
    p.events[m.index].Data = uint32(kind)
    p.events = append(p.events, Event{Kind: EventClose})
    return MarkClosed{index: m.index}
}

func (p *Parser) advance() {
    if !p.eof() {
        p.events = append(p.events, Event{
            Kind: EventToken,
            Data: uint32(p.pos),
        })
        p.pos++
        p.fuel = 256  // Reset fuel on progress
    }
}

func (p *Parser) at(kind TokenType) bool {
    return !p.eof() && p.tokens[p.pos].Type == kind
}

func (p *Parser) atAny(kinds []TokenType) bool {
    if p.eof() {
        return false
    }
    current := p.tokens[p.pos].Type
    for _, k := range kinds {
        if current == k {
            return true
        }
    }
    return false
}

func (p *Parser) expect(kind TokenType) {
    if p.at(kind) {
        p.advance()
        return
    }
    p.error(fmt.Sprintf("expected %s", kind))
}

func (p *Parser) advanceWithError(msg string) {
    m := p.open(NodeError)
    p.error(msg)
    if !p.eof() {
        p.advance()
    }
    p.close(m, NodeError)
}

func (p *Parser) error(msg string) {
    if p.eof() {
        return
    }
    tok := p.tokens[p.pos]
    p.errors = append(p.errors, ParseError{
        Span: Span{Start: tok.Position, End: tok.Position},
        Message: msg,
        Kind: ErrorUnexpectedToken,
    })
}
```

### Grammar Implementation

```go
// File = FnDecl*
func (p *Parser) file() {
    m := p.open(NodeFile)
    
    for !p.eof() {
        if p.at(FUN) {
            p.fnDecl()
        } else {
            p.advanceWithError("expected function declaration")
        }
    }
    
    p.close(m, NodeFile)
}

// FnDecl = 'fun' IDENTIFIER ParamList Block
func (p *Parser) fnDecl() {
    m := p.open(NodeFnDecl)
    
    p.expect(FUN)
    p.expect(IDENTIFIER)
    
    if p.at(LPAREN) {
        p.paramList()
    }
    
    if p.at(LBRACE) {
        p.block()
    }
    
    p.close(m, NodeFnDecl)
}

// ParamList = '(' (Param (',' Param)*)? ')'
func (p *Parser) paramList() {
    m := p.open(NodeParamList)
    
    p.expect(LPAREN)
    
    for !p.at(RPAREN) && !p.eof() {
        if p.at(IDENTIFIER) {
            p.param()
        } else if p.atAny(PARAM_RECOVERY) {
            break
        } else {
            p.advanceWithError("expected parameter")
        }
    }
    
    p.expect(RPAREN)
    p.close(m, NodeParamList)
}

// Param = IDENTIFIER (':' TypeExpr)? ','?
func (p *Parser) param() {
    m := p.open(NodeParam)
    
    p.expect(IDENTIFIER)
    
    if p.at(COLON) {
        p.advance()
        p.typeExpr()
    }
    
    if !p.at(RPAREN) {
        p.expect(COMMA)
    }
    
    p.close(m, NodeParam)
}

// Block = '{' Stmt* '}'
func (p *Parser) block() {
    m := p.open(NodeBlock)
    
    p.expect(LBRACE)
    
    for !p.at(RBRACE) && !p.eof() {
        if p.atAny(STMT_FIRST) {
            p.stmt()
        } else if p.atAny(STMT_RECOVERY) {
            break
        } else {
            p.advanceWithError("expected statement")
        }
    }
    
    p.expect(RBRACE)
    p.close(m, NodeBlock)
}

// Expression parsing uses Pratt parser for operator precedence
func (p *Parser) expr() {
    p.exprBp(0)
}

func (p *Parser) exprBp(minBp int) {
    // Pratt parser implementation
    // See matklad's "Simple but Powerful Pratt Parsing"
}
```

## Event-Based Plan Generation

The interpreter consumes parser events directly to generate execution plans without building an AST. This zero-copy pipeline enables fast plan generation with natural branch pruning.

### Direct Event Consumption

```go
func (i *Interpreter) generatePlan(events []Event) *ExecutionPlan {
    plan := &ExecutionPlan{Steps: make([]Step, 0, 100)}
    
    for i.pos < len(events) {
        event := events[i.pos]
        
        switch event.Kind {
        case EventOpen:
            switch event.Data {
            case NodeWhen:
                // Resolve condition at plan-time
                value := i.resolveValue("@var.ENV")
                
                // Process only matching branch
                for branch := range i.scanBranches() {
                    if branch.matches(value) {
                        i.processBranch(branch)  // Emit steps
                    } else {
                        i.skipBranch(branch)     // Prune events
                    }
                }
            
            case NodeFor:
                // Unroll loop at plan-time
                items := i.resolveValue("@var.SERVICES")
                loopBody := i.captureLoopBody()
                
                for _, item := range items {
                    i.setVar("service", item)
                    i.processEvents(loopBody)  // Replay with new binding
                }
            }
        }
    }
    
    return plan
}
```

### Branch Pruning

When conditionals are evaluated, only the selected branch is processed:

```opal
when @var.ENV {
    "production" -> kubectl apply -f k8s/prod/
    "staging" -> kubectl apply -f k8s/staging/
}
```

**Events (all branches present):**
```
[Open(When), Token(@var.ENV),
  Open(Branch), Token("production"), ..., Close(Branch),
  Open(Branch), Token("staging"), ..., Close(Branch),
Close(When)]
```

**Plan with ENV="production" (pruned):**
```
Step 1: kubectl apply -f k8s/prod/
// Staging branch pruned - events skipped
```

**Pruning implementation:**
```go
func (i *Interpreter) skipBranch(branch BranchInfo) {
    depth := 1
    for depth > 0 {
        i.pos++
        if events[i.pos].Kind == EventOpen { depth++ }
        if events[i.pos].Kind == EventClose { depth-- }
    }
    // Entire branch skipped - zero cost
}
```

### Loop Unrolling

Loops expand into concrete steps by replaying body events:

```opal
for service in ["api", "worker"] {
    kubectl scale deployment/@var.service --replicas=3
}
```

**Plan (unrolled):**
```
Step 1: kubectl scale deployment/api --replicas=3
Step 2: kubectl scale deployment/worker --replicas=3
```

The interpreter captures loop body events once, then replays them for each iteration with updated variable bindings.

### Parallel Resolution

Independent value decorators can resolve concurrently:

```opal
deploy: {
    @env.DATABASE_URL        # Resolve in parallel
    @aws.secret.api_key      # Resolve in parallel
    kubectl apply -f k8s/
}
```

All value decorators resolve before generating sequential execution steps.

### Performance Characteristics

**Pipeline efficiency:**
- Lexing: Zero-allocation tokenization
- Parsing: Lightweight event generation  
- Planning: Direct event consumption with pruning

**Memory profile:**
- Events: Compact (EventKind + Data)
- Plan: Minimal per-step overhead
- No AST: Significant memory savings

**Targets:**
- Lexer: >5000 lines/ms
- Parser: >3000 lines/ms (events)
- Plan generation: <10ms for typical scripts

## Implementation Plan

### Phase 1: Parser Foundation (Week 1)
- [ ] Event-based parse tree structure (`core/parser/tree.go`)
- [ ] Parser core with open/close/advance (`core/parser/parser.go`)
- [ ] Error recovery infrastructure
- [ ] Basic file/function parsing
- [ ] Golden tests for resilience

### Phase 2: Complete Grammar (Week 2)
- [ ] All statement types (let, for, if, when, try)
- [ ] Expression parsing (Pratt for precedence)
- [ ] Decorator syntax (@var.NAME, @retry(3) { })
- [ ] Comprehensive parser tests
- [ ] Error recovery tests

### Phase 3: Typed API (Week 3)
- [ ] Zero-cost wrapper types (`core/ast/`)
- [ ] Visitor pattern for tree traversal
- [ ] Position queries for LSP
- [ ] Symbol table extraction
- [ ] Comment association

### Phase 4: LSP Foundation (Week 4)
- [ ] NodeAt(offset) position query helper
- [ ] Basic symbol extraction (names + spans)
- [ ] Tree traversal utilities
- [ ] Integration tests with lexer

**Note**: Full LSP server, IR generation, and execution are separate future work sessions.

## Performance Validation

```go
func BenchmarkParser(b *testing.B) {
    input := generate10KLineFile()
    
    b.Run("lex+parse", func(b *testing.B) {
        b.ReportAllocs()
        
        for i := 0; i < b.N; i++ {
            lexer := NewLexer(input)
            tokens := lexer.GetTokens()  // ~2ms for 10K lines
            
            parser := NewParser(tokens)
            tree := parser.Parse()
            
            _ = tree
        }
    })
    
    b.ReportMetric(float64(len(input))/1000000, "MB/s")
}

// Fast parsing for large files
```

## Testing Strategy

### Golden Tests
```
tests/parser/golden/
  simple.opl          → simple.events
  errors.opl          → errors.events  (with error markers)
  complex.opl         → complex.events
  decorators.opl      → decorators.events
  control_flow.opl    → control_flow.events
```

Event format for golden tests:
```
Open(File)
  Open(FnDecl)
    Token(FUN, "fun")
    Token(IDENTIFIER, "greet")
    Open(ParamList)
      Token(LPAREN, "(")
      Open(Param)
        Token(IDENTIFIER, "name")
      Close
      Token(RPAREN, ")")
    Close
    Open(Block)
      Token(LBRACE, "{")
      Token(RBRACE, "}")
    Close
  Close
Close
```

### Resilience Tests

Test that parser produces useful partial trees:

```go
func TestParserResilience(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        wantNode NodeKind
        wantErr  bool
    }{
        {
            name: "incomplete function",
            input: "fun greet(name",
            wantNode: NodeFnDecl,  // Should still recognize as function
            wantErr: true,
        },
        {
            name: "missing brace",
            input: "fun f() { var x = 1",
            wantNode: NodeFnDecl,
            wantErr: true,
        },
        {
            name: "invalid expression",
            input: "fun f() { var x = + }",
            wantNode: NodeFnDecl,
            wantErr: true,
        },
        {
            name: "multiple functions with error",
            input: "fun f1(x: i32, fn f2() {}",
            wantNode: NodeFile,  // Should parse both functions
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tree := parseString(tt.input)
            
            // Should produce tree even with errors
            assert.NotNil(t, tree)
            
            // Should contain expected node type
            assert.True(t, containsNode(tree, tt.wantNode))
            
            // Should have errors if expected
            if tt.wantErr {
                assert.NotEmpty(t, tree.errors)
            }
        })
    }
}
```

### Property Tests

Invariants that must hold for all parse trees:

```go
func TestParseTreeInvariants(t *testing.T) {
    inputs := loadTestInputs()
    
    for _, input := range inputs {
        tree := parseString(input)
        
        // Every parse produces a tree
        assert.NotNil(t, tree)
        
        // Events are balanced (every Open has Close)
        assert.True(t, eventsBalanced(tree.events))
        
        // All tokens consumed
        assert.Equal(t, countTokenEvents(tree), len(tree.tokens))
        
        // Spans cover entire input (no gaps)
        assert.True(t, spansComplete(tree, input))
        
        // Position monotonicity (positions only increase)
        assert.True(t, positionsMonotonic(tree))
        
        // Comments preserved as tokens
        commentCount := countComments(tree.tokens)
        assert.Equal(t, commentCount, countCommentEvents(tree))
    }
}
```

## Key Design Decisions

### 1. Event-Based vs Node-Based
**Choice**: Event-based parse tree  
**Rationale**:
- Proven by rust-analyzer at scale
- Natural error recovery (partial trees)
- Complements zero-alloc lexer
- LSP can build views lazily
- Smaller memory footprint

### 2. Comment Handling
**Choice**: Keep as tokens, filter in typed API  
**Rationale**:
- Already tokenized (COMMENT type)
- Formatter needs them
- LSP hover can use them
- Zero cost to ignore in IR generation

### 3. Whitespace
**Choice**: `HasSpaceBefore` flag only  
**Rationale**:
- Already implemented in lexer
- Sufficient for formatter
- Keeps tokens small (4 bytes saved per token)
- Speed > full fidelity

### 4. Incremental Parsing
**Choice**: Not MVP, design for future  
**Rationale**:
- Full re-parse is fast enough for typical files
- LSP can cache parse trees per file
- Add later when profiling shows need
- Event-based design supports incremental updates

### 5. IR Separation
**Choice**: ParseTree → IR transformation  
**Rationale**:
- Different consumers need different representations
- LSP needs positions, IR needs semantics
- Clean separation of concerns
- IR can be optimized independently

### 6. Error Recovery Strategy
**Choice**: FIRST/FOLLOW sets + recovery sets  
**Rationale**:
- Proven approach (matklad's tutorial)
- Predictable behavior
- Easy to debug
- Works well with LL parsing

## References

- [Resilient LL Parsing Tutorial](https://matklad.github.io/2023/05/21/resilient-ll-parsing-tutorial.html) - matklad
- [Simple but Powerful Pratt Parsing](https://matklad.github.io/2020/04/13/simple-but-powerful-pratt-parsing.html) - matklad
- [Rust-Analyzer Syntax Trees](https://github.com/rust-lang/rust-analyzer/tree/master/crates/syntax)
- [Gopls Scalability](https://go.dev/blog/gopls-scalability) - Separate compilation approach

## Next Steps

1. Create `WORK.md` for tracking implementation
2. Implement Phase 1: Parser foundation
3. Write golden tests for basic cases
4. Iterate on error recovery
5. Benchmark and optimize hot paths