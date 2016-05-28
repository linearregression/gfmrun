package gfmxr

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/Sirupsen/logrus"
	"gopkg.in/urfave/cli.v2"
)

func NewCLI() *cli.App {
	app := cli.NewApp()
	app.Name = "gfmxr"
	app.Usage = "github-flavored markdown example runner"
	app.Authors = []*cli.Author{
		{
			Name:  "Dan Buch",
			Email: "dan@meatballhat.com",
		},
	}
	app.Version = VersionString
	app.Flags = []cli.Flag{
		&cli.StringSliceFlag{
			Name:    "sources",
			Aliases: []string{"s"},
			Usage:   "markdown source(s) to search for runnable examples",
			Value:   cli.NewStringSlice("README.md"),
			EnvVars: []string{"GFMXR_SOURCES", "SOURCES"},
		},
		&cli.IntFlag{
			Name:    "count",
			Aliases: []string{"c"},
			Usage:   "expected count of runnable examples (for verification)",
			EnvVars: []string{"GFMXR_COUNT", "COUNT"},
		},
		&cli.StringFlag{
			Name:    "languages",
			Aliases: []string{"L"},
			Usage:   "location of languages.yml file from linguist",
			Value:   DefaultLanguagesYml,
			EnvVars: []string{"GFMXR_LANGUAGES", "LANGUAGES"},
		},
		&cli.BoolFlag{
			Name:    "no-auto-pull",
			Aliases: []string{"N"},
			Value:   true,
			Usage:   "disable automatic pull of languages.yml when missing",
			EnvVars: []string{"GFMXR_NO_AUTO_PULL", "NO_AUTO_PULL"},
		},
		&cli.BoolFlag{
			Name:    "debug",
			Aliases: []string{"D"},
			Usage:   "show debug output",
			EnvVars: []string{"GFMXR_DEBUG", "DEBUG"},
		},
	}

	app.Commands = []*cli.Command{
		{
			Name:  "pull-languages",
			Usage: "explicitly download the latest languages.yml from the linguist source to $GFMXR_LANGUAGES (automatic unless \"--no-auto-pull\")",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "languages-url",
					Aliases: []string{"u"},
					Usage:   "source URL of languages.yml file from linguist",
					Value:   DefaultLanguagesYmlURL,
					EnvVars: []string{"GFMXR_LANGUAGES_URL", "LANGUAGES_URL"},
				},
			},
			Action: cliPullLanguages,
		},
		{
			Name:   "dump-languages",
			Usage:  "dump the parsed languages data structure as JSON",
			Hidden: true,
			Action: cliDumpLanguages,
		},
		{
			Name:   "list-frobs",
			Usage:  "list the known frobs and handled frob aliases",
			Hidden: true,
			Action: cliListFrobs,
		},
	}

	app.Action = cliRunExamples

	return app
}

func RunExamples(sources []string, expectedCount int, languagesFile string, autoPull bool, log *logrus.Logger) error {
	if sources == nil {
		sources = []string{}
	}

	if log == nil {
		log = logrus.New()
	}

	runner, err := NewRunner(sources, expectedCount, languagesFile, autoPull, log)
	if err != nil {
		return err
	}

	errs := runner.Run()

	if len(errs) > 0 {
		return multiError(errs)
	}

	return nil
}

func cliRunExamples(ctx *cli.Context) error {
	log := logrus.New()
	if ctx.Bool("debug") {
		log.Level = logrus.DebugLevel
	}

	err := RunExamples(ctx.StringSlice("sources"), ctx.Int("count"),
		ctx.String("languages"), ctx.Bool("no-auto-pull"), log)

	if err != nil {
		log.Error(err)
		return multiError([]error{err, cli.Exit("", 2)})
	}

	return nil
}

func cliListFrobs(ctx *cli.Context) error {
	langs, err := LoadLanguages(ctx.String("languages"))
	if err != nil {
		return err
	}

	known := map[string]bool{}

	for name, _ := range DefaultFrobs {
		for _, alias := range langs.Lookup(name).Aliases {
			known[alias] = true
		}
	}

	knownSlice := []string{}
	for lang := range known {
		knownSlice = append(knownSlice, lang)
	}

	sort.Strings(knownSlice)

	for _, lang := range knownSlice {
		fmt.Printf("%s\n", lang)
	}

	return nil
}

func cliDumpLanguages(ctx *cli.Context) error {
	langs, err := LoadLanguages(ctx.String("languages"))
	if err != nil {
		return multiError([]error{cli.Exit("failed to load languages", 4), err})
	}

	jsonBytes, err := json.MarshalIndent(langs.Map, "", "  ")
	if err != nil {
		return multiError([]error{cli.Exit("failed to marshal to json", 4), err})
	}

	fmt.Printf(string(jsonBytes) + "\n")
	return nil
}

func cliPullLanguages(ctx *cli.Context) error {
	return PullLanguagesYml(ctx.String("languages-url"), ctx.String("languages"))
}