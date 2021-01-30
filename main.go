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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/klog"
)

var Clientset *kubernetes.Clientset
var config *restclient.Config

func init() {
	// Creates the in-cluster config
	var err error
	config, err = restclient.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// Creates the clientset
	config.Burst = 100
	config.QPS = 100
	Clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
}

func main() {
	klog.Info("Hyperauth Log Collector Start!!")
	file, _ := os.OpenFile(
		"./logs/hyperauth.log",
		os.O_CREATE|os.O_RDWR|os.O_TRUNC,
		os.FileMode(0644),
	)
	defer file.Close()

	cronJobTail := cron.New()
	cronJobTail.AddFunc("@every 5s", func() { //test code
		nsName := os.Getenv("NAMESPACE")
		klog.Info("nsName : " + nsName)

		//Get Hyperauth Pod Name
		hyperauthPodName := getPodNameWithLabel(nsName, "app", "hyperauth")

		// Call k8s readPodLogs
		logString := getPodLog(nsName, hyperauthPodName)
		_, err := file.WriteString(logString)
		if err != nil {
			klog.Error(err)
		}
	})
	cronJobTail.Start()

	////////////////////////////////////////////////////////

	cronJobCollect := cron.New()
	// cronJobCollect.AddFunc("@every 1m", func() { //test code
	cronJobCollect.AddFunc("1 0 0 * * ?", func() { //every day 0am
		now := time.Now()
		klog.Info("Hyperauth Log Collector Start to Collect, " + "Current Time : " + now.Format("2006-01-02 15:04:05"))
		input, err := ioutil.ReadFile("./logs/hyperauth.log")
		if err != nil {
			klog.Error(err)
			return
		}

		err = ioutil.WriteFile("./logs/hyperauth_"+time.Now().AddDate(0, 0, -1).Format("2006-01-02")+".log", input, 0644)
		// err = ioutil.WriteFile("./logs/hyperauth_"+time.Now().Add(time.Minute*(-1)).Format("2006-01-02 15:04:05")+".log", input, 0644) //test code

		if err != nil {
			klog.Error(err)
			return
		}
		klog.Info("Log BackUp Success, " + "Current Time : " + now.Format("2006-01-02 15:04:05"))
		os.Truncate("./logs/hyperauth.log", 0)
		file.Seek(0, os.SEEK_SET)
	})
	cronJobCollect.Start()

	// for infinite loop
	for {
	}
}

func getPodLog(nsName string, podName string) string {

	podLogOpts := v1.PodLogOptions{
		Container: "hyperauth", // in case, Collector is in same pod with hyperauth
		// Follow:    true,
		SinceTime: &metav1.Time{
			time.Now().Add(time.Second * (-5)), //test code
		},
	}

	podLogReq := Clientset.CoreV1().Pods(nsName).GetLogs(podName, &podLogOpts)

	podLogs, err := podLogReq.Stream(context.TODO())
	if err != nil {
		klog.Error(err, "Error Getting pod log")
		return "error"
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		klog.Error(err, "Error Copy PodLog")
		return "error"
	}
	str := buf.String()
	klog.Info(" << Hyperauth Log " + time.Now().Format("2006-01-02 15:04:05") + " >>")
	klog.Info(str)
	return str
}

func getPodNameWithLabel(nsName string, key string, value string) string {
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{key: value}}
	pod, _ := Clientset.CoreV1().Pods(nsName).List(
		context.TODO(),
		metav1.ListOptions{
			LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
		},
	)
	for _, item := range pod.Items {
		klog.Info("pod item : " + item.Name)
	}

	podName := pod.Items[0].ObjectMeta.Name
	klog.Info("Hyperauth PodName : " + podName)
	return podName
}
