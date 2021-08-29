package cli

import (
	"github.com/soffa-io/soffa-core-go"
	"github.com/soffa-io/soffa-core-go/h"
	"github.com/soffa-io/soffa-core-go/log"
	"github.com/spf13/cobra"
	"net"
	"os"
)

func Execute(name string, version string, createApp func(env string) *soffa.App) {
	cobra.OnInitialize()
	var rootCmd = &cobra.Command{
		Use:     name,
		Version: version,
	}
	rootCmd.AddCommand(createServerCmd(createApp))
	rootCmd.AddCommand(createDbCommand(createApp))
	_ = rootCmd.Execute()
}

func createServerCmd(createApp func(env string) *soffa.App) *cobra.Command {
	var port int
	var randomPort bool
	var dbMigrations bool
	var envName string

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the service in server mode",
		Run: func(cmd *cobra.Command, args []string) {
			app := createApp(envName)
			if dbMigrations {
				app.MigrateDB()
			} else {
				log.Info("database migrations were skipped")
			}
			if randomPort {
				addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
				log.FatalIf(err)
				l, err := net.ListenTCP("tcp", addr)
				log.FatalIf(err)
				defer func(l *net.TCPListener) {
					_ = l.Close()
				}(l)
				port = l.Addr().(*net.TCPAddr).Port
			}
			app.Start(port)
		},
	}
	cmd.Flags().StringVarP(&envName, "env", "e", h.Getenv("ENV", "prod"), "active environment profile")
	cmd.Flags().IntVarP(&port, "port", "p", h.Getenvi("PORT", 8080), "server port")
	cmd.Flags().BoolVarP(&randomPort, "random-port", "r", false, "start the server on a random available port")
	cmd.Flags().BoolVarP(&dbMigrations, "persistence-migrations", "m", h.Getenvb("DB_MIGRATIONS", true), "apply database migrations")

	return cmd
}

func createDbCommand(createApp func(env string) *soffa.App) *cobra.Command {
	var configSource string
	var envName string

	cmd := &cobra.Command{
		Use:   "persistence:migrate",
		Short: "Run database migrations",
		Run: func(cmd *cobra.Command, args []string) {
			app := createApp(envName)
			app.MigrateDB()
		},
	}
	cmd.Flags().StringVarP(&configSource, "config", "c", os.Getenv("CONFIG_SOURCE"), "config source")
	cmd.Flags().StringVarP(&envName, "env", "e", h.Getenv("ENV", "prod"), "active environment profile")

	return cmd
}
