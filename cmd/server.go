// Copyright 2017 HootSuite Media Inc.
//
// Licensed under the Apache License, Version 2.0 (the License);
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an AS IS BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Modified hereafter by contributors to runatlantis/atlantis.

package cmd

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudposse/atlantis/server"
	"github.com/cloudposse/atlantis/server/events/vcs/bitbucketcloud"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// To add a new flag you must:
// 1. Add a const with the flag name (in alphabetic order).
// 2. Add a new field to server.UserConfig and set the mapstructure tag equal to the flag name.
// 3. Add your flag's description etc. to the stringFlags, intFlags, or boolFlags slices.
const (
	// Flag names.
	AllowForkPRsFlag           = "allow-fork-prs"
	AllowRepoConfigFlag        = "allow-repo-config"
	AtlantisURLFlag            = "atlantis-url"
	BitbucketBaseURLFlag       = "bitbucket-base-url"
	BitbucketTokenFlag         = "bitbucket-token"
	BitbucketUserFlag          = "bitbucket-user"
	BitbucketWebhookSecretFlag = "bitbucket-webhook-secret"
	ConfigFlag                 = "config"
	DataDirFlag                = "data-dir"
	GHHostnameFlag             = "gh-hostname"
	GHTeamWhitelistFlag        = "gh-team-whitelist"
	GHTokenFlag                = "gh-token"
	GHUserFlag                 = "gh-user"
	GHWebhookSecretFlag        = "gh-webhook-secret" // nolint: gosec
	GitlabHostnameFlag         = "gitlab-hostname"
	GitlabTokenFlag            = "gitlab-token"
	GitlabUserFlag             = "gitlab-user"
	GitlabWebhookSecretFlag    = "gitlab-webhook-secret" // nolint: gosec
	LogLevelFlag               = "log-level"
	PortFlag                   = "port"
	RepoConfigFlag             = "repo-config"
	RepoWhitelistFlag          = "repo-whitelist"
	RequireApprovalFlag        = "require-approval"
	SSLCertFileFlag            = "ssl-cert-file"
	SSLKeyFileFlag             = "ssl-key-file"
	WakeWordFlag               = "wake-word"

	// Flag defaults.
	DefaultBitbucketBaseURL = bitbucketcloud.BaseURL
	DefaultDataDir          = "~/.atlantis"
	DefaultGHHostname       = "github.com"
	DefaultGHTeamWhitelist  = "*:*"
	DefaultGitlabHostname   = "gitlab.com"
	DefaultLogLevel         = "info"
	DefaultPort             = 4141
	DefaultRepoConfig       = "atlantis.yaml"
	DefaultWakeWord         = "atlantis"
)

const redTermStart = "\033[31m"
const redTermEnd = "\033[39m"

