package parser

import (
	"bytes"
	"testing"

	"github.com/aledsdavies/opal/runtime/lexer"
)

// Fuzz tests for parser determinism and robustness.
//
// Eight specialized fuzz functions test different invariants:
//
// 1. FuzzParserDeterminism - Same input always produces identical output
// 2. FuzzParserNoPanic - Parser never panics on any input
// 3. FuzzParserEventBalance - Stack-based validation of open/close pairs
// 4. FuzzParserMemorySafety - Position bounds checking, monotonicity
// 5. FuzzParserPathologicalDepth - Handles deeply nested structures
// 6. FuzzParserErrorRecovery - Resilient parsing with error reporting
// 7. FuzzParserWhitespaceInvariance - Semantic equivalence across whitespace
// 8. FuzzParserSmokeTest - Catch-all for pathological inputs and edge cases
//
// These tests protect the events-first plan generation model.

// addSeedCorpus adds common test cases to all fuzz functions
func addSeedCorpus(f *testing.F) {
	// Basic syntax
	f.Add([]byte(""))
	f.Add([]byte("fun greet() {}"))
	f.Add([]byte("var x = 42"))
	f.Add([]byte("fun deploy(env) { kubectl apply }"))

	// Shell commands with dash arguments (test token emission)
	f.Add([]byte("wc -l file.txt"))
	f.Add([]byte("ls -la /tmp"))
	f.Add([]byte("kubectl apply -f deployment.yaml"))
	f.Add([]byte("curl -X POST --data @file.json https://api.example.com"))
	f.Add([]byte("grep -v pattern file.txt"))
	f.Add([]byte("tar -czf archive.tar.gz dir/"))
	f.Add([]byte("docker run --rm -it ubuntu bash"))

	// Redirect operators - valid
	f.Add([]byte(`echo "hello" > output.txt`))
	f.Add([]byte(`echo "world" >> output.txt`))
	f.Add([]byte(`cat file.txt > backup.txt`))
	f.Add([]byte(`ls -la > listing.txt`))
	f.Add([]byte(`echo "data" > @var.OUTPUT_FILE`))
	f.Add([]byte(`kubectl logs pod >> logs.txt`))
	f.Add([]byte(`echo "test" > /dev/null`))

	// Redirect with pipes
	f.Add([]byte(`cat data.txt | grep "error" > errors.txt`))
	f.Add([]byte(`echo "a" | wc -l >> count.txt`))

	// Redirect with AND/OR
	f.Add([]byte(`echo "a" > log.txt && echo "b"`))
	f.Add([]byte(`echo "a" >> log.txt || echo "failed"`))
	f.Add([]byte(`echo "a" > log.txt && echo "b" >> log.txt`))

	// Redirect with semicolon
	f.Add([]byte(`echo "a" > file1.txt ; echo "b" > file2.txt`))

	// Complex precedence with redirect
	f.Add([]byte(`echo "a" | grep "a" > file.txt && echo "b"`))
	f.Add([]byte(`cmd1 | cmd2 > out.txt || echo "failed"`))

	// Redirect - malformed (error recovery)
	f.Add([]byte(`echo "hello" >`))        // Missing target
	f.Add([]byte(`echo "hello" >>`))       // Missing target
	f.Add([]byte(`> output.txt`))          // Missing command
	f.Add([]byte(`>> output.txt`))         // Missing command
	f.Add([]byte(`echo "a" > > file.txt`)) // Double redirect operator

	// Control flow - valid if statements
	f.Add([]byte("fun test { if true { } }"))
	f.Add([]byte("fun test { if false { echo \"a\" } }"))
	f.Add([]byte("fun test { if x { } else { } }"))
	f.Add([]byte("fun test { if true { } else if false { } }"))
	f.Add([]byte("fun test { if @var.x { } }"))
	f.Add([]byte("if true { }"))                  // Top-level if (script mode)
	f.Add([]byte("if x { } else { echo \"a\" }")) // Top-level if-else

	// For loops - valid
	f.Add([]byte("fun test { for item in items { } }"))
	f.Add([]byte("fun test { for x in @var.list { echo @var.x } }"))
	f.Add([]byte("for item in items { }")) // Top-level for (script mode)

	// For loops - range expressions
	f.Add([]byte("fun test { for i in 1...10 { echo @var.i } }"))                      // Integer range
	f.Add([]byte("fun test { for i in @var.start...10 { } }"))                         // Decorator start
	f.Add([]byte("fun test { for i in 1...@var.end { } }"))                            // Decorator end
	f.Add([]byte("fun test { for i in @var.start...@var.end { } }"))                   // Both decorators
	f.Add([]byte("for port in 8000...8010 { echo \"Starting on @var.port\" }"))        // Top-level range
	f.Add([]byte("fun test { for i in 1...100 { kubectl scale --replicas=@var.i } }")) // Practical use case

	// Control flow - malformed (error recovery)
	f.Add([]byte("fun test { if }"))
	f.Add([]byte("fun test { if { } }"))
	f.Add([]byte("fun test { if true }"))
	f.Add([]byte("fun test { if true { } else }"))
	f.Add([]byte("fun test { else { } }"))
	f.Add([]byte("fun test { if \"str\" { } }"))
	f.Add([]byte("fun test { if 42 { } }"))
	f.Add([]byte("fun test { if true { fun helper() { } } }"))      // fun inside if
	f.Add([]byte("fun test { for }"))                               // Incomplete for
	f.Add([]byte("fun test { for item }"))                          // Missing in
	f.Add([]byte("fun test { for item in }"))                       // Missing collection
	f.Add([]byte("fun test { for item in items }"))                 // Missing block
	f.Add([]byte("fun test { for item in items { fun h() { } } }")) // fun inside for
	f.Add([]byte("fun test { for i in 1... { } }"))                 // Incomplete range (missing end)
	f.Add([]byte("fun test { for i in ...10 { } }"))                // Incomplete range (missing start)
	f.Add([]byte("fun test { for i in 1..10 { } }"))                // Wrong range operator (two dots)

	// Try/catch/finally - valid
	f.Add([]byte("fun test { try { echo \"a\" } catch { echo \"b\" } }"))
	f.Add([]byte("fun test { try { echo \"a\" } finally { echo \"c\" } }"))
	f.Add([]byte("fun test { try { echo \"a\" } catch { echo \"b\" } finally { echo \"c\" } }"))
	f.Add([]byte("fun test { try { echo \"a\" } }"))                  // try only (catch/finally optional)
	f.Add([]byte("try { kubectl apply } catch { kubectl rollback }")) // Top-level try

	// Try/catch/finally - malformed (error recovery)
	f.Add([]byte("fun test { try }"))                             // Missing try block
	f.Add([]byte("fun test { try catch { } }"))                   // Missing try block
	f.Add([]byte("fun test { try { } catch }"))                   // Missing catch block
	f.Add([]byte("fun test { try { } finally }"))                 // Missing finally block
	f.Add([]byte("fun test { try { fun h() { } } }"))             // fun inside try
	f.Add([]byte("fun test { try { } catch { fun h() { } } }"))   // fun inside catch
	f.Add([]byte("fun test { try { } finally { fun h() { } } }")) // fun inside finally
	f.Add([]byte("fun test { catch { } }"))                       // orphan catch
	f.Add([]byte("fun test { finally { } }"))                     // orphan finally
	f.Add([]byte("catch { }"))                                    // orphan catch at top level
	f.Add([]byte("finally { }"))                                  // orphan finally at top level

	// When pattern matching - valid
	f.Add([]byte(`fun test { when @var.ENV { "prod" -> echo "p" else -> echo "x" } }`))
	f.Add([]byte(`when @var.ENV { "production" -> kubectl apply else -> echo "skip" }`))                                    // Top-level when
	f.Add([]byte(`fun test { when @var.ENV { "prod" -> { kubectl apply echo "done" } } }`))                                 // Block body
	f.Add([]byte("fun test { when @var.ENV {\n\"prod\" -> echo \"p\"\n\"staging\" -> echo \"s\"\nelse -> echo \"x\"\n} }")) // Multiple arms

	// When pattern matching - regex patterns (Phase 2a)
	f.Add([]byte(`fun test { when @var.branch { r"^main$" -> echo "main" else -> echo "other" } }`))                             // Simple regex
	f.Add([]byte(`fun test { when @var.branch { r"^release/" -> echo "rel" r"^hotfix/" -> echo "fix" else -> echo "x" } }`))     // Multiple regex
	f.Add([]byte("fun test { when @var.branch {\nr\"^main$\" -> echo \"m\"\nr\"^dev-\" -> echo \"d\"\nelse -> echo \"x\"\n} }")) // Regex with newlines
	f.Add([]byte(`when @var.branch { r"^v[0-9]+\.[0-9]+\.[0-9]+$" -> echo "version tag" else -> echo "not a version" }`))        // Complex regex
	f.Add([]byte(`fun test { when @var.env { "prod" -> echo "p" r"^staging-" -> echo "s" else -> echo "x" } }`))                 // Mixed string and regex
	f.Add([]byte(`fun test { when @var.branch { r"^release/" -> { kubectl apply echo "deployed" } else -> echo "skip" } }`))     // Regex with block

	// When pattern matching - numeric range patterns (Phase 2b)
	f.Add([]byte(`fun test { when @var.status { 200...299 -> echo "success" else -> echo "error" } }`))                             // Simple range
	f.Add([]byte(`fun test { when @var.status { 200...299 -> echo "ok" 400...499 -> echo "client" 500...599 -> echo "server" } }`)) // Multiple ranges
	f.Add([]byte("fun test { when @var.status {\n200...299 -> echo \"ok\"\n400...499 -> echo \"err\"\nelse -> echo \"x\"\n} }"))    // Ranges with newlines
	f.Add([]byte(`when @var.port { 1...1024 -> echo "privileged" 1025...65535 -> echo "user" else -> echo "invalid" }`))            // Port ranges
	f.Add([]byte(`fun test { when @var.code { "success" -> echo "s" 200...299 -> echo "ok" else -> echo "x" } }`))                  // Mixed string and range
	f.Add([]byte(`fun test { when @var.status { 200...299 -> { kubectl apply echo "deployed" } else -> echo "skip" } }`))           // Range with block

	// When pattern matching - OR patterns (Phase 2c)
	f.Add([]byte(`fun test { when @var.env { "prod" | "production" -> echo "p" else -> echo "x" } }`))                                               // Simple OR
	f.Add([]byte(`fun test { when @var.env { "dev" | "development" | "local" -> echo "d" else -> echo "x" } }`))                                     // Multiple OR
	f.Add([]byte(`fun test { when @var.env { "prod" | r"^staging-" -> echo "deploy" else -> echo "skip" } }`))                                       // Mixed string and regex
	f.Add([]byte(`fun test { when @var.code { 200...299 | 300...399 -> echo "ok" else -> echo "err" } }`))                                           // OR with ranges
	f.Add([]byte(`when @var.branch { "main" | "master" | r"^release/" -> kubectl apply else -> echo "skip" }`))                                      // Top-level OR
	f.Add([]byte("fun test { when @var.env {\n\"prod\" | \"production\" -> echo \"p\"\n\"dev\" | \"local\" -> echo \"d\"\nelse -> echo \"x\"\n} }")) // OR with newlines

	// Unary expressions - valid
	f.Add([]byte("fun test { var x = -5 }"))                           // Negation with literal
	f.Add([]byte("fun test { var x = -@var.offset }"))                 // Negation with decorator
	f.Add([]byte("fun test { var result = -x + y }"))                  // Unary minus with binary addition
	f.Add([]byte("fun test { var result = -x * -y }"))                 // Multiple unary minus
	f.Add([]byte("fun test { if !true { echo \"no\" } }"))             // Logical NOT with literal
	f.Add([]byte("fun test { if !@var.ready { echo \"wait\" } }"))     // Logical NOT with decorator
	f.Add([]byte("fun test { if !ready && enabled { echo \"go\" } }")) // NOT with AND
	f.Add([]byte("fun test { var x = -1 + -2 }"))                      // Unary in binary expression
	f.Add([]byte("var offset = -10"))                                  // Top-level var with unary

	// Increment/decrement expressions - valid
	f.Add([]byte("fun test { var x = ++counter }"))                  // Prefix increment with identifier
	f.Add([]byte("fun test { var x = --counter }"))                  // Prefix decrement with identifier
	f.Add([]byte("fun test { var x = ++@var.count }"))               // Prefix increment with decorator
	f.Add([]byte("fun test { var x = --@var.count }"))               // Prefix decrement with decorator
	f.Add([]byte("fun test { var x = counter++ }"))                  // Postfix increment with identifier
	f.Add([]byte("fun test { var x = counter-- }"))                  // Postfix decrement with identifier
	f.Add([]byte("fun test { var x = @var.count++ }"))               // Postfix increment with decorator
	f.Add([]byte("fun test { var x = @var.count-- }"))               // Postfix decrement with decorator
	f.Add([]byte("fun test { var result = x++ + y }"))               // Postfix before addition
	f.Add([]byte("fun test { var result = ++x + y }"))               // Prefix before addition
	f.Add([]byte("fun test { var result = x++ * ++y }"))             // Mixed postfix and prefix
	f.Add([]byte("fun test { for i in 1...10 { var next = i++ } }")) // Increment in loop
	f.Add([]byte("var counter = 0"))                                 // Top-level var (for increment context)

	// Assignment operators - valid
	f.Add([]byte("fun test { total += 5 }"))                                  // Plus assign with literal
	f.Add([]byte("fun test { remaining -= @var.cost }"))                      // Minus assign with decorator
	f.Add([]byte("fun test { replicas *= 3 }"))                               // Multiply assign
	f.Add([]byte("fun test { batch_size /= 2 }"))                             // Divide assign
	f.Add([]byte("fun test { index %= 10 }"))                                 // Modulo assign
	f.Add([]byte("fun test { total += x + y }"))                              // Assignment with expression
	f.Add([]byte("fun test { for i in 1...10 { sum += i } }"))                // Assignment in loop (accumulation)
	f.Add([]byte("fun test { count += 1 count -= 1 }"))                       // Multiple assignments
	f.Add([]byte("fun test { var total = 0 for x in items { total += x } }")) // Accumulation pattern
	f.Add([]byte("fun test { replicas *= @var.ENVIRONMENTS.length }"))        // Scaling with decorator

	// When pattern matching - malformed (error recovery)
	f.Add([]byte("fun test { when { } }"))                                    // Missing expression
	f.Add([]byte("fun test { when @var.ENV }"))                               // Missing opening brace
	f.Add([]byte(`fun test { when @var.ENV { "prod" echo "x" } }`))           // Missing arrow
	f.Add([]byte(`fun test { when @var.ENV { "prod" -> echo "x" `))           // Missing closing brace
	f.Add([]byte(`fun test { when @var.ENV { "prod" -> fun h() { } } }`))     // fun inside when arm
	f.Add([]byte(`fun test { when @var.ENV { "prod" -> { fun h() { } } } }`)) // fun inside when block
	f.Add([]byte(`fun test { when @var.branch { r -> echo "x" } }`))          // Incomplete regex (missing string)
	f.Add([]byte(`fun test { when @var.branch { r"unclosed -> echo "x" } }`)) // Unclosed regex string
	f.Add([]byte(`fun test { when @var.status { 200... -> echo "x" } }`))     // Incomplete range (missing end)
	f.Add([]byte(`fun test { when @var.status { 200..299 -> echo "x" } }`))   // Wrong range operator (two dots)

	// Nested control flow combinations (metaprogramming nesting)
	f.Add([]byte("fun test { for x in items { if @var.x { echo \"found\" } } }"))                                           // for → if
	f.Add([]byte("fun test { for i in 1...3 { for j in 1...3 { echo \"@var.i,@var.j\" } } }"))                              // for → for
	f.Add([]byte("fun test { for env in envs { when @var.env { \"prod\" -> echo \"p\" else -> echo \"x\" } } }"))           // for → when
	f.Add([]byte(`fun test { when @var.env { "prod" -> if @var.ready { kubectl apply } else -> echo "skip" } }`))           // when → if
	f.Add([]byte("fun test { for item in items { try { kubectl apply } catch { echo \"failed\" } } }"))                     // for → try
	f.Add([]byte("fun test { try { for i in 1...3 { echo @var.i } } catch { echo \"err\" } }"))                             // try → for
	f.Add([]byte("fun test { if @var.deploy { for env in envs { kubectl apply -n @var.env } } }"))                          // if → for
	f.Add([]byte(`fun test { when @var.mode { "batch" -> for i in 1...10 { process @var.i } else -> echo "single" } }`))    // when → for
	f.Add([]byte("fun test { for x in 1...5 { when @var.x { 1...2 -> echo \"low\" 3...5 -> echo \"high\" } } }"))           // for → when (ranges)
	f.Add([]byte("fun test { try { when @var.env { \"prod\" -> kubectl apply else -> echo \"skip\" } } catch { } }"))       // try → when
	f.Add([]byte("fun test { for i in 1...3 { for j in 1...3 { for k in 1...3 { echo \"@var.i@var.j@var.k\" } } } }"))      // 3-level nesting
	f.Add([]byte("fun test { if true { if false { if @var.x { echo \"deep\" } } } }"))                                      // 3-level if nesting
	f.Add([]byte(`fun test { when @var.a { "x" -> when @var.b { "y" -> echo "xy" else -> echo "x" } else -> echo "?" } }`)) // when → when

	// Edge cases
	f.Add([]byte("fun"))                   // Incomplete
	f.Add([]byte("{}"))                    // Just braces
	f.Add([]byte("\"unterminated string")) // Unterminated

	// UTF-8 and line endings
	f.Add([]byte("fun test() {\r\n  echo \"hi\"\r\n}")) // CRLF
	f.Add([]byte("fun test { if\ntrue\n{\n}\n}"))       // If with newlines
	f.Add([]byte("fun test{if true{}}"))                // If no spaces
	f.Add([]byte("fun ðŸš€() {}"))                      // Emoji
	f.Add([]byte("\xff\xfe\xfd"))                       // Invalid UTF-8
}

