package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	_ "os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/heroku/docker-registry-client/registry"
	_ "golang.org/x/oauth2/google"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	httpaddress = ":8080"
)

func int32Ptr(i int32) *int32 { return &i }

func getApi(cr *gin.Context) {
	content, err := ioutil.ReadFile("uuid.txt")
	if err != nil {
		log.Println("getAPI error reading uuid")
		cr.String(http.StatusNotFound, "Error: %s", err.Error())
		return
	}
	log.Println("getAPI request received")

	//CORS
	cr.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	cr.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	cr.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	cr.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

	cr.JSON(http.StatusOK, gin.H{
		"Name":    "APIGW",
		"Version": string(content),
		"Date":    time.Now().String(),
	})
}

func getRepoImages(cr *gin.Context) {
	_, err := ioutil.ReadFile("uuid.txt")
	if err != nil {
		log.Println("getRepoImages error reading uuid")
		cr.String(http.StatusNotFound, "Error: %s", err.Error())
		return
	}
	log.Println("getRepoImages request received")
	url := "http://192.168.49.2:5000/"
	username := "" // anonymous
	password := "" // anonymous
	hub, err := registry.New(url, username, password)
	if err != nil {
		log.Println("getrepoimages")
		cr.String(http.StatusNotFound, "Error: %s", err.Error())
		return
	}

	repos, err := hub.Repositories()
	s := "[ "
	for id := range repos {
		if strings.Contains(repos[id][0:4], "iotd") {
			tags, _ := hub.Tags(repos[id])
			s = s + fmt.Sprintf(`{ "id" : "%d", "name" : "%s",`, id, repos[id])
			for id := range tags {
				manifest, _ := hub.Manifest(repos[id], tags[id])
				s = s + fmt.Sprintf(`"tag" : "%s", "arch": "%s" }`, tags[id], manifest.Architecture)
				break
			}
			s = s + ","
		}
	}
	s = s[:len(s)-1]
	s = s + " ]"
	if err != nil {
		cr.String(http.StatusNotFound, "Error: %s", err.Error())
		return
	}

	//CORS
	cr.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	cr.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	cr.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	cr.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

	cr.JSON(http.StatusOK, gin.H{"message": s})
}

func getList(cr *gin.Context) {
	config, err := clientcmd.BuildConfigFromFlags("", "kubeconfig")
	if err != nil {
		log.Println("getList error on kubeconfig file")
		cr.String(http.StatusNotFound, "Error: %s", err.Error())
		return
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Println("getList error on newconfig kubernetes")
		cr.String(http.StatusNotFound, "Error: %s", err.Error())
		return
	}

	deploymentsClient := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)

	list, err := deploymentsClient.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Println("getList error on listing deployments")
		cr.String(http.StatusNotFound, "Error: %s", err.Error())
		return
	}
	s := " { 'deployments': [ "
	for _, d := range list.Items {
		s = s + fmt.Sprintf("{ 'name' : %s, 'replicas' : %d },", d.Name, *d.Spec.Replicas)
	}
	s = s + "] }"
	log.Println("getList request received")

	//CORS
	cr.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	cr.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	cr.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	cr.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

	cr.JSON(http.StatusOK, gin.H{"message": s})
}

func create(cr *gin.Context) {
	config, err := clientcmd.BuildConfigFromFlags("", "kubeconfig")
	if err != nil {
		cr.String(http.StatusNotFound, "Error: %s", err.Error())
		return
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		cr.String(http.StatusNotFound, "Error: %s", err.Error())
		return
	}

	id := cr.Query("id")
	reg := cr.Query("reg")
	device := cr.Query("device")
	ec := cr.Query("ec")

	log.Printf("id:%s - reg:%s - device:%s - ec:%s", id, reg, device, ec)

	deploymentsClient := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: id,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": id,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": id,
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  id,
							Image: "localhost:5000/iot-linux-posix:v0.0.1",
							Env: []apiv1.EnvVar{
								{
									Name:  "REG",
									Value: reg,
								},
								{
									Name:  "DEVICE",
									Value: device,
								},
								{
									Name:  "EC",
									Value: ec,
								},
							},
							Command: []string{"/bin/sh"},
							Args:    []string{"-c", "echo \"-----BEGIN EC PRIVATE KEY-----\n$EC\n-----END EC PRIVATE KEY-----\" > ec_private.pem; ./iot_core_mqtt_client -p deviot2020-83f5d -d projects/deviot2020-83f5d/locations/us-central1/registries/$REG/devices/$DEVICE -t /devices/$DEVICE/state -f ec_private.pem"},
						},
					},
				},
			},
		},
	}

	result, err := deploymentsClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		cr.String(http.StatusNotFound, "Error: %s", err.Error())
		return
	}
	s := fmt.Sprintf("%q\n", result.GetObjectMeta().GetName())

	//CORS
	cr.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	cr.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	cr.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	cr.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

	cr.JSON(http.StatusOK, gin.H{"message": s})
}

func delete(cr *gin.Context) {
	config, err := clientcmd.BuildConfigFromFlags("", "kubeconfig")
	if err != nil {
		log.Println("getList error on kubeconfig file")
		cr.String(http.StatusNotFound, "Error: %s", err.Error())
		return
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Println("getList error on newconfig kubernetes")
		cr.String(http.StatusNotFound, "Error: %s", err.Error())
		return
	}
	deploymentsClient := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)
	id := cr.Query("id")
	deletePolicy := metav1.DeletePropagationForeground
	err2 := deploymentsClient.Delete(context.TODO(), id, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
	if err2 != nil {
		log.Println("delete deployment error")
		cr.String(http.StatusNotFound, "Error: %s", err2.Error())
		return
	}

	//CORS
	cr.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	cr.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	cr.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	cr.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

	cr.JSON(http.StatusOK, gin.H{"message": id})
}

func init() {
	log.SetPrefix("TRACE: ")
	log.SetFlags(log.Ldate | log.Lmicroseconds | log.Llongfile)
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	v1 := router.Group("/")
	{
		v1.GET("/api", getApi)
		v1.GET("/repoimages", getRepoImages)
		v1.GET("/list", getList)
		v1.GET("/create", create)
		v1.GET("/delete", delete)
	}

	log.Println("APIGW running on", httpaddress)
	router.Run(httpaddress)
}
