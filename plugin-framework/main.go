package main

import (
	"context"
	"fmt"
	"os"
	"time"

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
)

var (
	app = kingpin.New("plugin-framework", pluginFrameworkName)

	test = app.Command("test", "Run tests")

	fixtures = app.Command("fixtures", "Fixtures")

	generateFixture = fixtures.Command("generate", "Generate fixture")
	fixtureTemplate = generateFixture.Arg("template", "Fixture template name (use `fixtures list-templates` to get a name)").Required().String()
	fixtureName     = generateFixture.Arg("name", "Fixture name").Required().String()

	listTemplates = fixtures.Command("list-templates", "List fixture templates")
)

// runTests runs tests
func runTests(log logrus.FieldLogger) {
	ctx := context.Background()

	b, err := os.ReadFile(testWasm)
	if err != nil {
		log.Fatal(err)
	}

	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		log.Fatal(err)
	}

	asEnv := wasm.NewAssemblyScriptEnv(log)
	store := wasm.NewStore(wasm.NewBadgerPersistentStore(db), wasm.DecodeAssemblyScriptString)
	testRunner, err := wasm.NewTesting(fixturesDir)
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}

	c, err := pool.Get(ctx)
	if err != nil {
		log.Fatal(err)
	}

	ec := testRunner.For(c)
	if ec == nil {
		log.Fatal(trace.Errorf("testRunner context not found"))
	}

	err = ec.Run()
	if err != nil {
		log.Fatal(err)
	}
}

// genFixture generates fixture
func genFixture(log logrus.FieldLogger, template string, name string) {
	fmt.Println(pluginFrameworkName)
	fmt.Println()

	index, err := wasm.NewFixtureIndex(fixturesDir)
	if err != nil {
		log.Fatal(err)
	}

	f, err := index.Add(template, name)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Fixture generated: %v\n", f)
}

// printTemplates prints a list of defined fixture templates
func printTemplates(log logrus.FieldLogger) {
	fmt.Println(pluginFrameworkName)
	fmt.Println()

	b, err := wasm.NewTemplateBuilder()
	if err != nil {
		log.Fatal(err)
	}

	templates, err := b.All()
	if err != nil {
		log.Fatal(err)
	}

	for _, t := range templates {
		fmt.Printf("%v - %v (%v)\n", fmt.Sprintf("%v fixtures generate %v <name>", os.Args[0], t.Name), t.Description, t.Type)
	}
}

func main() {
	logger.Init()
	log := logger.Standard()

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case test.FullCommand():
		runTests(log)
	case generateFixture.FullCommand():
		genFixture(log, *fixtureTemplate, *fixtureName)
	case listTemplates.FullCommand():
		printTemplates(log)
	}
}
