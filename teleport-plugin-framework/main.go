package main

import (
	"context"
	"embed"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/dgraph-io/badger/v3"
	"github.com/gravitational/kingpin"
	"github.com/gravitational/teleport-plugins/lib/logger"
	"github.com/gravitational/teleport-plugins/lib/wasm"
	_ "github.com/gravitational/teleport/api/types"
	"github.com/gravitational/trace"
	"github.com/sirupsen/logrus"
)

const (
	defaultConcurrency = 1
	defaultTimeout     = time.Second * 30
	fixturesDir        = "fixtures"
	testWasm           = "build/test.wasm"
)

var (
	pluginFrameworkName = fmt.Sprintf("WASM plugin framework app %v", Version)

	app = kingpin.New("plugin-framework", pluginFrameworkName)

	new    = app.Command("new", "Generate plugin boilerplate")
	newDir = new.Arg("dir", "Path to the new plugin").String()

	test = app.Command("test", "Run tests")

	fixtures = app.Command("fixtures", "Fixtures")

	generateFixture = fixtures.Command("generate", "Generate fixture")
	fixtureTemplate = generateFixture.Arg("template", "Fixture template name (use `fixtures list-templates` to get a name)").Required().String()
	fixtureName     = generateFixture.Arg("name", "Fixture name").Required().String()

	listTemplates = fixtures.Command("list-templates", "List fixture templates")

	//go:embed assembly/*
	//go:embed vendor/*
	//go:embed boilerplate/*
	//go:embed fixtures/*
	//go:embed .eslintrc.json
	//go:embed .prettierrc
	fs embed.FS

	// perms represents permissions for target directory
	perms os.FileMode = 0777

	// nodeVersionGte minimal node version
	nodeVersionGte = "16.13"
	// nodeVersionLt max node version
	nodeVersionLt = "17"
)

// runTests runs tests
func runTests(log logrus.FieldLogger) {
	ctx := context.Background()

	b, err := os.ReadFile(testWasm)
	if err != nil {
		log.Fatal(trace.Wrap(err))
	}

	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		log.Fatal(trace.Wrap(err))
	}

	asEnv := wasm.NewAssemblyScriptEnv(log)
	store := wasm.NewStore(wasm.NewBadgerPersistentStore(db), wasm.DecodeAssemblyScriptString)
	testRunner, err := wasm.NewTesting(fixturesDir)
	if err != nil {
		log.Fatal(trace.Wrap(err))
	}
	pb := wasm.NewProtobufInterop()
	api := wasm.NewTeleportAPI(log, testRunner.MockAPIClient, pb)

	opts := wasm.ExecutionContextPoolOptions{
		Timeout:     defaultTimeout,
		Concurrency: defaultConcurrency,
		Bytes:       b,
	}

	pool, err := wasm.NewExecutionContextPool(opts, asEnv, testRunner, store, api, pb)
	if err != nil {
		log.Fatal(trace.Wrap(err))
	}

	c, err := pool.Get(ctx)
	if err != nil {
		log.Fatal(trace.Wrap(err))
	}

	ec := testRunner.For(c)
	if ec == nil {
		log.Fatal(trace.Errorf("testRunner context not found"))
	}

	err = ec.Run()
	if err != nil {
		log.Fatal(trace.Wrap(err))
	}
}

// genFixture generates fixture
func genFixture(log logrus.FieldLogger, template string, name string) {
	fmt.Println(pluginFrameworkName)
	fmt.Println()

	index, err := wasm.NewFixtureIndex(fixturesDir)
	if err != nil {
		log.Fatal(trace.Wrap(err))
	}

	f, err := index.Add(template, name)
	if err != nil {
		log.Fatal(trace.Wrap(err))
	}

	fmt.Printf("Fixture generated: %v\n", f)
}

// printTemplateList prints a list of defined fixture templates
func printTemplateList(log logrus.FieldLogger) {
	fmt.Println(pluginFrameworkName)
	fmt.Println()

	b, err := wasm.NewTemplateBuilder()
	if err != nil {
		log.Fatal(trace.Wrap(err))
	}

	templates, err := b.All()
	if err != nil {
		log.Fatal(trace.Wrap(err))
	}

	for _, t := range templates {
		fmt.Printf("%v - %v (%v)\n", fmt.Sprintf("%v fixtures generate %v <name>", os.Args[0], t.Name), t.Description, t.Type)
	}
}

