// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package app

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	cliflag "github.com/marmotedu/component-base/pkg/cli/flag"
	"github.com/marmotedu/component-base/pkg/cli/globalflag"
	"github.com/marmotedu/component-base/pkg/term"
	"github.com/marmotedu/component-base/pkg/version"
	"github.com/marmotedu/component-base/pkg/version/verflag"
	"github.com/marmotedu/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/marmotedu/iam/pkg/log"
)

var (
	progressMessage = color.GreenString("==>")

	usageTemplate = fmt.Sprintf(`%s{{if .Runnable}}
  %s{{end}}{{if .HasAvailableSubCommands}}
  %s{{end}}{{if gt (len .Aliases) 0}}

%s
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

%s
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

%s{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  %s {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

%s
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

%s
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

%s{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "%s --help" for more information about a command.{{end}}
`,
		color.CyanString("Usage:"),
		color.GreenString("{{.UseLine}}"),
		color.GreenString("{{.CommandPath}} [command]"),
		color.CyanString("Aliases:"),
		color.CyanString("Examples:"),
		color.CyanString("Available Commands:"),
		color.GreenString("{{rpad .Name .NamePadding }}"),
		color.CyanString("Flags:"),
		color.CyanString("Global Flags:"),
		color.CyanString("Additional help topics:"),
		color.GreenString("{{.CommandPath}} [command]"),
	)
)

// App is the main structure of a cli application.
// It is recommended that an app be created with the app.NewApp() function.
// App是cli应用的主要结构体
type App struct {
	basename    string //
	name        string // 应用名称
	description string // 应用描述
	options     CliOptions
	runFunc     RunFunc // 定义启用时的callback
	silence     bool
	noVersion   bool
	noConfig    bool
	commands    []*Command
	args        cobra.PositionalArgs // 位置参数
	cmd         *cobra.Command       // 应用的命令行
}

// Option defines optional parameters for initializing the application
// structure.
// 选项模式，定义匿名函数用来初始化应用的可选参数
type Option func(*App)

// WithOptions to open the application's function to read from the command line
// or read parameters from the configuration file.
// 打开应用的函数去读取命令行参数或者配置文件的参数
func WithOptions(opt CliOptions) Option {
	return func(a *App) { // 匿名函数的函数体是设置App的字段值
		a.options = opt
	}
}

// RunFunc defines the application's startup callback function.
// RunFunc 定义了应用运行时的callback函数
type RunFunc func(basename string) error

// WithRunFunc is used to set the application startup callback function option.
// WithRunFunc 用来设置应用启动时的callback函数选项
func WithRunFunc(run RunFunc) Option {
	return func(a *App) {
		a.runFunc = run
	}
}

// WithDescription is used to set the description of the application.
// WithDescription 用来设置app的描述信息
func WithDescription(desc string) Option {
	return func(a *App) {
		a.description = desc
	}
}

// WithSilence sets the application to silent mode, in which the program startup
// information, configuration information, and version information are not
// printed in the console.
// WithSilence设置app为静默模式，在该模式下程序启动信息、配置信息以及版本信息不会打印到console
func WithSilence() Option {
	return func(a *App) {
		a.silence = true
	}
}

// WithNoVersion set the application does not provide version flag.
// WithNoVersion设置app不提供版本的flag
func WithNoVersion() Option {
	return func(a *App) {
		a.noVersion = true
	}
}

// WithNoConfig set the application does not provide config flag.
// WithNoConfig 设置app不提供config的flag
func WithNoConfig() Option {
	return func(a *App) {
		a.noConfig = true
	}
}

// WithValidArgs set the validation function to valid non-flag arguments.
// WithValidArgs设置验证函数去验证非标签函数
func WithValidArgs(args cobra.PositionalArgs) Option {
	return func(a *App) {
		a.args = args
	}
}

// WithDefaultValidArgs set default validation function to valid non-flag arguments.
// WithDefaultValidArgs设置默认的验证函数去验证位置参数
func WithDefaultValidArgs() Option {
	return func(a *App) {
		a.args = func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				if len(arg) > 0 { // 确保参数为空，不为空则返回错误
					return fmt.Errorf("%q does not take any arguments, got %q", cmd.CommandPath(), args)
				}
			}

			return nil
		}
	}
}

// NewApp creates a new application instance based on the given application name,
// binary name, and other options.
// 创建一个应用实例，包含应用名称及其它选项。
func NewApp(name string, basename string, opts ...Option) *App {
	a := &App{ // 设置基本应用配置
		name:     name,
		basename: basename,
	}

	// 根据Options设置应用配置
	for _, o := range opts {
		o(a)
	}

	a.buildCommand() // 构建app的命令行参数

	return a
}

