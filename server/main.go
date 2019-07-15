package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/ericchiang/k8s"
	corev1 "github.com/ericchiang/k8s/apis/core/v1"
	"github.com/etcd-io/etcd/pkg/fileutil"
	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/rs/cors"
	"github.com/xumc/dockerdaydayup/server/proto"
	"google.golang.org/grpc"
	"io/ioutil"
	"log"
	"mime"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
)

var k8sClient *k8s.Client

func loadClient(kubeconfigPath string) (*k8s.Client, error) {
	data, err := ioutil.ReadFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("read kubeconfig: %v", err)
	}

	// Unmarshal YAML into a Kubernetes config object.
	var config k8s.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("unmarshal kubeconfig: %v", err)
	}
	return k8s.NewClient(&config)
}

type Server struct {
}

var tManager = NewTeleManager()

func (*Server) DigOut(ctx context.Context, req *proto.DigOutRequest) (*empty.Empty, error) {
	tele := NewTele(req.ServiceName, "8888", "80")

	if err := tManager.Add(tele); err != nil {
		log.Fatal("tManager add: ", err)
	}

	return &empty.Empty{}, nil
}

func (*Server) DeDigOut(ctx context.Context, req *proto.DigOutRequest) (*empty.Empty, error) {
	tele, err := tManager.FindByServiceName(req.ServiceName)
	if err == ServiceNotFoundErr {
		return nil, err
	}

	if err := tManager.Remove(tele); err != nil {
		log.Fatal("tManager Remove: ", err)
	}

	return &empty.Empty{}, nil
}

func (*Server) GetServices(context.Context, *empty.Empty) (*proto.ServicesReply, error) {
	var services corev1.ServiceList
	if err := k8sClient.List(context.Background(), "default", &services); err != nil {
		log.Fatalln(err)
		return nil, err
	}

	pServices := make([]*proto.Service, len(services.Items))
	for i, s := range services.Items {
		serviceName := s.GetMetadata().GetName()

		// find pods
		pods, err := getPodsOfService(serviceName)
		if err != nil {
			return nil, err
		}

		digOutStatus := getDigoutStatus(serviceName, pods)

		pServices[i] = &proto.Service{
			Id:   s.GetMetadata().GetUid(),
			Name: serviceName,
			DigoutStatus: digOutStatus,
		}
	}

	return &proto.ServicesReply{
		Items: pServices,
	}, nil
}

type StaticHandler struct {
	mux http.Handler
}

func (handler *StaticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	filePath := fmt.Sprintf("client/build%s", r.URL.Path)
	if !fileutil.Exist(filePath) {
		handler.mux.ServeHTTP(w, r)
		return
	}

	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatal(err)
	}

	ext := filepath.Ext(filePath)
	if ext != "" {
		w.Header().Set("Content-Type", mime.TypeByExtension(ext))
	}

	if _, err := w.Write(file); err != nil {
		log.Fatal(err)
	}
}

func main() {
	//client, err := k8s.NewInClusterClient()
	client, err := loadClient("/Users/xumc/.kube/config")
	if err != nil {
		log.Fatal(err)
	}
	k8sClient = client

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	muxOption := runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{OrigName: true, EmitDefaults: true})
	mux := runtime.NewServeMux(muxOption)
	opts := []grpc.DialOption{grpc.WithInsecure()}
	grpcServerEndpoint := "localhost:9090" // TODO
	err = proto.RegisterDdduServiceHandlerFromEndpoint(ctx, mux, grpcServerEndpoint, opts)
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()
	proto.RegisterDdduServiceServer(s, &Server{})

	grpcLis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		err = s.Serve(grpcLis)
		if err != nil {
			panic(fmt.Sprintf("can not serve %s", err.Error()))
		}

		wg.Done()
	}()

	wg.Add(1)
	go func() {

		staticHandler := &StaticHandler{
			mux: mux,
		}

		// TODO remove cors in production mode
		handler := cors.Default().Handler(staticHandler)

		log.Fatal(http.ListenAndServe(":8081", handler))

		wg.Done()
	}()

	wg.Wait() // TODO gracefully kill
}

func discover(client *k8s.Client) {
	fmt.Println("nodes:")
	var nodes corev1.NodeList
	if err := client.List(context.Background(), "", &nodes); err != nil {
		log.Fatal(err)
	}
	for _, node := range nodes.Items {
		fmt.Printf("name=%q schedulable=%t\n", *node.Metadata.Name, !*node.Spec.Unschedulable)
	}

	fmt.Println("pods:")
	var pods corev1.PodList
	if err := client.List(context.Background(), "default", &pods); err != nil {
		log.Fatalln(err)
	}
	for _, pod := range pods.Items {
		fmt.Printf("name=%q\n", *pod.Metadata.Name)
	}

	fmt.Println("services:")
	var services corev1.ServiceList
	if err := client.List(context.Background(), "default", &services); err != nil {
		log.Fatalln(err)
	}
	for _, s := range services.Items {
		fmt.Printf("name=%q\n", *s.Metadata.Name)
	}
}