// FuzzParserDeterminism verifies that parsing the same input twice
// produces identical event streams, tokens, and errors (full determinism).
func FuzzParserDeterminism(f *testing.F) {
	addSeedCorpus(f)

	f.Fuzz(func(t *testing.T, input []byte) {
		// Parse twice
		tree1 := Parse(input)
		tree2 := Parse(input)

		// Events must be identical
		if len(tree1.Events) != len(tree2.Events) {
			t.Errorf("Non-deterministic event count: %d vs %d",
				len(tree1.Events), len(tree2.Events))
			return
		}

		for i := range tree1.Events {
			if tree1.Events[i] != tree2.Events[i] {
				t.Errorf("Non-deterministic event at index %d: %+v vs %+v",
					i, tree1.Events[i], tree2.Events[i])
				return
			}
		}

		// Tokens must be identical (type, position, text)
		if len(tree1.Tokens) != len(tree2.Tokens) {
			t.Errorf("Non-deterministic token count: %d vs %d",
				len(tree1.Tokens), len(tree2.Tokens))
			return
		}

		for i := range tree1.Tokens {
			t1, t2 := tree1.Tokens[i], tree2.Tokens[i]
			if t1.Type != t2.Type || t1.Position != t2.Position ||
				!bytes.Equal(t1.Text, t2.Text) || t1.HasSpaceBefore != t2.HasSpaceBefore {
				t.Errorf("Non-deterministic token at index %d", i)
				return
			}
		}

		// Errors must be identical (message, count, order)
		if len(tree1.Errors) != len(tree2.Errors) {
			t.Errorf("Non-deterministic error count: %d vs %d",
				len(tree1.Errors), len(tree2.Errors))
			return
		}

		for i := range tree1.Errors {
			e1, e2 := tree1.Errors[i], tree2.Errors[i]
			if e1.Message != e2.Message ||
				e1.Position.Line != e2.Position.Line ||
				e1.Position.Column != e2.Position.Column ||
				e1.Position.Offset != e2.Position.Offset {
				t.Errorf("Non-deterministic error[%d]: %+v vs %+v", i, e1, e2)
				return
			}
		}
	})
}

