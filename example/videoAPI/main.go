package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/xumc/dockerdaydayup/example/videoapi/proto"
	"google.golang.org/grpc"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Video struct {
	ID   int64    `json:"id"`
	Name string `json:"name"`
	ViewCount int64
}

type HttpError struct {
	Code int `json:"code"`
	Message string `json:"message"`
}

func handleCheckAlive(w http.ResponseWriter, _ *http.Request){
	fmt.Fprintln(w,"service ok")
}

func handleVideos(w http.ResponseWriter, r *http.Request){
	host, err := os.Hostname()
	if err != nil {
		handleError(w, err)
		return
	}

	log.Printf("videos from host:%s\n", host)

	vs, vids, err := getVideosFromDB()
	if err != nil {
		log.Printf("getVideosFromDB error %v", err)
		handleError(w, err)
		return
	}

	reply, err := getViewCountsFromReportService(vids)
	if err != nil {
		log.Printf("getViewCountsFromReportServic error e%v", err)
		handleError(w, err)
		return
	}

	for i := range vs {
		vs[i].ViewCount = reply.Reply[i].GetViewCount()
	}


	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(vs)
	if err != nil {
		log.Println(err)
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)

	mux := http.NewServeMux()

	mux.HandleFunc("/check_alive", handleCheckAlive)

	mux.HandleFunc("/videos", handleVideos)

	go http.ListenAndServe(":80", mux)

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case <-exit:
		fmt.Println("Received CTRL-C")
		break
	}

	fmt.Println("Server exit")
}

func handleError(w http.ResponseWriter, err error) {
	log.Printf("error: %s\n", err.Error())
	w.Header().Set("Content-Type", "application/json")
	log.Println(json.NewEncoder(w).Encode(HttpError{5000, "internal error"}))
}

func getViewCountsFromReportService(vids []int64) (*proto.GetVideoReportResponse, error) {
	conn, err := grpc.Dial("video-api-video-report:80", grpc.WithInsecure())
	if err != nil {
		log.Printf("did not connect: %v", err)
		return nil, err
	}
	defer conn.Close()
	c := proto.NewReportServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reply, err := c.GetVideosViewCount(ctx, &proto.GetVideoReportRequest{
		Id: vids,
	})
	if err != nil {
		log.Printf("could not ping: %v", err)
		return nil, err
	}
	return reply, nil
}

func getVideosFromDB() ([]*Video, []int64, error) {
	db, err := sql.Open("mysql", "video-api-user:video-api-password@tcp(video-api-mysql:3306)/video-api-db")
	if err != nil {
		log.Println(err)
		return nil, nil, err
	}
	defer db.Close()

	results, err := db.Query("SELECT id, name FROM videos")
	if err != nil {
		log.Println(err)
		return nil, nil, err
	}
	defer results.Close()

	vs := make([]*Video, 0)
	vids := make([]int64, 0)
	for results.Next() {
		var v Video
		err = results.Scan(&v.ID, &v.Name)
		if err != nil {
			log.Println(err)
			return nil, nil, err
		}
		vs = append(vs, &v)
		vids = append(vids, v.ID)
	}

	return vs, vids, nil
}