const sudoPwd = "hello\n" // TODO

type tele struct {
	cmd *exec.Cmd

	serviceName string
	localPort   string
	remotePort  string
}

func NewTele(serviceName, localPort, remotePort string) *tele {
	return &tele{
		serviceName: serviceName,
		localPort:   localPort,
		remotePort:  remotePort,
	}
}

func (t *tele) run() error {
	path, err := exec.LookPath("telepresence")
	if err != nil {
		log.Fatal("LookPath: ", err)
		return err
	}

	args := []string{"-S", path, "--swap-deployment", t.serviceName, "--expose", fmt.Sprintf("%s:%s", t.localPort, t.remotePort), "--run", "bash", "--login"}
	t.cmd = exec.Command("sudo", args...)

	stdin, err := t.cmd.StdinPipe()
	if err != nil {
		log.Fatal("stdinPipe: ", err)
		return err
	}
	defer stdin.Close()
	_, err = stdin.Write([]byte(sudoPwd))
	if err != nil {
		log.Fatal("write to stdin: ", err)
		return err
	}

	err = t.cmd.Start()
	if err != nil {
		log.Fatal("cmd start: ", err)
		return err
	}

	err = t.cmd.Wait()
	if err != nil {
		log.Fatal("cmd wait: ", err)
		return err
	}

	return nil

}

func (t tele) kill() error {
	args := []string{"-S", "kill", strconv.Itoa(t.cmd.Process.Pid)}
	cmd := exec.Command("sudo", args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal("kill stdinPipe: ", err)
		return err
	}
	defer stdin.Close()
	_, err = stdin.Write([]byte(sudoPwd))
	if err != nil {
		log.Fatal("kill, write to stdin: ", err)
		return err
	}

	if err = cmd.Run(); err != nil {
		log.Fatal("kill run:", err)
		return err
	}

	return nil
}

type teleManager struct {
	teles []*tele

	sudoPwd string
}

func NewTeleManager() *teleManager {
	return &teleManager{
		teles:   make([]*tele, 0),
		sudoPwd: sudoPwd,
	}
}

func (tm *teleManager) Add(t *tele) error {
	go t.run()

	tm.teles = append(tm.teles, t)

	return nil
}

func (tm *teleManager) Remove(t *tele) error {
	index := sort.Search(len(tm.teles), func(i int) bool {
		return t.serviceName == tm.teles[i].serviceName
	})

	// TODO what if not found

	if err := t.kill(); err != nil {
		return err
	}

	tm.teles = append(tm.teles[:index], tm.teles[index+1:]...)

	return nil
}

var ServiceNotFoundErr = errors.New("service not found error")

func (tm *teleManager) FindByServiceName(s string) (*tele, error) {
	l := len(tm.teles)

	index := sort.Search(l, func(i int) bool {
		return tm.teles[i].serviceName == s
	})

	if index == l {
		return nil, ServiceNotFoundErr
	}

	return tm.teles[index], nil
}

func getPodsOfService(serviceName string) (*corev1.PodList, error) {
	l := new(k8s.LabelSelector)
	l.Eq("app.kubernetes.io/name", serviceName)

	var pods corev1.PodList
	if err := k8sClient.List(context.Background(), "default", &pods, l.Selector()); err != nil {
		log.Fatalln(err)
		return nil, err
	}

	return &pods, nil
}

func getDigoutStatus(serviceName string, pods *corev1.PodList) proto.DigOutStatus {
	var digOutStatus proto.DigOutStatus
	digOutStatus = proto.DigOutStatus_Unknown
	switch len(pods.Items) {
	case 1:
		_, ok := pods.Items[0].GetMetadata().GetLabels()["telepresence"]
		if ok {
			digOutStatus = proto.DigOutStatus_Open
		} else {
			digOutStatus = proto.DigOutStatus_Closed
		}

	case 2:
		for _, pod := range pods.Items {
			cstatus := getMainContainerStatus(serviceName, pod)

			if cstatus == "running" {
				_, isTeleContainer := pod.GetMetadata().GetLabels()["telepresence"]
				if isTeleContainer {
					digOutStatus = proto.DigOutStatus_Open
					break
				} else {
					digOutStatus = proto.DigOutStatus_Closed
					break
				}
			} else {
				continue
			}
		}

		// TODO detailing status logic
		default:
	}

	return digOutStatus
}

func getMainContainerStatus(serviceName string, pod *corev1.Pod) string {
	for _, cs := range pod.GetStatus().GetContainerStatuses() {
		if cs.GetName() == serviceName {
			return cs.GetState().String()
		}
	}

	return "unknown"
}