// FuzzParserNoPanic verifies the parser never panics on any input.
func FuzzParserNoPanic(f *testing.F) {
	addSeedCorpus(f)

	// Additional seeds specific to panic testing
	f.Add([]byte("\x00\x01\x02"))           // Binary data
	f.Add(bytes.Repeat([]byte("a"), 10000)) // Very long
	f.Add(bytes.Repeat([]byte("{"), 1000))  // Deep nesting

	// Decorator syntax
	f.Add([]byte("@timeout(5m) { }"))
	f.Add([]byte("@retry(3, 2s) { }"))
	f.Add([]byte("@retry(delay=2s, 3)"))
	f.Add([]byte("@timeout(5m) { @retry(3) { } }"))

	// Malformed decorators
	f.Add([]byte("@retry("))
	f.Add([]byte("@retry(3,"))
	f.Add([]byte("@retry(3, times=5)"))
	f.Add([]byte("@var."))

	f.Fuzz(func(t *testing.T, input []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Parser panicked: %v", r)
			}
		}()

		tree := Parse(input)
		if tree == nil {
			t.Error("Parse returned nil")
			return
		}

		// Growth cap: catch quadratic explosions
		// Generous heuristic: 10x input size + 1KB overhead
		maxStructures := 10*len(input) + 1024
		actualStructures := len(tree.Events) + len(tree.Tokens)
		if actualStructures > maxStructures {
			t.Errorf("Structure blow-up: %d events+tokens > %d (10x input + 1KB)",
				actualStructures, maxStructures)
		}
	})
}

