const assert = require('assert').strict
const stream = require('stream')

const sh = require('./index')

const syntax = sh.syntax
const parser = syntax.NewParser()
const printer = syntax.NewPrinter()

{
	// parsing a simple program
	const src = "echo 'foo'"
	var f = parser.Parse(src, "src")

	var stmts = f.StmtList.Stmts
	assert.equal(stmts.length, 1)

	var args = stmts[0].Cmd.Args
	assert.equal(args.length, 2)
	assert.equal(args[0].Parts.length, 1)
	assert.equal(args[0].Parts[0].Value, "echo")
}

{
	// accessing fields or methods creates separate objects
	const src = "echo 'foo'"
	var f = parser.Parse(src, "src")

	assert.equal(f.StmtList.Stmts == f.StmtList.Stmts, false)
	assert.equal(f.StmtList.Stmts === f.StmtList.Stmts, false)
	var stmts = f.StmtList.Stmts
	assert.equal(stmts == stmts, true)
	assert.equal(stmts === stmts, true)
}

{
	// parse errors
	const src = "echo ${"
	try {
		parser.Parse(src, "src")
		assert.fail("did not error")
	} catch (err) {
	}
}

{
	// node types, operators, and positions
	const src = "foo || bar"
	var f = parser.Parse(src, "src")

	var cmd = f.StmtList.Stmts[0].Cmd
	assert.equal(syntax.NodeType(cmd), "BinaryCmd")

	// TODO: see https://github.com/myitcv/gopherjs/issues/26
	// assert.equal(syntax.String(cmd.Op), "||")

	assert.equal(cmd.Pos().String(), "1:1")
	assert.equal(cmd.OpPos.String(), "1:5")
	assert.equal(cmd.OpPos.Line(), 1)
	assert.equal(cmd.OpPos.Col(), 5)
	assert.equal(cmd.OpPos.Offset(), 4)
}

{
	// running Walk
	const src = "foo bar"
	var f = parser.Parse(src, "src")

	var nilCount = 0
	var nonNilCount = 0
	var seenBar = false
	var seenCall = false
	syntax.Walk(f, function(node) {
		var typ = syntax.NodeType(node)
		if (node == null) {
			nilCount++
			assert.equal(typ, "nil")
		} else {
			nonNilCount++
			if (node.Value == "bar") {
				seenBar = true
			}
			assert.notEqual(typ, "nil")
		}
		if (typ == "CallExpr") {
			seenCall = true
		}
		return true
	})
	assert.equal(nonNilCount, 7)
	assert.equal(nilCount, 7)
	assert.equal(seenBar, true)
	assert.equal(seenCall, true)
}

{
	// printing
	const src = "echo      'foo'"
	var f = parser.Parse(src, "src")

	var out = printer.Print(f)
	assert.equal(out, "echo 'foo'\n")
}

{
	// parser options
	const parser = syntax.NewParser(
		syntax.KeepComments,
		syntax.Variant(syntax.LangMirBSDKorn),
		syntax.StopAt("$$")
	)
	const src = "echo ${|stmts;} # bar\n$$"
	var f = parser.Parse(src, "src")

	var out = printer.Print(f)
	assert.equal(out, "echo ${|stmts;} # bar\n")
}

{
	// parsing a readable stream
	const src = new stream.Readable
	src.push("echo foo")
	src.push(null)

	var f = parser.Parse(src, "src")

	var cmd = f.StmtList.Stmts[0].Cmd
	assert.equal(cmd.Args.length, 2)
}

{
	// using the parser interactively
	const inputs = [
		"foo\n",
		"bar; baz\n",
		"\n",
		"'incom\n",
		"plete'\n",
	]
	const wantCallbacks = [
		{"count": 1, "incomplete": false},
		{"count": 2, "incomplete": false},
		{"count": 0, "incomplete": false},
		{"count": 0, "incomplete": true},
		{"count": 1, "incomplete": false},
	]
	var gotCallbacks = []

	const src = {"read": function(size) {
		if (inputs.length == 0) {
			if (gotCallbacks.length == 0) {
				throw "did not see any callbacks before EOF"
			}
			return null // EOF
		}
		s = inputs[0]
		inputs.shift()
		return s
	}}

	parser.Interactive(src, function(stmts) {
		gotCallbacks.push({
			"count":      stmts.length,
			"incomplete": parser.Incomplete(),
		})
		return true
	})
	assert.deepEqual(gotCallbacks, wantCallbacks)
}
