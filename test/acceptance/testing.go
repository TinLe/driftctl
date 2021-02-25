package acceptance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/cloudskiff/driftctl/pkg/analyser"
	cmderrors "github.com/cloudskiff/driftctl/pkg/cmd/errors"
	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"

	"github.com/cloudskiff/driftctl/test"

	"github.com/spf13/cobra"

	"github.com/cloudskiff/driftctl/logger"
	"github.com/cloudskiff/driftctl/pkg/cmd"

	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-exec/tfinstall"
)

type AccCheck struct {
	PreExec  func()
	PostExec func()
	Env      map[string]string
	Check    func(result *ScanResult, stdout string, err error)
}

type AccTestCase struct {
	Paths                      []string
	Args                       []string
	OnStart                    func()
	OnEnd                      func()
	Checks                     []AccCheck
	tmpResultFilePath          string
	originalEnv                []string
	tf                         map[string]*tfexec.Terraform
	ShouldRefreshBeforeDestroy bool
}

func (c *AccTestCase) initTerraformExecutor() error {
	c.tf = make(map[string]*tfexec.Terraform, 1)
	for _, path := range c.Paths {
		execPath, err := tfinstall.LookPath().ExecPath(context.Background())
		if err != nil {
			return err
		}
		c.tf[path], err = tfexec.NewTerraform(path, execPath)
		if err != nil {
			return err
		}
		env := c.resolveTerraformEnv()
		if err := c.tf[path].SetEnv(env); err != nil {
			return err
		}
	}
	return nil
}

func (c *AccTestCase) createResultFile(t *testing.T) error {
	tmpDir := t.TempDir()
	file, err := ioutil.TempFile(tmpDir, "result")
	if err != nil {
		return err
	}
	defer file.Close()
	c.tmpResultFilePath = file.Name()
	return nil
}

func (c *AccTestCase) validate() error {
	if c.Checks == nil || len(c.Checks) == 0 {
		return fmt.Errorf("checks attribute must be defined")
	}

	if len(c.Paths) < 1 {
		return fmt.Errorf("Paths attribute must be defined")
	}

	for _, arg := range c.Args {
		if arg == "--output" || arg == "-o" {
			return fmt.Errorf("--output flag should not be defined in test case, it is automatically tested")
		}
	}

	return nil
}

func (c *AccTestCase) getResultFilePath() string {
	return c.tmpResultFilePath
}

func (c *AccTestCase) getResult(t *testing.T) *ScanResult {
	analysis := analyser.Analysis{}
	result, err := ioutil.ReadFile(c.getResultFilePath())
	if err != nil {
		return nil
	}

	if err := json.Unmarshal(result, &analysis); err != nil {
		return nil
	}

	return NewScanResult(t, analysis)
}

/**
 * Retrieve env from os.Environ() but override every variable prefixed with ACC_
 * e.g. ACC_AWS_PROFILE will override AWS_PROFILE
 */
func (c *AccTestCase) resolveTerraformEnv() map[string]string {

	environMap := make(map[string]string, len(os.Environ()))

	const PREFIX string = "ACC_"

	for _, e := range os.Environ() {
		envKeyValue := strings.SplitN(e, "=", 2)
		if strings.HasPrefix(envKeyValue[0], PREFIX) {
			varName := strings.TrimPrefix(envKeyValue[0], PREFIX)
			environMap[varName] = envKeyValue[1]
			continue
		}
		if _, exist := environMap[envKeyValue[0]]; !exist {
			environMap[envKeyValue[0]] = envKeyValue[1]
		}
	}

	return environMap
}

func (c *AccTestCase) terraformInit() error {
	if err := c.initTerraformExecutor(); err != nil {
		return err
	}
	for _, p := range c.Paths {
		_, err := os.Stat(path.Join(p, ".terraform"))
		if os.IsNotExist(err) {
			logrus.WithFields(logrus.Fields{
				"path": p,
			}).Debug("Running terraform init ...")
			stderr := new(bytes.Buffer)
			c.tf[p].SetStderr(stderr)
			if err := c.tf[p].Init(context.Background()); err != nil {
				return errors.Wrap(err, stderr.String())
			}
			logrus.WithFields(logrus.Fields{
				"path": p,
			}).Debug("Terraform init done")
		}
	}

	return nil
}

func (c *AccTestCase) terraformApply() error {
	for _, p := range c.Paths {
		logrus.WithFields(logrus.Fields{
			"p": p,
		}).Debug("Running terraform apply ...")
		stderr := new(bytes.Buffer)
		c.tf[p].SetStderr(stderr)
		if err := c.tf[p].Apply(context.Background()); err != nil {
			return errors.Wrap(err, stderr.String())
		}
		logrus.WithFields(logrus.Fields{
			"p": p,
		}).Debug("Terraform apply done")
	}

	return nil
}

func (c *AccTestCase) terraformDestroy() error {
	if c.ShouldRefreshBeforeDestroy {
		if err := c.terraformRefresh(); err != nil {
			return err
		}
	}

	for _, p := range c.Paths {
		logrus.WithFields(logrus.Fields{
			"p": p,
		}).Debug("Running terraform destroy ...")
		stderr := new(bytes.Buffer)
		c.tf[p].SetStderr(stderr)
		if err := c.tf[p].Destroy(context.Background()); err != nil {
			return errors.Wrap(err, stderr.String())
		}
		logrus.WithFields(logrus.Fields{
			"p": p,
		}).Debug("Terraform destroy done")
	}

	return nil
}

