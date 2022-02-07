// nolint:gochecknoglobals,exhaustivestruct,gochecknoinits
package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.xsfx.dev/wg-quicker/wgquick"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var (
	cfgDir    string
	iface     string
	verbose   bool
	protocol  int
	metric    int
	userspace bool
)

var rootCmd = &cobra.Command{
	Use: "wg-quicker",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if verbose {
			logrus.SetLevel(logrus.DebugLevel)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.Help(); err != nil {
			logrus.Fatal(err)
		}
		os.Exit(0)
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version informations",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("wg-quicker %s, commit %s, build on %s\n", version, commit, date) // nolint: forbidigo
	},
}

var upCmd = &cobra.Command{
	Use:   "up [config_file|interface]",
	Short: "Bringing interface up",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c, log := loadConfig(args[0])
		if err := wgquick.Up(c, iface, userspace, log); err != nil {
			logrus.WithError(err).Errorln("cannot up interface")
		}
	},
}

var downCmd = &cobra.Command{
	Use:   "down [config_file|interface]",
	Short: "Bringing interface down",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c, log := loadConfig(args[0])
		if err := wgquick.Down(c, iface, log); err != nil {
			logrus.WithError(err).Errorln("cannot down interface")
		}
	},
}

var syncCmd = &cobra.Command{
	Use:   "sync [config_file|interface]",
	Short: "Sync interface",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c, log := loadConfig(args[0])
		if err := wgquick.Sync(c, iface, userspace, log); err != nil {
			logrus.WithError(err).Errorln("cannot sync interface")
		}
	},
}

var showCmd = &cobra.Command{
	Use:   "show [interface]",
	Short: "Show current configuration",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		device := ""
		if len(args) == 1 {
			device = args[0]
		}

		if err := wgquick.Show(device); err != nil {
			logrus.WithError(err).Errorln("cannot show configuration")
		}
	},
}

func loadConfig(cfg string) (*wgquick.Config, logrus.FieldLogger) { //nolint:ireturn
	log := logrus.WithField("iface", iface)
	_, err := os.Stat(cfg)

	switch {
	case err == nil:
	case os.IsNotExist(err):
		if iface == "" {
			iface = cfg
			log = logrus.WithField("iface", iface)
		}

		cfg = fmt.Sprintf("%s/%s.conf", cfgDir, cfg)

		_, err = os.Stat(cfg)
		if err != nil {
			log.WithError(err).Errorln("cannot find config file")
		}
	default:
		logrus.WithError(err).Errorln("error while reading config file")
	}

	b, err := ioutil.ReadFile(cfg)
	if err != nil {
		logrus.WithError(err).Fatalln("cannot read file")
	}

	c := &wgquick.Config{}

	if err := c.UnmarshalText(b); err != nil {
		logrus.WithError(err).Fatalln("cannot parse config file")
	}

	c.RouteProtocol = protocol
	c.RouteMetric = metric

	return c, log
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgDir, "config-dir", "", "config directory (default is /etc/wireguard)")
	rootCmd.PersistentFlags().StringVarP(&iface, "iface", "i", "", "if interface name should differ from config")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose")
	rootCmd.PersistentFlags().IntVarP(&protocol, "route-protocol", "p", 0, "route protocol to use for our routes")
	rootCmd.PersistentFlags().IntVarP(&metric, "route-metric", "m", 0, "route metric to use for our routes")
	rootCmd.PersistentFlags().BoolVarP(
		&userspace,
		"userspace", "u",
		false,
		"enforce userspace implementation of wireguard",
	)
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(showCmd)
}

func initConfig() {
	if cfgDir == "" {
		cfgDir = "/etc/wireguard"
	}

	cfgDir = strings.TrimSuffix(cfgDir, "/")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}
