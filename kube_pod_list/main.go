package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"time"

	//"github.com/kr/pretty"
	flag "github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type CommandLineOptions struct {
	KubeConfig string
	Verbose    bool
}

func GetCmdOptOrDie() *CommandLineOptions {
	opt := CommandLineOptions{}

	configPathDefault := ""
	if home := homedir.HomeDir(); home != "" {
		configPathDefault = filepath.Join(home, ".kube", "config")
	}

	flag.StringVar(&opt.KubeConfig, "kubeconfig", configPathDefault,
		"path to kubectl config file to use",
	)
	flag.BoolVarP(&opt.Verbose, "verbose", "v", false, "log detailed info")

	flag.Parse()

	return &opt
}

type PodData struct {
	Name           string
	RestartCount   int32
	StartTime      time.Time
	ContainersWant int
	ContainersGot  int
	PodIP          string
	NodeName       string
	Phase          string
}

type PodReport struct {
	PodFormat string
	Pods      []*PodData
}

func NewPodReport(items []corev1.Pod) *PodReport {
	report := &PodReport{
		Pods: make([]*PodData, 0, len(items)),
	}

	for _, item := range items {
		var restartCount int32
		if len(item.Status.ContainerStatuses) > 0 {
			restartCount = item.Status.ContainerStatuses[0].RestartCount
		}

		podData := &PodData{
			Name:           item.ObjectMeta.Name,
			RestartCount:   restartCount,
			StartTime:      item.Status.StartTime.Time,
			Phase:          string(item.Status.Phase),
			PodIP:          item.Status.PodIP,
			NodeName:       item.Spec.NodeName,
			ContainersWant: len(item.Status.ContainerStatuses),
		}

		for _, cstatus := range item.Status.ContainerStatuses {
			if cstatus.Ready {
				podData.ContainersGot++
			}
		}

		report.Pods = append(report.Pods, podData)
	}

	report.SetReportFormat()
	report.SortReport()
	return report
}

func (report *PodReport) SortReport() {
	sort.SliceStable(report.Pods, func(i, j int) bool {
		return report.Pods[j].StartTime.Before(report.Pods[i].StartTime)
	})
}

func (report *PodReport) SetReportFormat() {
	maxNameLen := 0
	maxNodeLen := 0
	maxPodIpLen := 0
	maxPhaseLen := 0
	maxDurationLen := 0
	for _, item := range report.Pods {
		if len(item.Name) > maxNameLen {
			maxNameLen = len(item.Name)
		}
		if len(item.NodeName) > maxNodeLen {
			maxNodeLen = len(item.NodeName)
		}
		if len(item.PodIP) > maxPodIpLen {
			maxPodIpLen = len(item.PodIP)
		}
		if len(item.Phase) > maxPhaseLen {
			maxPhaseLen = len(item.Phase)
		}
		ct := DurationString(time.Since(item.StartTime))
		if len(ct) > maxDurationLen {
			maxDurationLen = len(ct)
		}
	}
	report.PodFormat = fmt.Sprintf("%%-%ds %%d/%%d Restarts: %%3d Start: %%%dv %%-%ds %%-%ds %%-%ds\n",
		maxNameLen,
		maxDurationLen,
		maxPhaseLen,
		maxPodIpLen,
		maxNodeLen,
	)
}

func (report *PodReport) String() string {
	result := []byte{}
	for _, pod := range report.Pods {
		result = append(result, fmt.Sprintf(report.PodFormat,
			pod.Name,
			pod.ContainersGot,
			pod.ContainersWant,
			pod.RestartCount,
			DurationString(time.Since(pod.StartTime)),
			pod.Phase,
			pod.PodIP,
			pod.NodeName,
		)...)
	}
	return string(result)
}

func DurationString(d time.Duration) string {
	days := int(d.Hours()) / 24
	d = d - time.Duration(days)*24*time.Hour

	hours := int(d.Hours())
	d = d - time.Duration(hours)*time.Hour

	minutes := int(d.Minutes())
	d = d - time.Duration(minutes)*time.Minute

	seconds := int(d.Seconds())

	return fmt.Sprintf("%dd:%02dh:%02dm:%02ds", days, hours, minutes, seconds)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	opt := GetCmdOptOrDie()

	if opt.Verbose {
		log.Printf("OPT: %+v", opt)
	}

	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: opt.KubeConfig}, nil)

	clientconfig, err := config.ClientConfig()

	if err != nil {
		log.Fatalf("kubeconfig load error: %v", err)
	}

	ns, _, err := config.Namespace()
	if err != nil {
		log.Fatalf("namespace error: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(clientconfig)
	if err != nil {
		log.Fatalf("NewForConfig error: %v", err)
	}

	pods, err := clientset.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Pods(%s) list error: %v", ns, err)
	}

	if opt.Verbose {
		log.Printf("There are %d pods in the cluster\n", len(pods.Items))
	}

	//log.Printf("pod: %s", pretty.Sprint(pods.Items[0]))

	report := NewPodReport(pods.Items)
	fmt.Print(report.String())
}
