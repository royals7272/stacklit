package engine

import (
	"testing"

	"github.com/glincker/stacklit/internal/config"
	"github.com/glincker/stacklit/internal/git"
	"github.com/glincker/stacklit/internal/graph"
	"github.com/glincker/stacklit/internal/monorepo"
	"github.com/glincker/stacklit/internal/parser"
)

func TestAssembleIndexFiltersTrimmedModuleReferences(t *testing.T) {
	files := []*parser.FileInfo{
		{Path: "src/api/index.ts", Language: "TypeScript", Imports: []string{"src/auth", "src/db"}, LineCount: 50},
		{Path: "src/auth/service.ts", Language: "TypeScript", Imports: []string{"src/db"}, LineCount: 40},
		{Path: "src/db/pool.ts", Language: "TypeScript", Exports: []string{"Pool"}, LineCount: 30},
	}

	g := graph.Build(files, graph.BuildOptions{MaxDepth: 4})
	cfg := config.DefaultConfig()
	cfg.MaxModules = 2

	idx := assembleIndex(
		".",
		&monorepo.Result{Type: "single"},
		[]string{"src/api/index.ts", "src/auth/service.ts", "src/db/pool.ts"},
		files,
		g,
		&git.Activity{},
		map[string][]byte{},
		cfg,
	)

	if len(idx.Modules) != 2 {
		t.Fatalf("expected 2 retained modules, got %d: %v", len(idx.Modules), idx.Modules)
	}
	if _, ok := idx.Modules["src/db"]; ok {
		t.Fatalf("expected trimmed module src/db to be omitted from modules, got %v", idx.Modules)
	}

	api := idx.Modules["src/api"]
	if len(api.DependsOn) != 1 || api.DependsOn[0] != "src/auth" {
		t.Fatalf("expected src/api depends_on to retain only src/auth, got %v", api.DependsOn)
	}

	auth := idx.Modules["src/auth"]
	if len(auth.DependsOn) != 0 {
		t.Fatalf("expected src/auth depends_on to drop trimmed src/db reference, got %v", auth.DependsOn)
	}
	if len(auth.DependedBy) != 1 || auth.DependedBy[0] != "src/api" {
		t.Fatalf("expected src/auth depended_by to contain only src/api, got %v", auth.DependedBy)
	}

	if len(idx.Dependencies.Edges) != 1 || idx.Dependencies.Edges[0] != ([2]string{"src/api", "src/auth"}) {
		t.Fatalf("expected only retained edge src/api -> src/auth, got %v", idx.Dependencies.Edges)
	}

	if len(idx.Dependencies.MostDepended) < 2 {
		t.Fatalf("expected ranked retained modules, got %v", idx.Dependencies.MostDepended)
	}
	if idx.Dependencies.MostDepended[0] != "src/auth" {
		t.Fatalf("expected src/auth to be most depended after trimming, got %v", idx.Dependencies.MostDepended)
	}
	for _, name := range idx.Dependencies.MostDepended {
		if name == "src/db" {
			t.Fatalf("expected trimmed module src/db to be absent from most_depended, got %v", idx.Dependencies.MostDepended)
		}
	}

	if len(idx.Dependencies.Isolated) != 0 {
		t.Fatalf("expected no isolated retained modules, got %v", idx.Dependencies.Isolated)
	}
}