// FuzzParserEventBalance verifies Open/Close events are properly nested
// using a stack (not just counts). Catches cross-closing and negative depth.
func FuzzParserEventBalance(f *testing.F) {
	addSeedCorpus(f)

	// Additional seeds specific to event balance testing
	nested := make([]byte, 0, 200)
	nested = append(nested, bytes.Repeat([]byte("{"), 100)...)
	nested = append(nested, bytes.Repeat([]byte("}"), 100)...)
	f.Add(nested)

	f.Fuzz(func(t *testing.T, input []byte) {
		tree := Parse(input)

		// Track nesting with a type-aware stack
		// Catches cross-closing: Open(Function) ... Close(Block)
		var stack []NodeKind

		for i, event := range tree.Events {
			switch event.Kind {
			case EventOpen:
				nodeType := NodeKind(event.Data)
				stack = append(stack, nodeType)

			case EventClose:
				if len(stack) == 0 {
					t.Errorf("Close event at index %d with empty stack", i)
					return
				}
				// Pop and verify matching type
				openType := stack[len(stack)-1]
				closeType := NodeKind(event.Data)
				if openType != closeType {
					t.Errorf("Type mismatch at event %d: Open(%v) closed by Close(%v)",
						i, openType, closeType)
					return
				}
				stack = stack[:len(stack)-1]

			case EventToken:
				// Token events don't affect nesting
			}
		}

		// Stack must be empty at end
		if len(stack) != 0 {
			t.Errorf("Unclosed constructs: %d nodes remain on stack", len(stack))
		}
	})
}