// generateBoilerplate generates boilerplate setup for plugin framework
func generateBoilerplate(dir string) {
	fmt.Println(pluginFrameworkName)
	fmt.Println()

	fmt.Println("[1] Checking node & yarn versions...")

	// Check node version
	out, err := exec.Command("node", "--version").Output()
	if err != nil {
		log.Fatalf("node executable is not found in the system %v : %v", out, trace.Wrap(err))
	}

	ver := strings.TrimSpace(string(out))

	s := fmt.Sprintf("    - Found node %v", ver)
	if !checkVersion(ver, nodeVersionGte, nodeVersionLt) {
		s = s + fmt.Sprintf(". node version is invalid: must be >= %v < %v. Please, use https://github.com/nvm-sh/nvm to install specific version.", nodeVersionGte, nodeVersionLt)
	}

	fmt.Println(s)

	// Check yarn version
	out, err = exec.Command("yarn", "-version").Output()
	if err != nil {
		log.Fatalf("yarn executable is not found in the system %v : %v", out, trace.Wrap(err))
	}

	fmt.Printf("    - Found yarn %v", string(out))

	// Create target directories
	fmt.Printf("[2] Creating target directory structure: %v, %v/assembly, %v/vendor\n", dir, dir, dir)

	_, err = os.Stat(dir)
	if !os.IsNotExist(err) {
		log.Fatalf("Error: target directory exists. Please specify the new path which does not exist.\n%v", trace.Wrap(err))
	}

	err = os.MkdirAll(dir, perms)
	if err != nil {
		log.Fatalf("Error: failed to create target directory. %v", trace.Wrap(err))
	}

	err = os.MkdirAll(path.Join(dir, "assembly"), perms)
	if err != nil {
		log.Fatalf("Error: failed to create target directory. %v", trace.Wrap(err))
	}

	err = os.MkdirAll(path.Join(dir, "vendor"), perms)
	if err != nil {
		log.Fatalf("Error: failed to create target directory. %v", trace.Wrap(err))
	}

	err = os.MkdirAll(path.Join(dir, "fixtures"), perms)
	if err != nil {
		log.Fatalf("Error: failed to create target directory. %v", trace.Wrap(err))
	}

	// Write target files
	fmt.Printf("[3] Writing template files\n")

	writeDirFromFS(dir, "assembly")
	writeDirFromFS(dir, "vendor")
	writeDirFromFS(dir, "fixtures")

	writeFileFromFS(dir, "boilerplate/package.json", "package.json")
	writeFileFromFS(dir, "boilerplate/asconfig.json", "asconfig.json")
	writeFileFromFS(dir, "boilerplate/tsconfig.json", "tsconfig.json")
	writeFileFromFS(dir, ".prettierrc", ".prettierrc")
	writeFileFromFS(dir, ".eslintrc.json", ".eslintrc.json")
	writeFileFromFS(dir, "boilerplate/gitignore", ".gitignore")

	// Yarn install in target dir
	fmt.Printf("[4] Running yarn install in %v...\n\n", dir)

	cmd := exec.Command("yarn")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()

	fmt.Println()

	fmt.Printf("[5] Copying terraform-plugin-framework binary %v.\n\n", dir)

	targetBinary := path.Join(dir, path.Base(os.Args[0]))
	err = cp(os.Args[0], targetBinary)
	if err != nil {
		log.Fatal(trace.Wrap(err))
	}

	err = os.Chmod(targetBinary, perms)
	if err != nil {
		log.Fatal(trace.Wrap(err))
	}

	fmt.Printf("Now you can run: cd %v && yarn test\n\n", dir)

	// TODO: Replace with the actual docs
	fmt.Println("Please, check Teleport Plugin Framework documentation: https://goteleport.com/docs")

	fmt.Printf("Happy hacking!")

	fmt.Println()
}

// cp copies src file to dst
func cp(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

// checkVersion checks that a version is within the bounds
func checkVersion(version, gte, lt string) bool {
	cgte, err := semver.NewConstraint(fmt.Sprintf(">= %v", gte))
	if err != nil {
		log.Fatal(trace.Wrap(err))
	}

	clt, err := semver.NewConstraint(fmt.Sprintf("< %v", lt))
	if err != nil {
		log.Fatal(trace.Wrap(err))
	}

	v, err := semver.NewVersion(strings.TrimSpace(version))
	if err != nil {
		log.Fatal(trace.Wrap(err))
	}

	return cgte.Check(v) && clt.Check(v)
}

// writeDirFromFS copies directory from embedded fs to target directory on disk
func writeDirFromFS(dir string, sub string) {
	files, err := fs.ReadDir(sub)
	if err != nil {
		log.Fatal(trace.Wrap(err))
	}

	for _, f := range files {
		writeFileFromFS(dir, path.Join(sub, f.Name()), path.Join(sub, f.Name()))
	}
}

// writeFileFromFS copies file from embedded fs to target directory on disk
func writeFileFromFS(dir string, name string, targetName string) {
	b, err := fs.ReadFile(name)
	if err != nil {
		log.Fatal(trace.Wrap(err))
	}

	err = os.WriteFile(path.Join(dir, targetName), b, perms)
	if err != nil {
		log.Fatalf("Error writing target file: %v : %v", name, trace.Wrap(err))
	}
}

func main() {
	logger.Init()
	log := logger.Standard()

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case new.FullCommand():
		generateBoilerplate(*newDir)
	case test.FullCommand():
		runTests(log)
	case generateFixture.FullCommand():
		genFixture(log, *fixtureTemplate, *fixtureName)
	case listTemplates.FullCommand():
		printTemplateList(log)
	}
}
