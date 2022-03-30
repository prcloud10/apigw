package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "os"
	"time"

	"github.com/gin-gonic/gin"
	_ "golang.org/x/oauth2/google"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
)

const (
	httpaddress = ":8081"
)

func int32Ptr(i int32) *int32 { return &i }

func getApi(cr *gin.Context) {

	//CORS
	cr.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	cr.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	cr.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	cr.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

	cr.JSON(http.StatusOK, gin.H{
		"Name":    "APIGW",
		"Version": "0.1",
		"Date":    time.Now().String(),
	})
}

func getList(cr *gin.Context) {

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Println("getList error on newconfig kubernetes")
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
		v1.GET("/list", getList)
	}

	log.Println("APIGW running on", httpaddress)
	router.Run(httpaddress)
}