// FuzzParserMemorySafety verifies positions are valid and monotonic.
func FuzzParserMemorySafety(f *testing.F) {
	addSeedCorpus(f)

	// Additional seeds specific to memory safety testing
	f.Add(bytes.Repeat([]byte("a"), 10000)) // Very long
	f.Add(bytes.Repeat([]byte("{"), 1000))  // Deep nesting
	longIdent := append([]byte("var "), bytes.Repeat([]byte("x"), 1000)...)
	longIdent = append(longIdent, []byte(" = 42")...)
	f.Add(longIdent)

	f.Fuzz(func(t *testing.T, input []byte) {
		tree := Parse(input)

		// Verify event→token indices are valid
		for i, event := range tree.Events {
			if event.Kind == EventToken {
				tokenIdx := int(event.Data)
				if tokenIdx < 0 || tokenIdx >= len(tree.Tokens) {
					t.Errorf("Event %d references invalid token index %d (have %d tokens)",
						i, tokenIdx, len(tree.Tokens))
				}
			}
		}

		// Verify token positions are valid
		for i, token := range tree.Tokens {
			// Line and column must be >= 1
			if token.Position.Line < 1 {
				t.Errorf("Token %d has invalid line %d (must be >= 1)",
					i, token.Position.Line)
			}
			if token.Position.Column < 1 {
				t.Errorf("Token %d has invalid column %d (must be >= 1)",
					i, token.Position.Column)
			}

			// Offset must be within source bounds
			if token.Position.Offset < 0 || token.Position.Offset > len(input) {
				t.Errorf("Token %d offset %d out of bounds (source length %d)",
					i, token.Position.Offset, len(input))
			}
		}

		// Verify positions are monotonic (offsets increasing)
		for i := 1; i < len(tree.Tokens); i++ {
			curr, prev := tree.Tokens[i], tree.Tokens[i-1]
			if curr.Position.Offset < prev.Position.Offset {
				t.Errorf("Tokens %d and %d not monotonic: offset %d then %d",
					i-1, i, prev.Position.Offset, curr.Position.Offset)
			}
		}

		// Verify line/column coherence (line never decreases, column resets after newline)
		if len(tree.Tokens) > 0 {
			last := tree.Tokens[0].Position
			for i := 1; i < len(tree.Tokens); i++ {
				p := tree.Tokens[i].Position
				// Offset must increase
				if p.Offset < last.Offset {
					t.Errorf("Non-monotonic offset at token %d: %d -> %d", i, last.Offset, p.Offset)
				}
				// Line must not decrease
				if p.Line < last.Line {
					t.Errorf("Line decreased at token %d: %d -> %d", i, last.Line, p.Line)
				}
				// If same line, column must not decrease
				if p.Line == last.Line && p.Column < last.Column {
					t.Errorf("Column decreased at token %d: line %d, col %d -> %d",
						i, p.Line, last.Column, p.Column)
				}
				last = p
			}
		}

		// Verify token reference monotonicity in events
		// EventToken indices should be non-decreasing (events reference tokens in order)
		lastTok := -1
		for i, ev := range tree.Events {
			if ev.Kind == EventToken {
				idx := int(ev.Data)
				if idx < lastTok {
					t.Errorf("Token refs not non-decreasing at event %d: %d -> %d", i, lastTok, idx)
				}
				lastTok = idx
			}
		}
	})
}

