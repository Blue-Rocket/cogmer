// Command suiterun runs the module's tests in several processes at once and reports them
// as one run. scripts/test-and-record.sh and .githooks/pre-push run the suite through it.
//
//	go run ./cmd/suiterun [-short] [-shards N] <times file>
//
// The tests of one package run in one process under go test, one after another, since
// most of them set environment variables, which a process shares. suiterun compiles each
// package's tests once, splits the tests into one group for each process, balanced by the
// times the last run recorded, and runs the groups at once, each in a process of its own
// with an environment of its own. It prints what go test would print, and writes every
// test's time, slowest first, to the times file.
//
// It exits 1 when a test or a build failed, and 2 when it could not run the suite.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// stampedBuildFlags must match stampedBuildFlags in cmd/cogmer/testhelpers_test.go. The
// whole suite gives every process the one build made with them (D-293, a test gives a
// build its version by overwriting the stamped bytes), so that each process does not make
// its own. buildVersion checks the flags this build was made with against its own and
// fails every test that asks for a version when they differ.
var stampedBuildFlags = "-X main.version=cogmer-test-stamp" + strings.Repeat("~", 47) + " -X main.noBrowser=1"

type pkg struct {
	ImportPath   string
	Dir          string
	TestGoFiles  []string
	XTestGoFiles []string
}

type event struct {
	Action  string
	Package string
	Test    string
	Elapsed *float64
	Output  string
}

type timing struct {
	seconds float64
	name    string
}

func main() {
	short := flag.Bool("short", false, "run the tests with -short")
	shards := flag.Int("shards", 2*runtime.NumCPU(), "how many test processes to run at once; the tests mostly wait, so twice the cores was fastest on four")
	flag.Parse()
	if flag.NArg() != 1 || *shards < 1 {
		fmt.Fprintln(os.Stderr, "usage: suiterun [-short] [-shards N] <times file>")
		os.Exit(2)
	}
	timesFile := flag.Arg(0)
	failed, err := run(*short, *shards, timesFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "suiterun:", err)
		os.Exit(2)
	}
	if failed {
		os.Exit(1)
	}
}

func run(short bool, shards int, timesFile string) (bool, error) {
	work, err := os.MkdirTemp("", "suiterun-")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(work)

	out, err := exec.Command("go", "list", "-json", "./...").Output()
	if err != nil {
		return false, fmt.Errorf("go list: %v", err)
	}
	var pkgs []pkg
	dec := json.NewDecoder(bytes.NewReader(out))
	for dec.More() {
		var p pkg
		if err := dec.Decode(&p); err != nil {
			return false, err
		}
		pkgs = append(pkgs, p)
	}

	// Compile every package's tests, and the one build the whole suite shares, at once.
	env := os.Environ()
	var wg sync.WaitGroup
	var mu sync.Mutex
	failed := false
	binaries := map[string]string{}
	var buildErrs []string
	for i, p := range pkgs {
		if len(p.TestGoFiles)+len(p.XTestGoFiles) == 0 {
			fmt.Printf("?   \t%s\t[no test files]\n", p.ImportPath)
			continue
		}
		bin := filepath.Join(work, fmt.Sprintf("pkg%d.test", i))
		wg.Add(1)
		go func(p pkg) {
			defer wg.Done()
			out, err := exec.Command("go", "test", "-c", "-o", bin, p.ImportPath).CombinedOutput()
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				failed = true
				buildErrs = append(buildErrs, string(out)+fmt.Sprintf("FAIL\t%s [build failed]\n", p.ImportPath))
				return
			}
			binaries[p.ImportPath] = bin
		}(p)
	}
	// The short suite asks for a version too, so the build is made in either case, while
	// the tests compile.
	{
		stamped := filepath.Join(work, "stamped", "cogmer"+exeSuffix())
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, err := exec.Command("go", "build", "-ldflags", stampedBuildFlags, "-o", stamped, "./cmd/cogmer").CombinedOutput()
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				failed = true
				buildErrs = append(buildErrs, string(out)+"FAIL\tthe shared build of ./cmd/cogmer failed\n")
				return
			}
			env = append(env, "COGMER_TEST_BUILD="+stamped)
		}()
	}
	wg.Wait()
	for _, e := range buildErrs {
		fmt.Print(e)
	}

	last := lastTimes(timesFile)
	var times []timing
	for _, p := range pkgs {
		bin, ok := binaries[p.ImportPath]
		if !ok {
			continue
		}
		names, err := listTests(bin, p.Dir)
		if err != nil {
			return false, fmt.Errorf("%s: %v", p.ImportPath, err)
		}
		start := time.Now()
		groups := split(names, shards, func(n string) float64 {
			if s, ok := last[p.ImportPath+" "+n]; ok {
				return s
			}
			return 0.1
		})
		results := make([]shardResult, len(groups))
		var swg sync.WaitGroup
		for i, g := range groups {
			swg.Add(1)
			go func(i int, g []string) {
				defer swg.Done()
				results[i] = runShard(bin, p, g, short, env)
			}(i, g)
		}
		swg.Wait()
		pkgFailed := false
		for _, r := range results {
			fmt.Print(r.report)
			times = append(times, r.times...)
			if r.failed {
				pkgFailed = true
			}
		}
		if pkgFailed {
			failed = true
			fmt.Printf("FAIL\t%s\t%.3fs\n", p.ImportPath, time.Since(start).Seconds())
		} else {
			fmt.Printf("ok  \t%s\t%.3fs\n", p.ImportPath, time.Since(start).Seconds())
		}
	}

	sort.Slice(times, func(i, j int) bool { return times[i].seconds > times[j].seconds })
	var b strings.Builder
	fmt.Fprintln(&b, "# seconds each test took, slowest first; a subtest's time is also in its parent's")
	for _, t := range times {
		fmt.Fprintf(&b, "%8.2f  %s\n", t.seconds, t.name)
	}
	if err := os.WriteFile(timesFile, []byte(b.String()), 0o644); err != nil {
		return failed, err
	}
	return failed, nil
}

