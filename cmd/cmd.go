package cmd

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/urfave/cli"

	"github.com/go-spirit/spirit"
	"github.com/go-spirit/spirit/component"
	"github.com/go-spirit/spirit/doc"
)

var (
	DefaultCmd = newCmd()
)

type Cmd interface {
	App() *cli.App
	Init(opts ...Option) error
	Options() Options
}

func init() {
	rand.Seed(time.Now().Unix())
	help := cli.HelpPrinter

	cli.HelpPrinter = func(writer io.Writer, templ string, data interface{}) {
		help(writer, templ, data)
		os.Exit(0)
	}
}

type cmd struct {
	opts Options
	app  *cli.App
}

func newCmd(opts ...Option) Cmd {
	options := Options{}

	for _, o := range opts {
		o(&options)
	}

	cmd := new(cmd)
	cmd.opts = options
	cmd.app = cli.NewApp()
	cmd.app.Usage = ""
	cmd.app.Description = "go-spirit is a flow-based, actor-like message process framework"

	cmd.app.Commands = cli.Commands{
		&cli.Command{
			Name:  "list",
			Usage: "list what you want to known",
			Subcommands: cli.Commands{
				&cli.Command{
					Name:   "components",
					Action: cmd.components,
				},
			},
		},
		&cli.Command{
			Name:   "document",
			Usage:  "print document",
			Action: cmd.document,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "name",
					Usage: "component name",
				},
			},
		},
		&cli.Command{
			Name:   "run",
			Usage:  "run components",
			Action: cmd.run,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "config",
					Usage: "config file",
				},
				&cli.StringFlag{
					Name:  "config-provider,cp",
					Usage: "the config provider you want to use",
					Value: "hocon",
				},
			},
		},
	}

	if cmd.app.Version == "0.0.0" {
		cmd.app.HideVersion = true
	}

	return cmd
}

func (p *cmd) components(ctx *cli.Context) error {

	compDescribes := component.ListComponents()
	buf := bytes.NewBuffer(nil)

	for _, comp := range compDescribes {
		buf.WriteString(fmt.Sprintf("- %s: %s\n", comp.Name, comp.RegisterFunc))
	}

	fmt.Print(buf.String())

	return nil
}

func (p *cmd) document(ctx *cli.Context) (err error) {

	name := ctx.String("name")

	if len(name) == 0 {
		return
	}

	documenter, exist := doc.GetDocumenter(name)
	if !exist {
		err = fmt.Errorf("documenter of %s not exist", name)
		return
	}

	document := documenter.Document()

	docStr, err := document.JSON()

	if err != nil {
		return
	}

	fmt.Println(docStr)

	return
}

func (p *cmd) run(ctx *cli.Context) (err error) {

	conf := ctx.String("config")
	configProvider := ctx.String("config-provider")

	s, err := spirit.New(
		spirit.ConfigFile(conf, configProvider),
	)

	if err != nil {
		return
	}

	err = s.Run()

	return
}

func (p *cmd) App() *cli.App {
	return p.app
}

func (p *cmd) Options() Options {
	return p.opts
}

func (p *cmd) Init(opts ...Option) error {
	for _, o := range opts {
		o(&p.opts)
	}

	p.app.RunAndExitOnError()
	return nil
}

func DefaultOptions() Options {
	return DefaultCmd.Options()
}

func App() *cli.App {
	return DefaultCmd.App()
}

func Init(opts ...Option) error {
	return DefaultCmd.Init(opts...)
}

func NewCmd(opts ...Option) Cmd {
	return newCmd(opts...)
}
