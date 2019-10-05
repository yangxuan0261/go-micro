package main

import (
	"encoding/json"
	"log"
	"strings"

	demo "GoMicro/test_etcdv3/test_etcdv3_06/srv/proto/demo"
	"context"

	"github.com/micro/go-micro"
	"github.com/micro/go-micro/errors"
	_ "github.com/micro/go-plugins/registry/etcdv3"
	api "github.com/micro/micro/api/proto"
)

type Say struct {
	Client demo.SayService
}

func (s *Say) Hello(ctx context.Context, req *api.Request, rsp *api.Response) error {
	log.Print("Received Say.Hello API request")

	name, ok := req.Get["name"]
	if !ok || len(name.Values) == 0 {
		return errors.BadRequest("go.micro.api2.greeter", "Name cannot be blank")
	}

	response, err := s.Client.Hello(ctx, &demo.Request{
		Name: strings.Join(name.Values, " "),
	})
	if err != nil {
		return err
	}

	rsp.StatusCode = 200
	b, _ := json.Marshal(map[string]string{
		"message": response.Msg,
		"api":     "api two",
	})
	rsp.Body = string(b)

	return nil
}

func main() {
	service := micro.NewService(
		micro.Name("go.micro.api.greeter"),
	)

	service.Init()

	service.Server().Handle(
		service.Server().NewHandler(
			&Say{
				Client: demo.NewSayService("go.micro.srv.greeter", service.Client())},
		),
	)

	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
