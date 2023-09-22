package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/kio"
)

func main() {
	lg := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{}))
	flag.Parse()
	for i, arg := range flag.Args() {
		lg := lg.With(slog.Int("arg-index", i), slog.String("arg-name", arg))
		process(lg, arg)
	}
}

func process(lg *slog.Logger, arg string) (err error) {
	ctx := context.Background()

	var r io.ReadCloser = os.Stdin
	if arg != "-" {
		r, err = os.Open(arg)
		if err != nil {
			lg.LogAttrs(ctx, slog.LevelError, "open", slog.String("error", err.Error()))
			return err
		}
	}
	defer r.Close()

	b, err := io.ReadAll(r)
	if err != nil {
		lg.LogAttrs(ctx, slog.LevelError, "read", slog.String("error", err.Error()))
		return err
	}

	nodes, err := kio.FromBytes(b)
	if err != nil {
		lg.LogAttrs(ctx, slog.LevelError, "parse yaml", slog.String("error", err.Error()))
		return err
	}

	rep := strings.NewReplacer("/", "_", ".", "_")

	for i, node := range nodes {
		lg := lg.With(slog.Int("document-index", i))
		meta, err := node.GetMeta()
		if err != nil {
			lg.LogAttrs(ctx, slog.LevelError, "get meta", slog.String("error", err.Error()))
			continue
		}

		filename := fmt.Sprintf(
			"%s__%s__%s__%s.yaml",
			strings.ToLower(rep.Replace(meta.APIVersion)),
			strings.ToLower(meta.Kind),
			meta.Namespace,
			meta.Name,
		)
		lg = lg.With(
			slog.String("apiVersion", meta.APIVersion),
			slog.String("kind", meta.Kind),
			slog.String("namespace", meta.Namespace),
			slog.String("name", meta.Name),
			slog.String("filename", filename),
		)
		err = os.WriteFile(filename, []byte(node.MustString()), 0o644)
		if err != nil {
			lg.LogAttrs(ctx, slog.LevelError, "write file", slog.String("error", err.Error()))
			continue
		}
	}
	return nil
}
