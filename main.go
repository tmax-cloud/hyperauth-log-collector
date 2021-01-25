package main

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/robfig/cron"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/klog"
)

var Clientset *kubernetes.Clientset
var config *restclient.Config

func init() {

	// var kubeconfig *string
	// if home := homedir.HomeDir(); home != "" {
	// 	kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "/root/.kube")
	// } else {
	// 	kubeconfig = flag.String("kubeconfig", "", "/root/.kube")
	// }
	// flag.Parse()

	// var err error
	// config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	// if err != nil {
	// 	klog.Errorln(err)
	// 	panic(err)
	// }
	// config.Burst = 100
	// config.QPS = 100
	// Clientset, err = kubernetes.NewForConfig(config)
	// if err != nil {
	// 	klog.Errorln(err)
	// 	panic(err)
	// }

	// If api-server on POD, activate below code and delete above
	// creates the in-cluster config
	var err error
	config, err = restclient.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	config.Burst = 100
	config.QPS = 100
	Clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

}

func main() {
	klog.Info("Hyperauth Log Collector Start!!")

	// Logging Cron Job
	cronJob := cron.New()
	cronJob.AddFunc("1 0 0 * * ?", func() { //every day 0am
		// cronJob.AddFunc("@every 1m", func() { //test code
		now := time.Now()
		klog.Info("Hyperauth Log Collector Start to Collect, " + "Current Time : " + now.Format("2006-01-02 15:04:05"))
		nsName := os.Getenv("NAMESPACE")
		klog.Info("nsName : " + nsName)

		//Get Hyperauth Pod Name
		hyperauthPodName := getPodNameWithLabel(nsName, "app", "hyperauth")

		// Call k8s readPodLogs
		podLog := getPodLog(nsName, hyperauthPodName)

		input := []byte(podLog)
		err := ioutil.WriteFile("./logs/hyperauth_"+time.Now().AddDate(0, 0, -1).Format("2006-01-02")+".log", input, 0644)
		// err := ioutil.WriteFile("./logs/hyperauth_"+time.Now().Add(time.Minute*(-1)).Format("2006-01-02 15:04:05")+".log", input, 0644) //test code

		if err != nil {
			klog.Error(err)
			return
		}
		klog.Info("Log BackUp Success, " + "Current Time : " + now.Format("2006-01-02 15:04:05"))
	})
	cronJob.Start()

	// for infinite loop
	for {
	}

	// fmt.Scanln()
	// mux := http.NewServeMux()
	// if err := http.ListenAndServe(":80", mux); err != nil {
	// 	klog.Errorf("Failed to listen and serve Hypercloud5-API server: %s", err)
	// }

}

func getPodLog(nsName string, podName string) string {
	podLogOpts := v1.PodLogOptions{
		Container: "hyperauth", // in case, Collector is in same pod with hyperauth
		// Follow: true,
		SinceTime: &metav1.Time{
			time.Now().AddDate(0, 0, -1),
			// time.Now().Add(time.Minute * (-1)), //test code
		},
	}

	podLogReq := Clientset.CoreV1().Pods(nsName).GetLogs(podName, &podLogOpts)
	podLogs, err := podLogReq.Stream(context.TODO())
	if err != nil {
		klog.Error(err, "Error creating", "./logs/api-server")
		return "error in opening stream"
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "error in copy information from podLogs to buf"
	}
	str := buf.String()
	klog.Info(" << Hyperauth Log >>")
	klog.Info(str)
	return str
}

func getPodNameWithLabel(nsName string, key string, value string) string {
	// labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{key: value}}
	pod, _ := Clientset.CoreV1().Pods(nsName).List(
		context.TODO(),
		metav1.ListOptions{
			// LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
		},
	)
	// klog.Info("PODs : " + pod.String())
	// for _, item := range pod.Items {
	// 	klog.Info("pod item : " + item.Name)
	// }

	podName := pod.Items[0].ObjectMeta.Name
	klog.Info("Hyperauth PodName : " + podName)
	return podName
}
