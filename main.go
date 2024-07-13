package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"

	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"github.com/warden-protocol/wardenprotocol/shield"
	"github.com/warden-protocol/wardenprotocol/shield/ast"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
<!DOCTYPE html>
<html>
	<head>
		<title>Shield UI</title>
		<meta charset="utf-8">
		<meta name="viewport" content="width=device-width, initial-scale=1">
		<style>
			body {
				font-family: sans-serif;
			}

			input[type="text"] {
				width: 100%;
				padding: 12px 20px;
				margin: 8px 0;
				box-sizing: border-box;
				border: 2px solid #ccc;
				border-radius: 4px;
			}

			button {
				background-color: #4CAF50;
				color: white;
				padding: 12px 20px;
				margin: 8px 0;
				border: none;
				border-radius: 4px;
				cursor: pointer;
			}

			button:hover {
				background-color: #45a049;
			}
		</style>
	</head>
	<body>
		<form action="/graph" method="get">
			<input type="text" name="definition" placeholder="definition" />
			<button type="submit">Submit</button>
		</form>
	</body>
</html>
`))
	})

	http.HandleFunc("/graph", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		def := r.Form.Get("definition")

		expr, err := shield.Parse(def)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("error parsing definition `%s`:\n %v", def, err)))
			return
		}

		g := graphviz.New()
		graph, err := g.Graph()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		defer func() {
			if err := graph.Close(); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
				return
			}
			g.Close()
		}()

		if _, err := gen(expr, graph); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		// n, err := createNode(graph, "n")
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// m, err := createNode(graph, "m")
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// e, err := createEdge(graph, "e", n, m)
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// e.SetLabel("e")

		w.Header().Set("Content-Type", "image/svg+xml")
		if err := g.Render(graph, "svg", w); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
	})

	fmt.Println("Listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func gen(expr *ast.Expression, graph *cgraph.Graph) (*cgraph.Node, error) {
	if expr == nil {
		return nil, fmt.Errorf("expr is nil")
	}

	if graph == nil {
		return nil, fmt.Errorf("graph is nil")
	}

	switch e := expr.Value.(type) {
	case *ast.Expression_Identifier:
		return createNode(graph, e.Identifier.Value)
	case *ast.Expression_BooleanLiteral:
		return createNode(graph, fmt.Sprint(e.BooleanLiteral.Value))
	case *ast.Expression_IntegerLiteral:
		return createNode(graph, fmt.Sprint(e.IntegerLiteral.Value))
	case *ast.Expression_StringLiteral:
		return createNode(graph, fmt.Sprint(e.StringLiteral.Value))
	case *ast.Expression_ArrayLiteral:
		parent, err := createNode(graph, "array")
		if err != nil {
			return nil, err
		}
		parent.SetColor("red")
		for _, e := range e.ArrayLiteral.Elements {
			child, err := gen(e, graph)
			if err != nil {
				return nil, err
			}
			if _, err := createEdge(graph, "", parent, child); err != nil {
				return nil, err
			}
		}
		return parent, nil
	case *ast.Expression_InfixExpression:
		left, err := gen(e.InfixExpression.Left, graph)
		if err != nil {
			return nil, err
		}
		right, err := gen(e.InfixExpression.Right, graph)
		if err != nil {
			return nil, err
		}
		op, err := createNode(graph, e.InfixExpression.Operator)
		if err != nil {
			return nil, err
		}
		op.SetColor("blue")
		if _, err := createEdge(graph, "", op, left); err != nil {
			return nil, err
		}
		if _, err := createEdge(graph, "", op, right); err != nil {
			return nil, err
		}
		return op, nil
	case *ast.Expression_PrefixExpression:
		right, err := gen(e.PrefixExpression.Right, graph)
		if err != nil {
			return nil, err
		}
		op, err := createNode(graph, e.PrefixExpression.Operator)
		if err != nil {
			return nil, err
		}
		op.SetColor("blue")
		if _, err := createEdge(graph, "", op, right); err != nil {
			return nil, err
		}
		return op, nil
	case *ast.Expression_CallExpression:
		parent, err := createNode(graph, "call")
		if err != nil {
			return nil, err
		}
		parent.SetColor("green")

		identifier, err := createNode(graph, e.CallExpression.Function.Value)
		if err != nil {
			return nil, err
		}
		if _, err := createEdge(graph, "fn", parent, identifier); err != nil {
			return nil, err
		}

		args, err := createNode(graph, "args")
		if err != nil {
			return nil, err
		}
		if _, err := createEdge(graph, "", parent, args); err != nil {
			return nil, err
		}

		for _, e := range e.CallExpression.Arguments {
			child, err := gen(e, graph)
			if err != nil {
				return nil, err
			}
			if _, err := createEdge(graph, "", args, child); err != nil {
				return nil, err
			}
		}
		return parent, nil
	}

	return nil, nil
}

func createNode(graph *cgraph.Graph, name string) (*cgraph.Node, error) {
	randomName := fmt.Sprintf("node_%d", rand.Int())
	node, err := graph.CreateNode(randomName)
	if err != nil {
		return nil, err
	}
	node.SetLabel(name)
	return node, nil
}

func createEdge(graph *cgraph.Graph, name string, from *cgraph.Node, to *cgraph.Node) (*cgraph.Edge, error) {
	randomName := fmt.Sprintf("edge_%d", rand.Int())
	edge, err := graph.CreateEdge(randomName, from, to)
	if err != nil {
		return nil, err
	}
	if name != "" {
		edge.SetLabel(name)
	}
	return edge, nil
}
