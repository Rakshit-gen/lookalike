package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"net/url"
	"os"
	"path/filepath"

	"github.com/Rakshit-gen/lookalike/internal/group"
)

var reportPage = template.Must(template.New("report").Funcs(template.FuncMap{
	"fileURL": func(root, rel string) template.URL {
		u := url.URL{Scheme: "file", Path: filepath.ToSlash(filepath.Join(root, filepath.FromSlash(rel)))}
		return template.URL(u.String())
	},
	"size": humanSize,
}).Parse(`<!doctype html>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>lookalike: {{.Root}}</title>
<style>
  :root { color-scheme: light dark; --bg: #fff; --fg: #1b1b1b; --muted: #666; --line: #ddd; --keep: #1a7f37; }
  @media (prefers-color-scheme: dark) { :root { --bg: #161616; --fg: #e8e8e8; --muted: #999; --line: #333; --keep: #3fb950; } }
  body { margin: 0 auto; max-width: 1100px; padding: 16px; font: 14px/1.4 system-ui, sans-serif; background: var(--bg); color: var(--fg); }
  h1 { font-size: 18px; word-break: break-all; }
  .meta { color: var(--muted); }
  section { border-top: 1px solid var(--line); padding: 12px 0; }
  .photos { display: flex; flex-wrap: wrap; gap: 12px; }
  figure { margin: 0; width: 200px; }
  img { width: 200px; height: 150px; object-fit: contain; background: var(--line); border: 3px solid transparent; }
  .keep img { border-color: var(--keep); }
  figcaption { font-size: 12px; word-break: break-all; }
  .keep figcaption::before { content: "keep: "; color: var(--keep); font-weight: 600; }
</style>
<h1>{{.Root}}</h1>
<p class="meta">{{len .Groups}} groups at similarity {{printf "%.2f" .Threshold}} or more. The green copy is the one lookalike move would keep.</p>
{{range .Groups}}<section>
  <p class="meta">{{len .Photos}} photos, similarity {{printf "%.3f" .Loosest}} to {{printf "%.3f" .Closest}}</p>
  <div class="photos">{{range $i, $p := .Photos}}
    <figure{{if eq $i 0}} class="keep"{{end}}>
      <a href="{{fileURL $.Root $p.Path}}"><img loading="lazy" src="{{fileURL $.Root $p.Path}}" alt=""></a>
      <figcaption>{{$p.Path}}<br>{{$p.Width}}x{{$p.Height}}, {{size $p.Size}}</figcaption>
    </figure>{{end}}
  </div>
</section>{{end}}
`))

func runReport(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("report", flag.ExitOnError)
	threshold := fs.Float64("min", defaultThreshold, "minimum similarity (0 to 1) for two photos to be grouped")
	rotated := fs.Bool("rotated", false, "also match copies turned a quarter or half turn")
	out := fs.String("o", "lookalike-report.html", "where to write the report")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, "usage: lookalike report [flags] FOLDER\n\n"+
			"Writes an HTML page showing each group's photos side by side. Open it in a\n"+
			"browser on the same computer; it links to the photos on disk.\n\n")
		fs.PrintDefaults()
	}
	fs.Parse(args)
	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(2)
	}
	lib, err := openLibrary(ctx, fs.Arg(0))
	if err != nil {
		return err
	}
	groups, err := group.Find(ctx, nuclaClient(), lib, group.Options{Threshold: *threshold, Neighbors: 10, Rotated: *rotated})
	if err != nil {
		return err
	}
	f, err := os.Create(*out)
	if err != nil {
		return err
	}
	err = reportPage.Execute(f, map[string]any{"Root": lib.Root, "Groups": groups, "Threshold": *threshold})
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	fmt.Printf("wrote %s (%d groups)\n", *out, len(groups))
	return nil
}
