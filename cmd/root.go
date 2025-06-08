package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	FlagAccessKeyID     = "access-key-id"
	FlagAddress         = "address"
	FlagBucket          = "bucket"
	FlagBufferSize      = "buffer-size"
	FlagKey             = "key"
	FlagS3Host          = "s3-host"
	FlagSecretAccessKey = "secret-access-key"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "awsfiles",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		setSlogDebug(cmd)
		mergeCobraAndViper(cmd)
		var cfg Config
		err := viper.Unmarshal(&cfg)
		if err != nil && cmd.Flag("debug").Value.String() == "true" {
			slog.Error("Error binding flag", "error", err)
		}
		if isDebug(cmd) {
			slog.Debug("variable:", "config", fmt.Sprintf("%+v", cfg))
		}
		for key, value := range viper.GetViper().AllSettings() {
			slog.Info("variable:", key, value)
		}
		getObject(cfg)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "Config", "", "Config file (default is $HOME/.awsfiles.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("debug", "d", false, "Help message for debug")
	rootCmd.PersistentFlags().String(FlagAccessKeyID, "", "AccessKeyID")
	rootCmd.PersistentFlags().String(FlagSecretAccessKey, "", "SecretAccessKey")
	rootCmd.PersistentFlags().String(FlagAddress, "", "Address as host:port")
	rootCmd.PersistentFlags().String(FlagS3Host, "", "S3 host URL")
	rootCmd.PersistentFlags().String(FlagBucket, "", "Bucket")
	rootCmd.PersistentFlags().String(FlagKey, "", "Key")
	rootCmd.PersistentFlags().Int(FlagBufferSize, 16384, "Buffers size")
}

// initConfig reads in Config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use Config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search Config in home directory with name ".awsfiles" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".awsfiles")
	}
	viper.AutomaticEnv() // read in environment variables that match

	// If a Config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using Config file:", viper.ConfigFileUsed())
	}
}

func setSlogDebug(cmd *cobra.Command) {
	if isDebug(cmd) {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
}

func mergeCobraAndViper(cmd *cobra.Command) {
	for key, _ := range viper.GetViper().AllSettings() {
		viperBindPFlag(key, SnakeCaseToKebabCase(key), cmd)
	}
	cmd.Flags().VisitAll(mergeCobraAndViperFunc(cmd))
}

func mergeCobraAndViperFunc(cmd *cobra.Command) func(f *pflag.Flag) {
	return func(f *pflag.Flag) {
		if isDebug(cmd) {
			slog.Debug(
				"variable:", "flag",
				fmt.Sprintf("FLAG: (Type: %s, Changed: %v) --%s=%q",
					f.Value.Type(), f.Changed, f.Name, f.Value.String(),
				))
		}
		if f.Changed || viper.Get(KebabCaseToSnakeCase(f.Name)) == nil {
			switch f.Value.Type() {
			case "bool":
				viper.Set(KebabCaseToSnakeCase(f.Name), f.Value.String() == "true")
			case "int":
				value, _ := strconv.Atoi(f.Value.String())
				if value > 0 {
					viper.Set(KebabCaseToSnakeCase(f.Name), value)
				} else {
					value, _ = strconv.Atoi(f.DefValue)
					viper.Set(KebabCaseToSnakeCase(f.Name), value)
				}
			case "string":
				viper.Set(KebabCaseToSnakeCase(f.Name), f.Value.String())
			}
		}
	}
}

func viperBindPFlag(key, flag string, cmd *cobra.Command) {
	err := viper.BindPFlag(key, cmd.Flags().Lookup(flag))
	if err != nil && isDebug(cmd) {
		slog.Error("Error binding flag", "error", err)
	}
}

func isDebug(cmd *cobra.Command) bool {
	return cmd.Flag("debug").Value.String() == "true"
}

func KebabCaseToSnakeCase(s string) string {
	result := strings.ReplaceAll(s, "-", "_")
	return strings.ToLower(result)
}

func SnakeCaseToKebabCase(s string) string {
	result := strings.ReplaceAll(s, "_", "-")
	return strings.ToLower(result)
}