var stringFlags = []stringFlag{
	{
		name:        AtlantisURLFlag,
		description: "URL that Atlantis can be reached at. Defaults to http://$(hostname):$port where $port is from --" + PortFlag + ".",
	},
	{
		name:        BitbucketUserFlag,
		description: "Bitbucket username of API user.",
	},
	{
		name:        BitbucketTokenFlag,
		description: "Bitbucket app password of API user. Can also be specified via the ATLANTIS_BITBUCKET_TOKEN environment variable.",
	},
	{
		name: BitbucketBaseURLFlag,
		description: "Base URL of Bitbucket Server (aka Stash) installation." +
			" Must include scheme, ex. 'http://bitbucket.corp:7990' or 'https://bitbucket.corp'." +
			" If using Bitbucket Cloud (bitbucket.org), do not set.",
		defaultValue: DefaultBitbucketBaseURL,
	},
	{
		name: BitbucketWebhookSecretFlag,
		description: "Secret used to validate Bitbucket webhooks. Only Bitbucket Server supports webhook secrets." +
			" SECURITY WARNING: If not specified, Atlantis won't be able to validate that the incoming webhook call came from Bitbucket. " +
			"This means that an attacker could spoof calls to Atlantis and cause it to perform malicious actions. " +
			"Should be specified via the ATLANTIS_BITBUCKET_WEBHOOK_SECRET environment variable.",
	},
	{
		name:        ConfigFlag,
		description: "Path to config file. All flags can be set in a YAML config file instead.",
	},
	{
		name:         DataDirFlag,
		description:  "Path to directory to store Atlantis data.",
		defaultValue: DefaultDataDir,
	},
	{
		name:         GHHostnameFlag,
		description:  "Hostname of your Github Enterprise installation. If using github.com, no need to set.",
		defaultValue: DefaultGHHostname,
	},
	{
		name: GHTeamWhitelistFlag,
		description: "Comma separated list of key-value pairs representing the GitHub teams and the operations that the members of a particular team are allowed to perform. " +
			"The format is {team}:{command},{team}:{command}, ex. dev:plan,ops:apply,admin:destroy,devops:*. " +
			"This example means to give the users from the 'dev' GitHub team the permissions to execute the 'plan' command, " +
			"give the 'ops' team the permissions to execute the 'apply' command, " +
			"give the 'admin' team the permissions to execute the 'destroy' command, " +
			"and allow the 'devops' team to perform any operation. If this argument is not provided, the default value (*:*) will be used and the default behavior will be to not check permissions " +
			"and to allow users from any team to perform any operation.",
		defaultValue: DefaultGHTeamWhitelist,
	},
	{
		name:        GHUserFlag,
		description: "GitHub username of API user.",
	},
	{
		name:        GHTokenFlag,
		description: "GitHub token of API user. Can also be specified via the ATLANTIS_GH_TOKEN environment variable.",
	},
	{
		name: GHWebhookSecretFlag,
		description: "Secret used to validate GitHub webhooks (see https://developer.github.com/webhooks/securing/)." +
			" SECURITY WARNING: If not specified, Atlantis won't be able to validate that the incoming webhook call came from GitHub. " +
			"This means that an attacker could spoof calls to Atlantis and cause it to perform malicious actions. " +
			"Should be specified via the ATLANTIS_GH_WEBHOOK_SECRET environment variable.",
	},
	{
		name:         GitlabHostnameFlag,
		description:  "Hostname of your GitLab Enterprise installation. If using gitlab.com, no need to set.",
		defaultValue: DefaultGitlabHostname,
	},
	{
		name:        GitlabUserFlag,
		description: "GitLab username of API user.",
	},
	{
		name:        GitlabTokenFlag,
		description: "GitLab token of API user. Can also be specified via the ATLANTIS_GITLAB_TOKEN environment variable.",
	},
	{
		name: GitlabWebhookSecretFlag,
		description: "Optional secret used to validate GitLab webhooks." +
			" SECURITY WARNING: If not specified, Atlantis won't be able to validate that the incoming webhook call came from GitLab. " +
			"This means that an attacker could spoof calls to Atlantis and cause it to perform malicious actions. " +
			"Should be specified via the ATLANTIS_GITLAB_WEBHOOK_SECRET environment variable.",
	},
	{
		name:         LogLevelFlag,
		description:  "Log level. Either debug, info, warn, or error.",
		defaultValue: DefaultLogLevel,
	},
	{
		name: RepoConfigFlag,
		description: "Optional path to the Atlantis YAML config file contained in each repo that this server should use. " +
			"This allows different Atlantis servers to point at different configs in the same repo.",
		defaultValue: DefaultRepoConfig,
	},
	{
		name: RepoWhitelistFlag,
		description: "Comma separated list of repositories that Atlantis will operate on. " +
			"The format is {hostname}/{owner}/{repo}, ex. github.com/runatlantis/atlantis. '*' matches any characters until the next comma and can be used for example to whitelist " +
			"all repos: '*' (not recommended), an entire hostname: 'internalgithub.com/*' or an organization: 'github.com/runatlantis/*'." +
			" For Bitbucket Server, {hostname} is the domain without scheme and port, {owner} is the name of the project (not the key), and {repo} is the repo name.",
	},
	{
		name:        SSLCertFileFlag,
		description: "File containing x509 Certificate used for serving HTTPS. If the cert is signed by a CA, the file should be the concatenation of the server's certificate, any intermediates, and the CA's certificate.",
	},
	{
		name:        SSLKeyFileFlag,
		description: fmt.Sprintf("File containing x509 private key matching --%s.", SSLCertFileFlag),
	},
	{
		name: WakeWordFlag,
		description: "Wake word for this server to listen to. Default is 'atlantis'. " +
			"This allows different wake commands (e.g. 'staging' or 'prod') to be used for different stages if more than one server operates on the same repo.",
		defaultValue: DefaultWakeWord,
	},
}
var boolFlags = []boolFlag{
	{
		name:         AllowForkPRsFlag,
		description:  "Allow Atlantis to run on pull requests from forks. A security issue for public repos.",
		defaultValue: false,
	},
	{
		name: AllowRepoConfigFlag,
		description: "Allow repositories to use atlantis repo config YAML files to customize the commands Atlantis runs." +
			" Should only be enabled in a trusted environment since it enables a pull request to run arbitrary commands" +
			" on the Atlantis server.",
		defaultValue: false,
	},
	{
		name:         RequireApprovalFlag,
		description:  "Require pull requests to be \"Approved\" before allowing the apply command to be run.",
		defaultValue: false,
	},
}
var intFlags = []intFlag{
	{
		name:         PortFlag,
		description:  "Port to bind to.",
		defaultValue: DefaultPort,
	},
}