// FuzzParserPathologicalDepth verifies the parser handles deep nesting
// without panicking (prevents exponential backtracking).
func FuzzParserPathologicalDepth(f *testing.F) {
	addSeedCorpus(f)

	// Additional seeds specific to pathological depth testing
	nested1 := make([]byte, 0, 200)
	nested1 = append(nested1, bytes.Repeat([]byte("{"), 100)...)
	nested1 = append(nested1, bytes.Repeat([]byte("}"), 100)...)
	f.Add(nested1)

	nested2 := make([]byte, 0, 1000)
	nested2 = append(nested2, bytes.Repeat([]byte("fun f() { "), 50)...)
	nested2 = append(nested2, bytes.Repeat([]byte("}"), 50)...)
	f.Add(nested2)

	nested3 := make([]byte, 0, 1000)
	nested3 = append(nested3, []byte("fun test { ")...)
	nested3 = append(nested3, bytes.Repeat([]byte("if true { "), 50)...)
	nested3 = append(nested3, bytes.Repeat([]byte("} "), 50)...)
	nested3 = append(nested3, []byte("}")...)
	f.Add(nested3)

	f.Fuzz(func(t *testing.T, input []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Parser panicked on deep nesting: %v", r)
			}
		}()

		tree := Parse(input)

		// Count max depth in event stream
		maxDepth := 0
		currentDepth := 0

		for _, event := range tree.Events {
			switch event.Kind {
			case EventOpen:
				currentDepth++
				if currentDepth > maxDepth {
					maxDepth = currentDepth
				}
			case EventClose:
				currentDepth--
			}
		}

		// Parser should handle reasonable depth (1000 levels)
		// If depth exceeds this, should produce error, not panic
		const maxReasonableDepth = 1000
		if maxDepth > maxReasonableDepth && len(tree.Errors) == 0 {
			t.Logf("Warning: Very deep nesting (%d levels) without error", maxDepth)
		}
	})
}

// FuzzParserErrorRecovery verifies resilient parsing (errors, not crashes).
func FuzzParserErrorRecovery(f *testing.F) {
	addSeedCorpus(f)

	// Additional seeds specific to error recovery testing
	f.Add([]byte("fun"))        // Incomplete
	f.Add([]byte("fun greet(")) // Unclosed
	f.Add([]byte("@"))          // Lone decorator
	f.Add([]byte("var = 42"))   // Missing name

	f.Fuzz(func(t *testing.T, input []byte) {
		tree := Parse(input)

		// Errors should have messages
		for i, err := range tree.Errors {
			if err.Message == "" {
				t.Errorf("Error %d has empty message", i)
			}
		}
	})
}

// TestFuzzCorpusMinimization verifies the fuzz tests work correctly.
func TestFuzzCorpusMinimization(t *testing.T) {
	inputs := [][]byte{
		[]byte(""),
		[]byte("fun greet() {}"),
		[]byte("invalid syntax"),
		[]byte("@retry(3) { }"),
	}

	for _, input := range inputs {
		// Verify determinism
		tree1 := Parse(input)
		tree2 := Parse(input)

		if len(tree1.Events) != len(tree2.Events) {
			t.Errorf("Determinism failed for: %q", input)
		}

		// Verify no panic
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Panic on: %q", input)
				}
			}()
			Parse(input)
		}()

		// Verify event balance
		depth := 0
		for _, event := range tree1.Events {
			switch event.Kind {
			case EventOpen:
				depth++
			case EventClose:
				depth--
			}
			if depth < 0 {
				t.Errorf("Negative depth on: %q", input)
				break
			}
		}
		if depth != 0 {
			t.Errorf("Unbalanced events (depth=%d) on: %q", depth, input)
		}
	}
}

