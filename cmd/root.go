/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/404LifeFound/es-snapshot-restore/config"
	posflag "github.com/404LifeFound/koanf-pflags-provider"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "es-snapshot-restore",
		Short: "restore elasticsearch index from snapshot",
		Long:  "A application can restore elasticsearch index from snapshot via command or http server",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			LoadConfig(cmd)
			cfg, _ := json.Marshal(config.GlobalConfig)
			log.Debug().Msgf("config from struct is: %s", string(cfg))
		},
		Run: func(cmd *cobra.Command, args []string) {
			log.Debug().Msgf("run: %s", cmd.Name())
		},
	}
	rootCmd.PersistentFlags().String(config.FLAG_CONFIG_FILE, config.CONFIG_FILE_PATH, "config file path")
	rootCmd.PersistentFlags().String(config.FLAG_ENV_PREFIX, config.ENV_PREFIX, "env prefix")

	rootCmd.PersistentFlags().String("es-host", "127.0.0.1", "es host")
	rootCmd.PersistentFlags().Int("es-port", 9200, "es port")
	rootCmd.PersistentFlags().String("es-protocol", "https", "es protocol,https or http")
	rootCmd.PersistentFlags().String("es-username", "", "es username")
	rootCmd.PersistentFlags().String("es-password", "", "es password")

	rootCmd.AddCommand(
		NewServerCmd(),
	)

	return rootCmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd := NewRootCmd()
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func LoadConfig(cmd *cobra.Command) {
	k := koanf.New(".")
	// get config file, if pass --config-file then use it, else use CONFIG_FILE environment, or use default --config-file
	var configFile string
	var envPrefix string
	if cmd.Flags().Lookup(config.FLAG_CONFIG_FILE).Changed {
		configFile, _ = cmd.Flags().GetString(config.FLAG_CONFIG_FILE)
	} else {
		configFile = os.Getenv(config.ENV_CONFIG_FILE)
		if configFile == "" {
			configFile, _ = cmd.Flags().GetString(config.FLAG_CONFIG_FILE)
		}
	}
	log.Debug().Msgf("config file is: %s", configFile)

	// Load YAMl config.
	if err := k.Load(file.Provider(configFile), yaml.Parser()); err != nil {
		log.Error().Err(err).Msgf("faild to load config file: %s", configFile)
	}
	log.Debug().Msgf("config from config file %s is %v", configFile, k.All())

	// get env prefix, flag > file, default is flag
	if cmd.Flags().Lookup(config.FLAG_ENV_PREFIX).Changed {
		envPrefix, _ = cmd.Flags().GetString(config.FLAG_ENV_PREFIX)
	} else if k.String("env.prefix") != "" {
		envPrefix = k.String("env.prefix")
	} else {
		envPrefix, _ = cmd.Flags().GetString(config.FLAG_ENV_PREFIX)
	}
	log.Debug().Msgf("env prefix is: %s", envPrefix)

	k.Load(env.Provider(".", env.Opt{
		Prefix: envPrefix,
		TransformFunc: func(k, v string) (string, any) {
			k = strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(k, envPrefix)), "_", ".")
			if strings.Contains(v, " ") {
				return k, strings.Split(v, " ")
			}
			return k, v
		},
	}), nil)

	log.Debug().Msgf("config from file and env is %v", k.All())

	if err := k.Load(posflag.Provider(
		posflag.WithFlagset(cmd.Flags()),
		posflag.WithKo(k),
		posflag.WithFlagCB(func(f *pflag.Flag) (string, any) {
			log.Debug().Msgf("flag: %v, value: %v", f.Name, f.Value)
			k := strings.ReplaceAll(strings.ToLower(f.Name), "-", ".")
			v := posflag.FlagVal(cmd.Flags(), f)
			return k, v
		}),
	), nil); err != nil {
		log.Error().Err(err).Msg("faild to load flags")
	}
	log.Debug().Msgf("config from file and env and flags is %v", k.All())

	// unflatten map "kibana.host" ==》 "kibana: host: xx", but finally,it will include both, "kibana.host","kibana: host"
	if err := k.Load(confmap.Provider(k.All(), "."), nil); err != nil {
		log.Error().Err(err).Msg("faild to load all koanf conf to confmap")
	}

	if err := k.UnmarshalWithConf("", &config.GlobalConfig, koanf.UnmarshalConf{
		Tag:       "koanf",
		FlatPaths: false,
	}); err != nil {
		log.Error().Err(err).Msg("failed to unmarshal config to struct")
	}
}