type stringFlag struct {
	name         string
	description  string
	defaultValue string
}
type intFlag struct {
	name         string
	description  string
	defaultValue int
}
type boolFlag struct {
	name         string
	description  string
	defaultValue bool
}

// ServerCmd is an abstraction that helps us test. It allows
// us to mock out starting the actual server.
type ServerCmd struct {
	ServerCreator ServerCreator
	Viper         *viper.Viper
	// SilenceOutput set to true means nothing gets printed.
	// Useful for testing to keep the logs clean.
	SilenceOutput   bool
	AtlantisVersion string
}

// ServerCreator creates servers.
// It's an abstraction to help us test.
type ServerCreator interface {
	NewServer(userConfig server.UserConfig, config server.Config) (ServerStarter, error)
}

// DefaultServerCreator is the concrete implementation of ServerCreator.
type DefaultServerCreator struct{}

// ServerStarter is for starting up a server.
// It's an abstraction to help us test.
type ServerStarter interface {
	Start() error
}

// NewServer returns the real Atlantis server object.
func (d *DefaultServerCreator) NewServer(userConfig server.UserConfig, config server.Config) (ServerStarter, error) {
	return server.NewServer(userConfig, config)
}

// Init returns the runnable cobra command.
func (s *ServerCmd) Init() *cobra.Command {
	c := &cobra.Command{
		Use:           "server",
		Short:         "Start the atlantis server",
		Long:          `Start the atlantis server and listen for webhook calls.`,
		SilenceErrors: true,
		SilenceUsage:  s.SilenceOutput,
		PreRunE: s.withErrPrint(func(cmd *cobra.Command, args []string) error {
			return s.preRun()
		}),
		RunE: s.withErrPrint(func(cmd *cobra.Command, args []string) error {
			return s.run()
		}),
	}

	// Configure viper to accept env vars prefixed with ATLANTIS_ that can be
	// used instead of flags.
	s.Viper.SetEnvPrefix("ATLANTIS")
	s.Viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	s.Viper.AutomaticEnv()
	s.Viper.SetTypeByDefaultValue(true)

	// Replace the call in their template to use the usage function that wraps
	// columns to make for a nicer output.
	usageWithWrappedCols := strings.Replace(c.UsageTemplate(), ".FlagUsages", ".FlagUsagesWrapped 120", -1)
	c.SetUsageTemplate(usageWithWrappedCols)

	// If a user passes in an invalid flag, tell them what the flag was.
	c.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		fmt.Fprintf(os.Stderr, "\033[31mError: %s\033[39m\n\n", err.Error())
		return err
	})

	// Set string flags.
	for _, f := range stringFlags {
		usage := f.description
		if f.defaultValue != "" {
			usage = fmt.Sprintf("%s (default \"%s\")", usage, f.defaultValue)
		}
		c.Flags().String(f.name, "", usage+"\n")
		s.Viper.BindPFlag(f.name, c.Flags().Lookup(f.name)) // nolint: errcheck
	}

	// Set int flags.
	for _, f := range intFlags {
		usage := f.description
		if f.defaultValue != 0 {
			usage = fmt.Sprintf("%s (default %d)", usage, f.defaultValue)
		}
		c.Flags().Int(f.name, 0, usage+"\n")
		s.Viper.BindPFlag(f.name, c.Flags().Lookup(f.name)) // nolint: errcheck
	}

	// Set bool flags.
	for _, f := range boolFlags {
		c.Flags().Bool(f.name, f.defaultValue, f.description+"\n")
		s.Viper.BindPFlag(f.name, c.Flags().Lookup(f.name)) // nolint: errcheck
	}

	return c
}