func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

// listTests names the package's top-level tests and examples, which are what -test.run
// selects.
func listTests(bin, dir string) ([]string, error) {
	cmd := exec.Command(bin, "-test.list", ".")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, n := range strings.Fields(string(out)) {
		if strings.HasPrefix(n, "Test") || strings.HasPrefix(n, "Example") {
			names = append(names, n)
		}
	}
	return names, nil
}

// lastTimes reads the times the last run wrote, keyed by package and test.
func lastTimes(file string) map[string]float64 {
	m := map[string]float64{}
	data, err := os.ReadFile(file)
	if err != nil {
		if data, err = os.ReadFile(filepath.Join(filepath.Dir(file), "latest")); err != nil {
			return m
		}
	}
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Fields(line)
		if len(f) != 3 || strings.HasPrefix(line, "#") {
			continue
		}
		var s float64
		if _, err := fmt.Sscanf(f[0], "%g", &s); err == nil {
			m[f[1]+" "+f[2]] = s
		}
	}
	return m
}

// split puts the slowest tests first, each into the group with the least time so far.
func split(names []string, n int, cost func(string) float64) [][]string {
	sorted := append([]string(nil), names...)
	sort.Slice(sorted, func(i, j int) bool { return cost(sorted[i]) > cost(sorted[j]) })
	if n > len(sorted) {
		n = len(sorted)
	}
	groups := make([][]string, n)
	total := make([]float64, n)
	for _, name := range sorted {
		k := 0
		for i := range total {
			if total[i] < total[k] {
				k = i
			}
		}
		groups[k] = append(groups[k], name)
		total[k] += cost(name)
	}
	return groups
}

type shardResult struct {
	report string
	times  []timing
	failed bool
}

// runShard runs one group of tests through test2json, and keeps what go test would print:
// a failed test's output, and a package's own output.
func runShard(bin string, p pkg, names []string, short bool, env []string) shardResult {
	args := []string{"tool", "test2json", "-t", "-p", p.ImportPath, bin,
		"-test.v=test2json", "-test.timeout=10m", "-test.run", "^(" + strings.Join(names, "|") + ")$"}
	if short {
		args = append(args, "-test.short")
	}
	cmd := exec.Command("go", args...)
	cmd.Dir = p.Dir
	cmd.Env = env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return shardResult{report: err.Error() + "\n", failed: true}
	}
	if err := cmd.Start(); err != nil {
		return shardResult{report: err.Error() + "\n", failed: true}
	}
	var r shardResult
	var report strings.Builder
	held := map[string][]string{}
	in := bufio.NewScanner(stdout)
	in.Buffer(make([]byte, 1024*1024), 64*1024*1024)
	for in.Scan() {
		var e event
		if err := json.Unmarshal(in.Bytes(), &e); err != nil {
			report.WriteString(in.Text() + "\n")
			continue
		}
		switch {
		case e.Test == "" && e.Action == "output":
			o := e.Output
			if !strings.HasPrefix(o, "=== ") && !strings.HasPrefix(o, "--- PASS") && o != "PASS\n" &&
				o != "FAIL\n" && !strings.HasPrefix(o, "ok  \t") && !strings.HasPrefix(o, "FAIL\t") {
				report.WriteString(o)
			}
		case e.Test == "" && e.Action == "fail":
			r.failed = true
		case e.Action == "output":
			held[e.Test] = append(held[e.Test], e.Output)
		case e.Test != "" && (e.Action == "pass" || e.Action == "fail" || e.Action == "skip"):
			if e.Action == "fail" {
				r.failed = true
				report.WriteString(strings.Join(held[e.Test], ""))
			}
			delete(held, e.Test)
			if e.Elapsed != nil && e.Action != "skip" {
				r.times = append(r.times, timing{*e.Elapsed, p.ImportPath + " " + e.Test})
			}
		}
	}
	if err := cmd.Wait(); err != nil {
		r.failed = true
		if stderr.Len() > 0 {
			report.WriteString(stderr.String())
		}
	}
	r.report = report.String()
	return r
}
