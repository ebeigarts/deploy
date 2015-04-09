package github

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/remind101/deploy/Godeps/_workspace/src/github.com/howeyc/gopass"
	"github.com/remind101/deploy/Godeps/_workspace/src/github.com/github/hub/ui"
	"github.com/remind101/deploy/Godeps/_workspace/src/github.com/github/hub/utils"
)

var defaultConfigsFile string

func init() {
	homeDir := os.Getenv("HOME")

	if homeDir == "" {
		if u, err := user.Current(); err == nil {
			homeDir = u.HomeDir
		}
	}

	if homeDir == "" {
		utils.Check(fmt.Errorf("Can't get current user's home dir"))
	}

	defaultConfigsFile = filepath.Join(homeDir, ".config", "hub")
}

type yamlHost struct {
	User       string `yaml:"user"`
	OAuthToken string `yaml:"oauth_token"`
	Protocol   string `yaml:"protocol"`
}

type yamlConfig map[string][]yamlHost

type Host struct {
	Host        string `toml:"host"`
	User        string `toml:"user"`
	AccessToken string `toml:"access_token"`
	Protocol    string `toml:"protocol"`
}

type Config struct {
	Hosts []Host `toml:"hosts"`
}

func (c *Config) PromptForHost(host string) (h *Host, err error) {
	h = c.Find(host)
	if h != nil {
		return
	}

	user := c.PromptForUser(host)
	pass := c.PromptForPassword(host, user)

	client := NewClient(host)
	var code, token string
	for {
		token, err = client.FindOrCreateToken(user, pass, code)
		if err == nil {
			break
		}

		if ae, ok := err.(*AuthError); ok && ae.IsRequired2FACodeError() {
			if code != "" {
				ui.Errorln("warning: invalid two-factor code")
			}
			code = c.PromptForOTP()
		} else {
			break
		}
	}

	if err != nil {
		return
	}

	client.Host.AccessToken = token
	currentUser, err := client.CurrentUser()
	if err != nil {
		return
	}

	h = &Host{
		Host:        host,
		User:        currentUser.Login,
		AccessToken: token,
		Protocol:    "https",
	}
	c.Hosts = append(c.Hosts, *h)
	err = newConfigService().Save(configsFile(), c)

	return
}

func (c *Config) PromptForUser(host string) (user string) {
	user = os.Getenv("GITHUB_USER")
	if user != "" {
		return
	}

	ui.Printf("%s username: ", host)
	user = c.scanLine()

	return
}

func (c *Config) PromptForPassword(host, user string) (pass string) {
	pass = os.Getenv("GITHUB_PASSWORD")
	if pass != "" {
		return
	}

	ui.Printf("%s password for %s (never stored): ", host, user)
	if isTerminal(os.Stdout.Fd()) {
		pass = string(gopass.GetPasswd())
	} else {
		pass = c.scanLine()
	}

	return
}

func (c *Config) PromptForOTP() string {
	fmt.Print("two-factor authentication code: ")
	return c.scanLine()
}

func (c *Config) scanLine() string {
	var line string
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		line = scanner.Text()
	}
	utils.Check(scanner.Err())

	return line
}

func (c *Config) Find(host string) *Host {
	for _, h := range c.Hosts {
		if h.Host == host {
			return &h
		}
	}

	return nil
}

func (c *Config) selectHost() *Host {
	options := len(c.Hosts)

	if options == 1 {
		return &c.Hosts[0]
	}

	prompt := "Select host:\n"
	for idx, host := range c.Hosts {
		prompt += fmt.Sprintf(" %d. %s\n", idx+1, host.Host)
	}
	prompt += fmt.Sprint("> ")

	ui.Printf(prompt)
	index := c.scanLine()
	i, err := strconv.Atoi(index)
	if err != nil || i < 1 || i > options {
		utils.Check(fmt.Errorf("Error: must enter a number [1-%d]", options))
	}

	return &c.Hosts[i-1]
}

func configsFile() string {
	configsFile := os.Getenv("HUB_CONFIG")
	if configsFile == "" {
		configsFile = defaultConfigsFile
	}

	return configsFile
}

func CurrentConfig() *Config {
	c := &Config{}
	newConfigService().Load(configsFile(), c)

	return c
}

func (c *Config) DefaultHost() (host *Host, err error) {
	if GitHubHostEnv != "" {
		host, err = c.PromptForHost(GitHubHostEnv)
	} else if len(c.Hosts) > 0 {
		host = c.selectHost()
	} else {
		host, err = c.PromptForHost(DefaultGitHubHost())
	}

	return
}

// Public for testing purpose
func CreateTestConfigs(user, token string) *Config {
	f, _ := ioutil.TempFile("", "test-config")
	defaultConfigsFile = f.Name()

	host := Host{
		User:        "jingweno",
		AccessToken: "123",
		Host:        GitHubHost,
	}

	c := &Config{Hosts: []Host{host}}
	err := newConfigService().Save(f.Name(), c)
	if err != nil {
		panic(err)
	}

	return c
}