func (s *ServerCmd) preRun() error {
	// If passed a config file then try and load it.
	configFile := s.Viper.GetString(ConfigFlag)
	if configFile != "" {
		s.Viper.SetConfigFile(configFile)
		if err := s.Viper.ReadInConfig(); err != nil {
			return errors.Wrapf(err, "invalid config: reading %s", configFile)
		}
	}
	return nil
}

func (s *ServerCmd) run() error {
	var userConfig server.UserConfig
	if err := s.Viper.Unmarshal(&userConfig); err != nil {
		return err
	}
	s.setDefaults(&userConfig)
	if err := s.validate(userConfig); err != nil {
		return err
	}
	if err := s.setAtlantisURL(&userConfig); err != nil {
		return err
	}
	if err := s.setDataDir(&userConfig); err != nil {
		return err
	}
	s.securityWarnings(&userConfig)
	s.trimAtSymbolFromUsers(&userConfig)

	// Config looks good. Start the server.
	server, err := s.ServerCreator.NewServer(userConfig, server.Config{
		AllowForkPRsFlag:    AllowForkPRsFlag,
		AllowRepoConfigFlag: AllowRepoConfigFlag,
		AtlantisVersion:     s.AtlantisVersion,
	})
	if err != nil {
		return errors.Wrap(err, "initializing server")
	}
	return server.Start()
}

func (s *ServerCmd) setDefaults(c *server.UserConfig) {
	if c.DataDir == "" {
		c.DataDir = DefaultDataDir
	}
	if c.GithubHostname == "" {
		c.GithubHostname = DefaultGHHostname
	}
	if c.GitlabHostname == "" {
		c.GitlabHostname = DefaultGitlabHostname
	}
	if c.BitbucketBaseURL == "" {
		c.BitbucketBaseURL = DefaultBitbucketBaseURL
	}
	if c.LogLevel == "" {
		c.LogLevel = DefaultLogLevel
	}
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.GithubTeamWhitelist == "" {
		c.GithubTeamWhitelist = DefaultGHTeamWhitelist
	}
	if c.RepoConfig == "" {
		c.RepoConfig = DefaultRepoConfig
	}
	if c.WakeWord == "" {
		c.WakeWord = DefaultWakeWord
	}
	if c.CustomStageNames == nil || len(c.CustomStageNames) == 0 {
		c.CustomStageNames = []string{}
	}
}

func (s *ServerCmd) validate(userConfig server.UserConfig) error {
	logLevel := userConfig.LogLevel
	if logLevel != "debug" && logLevel != "info" && logLevel != "warn" && logLevel != "error" {
		return errors.New("invalid log level: not one of debug, info, warn, error")
	}

	if (userConfig.SSLKeyFile == "") != (userConfig.SSLCertFile == "") {
		return fmt.Errorf("--%s and --%s are both required for ssl", SSLKeyFileFlag, SSLCertFileFlag)
	}

	// The following combinations are valid.
	// 1. github user and token set
	// 2. gitlab user and token set
	// 3. bitbucket user and token set
	// 4. any combination of the above
	vcsErr := fmt.Errorf("--%s/--%s or --%s/--%s or --%s/--%s must be set", GHUserFlag, GHTokenFlag, GitlabUserFlag, GitlabTokenFlag, BitbucketUserFlag, BitbucketTokenFlag)
	if ((userConfig.GithubUser == "") != (userConfig.GithubToken == "")) || ((userConfig.GitlabUser == "") != (userConfig.GitlabToken == "")) || ((userConfig.BitbucketUser == "") != (userConfig.BitbucketToken == "")) {
		return vcsErr
	}
	// At this point, we know that there can't be a single user/token without
	// its partner, but we haven't checked if any user/token is set at all.
	if userConfig.GithubUser == "" && userConfig.GitlabUser == "" && userConfig.BitbucketUser == "" {
		return vcsErr
	}

	if userConfig.RepoWhitelist == "" {
		return fmt.Errorf("--%s must be set for security purposes", RepoWhitelistFlag)
	}
	if strings.Contains(userConfig.RepoWhitelist, "://") {
		return fmt.Errorf("--%s cannot contain ://, should be hostnames only", RepoWhitelistFlag)
	}

	if userConfig.BitbucketBaseURL == DefaultBitbucketBaseURL && userConfig.BitbucketWebhookSecret != "" {
		return fmt.Errorf("--%s cannot be specified for Bitbucket Cloud because it is not supported by Bitbucket", BitbucketWebhookSecretFlag)
	}

	parsed, err := url.Parse(userConfig.BitbucketBaseURL)
	if err != nil {
		return fmt.Errorf("error parsing --%s flag value %q: %s", BitbucketWebhookSecretFlag, userConfig.BitbucketBaseURL, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("--%s must have http:// or https://, got %q", BitbucketBaseURLFlag, userConfig.BitbucketBaseURL)
	}

	// Cannot accept custom repo config if we know repo configs are disabled
	if (userConfig.RepoConfig != DefaultRepoConfig) && (!userConfig.AllowRepoConfig) {
		return fmt.Errorf("custom --%s cannot be specified if --%s is false", RepoConfigFlag, AllowRepoConfigFlag)
	}

	return nil
}

// setAtlantisURL sets the externally accessible URL for atlantis.
func (s *ServerCmd) setAtlantisURL(userConfig *server.UserConfig) error {
	if userConfig.AtlantisURL == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("Failed to determine hostname: %v", err)
		}
		userConfig.AtlantisURL = fmt.Sprintf("http://%s:%d", hostname, userConfig.Port)
	}
	return nil
}

