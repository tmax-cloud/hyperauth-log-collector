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
	file1, _ := os.OpenFile(
		"./logs/hyperauth_1.log",
		os.O_CREATE|os.O_RDWR,
		os.FileMode(0644),
	)
	defer file1.Close()

	file2, _ := os.OpenFile(
		"./logs/hyperauth_2.log",
		os.O_CREATE|os.O_RDWR,
		os.FileMode(0644),
	)
	defer file2.Close()

	cronJobTail := cron.New()
	cronJobTail.AddFunc("@every 5s", func() {
		nsName := os.Getenv("NAMESPACE")
		klog.Info("nsName : " + nsName)
		key := "app"
		value := "hyperauth"
		labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{key: value}}
		pod, _ := Clientset.CoreV1().Pods(nsName).List(
			context.TODO(),
			metav1.ListOptions{
				LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
			},
		)

		if pod.Items != nil && len(pod.Items) > 0 {
			podName1 := pod.Items[0].Name
			klog.Info("Hyperauth Pod 1 Name : " + podName1)
			logString1 := getPodLog(nsName, podName1)
			_, err := file1.WriteString(logString1)
			if err != nil {
				klog.Error(err)
			}
			if len(pod.Items) > 1 {
				podName2 := pod.Items[1].Name
				klog.Info("Hyperauth Pod 2 Name : " + podName2)
				logString2 := getPodLog(nsName, podName2)
				_, err = file2.WriteString(logString2)
				if err != nil {
					klog.Error(err)
				}
			}
		}
	})
	cronJobTail.Start()

	////////////////////////////////////////////////////////

	cronJobCollect := cron.New()
	// cronJobCollect.AddFunc("@every 1m", func() { //test code
	cronJobCollect.AddFunc("1 0 0 * * ?", func() { //every day 0am
		now := time.Now()
		klog.Info("Hyperauth Log Collector Start to Collect, " + "Current Time : " + now.Format("2006-01-02 15:04:05"))
		input1, err1 := ioutil.ReadFile(file1.Name())
		if err1 != nil {
			klog.Error(err1)
			return
		}
		os.Mkdir("./logs/1", 0755)

		input2, err2 := ioutil.ReadFile(file2.Name())
		if err2 != nil {
			klog.Error(err2)
			return
		}
		os.Mkdir("./logs/2", 0755)

		err1 = ioutil.WriteFile("./logs/1/hyperauth_1_"+now.AddDate(0, 0, -1).Format("2006-01-02")+".log", input1, 0644)
		err2 = ioutil.WriteFile("./logs/2/hyperauth_2_"+now.AddDate(0, 0, -1).Format("2006-01-02")+".log", input2, 0644)

		// err1 = ioutil.WriteFile("./logs/1/hyperauth_1_"+now.Add(time.Minute*(-1)).Format("2006-01-02 15:04:05")+".log", input1, 0644) //test code
		// err2 = ioutil.WriteFile("./logs/2/hyperauth_2_"+now.Add(time.Minute*(-1)).Format("2006-01-02 15:04:05")+".log", input2, 0644) //test code

		if err1 != nil {
			klog.Error(err1)
			return
		}
		if err2 != nil {
			klog.Error(err2)
			return
		}
		klog.Info("Log BackUp Success Write file , 1/hyperauth_1_" + now.AddDate(0, 0, -1).Format("2006-01-02") + ".log , Current Time : " + now.Format("2006-01-02 15:04:05"))
		klog.Info("Log BackUp Success Write file , 2/hyperauth_2_" + now.AddDate(0, 0, -1).Format("2006-01-02") + ".log , Current Time : " + now.Format("2006-01-02 15:04:05"))

		file1.Truncate(0)
		file2.Truncate(0)

		// os.Truncate("./logs/hyperauth.log", 0)
		file1.Seek(0, os.SEEK_SET)
		file2.Seek(0, os.SEEK_SET)
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
