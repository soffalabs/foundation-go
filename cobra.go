package soffa

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

func CreateAppCli(name string, description string, initializer AppCreator) {
	cobra.OnInitialize()

	var rootCmd = &cobra.Command{
		Use:   name,
		Short: description,
	}

	rootCmd.AddCommand(createServerCmd(initializer))
	rootCmd.AddCommand(createDbCommand(initializer))
	
	_ = rootCmd.Execute()
}

type AppCreator = func(env string, configSource string, router bool) *App

func createServerCmd(initializer AppCreator) *cobra.Command {
	var port int
	var configSource string
	var dbMigrations bool
	var env string

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the service in server mode",
		Run: func(cmd *cobra.Command, args []string) {
			app := initializer(env, configSource, true)
			if dbMigrations {
				app.ApplyMigrations()
			} else {
				log.Info("database migrations were skipped")
			}
			app.Start(port)
		},
	}
	cmd.Flags().StringVarP(&env, "env", "e", Getenv(os.Getenv("ENV"), "dev", true), "active environment profile")
	cmd.Flags().StringVarP(&configSource, "config", "c", os.Getenv("CONFIG_SOURCE"), "config source")
	cmd.Flags().IntVarP(&port, "port", "p", Getenvi("PORT", 8080), "server port")
	cmd.Flags().BoolVarP(&dbMigrations, "db-migrations", "m", Getenvb("DB_MIGRATIONS", true), "apply database migrations")

	return cmd
}

func createDbCommand(initializer AppCreator) *cobra.Command {
	var configSource string
	var env string

	cmd := &cobra.Command{
		Use:   "db:migrate",
		Short: "Run database migrations",
		Run: func(cmd *cobra.Command, args []string) {
			app := initializer(env, configSource, false)
			app.ApplyMigrations()
		},
	}
	cmd.Flags().StringVarP(&configSource, "config", "c", os.Getenv("CONFIG_SOURCE"), "config source")
	cmd.Flags().StringVarP(&env, "env", "e", Getenv(os.Getenv("ENV"), "dev", true), "active environment profile")
	
	return cmd
}