func (c *AccTestCase) terraformRefresh() error {
	for _, p := range c.Paths {
		logrus.WithFields(logrus.Fields{
			"p": p,
		}).Debug("Running terraform refresh ...")
		stderr := new(bytes.Buffer)
		c.tf[p].SetStderr(stderr)
		if err := c.tf[p].Refresh(context.Background()); err != nil {
			return errors.Wrap(err, stderr.String())
		}
		logrus.WithFields(logrus.Fields{
			"p": p,
		}).Debug("Terraform refresh done")
	}

	return nil
}

func runDriftCtlCmd(driftctlCmd *cmd.DriftctlCmd) (*cobra.Command, string, error) {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	cmd, cmdErr := driftctlCmd.ExecuteC()
	// Ignore not in sync errors in acceptance test context
	if _, isNotInSyncErr := cmdErr.(cmderrors.InfrastructureNotInSync); isNotInSyncErr {
		cmdErr = nil
	}
	outC := make(chan string)
	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- buf.String()
	}()

	// back to normal state
	w.Close()
	os.Stdout = old // restoring the real stdout
	out := <-outC
	return cmd, out, cmdErr
}

func (c *AccTestCase) useTerraformEnv() {
	c.originalEnv = os.Environ()
	environMap := c.resolveTerraformEnv()
	env := make([]string, 0, len(environMap))
	for k, v := range environMap {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	c.setEnv(env)
}

func (c *AccTestCase) restoreEnv() {
	if c.originalEnv != nil {
		logrus.Debug("Restoring original environment ...")
		os.Clearenv()
		c.setEnv(c.originalEnv)
		c.originalEnv = nil
	}
}

func (c *AccTestCase) setEnv(env []string) {
	os.Clearenv()
	for _, e := range env {
		envKeyValue := strings.SplitN(e, "=", 2)
		os.Setenv(envKeyValue[0], envKeyValue[1])
	}
}

func Run(t *testing.T, c AccTestCase) {

	if os.Getenv("DRIFTCTL_ACC") == "" {
		t.Skip()
	}

	if err := c.validate(); err != nil {
		t.Fatal(err)
	}

	if c.OnStart != nil {
		c.useTerraformEnv()
		c.OnStart()
		if c.OnEnd != nil {
			defer func() {
				c.useTerraformEnv()
				c.OnEnd()
				c.restoreEnv()
			}()
		}
		c.restoreEnv()
	}

	// Disable terraform version checks
	// @link https://www.terraform.io/docs/commands/index.html#upgrade-and-security-bulletin-checks
	checkpoint := os.Getenv("CHECKPOINT_DISABLE")
	os.Setenv("CHECKPOINT_DISABLE", "true")

	// Execute terraform init if .terraform folder is not found in test folder
	err := c.terraformInit()
	if err != nil {
		t.Fatal(err)
	}

	err = c.terraformApply()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		c.restoreEnv()
		err := c.terraformDestroy()
		os.Setenv("CHECKPOINT_DISABLE", checkpoint)
		if err != nil {
			t.Fatal(err)
		}
	}()

	logger.Init()

	err = c.createResultFile(t)
	if err != nil {
		t.Fatal(err)
	}
	if c.Args != nil {
		c.Args = append([]string{""}, c.Args...)
		isFromSet := false
		for _, arg := range c.Args {
			if arg == "--from" || arg == "-f" {
				isFromSet = true
				break
			}
		}
		if !isFromSet {
			for _, p := range c.Paths {
				c.Args = append(c.Args,
					"--from", fmt.Sprintf("tfstate://%s", path.Join(p, "terraform.tfstate")),
				)
			}
		}
		c.Args = append(c.Args,
			"--output", fmt.Sprintf("json://%s", c.getResultFilePath()),
		)
	}
	os.Args = c.Args

	for _, check := range c.Checks {
		driftctlCmd := cmd.NewDriftctlCmd(test.Build{})
		if check.Check == nil {
			t.Fatal("Check attribute must be defined")
		}
		if len(check.Env) > 0 {
			for key, value := range check.Env {
				os.Setenv(key, value)
			}
		}
		if check.PreExec != nil {
			c.useTerraformEnv()
			check.PreExec()
			c.restoreEnv()
		}
		_, out, cmdErr := runDriftCtlCmd(driftctlCmd)
		if len(check.Env) > 0 {
			for key := range check.Env {
				_ = os.Unsetenv(key)
			}
		}
		check.Check(c.getResult(t), out, cmdErr)
		if check.PostExec != nil {
			check.PostExec()
		}
	}
}

func RetryFor(timeout time.Duration, f func(c chan struct{}) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	doneCh := make(chan struct{}, 1)
	errCh := make(chan error, 1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := f(doneCh); err != nil {
					errCh <- err
					return
				}
				time.Sleep(1 * time.Second)
			}
		}
	}()

	select {
	case <-doneCh:
		return nil
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
