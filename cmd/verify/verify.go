// Package verify provides the "verify" command.
package verify

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	cmd_generate "github.com/safe-waters/docker-lock/cmd/generate"
	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/registry"
	"github.com/safe-waters/docker-lock/pkg/verify"
	"github.com/safe-waters/docker-lock/pkg/verify/diff"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const namespace = "verify"

// NewVerifyCmd creates the command 'verify' used in 'docker lock verify'.
func NewVerifyCmd(client *registry.HTTPClient) (*cobra.Command, error) {
	verifyCmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify that a Lockfile is up-to-date",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return bindPFlags(cmd, []string{
				"lockfile-name",
				"config-file",
				"env-file",
				"ignore-missing-digests",
				"exclude-tags",
			})
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			flags, err := parseFlags()
			if err != nil {
				return err
			}

			verifier, err := SetupVerifier(client, flags)
			if err != nil {
				return err
			}

			reader, err := os.Open(flags.LockfileName)
			if err != nil {
				return err
			}
			defer reader.Close()

			return verifier.VerifyLockfile(reader)
		},
	}
	verifyCmd.Flags().String(
		"lockfile-name", "docker-lock.json", "Lockfile to read from",
	)
	verifyCmd.Flags().String(
		"config-file", cmd_generate.DefaultConfigPath(),
		"Path to config file for auth credentials",
	)
	verifyCmd.Flags().String(
		"env-file", ".env", "Path to .env file",
	)
	verifyCmd.Flags().Bool(
		"ignore-missing-digests", false,
		"Do not fail if unable to find digests",
	)
	verifyCmd.Flags().Bool(
		"exclude-tags", false, "Exclude image tags from verification",
	)

	return verifyCmd, nil
}

// SetupVerifier creates a Verifier configured for docker-lock's cli.
func SetupVerifier(
	client *registry.HTTPClient,
	flags *Flags,
) (*verify.Verifier, error) {
	if flags == nil {
		return nil, errors.New("flags cannot be nil")
	}

	if err := cmd_generate.DefaultLoadEnv(flags.EnvPath); err != nil {
		return nil, err
	}

	existingLByt, err := ioutil.ReadFile(flags.LockfileName)
	if err != nil {
		return nil, err
	}

	var existingLockfile generate.Lockfile
	if err = json.Unmarshal(existingLByt, &existingLockfile); err != nil {
		return nil, err
	}

	dockerfilePaths := make([]string, len(existingLockfile.DockerfileImages))
	composefilePaths := make([]string, len(existingLockfile.ComposefileImages))
	kubernetesfilePaths := make(
		[]string, len(existingLockfile.KubernetesfileImages),
	)

	var i, j, k int

	for p := range existingLockfile.DockerfileImages {
		dockerfilePaths[i] = p
		i++
	}

	for p := range existingLockfile.ComposefileImages {
		composefilePaths[j] = p
		j++
	}

	for p := range existingLockfile.KubernetesfileImages {
		kubernetesfilePaths[k] = p
		k++
	}

	generatorFlags, err := cmd_generate.NewFlags(
		".", "", flags.ConfigPath, flags.EnvPath, flags.IgnoreMissingDigests,
		dockerfilePaths, composefilePaths, kubernetesfilePaths, nil, nil, nil,
		false, false, false, len(dockerfilePaths) == 0,
		len(composefilePaths) == 0, len(kubernetesfilePaths) == 0,
	)
	if err != nil {
		return nil, err
	}

	generator, err := cmd_generate.SetupGenerator(client, generatorFlags)
	if err != nil {
		return nil, err
	}

	dockerfileDifferentiator := &diff.DockerfileDifferentiator{
		ExcludeTags: flags.ExcludeTags,
	}
	composefileDifferentiator := &diff.ComposefileDifferentiator{
		ExcludeTags: flags.ExcludeTags,
	}
	kubernetesfileDifferentiator := &diff.KubernetesfileDifferentiator{
		ExcludeTags: flags.ExcludeTags,
	}

	return verify.NewVerifier(
		generator, dockerfileDifferentiator,
		composefileDifferentiator, kubernetesfileDifferentiator,
	)
}

func bindPFlags(cmd *cobra.Command, flagNames []string) error {
	for _, name := range flagNames {
		if err := viper.BindPFlag(
			fmt.Sprintf("%s.%s", namespace, name), cmd.Flags().Lookup(name),
		); err != nil {
			return err
		}
	}

	return nil
}

func parseFlags() (*Flags, error) {
	lockfileName := viper.GetString(
		fmt.Sprintf("%s.%s", namespace, "lockfile-name"),
	)
	configPath := viper.GetString(
		fmt.Sprintf("%s.%s", namespace, "config-file"),
	)
	envPath := viper.GetString(
		fmt.Sprintf("%s.%s", namespace, "env-file"),
	)
	ignoreMissingDigests := viper.GetBool(
		fmt.Sprintf("%s.%s", namespace, "ignore-missing-digests"),
	)
	excludeTags := viper.GetBool(
		fmt.Sprintf("%s.%s", namespace, "exclude-tags"),
	)

	return NewFlags(
		lockfileName, configPath, envPath, ignoreMissingDigests, excludeTags,
	)
}
