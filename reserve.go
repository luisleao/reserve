package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/s4y/go-sse"
	"github.com/s4y/reserve/httpsuffixer"
	"github.com/s4y/reserve/watcher"
)

var gFilters = map[string][]byte{
	"text/html": []byte(`
<script>
'use strict';
(() => {
	const newHookForExtension = {
		'css': f => {
			for (let el of document.querySelectorAll('link[rel=stylesheet]')) {
				if (el.href != f)
					continue;
				return () => {
					return fetch(f, { cache: 'reload' })
						.then(r => r.blob())
						.then(blob => {
							el.href = URL.createObjectURL(blob);
						});
				};
				break;
			}
		},
		'js': f => {
		return () => {
			return fetch(f, { cache: 'reload' })
				.then(r => r.blob())
				.then(blob => Promise.all([
					import(` + "`" + `${f}?live_module` + "`" + `),
					import(URL.createObjectURL(blob)),
				]))
				.then(mods => {
					const oldm = mods[0];
					const newm = mods[1];
					if (!oldm.__reserve_setters)
						location.reload(true);
					for (const k in newm) {
						const oldproto = oldm[k].prototype;
						const newproto = newm[k].prototype;
						if (oldproto) {
							for (const protok of Object.getOwnPropertyNames(oldproto)) {
								Object.defineProperty(oldproto, protok, { value: function (...args) {
									Object.setPrototypeOf(this, newproto);
									if (this.adopt)
										this.adopt(oldproto);
									this[protok](...args);
								} });
							}
						}
						const setter = oldm.__reserve_setters[k];
						if (!setter)
							location.reload(true);
						setter(newm[k]);
					}
				})
			};
		}
	};
	const hooks = Object.create(null);

	const es = new EventSource("/.reserve/changes");
	es.addEventListener('change', e => {
		const target = new URL(e.data, location.href).href;
		if (!(target in hooks)) {
			const ext = target.split('/').pop().split('.').pop();
			if (newHookForExtension[ext])
				hooks[target] = newHookForExtension[ext](target);
		}
		if (hooks[target]) {
			if (hooks[target]() !== false)
				return;
		}
		location.reload(true);
	});

	let wasOpen = false;
	es.addEventListener('open', e => {
		if (wasOpen)
			location.reload(true);
		wasOpen = true;
	});

	let stdin = new EventSource("/.reserve/stdin");
	stdin.addEventListener("line", e => {
		const ev = new CustomEvent('stdin');
		ev.data = e.data;
		window.dispatchEvent(ev);
	});
})();
</script>
`),
}

func jsWrapper(orig_filename string) string {
	f := template.JSEscapeString(orig_filename)
	return `
export * from "` + f + `"

import * as mod from "` + f + `"
let _default = mod.default
export {_default as default}

export const __reserve_setters = {}
for (const k in mod)
  __reserve_setters[k] = eval(` + "`" + `v => ${k == 'default' ? '_default' : k} = v` + "`" + `)
	`
}

func main() {
	httpAddr := flag.String("http", "127.0.0.1:8080", "Listening address")
	flag.Parse()
	fmt.Printf("http://%s/\n", *httpAddr)

	ln, err := net.Listen("tcp", *httpAddr)
	if err != nil {
		log.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	changeServer := sse.Server{}

	suffixer := httpsuffixer.SuffixServer{gFilters}

	watcher := watcher.NewWatcher(cwd)
	go func() {
		for change := range watcher.Changes {
			if strings.HasPrefix(path.Base(change), ".") {
				continue
			}
			changeServer.Broadcast(sse.Event{Name: "change", Data: "/" + change})
		}
	}()

	stdinServer := sse.Server{}
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			stdinServer.Broadcast(sse.Event{Name: "line", Data: scanner.Text()})
		}
		os.Exit(0)
	}()

	fileServer := suffixer.WrapServer(http.FileServer(http.Dir(cwd)))

	log.Fatal(http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.reserve/changes" {
			changeServer.ServeHTTP(w, r)
		} else if r.URL.Path == "/.reserve/stdin" {
			stdinServer.ServeHTTP(w, r)
		} else if _, exists := r.URL.Query()["live_module"]; exists {
			w.Header().Set("Content-Type", "application/javascript")
			w.Write([]byte(jsWrapper(r.URL.Path)))
		} else {
			fileServer.ServeHTTP(w, r)
			// w.Write([]byte("outer fn was here"))
		}
	})))
}
