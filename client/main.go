package main

import (
	"context"
	"os"
	"time"

	"github.com/metal-stack/masterdata-api/pkg/auth"

	v1 "github.com/metal-stack/go-ipam/api/v1"
	"github.com/metal-stack/go-ipam/pkg/client"
	"go.uber.org/zap"
)

const grpcRequestTimeout = 5 * time.Second

func main() {

	logger, _ := zap.NewProduction()
	logger.Info("Starting Client")

	hmacKey := os.Getenv("HMAC_KEY")
	if hmacKey == "" {
		hmacKey = auth.HmacDefaultKey
	}

	c, err := client.NewClient(context.TODO(), "localhost", 50051, "certs/client.pem", "certs/client-key.pem", "certs/ca.pem", hmacKey, logger)
	if err != nil {
		logger.Fatal(err.Error())
	}
	defer func() {
		err = c.Close()
		if err != nil {
			logger.Fatal(err.Error())
		}
	}()
	run(c, logger)

	logger.Info("Success")
}

func run(c client.Client, log *zap.Logger) {

	ctx, cancel := context.WithTimeout(context.Background(), grpcRequestTimeout)
	defer cancel()

	// create
	pcr := &v1.PrefixCreateRequest{
		Cidr: "192.168.0.0/16",
	}
	res, err := c.Ipam().Create(ctx, pcr)
	if err != nil {
		log.Fatal("could not create prefix", zap.Error(err))
	}
	log.Info("created prefix", zap.Stringer("prefix", res))

	// get
	_, err = c.Ipam().Get(ctx, &v1.PrefixGetRequest{Cidr: "192.168.0.0/16"})
	if err != nil {
		log.Fatal("created prefix notfound", zap.Error(err))
	}

	// child prefix
	cp, err := c.Ipam().AcquireChild(ctx, &v1.AcquireChildRequest{Cidr: "192.168.0.0/16", Length: 18})
	if err != nil {
		log.Fatal("acquire child prefix not working", zap.Error(err))
	}
	log.Info("created child prefix", zap.Stringer("prefix", cp))
}
