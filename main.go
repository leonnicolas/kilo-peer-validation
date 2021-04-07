package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	kilo "github.com/squat/kilo/pkg/k8s/apis/kilo/v1alpha1"
	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	certPath    *string = flag.String("cert-file", "", "file path to certificat file")
	keyPath     *string = flag.String("key-file", "", "file path to key file")
	metricsAddr *string = flag.String("metrics-address", ":9090", "The metrics server will be listening to that address with port\ne.g. 172.0.0.1:9090")
)

var deserializer = serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()

var (
	validationCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "https_admission_requests_total",
			Help: "The number of received admission reviews requests",
		},
		[]string{"operation", "response"},
	)
	errorCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "errors_total",
			Help: "The total number of errors",
		},
	)
)

func validationHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Handling request from %s\n", r.RemoteAddr)
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errorCounter.Inc()
		log.Printf("failed to parse body from incoming request from %s\n", r.RemoteAddr)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var admissionReview v1.AdmissionReview

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		errorCounter.Inc()
		msg := fmt.Sprintf("Content-Type=%s, expect application/json", contentType)
		log.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	response := v1.AdmissionReview{}

	_, gvk, err := deserializer.Decode(body, nil, &admissionReview)
	if err != nil {
		errorCounter.Inc()
		msg := fmt.Sprintf("Request could not be decoded: %v", err)
		log.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	if *gvk != v1.SchemeGroupVersion.WithKind("AdmissionReview") {
		errorCounter.Inc()
		log.Println("only api v1 is supported")
		http.Error(w, "only api v1 is supported", http.StatusBadRequest)
		return
	} else {
		response.SetGroupVersionKind(*gvk)
		response.Response = &v1.AdmissionResponse{
			UID: admissionReview.Request.UID,
		}
	}

	rawExtension := admissionReview.Request.Object
	var peer kilo.Peer

	if err = json.Unmarshal(rawExtension.Raw, &peer); err != nil {
		errorCounter.Inc()
		msg := fmt.Sprintf("could not unmarshal extension to peer spec: %v:", err)
		log.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	if err := peer.Validate(); err == nil {
		validationCounter.With(prometheus.Labels{"operation": string(admissionReview.Request.Operation), "response": "allowed"}).Inc()
		response.Response.Allowed = true
	} else {
		validationCounter.With(prometheus.Labels{"operation": string(admissionReview.Request.Operation), "response": "denied"}).Inc()
		response.Response.Result = &metav1.Status{
			Message: err.Error(),
		}
	}

	res, err := json.Marshal(response)
	if err != nil {
		errorCounter.Inc()
		msg := fmt.Sprintf("failed to marshal response: %v", err)
		log.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(res); err != nil {
		log.Printf("failed to write response: %v\n", err)
	}
}

func main() {
	flag.Parse()

	r := prometheus.NewRegistry()
	r.MustRegister(
		errorCounter,
		validationCounter,
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)
	mm := http.NewServeMux()
	mm.Handle("/metrics", promhttp.HandlerFor(r, promhttp.HandlerOpts{}))
	go http.ListenAndServe(*metricsAddr, mm)

	mux := http.NewServeMux()
	mux.HandleFunc("/validate", validationHandler)
	server := &http.Server{
		Addr:    ":8443",
		Handler: mux,
	}
	log.Fatal(server.ListenAndServeTLS(*certPath, *keyPath))
}
