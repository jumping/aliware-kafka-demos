package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"services"
	"strconv"
	"time"

	"github.com/Shopify/sarama"
	log "github.com/sirupsen/logrus"
)

var cfg *configs.MqConfig
var producer sarama.SyncProducer

func init() {

	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	log.Info("init kafka producer, it may take a few seconds to init the connection")

	var err error

	cfg = &configs.MqConfig{}
	configs.LoadJsonConfig(cfg, "kafka.json")

	mqConfig := sarama.NewConfig()
	mqConfig.Net.SASL.Enable = true
	mqConfig.Net.SASL.User = cfg.Ak
	mqConfig.Net.SASL.Password = cfg.Password
	mqConfig.Net.SASL.Handshake = true

	certBytes, err := ioutil.ReadFile(configs.GetFullPath(cfg.CertFile))
	clientCertPool := x509.NewCertPool()
	ok := clientCertPool.AppendCertsFromPEM(certBytes)
	if !ok {
		panic("kafka producer failed to parse root certificate")
	}

	mqConfig.Net.TLS.Config = &tls.Config{
		//Certificates:       []tls.Certificate{},
		RootCAs:            clientCertPool,
		InsecureSkipVerify: true,
	}

	mqConfig.Net.TLS.Enable = true
	mqConfig.Producer.Return.Successes = true

	if err = mqConfig.Validate(); err != nil {
		msg := fmt.Sprintf("Kafka producer config invalidate. config: %v. err: %v", *cfg, err)
		fmt.Println(msg)
		panic(msg)
	}

	producer, err = sarama.NewSyncProducer(cfg.Servers, mqConfig)
	if err != nil {
		msg := fmt.Sprintf("Kafak producer create fail. err: %v", err)
		fmt.Println(msg)
		panic(msg)
	}

}

func produce(topic string, key string, content string) error {
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.StringEncoder(content),
	}

	_, _, err := producer.SendMessage(msg)
	if err != nil {
		msg := fmt.Sprintf("Send Error topic: %v. key: %v. content: %v", topic, key, content)
		fmt.Println(msg)
		return err
	}
	fmt.Printf("Send OK topic:%s key:%s value:%s\n", topic, key, content)

	return nil
}

func main() {
	//the key of the kafka messages
	//do not set the same the key for all messages, it may cause partition im-balance
	key := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	ticker := time.NewTicker(time.Second * 5)
	value := "This is a kafka message! --JumpingQu(jumping.qu@chiefclouds.com)"
	go func() {
		for t := range ticker.C {
			produce(cfg.Topics[0], key, fmt.Sprintf("%s %d\n", value, t))
			log.Info(t)
		}
	}()
	time.Sleep(time.Second * 200)
	ticker.Stop()
}