// 创建应用的命令行参数
func (a *App) buildCommand() {
	cmd := cobra.Command{
		Use:   FormatBaseName(a.basename), // usage中的一行使用信息，basename在linux下可以不用做处理
		Short: a.name,                     // help输出的简短描述
		Long:  a.description,              // `help command`输出的详细描述
		// stop printing usage when the command errors
		SilenceUsage:  true,   // 发生错误时不打印usage
		SilenceErrors: true,   // 静默下游错误
		Args:          a.args, // 位置参数
	}
	// cmd.SetUsageTemplate(usageTemplate)
	cmd.SetOut(os.Stdout)          // 设置usage的输出位置
	cmd.SetErr(os.Stderr)          // 错误信息的输出位置
	cmd.Flags().SortFlags = true   // 为help/usage信息中的flags排序
	cliflag.InitFlags(cmd.Flags()) // 初始化pflag.FlagSet，添加格式化函数对传入的flag的格式进行转换，把`_`转换成`-`,把标准库flag的FlagSet添加到pflag中的FlagSet中

	if len(a.commands) > 0 { // 如果app中含有子command则添加到cmd中
		for _, command := range a.commands {
			cmd.AddCommand(command.cobraCommand())
		}
		cmd.SetHelpCommand(helpCommand(FormatBaseName(a.basename))) // 设置help信息的Command
	}
	if a.runFunc != nil { // 设置运行函数，设置RunE的步骤可以放在return前更容易理解
		cmd.RunE = a.runCommand
	}

	var namedFlagSets cliflag.NamedFlagSets
	if a.options != nil { // 将Options配置的Flag添加到FlagSet中
		namedFlagSets = a.options.Flags()
		fs := cmd.Flags()
		for _, f := range namedFlagSets.FlagSets {
			fs.AddFlagSet(f)
		}
	}

	if !a.noVersion { // 添加version相关的Flag到global FlagSet中
		verflag.AddFlags(namedFlagSets.FlagSet("global"))
	}
	if !a.noConfig { // 从配置文件中读取配置，添加config相关的Flag到global FlagSet中
		addConfigFlag(a.basename, namedFlagSets.FlagSet("global"))
	}
	globalflag.AddGlobalFlags(namedFlagSets.FlagSet("global"), cmd.Name())
	// add new global flagset to cmd FlagSet
	cmd.Flags().AddFlagSet(namedFlagSets.FlagSet("global")) // 添加global FlagSet到 cmd FlagSet中

	addCmdTemplate(&cmd, namedFlagSets) // 设置cmd的help/usage
	a.cmd = &cmd
}

// Run is used to launch the application.
func (a *App) Run() {
	if err := a.cmd.Execute(); err != nil {
		fmt.Printf("%v %v\n", color.RedString("Error:"), err)
		os.Exit(1)
	}
}

// Command returns cobra command instance inside the application.
func (a *App) Command() *cobra.Command {
	return a.cmd
}

// runCommand 运行app的Command命令
func (a *App) runCommand(cmd *cobra.Command, args []string) error {
	printWorkingDir()               // 打印工作目录
	cliflag.PrintFlags(cmd.Flags()) // 打印FlagSet中的所有Flag
	if !a.noVersion {               // 打印版本信息
		// display application version information
		verflag.PrintAndExitIfRequested()
	}

	if !a.noConfig {
		// 把FlagSet中的所有Flag绑定到配置中，每个Flag的long名称作为配置的key
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return err
		}
		// 把配置反序列化到options中
		if err := viper.Unmarshal(a.options); err != nil {
			return err
		}
	}

	if !a.silence {
		log.Infof("%v Starting %s ...", progressMessage, a.name)
		if !a.noVersion {
			log.Infof("%v Version: `%s`", progressMessage, version.Get().ToJSON())
		}
		if !a.noConfig {
			log.Infof("%v Config file used: `%s`", progressMessage, viper.ConfigFileUsed())
		}
	}
	if a.options != nil {
		if err := a.applyOptionRules(); err != nil {
			return err
		}
	}
	// run application
	if a.runFunc != nil {
		return a.runFunc(a.basename)
	}

	return nil
}

func (a *App) applyOptionRules() error {
	if completeableOptions, ok := a.options.(CompleteableOptions); ok {
		if err := completeableOptions.Complete(); err != nil {
			return err
		}
	}

	if errs := a.options.Validate(); len(errs) != 0 {
		return errors.NewAggregate(errs)
	}

	if printableOptions, ok := a.options.(PrintableOptions); ok && !a.silence {
		log.Infof("%v Config: `%s`", progressMessage, printableOptions.String())
	}

	return nil
}

func printWorkingDir() {
	wd, _ := os.Getwd()
	log.Infof("%v WorkingDir: %s", progressMessage, wd)
}

// 设置cmd的help/usage
func addCmdTemplate(cmd *cobra.Command, namedFlagSets cliflag.NamedFlagSets) {
	usageFmt := "Usage:\n  %s\n"
	cols, _, _ := term.TerminalSize(cmd.OutOrStdout())
	cmd.SetUsageFunc(func(cmd *cobra.Command) error { // 设置usage函数
		fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStderr(), namedFlagSets, cols)

		return nil
	})
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) { // 设置help函数
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
		cliflag.PrintSections(cmd.OutOrStdout(), namedFlagSets, cols)
	})
}
