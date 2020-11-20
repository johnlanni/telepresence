package connector

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/datawire/ambassador/pkg/dlog"
	"github.com/sirupsen/logrus"

	"github.com/datawire/telepresence2/pkg/client"

	"github.com/datawire/telepresence2/pkg/version"

	"github.com/datawire/ambassador/pkg/kates"

	"github.com/datawire/ambassador/pkg/dtest"
)

var kubeconfig string
var namespace string
var registry string
var testVersion = "v0.1.2-test"

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard) // We want success or failure, not an abundance of output
	kubeconfig = dtest.Kubeconfig()
	namespace = fmt.Sprintf("telepresence-%d", os.Getpid())
	registry = dtest.DockerRegistry()
	version.Version = testVersion

	os.Setenv("DTEST_KUBECONFIG", kubeconfig)
	os.Setenv("KO_DOCKER_REPO", registry)
	os.Setenv("TELEPRESENCE_REGISTRY", registry)
	dtest.WithMachineLock(func() {
		capture(nil, "kubectl", "--kubeconfig", kubeconfig, "create", "namespace", namespace)
		defer capture(nil, "kubectl", "--kubeconfig", kubeconfig, "delete", "namespace", namespace, "--wait=false")
		os.Exit(m.Run())
	})
}

func showArgs(exe string, args []string) {
	fmt.Print("+ ")
	fmt.Print(exe)
	for _, arg := range args {
		fmt.Print(" ", arg)
	}
	fmt.Println()
}

func capture(t *testing.T, exe string, args ...string) string {
	showArgs(exe, args)
	cmd := exec.Command(exe, args...)
	out, err := cmd.CombinedOutput()
	sout := string(out)
	if err != nil {
		if t != nil {
			t.Fatalf("%s\n%s", sout, err.Error())
		} else {
			log.Fatalf("%s\n%s", sout, err.Error())
		}
	}
	return sout
}

func captureOut(t *testing.T, exe string, args ...string) string {
	showArgs(exe, args)
	cmd := exec.Command(exe, args...)
	out, err := cmd.Output()
	sout := string(out)
	if err != nil {
		if t != nil {
			t.Fatalf("%s\n%s", sout, err.Error())
		} else {
			log.Fatalf("%s\n%s", sout, err.Error())
		}
	}
	return sout
}

var imageName string

func publishManager(t *testing.T) {
	if imageName != "" {
		return
	}
	t.Helper()
	_ = os.Chdir("../../..") // ko must be executed from root to find the .ko.yaml config
	imageName = strings.TrimSpace(captureOut(t, "ko", "publish", "--local", "./cmd/traffic"))
	tag := fmt.Sprintf("%s/tel2:%s", registry, client.Version())
	capture(t, "docker", "tag", imageName, tag)
	capture(t, "docker", "push", tag)
}

func removeManager(t *testing.T) {
	// Remove service and deployment
	cmd := exec.Command("kubectl", "--kubeconfig", kubeconfig, "--namespace", namespace, "delete", "svc,deployment", "traffic-manager")
	_, _ = cmd.Output()

	// Wait until getting them fails
	gone := false
	for cnt := 0; cnt < 10; cnt++ {
		cmd = exec.Command("kubectl", "--kubeconfig", kubeconfig, "--namespace", namespace, "get", "deployment", "traffic-manager")
		if err := cmd.Run(); err != nil {
			gone = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !gone {
		t.Fatal("timeout waiting for deployment to vanish")
	}
	gone = false
	for cnt := 0; cnt < 10; cnt++ {
		cmd = exec.Command("kubectl", "--kubeconfig", kubeconfig, "--namespace", namespace, "get", "svc", "traffic-manager")
		if err := cmd.Run(); err != nil {
			gone = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !gone {
		t.Fatal("timeout waiting for service to vanish")
	}
}

func testContext() context.Context {
	return dlog.WithLogger(context.Background(), dlog.WrapLogrus(logrus.StandardLogger()))
}
func Test_findTrafficManager_notPresent(t *testing.T) {
	c := testContext()
	kc, err := newKCluster(kubeconfig, "", namespace)
	if err != nil {
		t.Fatal(err)
	}
	ti, err := newTrafficManagerInstaller(kc)
	if err != nil {
		t.Fatal(err)
	}
	version.Version = "v0.0.0-bogus"
	defer func() { version.Version = testVersion }()

	if _, err = ti.findDeployment(c, appName); err != nil {
		if !kates.IsNotFound(err) {
			t.Fatal(err)
		}
	}
	t.Fatal("expected find to return not-found error")
}

func Test_findTrafficManager_present(t *testing.T) {
	c := testContext()
	publishManager(t)
	defer removeManager(t)
	kc, err := newKCluster(kubeconfig, "", namespace)
	if err != nil {
		t.Fatal(err)
	}
	ti, err := newTrafficManagerInstaller(kc)
	if err != nil {
		t.Fatal(err)
	}
	err = ti.createManagerDeployment(c)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ti.findDeployment(c, appName)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_ensureTrafficManager_notPresent(t *testing.T) {
	c := testContext()
	publishManager(t)
	defer removeManager(t)
	kc, err := newKCluster(kubeconfig, "", namespace)
	if err != nil {
		t.Fatal(err)
	}
	ti, err := newTrafficManagerInstaller(kc)
	if err != nil {
		t.Fatal(err)
	}
	sshd, api, err := ti.ensureManager(c)
	if err != nil {
		t.Fatal(err)
	}
	if sshd != 8022 {
		t.Fatal("expected sshd port to be 8082")
	}
	if api != 8081 {
		t.Fatal("expected api port to be 8081")
	}
}
