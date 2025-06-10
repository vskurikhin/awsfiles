package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	FlagAccessKeyID        = "access-key-id"
	FlagAddress            = "address"
	FlagBucket             = "bucket"
	FlagBufferSize         = "buffer-size"
	FlagCAFile             = "ca-file"
	FlagDebug              = "debug"
	FlagInsecureSkipVerify = "insecure-skip-verify"
	FlagKey                = "key"
	FlagS3Host             = "s3-host"
	FlagSecretAccessKey    = "secret-access-key"
	FlagServerName         = "server-name"
	FlagVerbose            = "verbose"
)

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "Config", "", "Config file (default is $HOME/.awsfiles.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.PersistentFlags().Bool(FlagInsecureSkipVerify, false, "Controls whether a client verifies the server's certificate chain and host name.")
	rootCmd.PersistentFlags().BoolP(FlagDebug, "d", false, "Help message for debug")
	rootCmd.PersistentFlags().BoolP(FlagVerbose, "v", false, "Verbose")

	rootCmd.PersistentFlags().Int(FlagBufferSize, 16384, "Buffers size")

	rootCmd.PersistentFlags().String(FlagAddress, "", "Address as host:port")
	rootCmd.PersistentFlags().String(FlagCAFile, "", "CA File")
	rootCmd.PersistentFlags().String(FlagServerName, "", "TLS servername")

	rootCmd.PersistentFlags().StringP(FlagAccessKeyID, "a", "", "AccessKeyID")
	rootCmd.PersistentFlags().StringP(FlagS3Host, "u", "", "S3 host URL")
	rootCmd.PersistentFlags().StringP(FlagSecretAccessKey, "s", "", "SecretAccessKey")

	getObjectCmd.Flags().StringP(FlagBucket, "b", "", "Bucket")
	getObjectCmd.Flags().StringP(FlagKey, "k", "", "Key")

	rootCmd.AddCommand(getObjectCmd)
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
