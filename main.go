package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/cmd/util/editor"
	"sigs.k8s.io/yaml"
)

func main() {
	flagHelp := false
	flags := pflag.NewFlagSet("kubectl-edit-secret", pflag.ExitOnError)
	flags.BoolVarP(&flagHelp, "help", "h", flagHelp, "Print usage.")
	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "      %s [OPTIONS] SECRET_NAME\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "OPTIONS:\n")
		flags.PrintDefaults()
	}
	pflag.CommandLine = flags
	configFlags := genericclioptions.NewConfigFlags(true)
	configFlags.AddFlags(flags)
	pflag.Parse()

	if flagHelp {
		flags.Usage()
		return
	}
	args := flags.Args()
	if len(args) != 1 {
		flags.Usage()
		os.Exit(1)
	}

	secretName := args[0]
	namespace, err := flags.GetString("namespace")
	must(err, "Load namespace failed.")
	if namespace == "" {
		namespace = "default"
	}

	config, err := configFlags.ToRawKubeConfigLoader().ClientConfig()
	must(err, "Load kubeconfig failed.")
	clientset, err := kubernetes.NewForConfig(config)
	must(err, "Init kubeclient failed.")
	ctx := context.Background()

	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	must(err, "Fetch %s/%s failed.", namespace, secretName)

	secret, file, err := edit(secret)
	if err == nil {
		_, err = clientset.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	}
	if err == nil {
		fmt.Fprintf(os.Stdout, "%s/%s edited\n", namespace, secretName)
		return
	}
	if errors.Is(err, errNoContentChanged) {
		fmt.Fprintf(os.Stdout, "Edit cancelled, no changes made.\n")
		return
	}

	format := "Edit cancelled, no valid changes were saved.\n"
	if file != "" {
		format += "A copy of your changes has been stored to: %s.\n"
		must(err, format, file)
	} else {
		must(err, format)
	}
}

func must(err error, format string, args ...interface{}) {
	if err == nil {
		return
	}
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	format += "error: %v\n"

	fmt.Fprintf(os.Stderr, format, append(args, err)...)
	os.Exit(1)
}

var errNoContentChanged = errors.New("no content changed")

func edit(secret *corev1.Secret) (*corev1.Secret, string, error) {
	if secret.StringData == nil {
		secret.StringData = map[string]string{}
	}
	// Make secret visible.
	for key, value := range secret.Data {
		secret.StringData[key] = string(value)
	}
	secret.Data = nil

	data, err := yaml.Marshal(secret)
	if err != nil {
		return nil, "", err
	}
	edit := editor.NewDefaultEditor([]string{"KUBE_EDITOR", "EDITOR"})
	edited, file, err := edit.LaunchTempFile(fmt.Sprintf("%s-edit-", filepath.Base(os.Args[0])), ".yaml", bytes.NewBuffer(data))
	if err != nil {
		return nil, file, err
	}

	if bytes.Equal(cmdutil.StripComments(data), cmdutil.StripComments(edited)) {
		return nil, file, errNoContentChanged
	}

	editedSecret := &corev1.Secret{}
	if err := yaml.Unmarshal(edited, editedSecret); err != nil {
		return nil, file, err
	}
	if secret.ResourceVersion != editedSecret.ResourceVersion {
		return nil, file, errors.New("metadata.resourceVersion shouldn't be changed")
	}
	if secret.Namespace != editedSecret.Namespace || secret.Name != editedSecret.Name {
		return nil, file, errors.New("metadata.namespace or metadata.name shouldn't be changed")
	}
	return editedSecret, file, nil
}