// FuzzParserWhitespaceInvariance ensures spaces/tabs between tokens
// do not affect the semantic token+event streams. Newlines are preserved.
// This is critical for plan hashing stability.
//
// The test reconstructs input by preserving HasSpaceBefore flags (token boundaries)
// while varying the amount/type of whitespace. This ensures:
// - Tokens don't merge (+ + stays separate, doesn't become ++)
// - Amount of whitespace doesn't matter (1 space vs 10 spaces)
// - Newlines are preserved (they're semantic in Opal)
func FuzzParserWhitespaceInvariance(f *testing.F) {
	addSeedCorpus(f)

	// Additional seeds specific to whitespace invariance testing
	f.Add([]byte("fun greet(name){echo name}"))
	f.Add([]byte("var x=1\nvar y=2\nfun f(){x+y}"))
	f.Add([]byte("fun test{if true{}}"))
	f.Add([]byte("fun\u200Bz(){}"))  // ZWSP
	f.Add([]byte("x\u00A0=\u00A01")) // NBSP

	f.Fuzz(func(t *testing.T, input []byte) {
		orig := Parse(input)

		// If parser didn't produce tokens, nothing to test
		if len(orig.Tokens) == 0 {
			return
		}

		// Skip inputs that are mostly ILLEGAL tokens (invalid syntax)
		// Whitespace changes can alter tokenization of invalid syntax
		// Example: "& & &" (3 ILLEGAL) → "& &&" (1 ILLEGAL + 1 AND_AND) when spaces removed
		illegalCount := 0
		for _, tk := range orig.Tokens {
			if tk.Type == lexer.ILLEGAL {
				illegalCount++
			}
		}
		// If half or more of the non-EOF tokens are ILLEGAL, skip
		// (whitespace changes can alter tokenization of invalid syntax)
		nonEOF := len(orig.Tokens)
		if nonEOF > 0 && orig.Tokens[nonEOF-1].Type == lexer.EOF {
			nonEOF--
		}
		if nonEOF > 0 && illegalCount*2 >= nonEOF {
			return
		}

		// Helper: semantic token view (ignore positions)
		type semanticToken struct {
			Type lexer.TokenType
			Text string
		}
		semanticTokens := func(tokens []lexer.Token) []semanticToken {
			out := make([]semanticToken, len(tokens))
			for i, tk := range tokens {
				out[i] = semanticToken{Type: tk.Type, Text: string(tk.Text)}
			}
			return out
		}

		// Helper: semantic event view (ignore positions)
		type semanticEvent struct {
			Kind EventKind
			Data uint32
		}
		semanticEvents := func(events []Event) []semanticEvent {
			out := make([]semanticEvent, len(events))
			for i, ev := range events {
				out[i] = semanticEvent(ev)
			}
			return out
		}

		// Reconstruct input by trusting HasSpaceBefore flags
		// Key insight: HasSpaceBefore already tells us if there was whitespace
		// We just vary the amount/type while preserving token boundaries
		var buf bytes.Buffer

		// Deterministic RNG based on input
		seed := int64(0)
		for _, b := range input {
			seed = seed*31 + int64(b)
		}
		rng := struct{ state int64 }{state: seed}
		randInt := func(n int) int {
			rng.state = rng.state*1103515245 + 12345
			val := (rng.state / 65536) % int64(n)
			if val < 0 {
				val = -val
			}
			return int(val)
		}

		// Helper: get token text (from tk.Text or infer from type)
		getTokenText := func(tk lexer.Token) []byte {
			if len(tk.Text) > 0 {
				return tk.Text
			}
			// Token has nil Text - infer from type
			switch tk.Type {
			case lexer.LPAREN:
				return []byte("(")
			case lexer.RPAREN:
				return []byte(")")
			case lexer.LBRACE:
				return []byte("{")
			case lexer.RBRACE:
				return []byte("}")
			case lexer.LSQUARE:
				return []byte("[")
			case lexer.RSQUARE:
				return []byte("]")
			case lexer.COMMA:
				return []byte(",")
			case lexer.COLON:
				return []byte(":")
			case lexer.DOT:
				return []byte(".")
			case lexer.AT:
				return []byte("@")
			case lexer.SEMICOLON:
				return []byte(";")
			case lexer.PLUS:
				return []byte("+")
			case lexer.MINUS:
				return []byte("-")
			case lexer.MULTIPLY:
				return []byte("*")
			case lexer.DIVIDE:
				return []byte("/")
			case lexer.MODULO:
				return []byte("%")
			case lexer.LT:
				return []byte("<")
			case lexer.GT:
				return []byte(">")
			case lexer.APPEND:
				return []byte(">>")
			case lexer.NOT:
				return []byte("!")
			case lexer.EQUALS:
				return []byte("=")
			case lexer.PIPE:
				return []byte("|")
			case lexer.EQ_EQ:
				return []byte("==")
			case lexer.NOT_EQ:
				return []byte("!=")
			case lexer.LT_EQ:
				return []byte("<=")
			case lexer.GT_EQ:
				return []byte(">=")
			case lexer.AND_AND:
				return []byte("&&")
			case lexer.OR_OR:
				return []byte("||")
			case lexer.INCREMENT:
				return []byte("++")
			case lexer.DECREMENT:
				return []byte("--")
			case lexer.PLUS_ASSIGN:
				return []byte("+=")
			case lexer.MINUS_ASSIGN:
				return []byte("-=")
			case lexer.MULTIPLY_ASSIGN:
				return []byte("*=")
			case lexer.DIVIDE_ASSIGN:
				return []byte("/=")
			case lexer.MODULO_ASSIGN:
				return []byte("%=")
			case lexer.ARROW:
				return []byte("->")
			case lexer.FUN:
				return []byte("fun")
			case lexer.VAR:
				return []byte("var")
			case lexer.FOR:
				return []byte("for")
			case lexer.IN:
				return []byte("in")
			case lexer.IF:
				return []byte("if")
			case lexer.ELSE:
				return []byte("else")
			case lexer.WHEN:
				return []byte("when")
			case lexer.TRY:
				return []byte("try")
			case lexer.CATCH:
				return []byte("catch")
			case lexer.FINALLY:
				return []byte("finally")
			case lexer.NEWLINE:
				return []byte("\n")
			case lexer.COMMENT:
				// Comments have text in tk.Text, handled specially
				return nil
			case lexer.ILLEGAL:
				// ILLEGAL tokens have text in tk.Text
				return nil
			default:
				return nil
			}
		}

		// Track cursor in original input to extract newlines
		cursor := 0

		for _, tk := range orig.Tokens {
			// Extract newlines from gap before this token
			if tk.Position.Offset > cursor {
				gap := input[cursor:tk.Position.Offset]
				for _, b := range gap {
					if b == '\n' || b == '\r' {
						buf.WriteByte(b)
					}
				}
			}

			// If token had whitespace before it, emit random amount
			if tk.HasSpaceBefore {
				n := 1 + randInt(3) // 1-3 spaces/tabs
				for j := 0; j < n; j++ {
					if randInt(2) == 0 {
						buf.WriteByte(' ')
					} else {
						buf.WriteByte('\t')
					}
				}
			}

			// Write token text
			if tk.Type == lexer.COMMENT {
				// Comments need full reconstruction: /* content */ or // content
				// Check source to determine comment type
				offset := tk.Position.Offset
				if offset+1 < len(input) && input[offset] == '/' && input[offset+1] == '/' {
					// Line comment: // + content
					buf.WriteString("//")
					buf.Write(tk.Text)
					cursor = offset + 2 + len(tk.Text)
				} else if offset+1 < len(input) && input[offset] == '/' && input[offset+1] == '*' {
					// Block comment: /* + content + */ (if terminated)
					buf.WriteString("/*")
					buf.Write(tk.Text)
					// Check if terminated by looking at source
					terminated := false
					if len(input) >= offset+2+len(tk.Text)+2 {
						checkPos := offset + 2 + len(tk.Text)
						if checkPos+1 < len(input) && input[checkPos] == '*' && input[checkPos+1] == '/' {
							terminated = true
						}
					}
					if terminated {
						buf.WriteString("*/")
						cursor = offset + 2 + len(tk.Text) + 2
					} else {
						cursor = offset + 2 + len(tk.Text)
					}
				}
			} else if len(tk.Text) > 0 {
				// Token has explicit text (identifiers, strings, numbers)
				buf.Write(tk.Text)
				cursor = tk.Position.Offset + len(tk.Text)
			} else {
				// Token has nil Text (operators, keywords) - get from type
				tokenText := getTokenText(tk)
				if tokenText != nil {
					buf.Write(tokenText)
					cursor = tk.Position.Offset + len(tokenText)
				}
			}
		}

		noised := buf.Bytes()
		got := Parse(noised)

		// Compare semantic tokens (type + text only, ignore position)
		st1, st2 := semanticTokens(orig.Tokens), semanticTokens(got.Tokens)
		if len(st1) != len(st2) {
			t.Errorf("Token count changed with whitespace: %d -> %d", len(st1), len(st2))
			t.Errorf("Original input: %q", input)
			t.Errorf("Noised input: %q", noised)
			t.Errorf("Original tokens: %v", st1)
			t.Errorf("Noised tokens: %v", st2)
			return
		}
		for i := range st1 {
			if st1[i] != st2[i] {
				t.Errorf("Token %d changed with whitespace: %+v -> %+v", i, st1[i], st2[i])
				return
			}
		}

		// Compare semantic events (kind + data only, ignore positions)
		se1, se2 := semanticEvents(orig.Events), semanticEvents(got.Events)
		if len(se1) != len(se2) {
			t.Errorf("Event count changed with whitespace: %d -> %d", len(se1), len(se2))
			return
		}
		for i := range se1 {
			if se1[i] != se2[i] {
				t.Errorf("Event %d changed with whitespace: %+v -> %+v", i, se1[i], se2[i])
				return
			}
		}
	})
}

// FuzzParserSmokeTest is a simple smoke test that just runs Parse() on random bytes.
// No invariants checked - just looking for crashes the other 7 fuzz functions might miss.
// This is the "catch-all" fuzzer for unexpected edge cases.
func FuzzParserSmokeTest(f *testing.F) {
	addSeedCorpus(f)

	// Additional seeds for smoke testing
	f.Add([]byte("\x00"))                                         // Null byte
	f.Add([]byte("\x00\x00\x00"))                                 // Multiple nulls
	f.Add(bytes.Repeat([]byte("\x00"), 1000))                     // Many nulls
	f.Add(bytes.Repeat([]byte("a"), 100000))                      // Very long input
	f.Add(bytes.Repeat([]byte("{"), 10000))                       // Deep nesting
	f.Add([]byte("@" + string(bytes.Repeat([]byte("a"), 10000)))) // Long identifier

	f.Fuzz(func(t *testing.T, input []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Parser panicked on smoke test: %v\nInput: %q", r, input)
			}
		}()

		// Just parse - no invariants
		tree := Parse(input)

		// Basic sanity: tree should not be nil
		if tree == nil {
			t.Error("Parse returned nil")
		}
	})
}
