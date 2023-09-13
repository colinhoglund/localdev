package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/coredns/corefile-migration/migration/corefile"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	kindcluster "sigs.k8s.io/kind/pkg/cluster"
	kindcmd "sigs.k8s.io/kind/pkg/cmd"
)

type kindCommand struct {
	k8sVersion  string
	configFile  string
	clusterName string
	domain      string
}

func newKindCommand() *cobra.Command {
	kind := &kindCommand{}

	cmd := &cobra.Command{
		Use:   "kind",
		Short: "Tools for building local kind clusters",
	}

	cmd.PersistentFlags().StringVar(&kind.clusterName, "cluster-name", "kind", "kind cluster name")

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Bootstrap a local kind cluster",
		RunE:  kind.startE,
	}

	startCmd.Flags().StringVar(&kind.configFile, "config-file", "", "kind cluster configuration file (optional)")
	startCmd.Flags().StringVar(&kind.k8sVersion, "k8s-version", "", "k8s version (default: latest)")

	delCommand := &cobra.Command{
		Use:   "delete",
		Short: "Delete a local kind cluster",
		RunE:  kind.deleteE,
	}

	patchCommand := &cobra.Command{
		Use:   "patch-coredns <NAMESPACE> <SERVICE>",
		Short: "Patches the default coredns Corefile to forward DNS queries to a custom DNS service",
		Args:  cobra.ExactArgs(2),
		RunE:  kind.patchE,
	}

	patchCommand.Flags().StringVar(&kind.domain, "domain", "example.com", "Custom domain name")

	cmd.AddCommand(startCmd, delCommand, patchCommand)

	return cmd
}

func (k *kindCommand) startE(cmd *cobra.Command, args []string) error {
	kubeconfig := k.kubeconfigOrDie()

	opts := []kindcluster.CreateOption{
		kindcluster.CreateWithKubeconfigPath(kubeconfig),
		kindcluster.CreateWithNodeImage(fmt.Sprintf("kindest/node:%s", k.k8sVersion)),
	}

	if k.configFile != "" {
		opts = append(opts, kindcluster.CreateWithConfigFile(k.configFile))
	}

	provider := k.providerPodman()

	if err := provider.Create(k.clusterName, opts...); err != nil {
		return err
	}

	return provider.ExportKubeConfig(k.clusterName, kubeconfig, false)
}

func (k *kindCommand) deleteE(cmd *cobra.Command, args []string) error {
	return k.providerPodman().Delete(k.clusterName, k.kubeconfigOrDie())
}

func (k *kindCommand) patchE(cmd *cobra.Command, args []string) error {
	forwardServiceNamespace := args[0]
	forwardServiceName := args[1]
	forwardServerKey := fmt.Sprintf("%s:53", k.domain)

	ctx := context.TODO()
	clientset := k.kubeClientsetOrDie()

	corednsConfigmap, err := clientset.CoreV1().ConfigMaps("kube-system").Get(ctx, "coredns", metav1.GetOptions{})
	if err != nil {
		return err
	}

	corefileData, ok := corednsConfigmap.Data["Corefile"]
	if !ok {
		return fmt.Errorf("no corefile found")
	}

	corednsCorefile, err := corefile.New(corefileData)
	if err != nil {
		return err
	}

	for _, server := range corednsCorefile.Servers {
		if slices.Contains(server.DomPorts, forwardServerKey) {
			// skip patching
			return nil
		}
	}

	forwardService, err := clientset.CoreV1().Services(forwardServiceNamespace).Get(ctx, forwardServiceName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	ipaddr := forwardService.Spec.ClusterIP
	if ipaddr == "" {
		return fmt.Errorf("ClusterIP is empty for %s/%s", forwardServiceNamespace, forwardServiceName)
	}

	forwardServer := &corefile.Server{
		DomPorts: []string{forwardServerKey},
		Plugins: []*corefile.Plugin{
			{Name: "errors"},
			{Name: "cache", Args: []string{"30"}},
			{Name: "forward", Args: []string{".", ipaddr}},
		},
	}

	corednsCorefile.Servers = append(corednsCorefile.Servers, forwardServer)
	corednsConfigmap.Data["Corefile"] = corednsCorefile.ToString()

	if _, err := clientset.CoreV1().ConfigMaps("kube-system").Update(ctx, corednsConfigmap, metav1.UpdateOptions{}); err != nil {
		return err
	}

	return nil
}

func (k *kindCommand) providerPodman() *kindcluster.Provider {
	return kindcluster.NewProvider(kindcluster.ProviderWithLogger(kindcmd.NewLogger()), kindcluster.ProviderWithPodman())
}

func (k *kindCommand) kubecontext() string {
	return fmt.Sprintf("kind-%s", k.clusterName)
}

func (k *kindCommand) kubeconfigOrDie() string {
	userHome, err := homedir.Dir()
	if err != nil {
		panic(err)
	}

	return filepath.Join(userHome, ".kube", fmt.Sprintf("config.%s", k.kubecontext()))
}

func (k *kindCommand) kubeClientsetOrDie() kubernetes.Interface {
	k8sFlags := genericclioptions.NewConfigFlags(true)
	k8sFlags.KubeConfig = ptr.To(k.kubeconfigOrDie())
	k8sFlags.Context = ptr.To(k.kubecontext())

	restConfig, err := k8sFlags.ToRESTConfig()
	if err != nil {
		panic(err)
	}

	return kubernetes.NewForConfigOrDie(restConfig)
}