// setDataDir checks if ~ was used in data-dir and converts it to the actual
// home directory. If we don't do this, we'll create a directory called "~"
// instead of actually using home. It also converts relative paths to absolute.
func (s *ServerCmd) setDataDir(userConfig *server.UserConfig) error {
	finalPath := userConfig.DataDir

	// Convert ~ to the actual home dir.
	if strings.HasPrefix(finalPath, "~/") {
		var err error
		finalPath, err = homedir.Expand(finalPath)
		if err != nil {
			return errors.Wrap(err, "determining home directory")
		}
	}

	// Convert relative paths to absolute.
	finalPath, err := filepath.Abs(finalPath)
	if err != nil {
		return errors.Wrap(err, "making data-dir absolute")
	}
	userConfig.DataDir = finalPath
	return nil
}

// trimAtSymbolFromUsers trims @ from the front of the github and gitlab usernames
func (s *ServerCmd) trimAtSymbolFromUsers(userConfig *server.UserConfig) {
	userConfig.GithubUser = strings.TrimPrefix(userConfig.GithubUser, "@")
	userConfig.GitlabUser = strings.TrimPrefix(userConfig.GitlabUser, "@")
	userConfig.BitbucketUser = strings.TrimPrefix(userConfig.BitbucketUser, "@")
}

func (s *ServerCmd) securityWarnings(userConfig *server.UserConfig) {
	if userConfig.GithubUser != "" && userConfig.GithubWebhookSecret == "" && !s.SilenceOutput {
		fmt.Fprintf(os.Stderr, "%s[WARN] No GitHub webhook secret set. This could allow attackers to spoof requests from GitHub.%s\n", redTermStart, redTermEnd)
	}
	if userConfig.GitlabUser != "" && userConfig.GitlabWebhookSecret == "" && !s.SilenceOutput {
		fmt.Fprintf(os.Stderr, "%s[WARN] No GitLab webhook secret set. This could allow attackers to spoof requests from GitLab.%s\n", redTermStart, redTermEnd)
	}
	if userConfig.BitbucketUser != "" && userConfig.BitbucketBaseURL != DefaultBitbucketBaseURL && userConfig.BitbucketWebhookSecret == "" && !s.SilenceOutput {
		fmt.Fprintf(os.Stderr, "%s[WARN] No Bitbucket webhook secret set. This could allow attackers to spoof requests from Bitbucket.%s\n", redTermStart, redTermEnd)
	}
	if userConfig.BitbucketUser != "" && userConfig.BitbucketBaseURL == DefaultBitbucketBaseURL && !s.SilenceOutput {
		fmt.Fprintf(os.Stderr, "%s[WARN] Bitbucket Cloud does not support webhook secrets. This could allow attackers to spoof requests from Bitbucket. Ensure you are whitelisting Bitbucket IPs.%s\n", redTermStart, redTermEnd)
	}
}

// withErrPrint prints out any errors to a terminal in red.
func (s *ServerCmd) withErrPrint(f func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := f(cmd, args)
		if err != nil && !s.SilenceOutput {
			fmt.Fprintf(os.Stderr, "%s[ERROR] %s%s\n\n", redTermStart, err.Error(), redTermEnd)
		}
		return err
	}
}
