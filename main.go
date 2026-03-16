package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.bug.st/serial"
)

var reg = prometheus.NewRegistry()

var (
	voltage = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "osp3_voltage_volts",
		Help: "Voltage in Volts",
	}, []string{"channel"})

	current = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "osp3_current_amps",
		Help: "Current in Amperes",
	}, []string{"channel"})

	power = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "osp3_power_watts",
		Help: "Power in Watts",
	}, []string{"channel"})

	status = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "osp3_status",
		Help: "Channel status (1 for ON, 0 for OFF)",
	}, []string{"channel"})
)

func init() {
	reg.MustRegister(voltage, current, power, status)
}
func main() {
	envDevice := getEnv("DEVICE", "/dev/ttyUSB0")
	envBaud := getEnv("BAUD", "115200")
	envPort := getEnv("PORT", "9120")

	devicePtr := flag.String("device", envDevice, "Serial device path")
	baudStr := flag.String("baud", envBaud, "Serial baud rate")
	portStr := flag.String("port", envPort, "HTTP port for Prometheus metrics")

	baudInt, err := strconv.Atoi(*baudStr)
	if err != nil {
		log.Fatalf("Invalid baud rate: %v", err)
	}

	portInt, err := strconv.Atoi(*portStr)
	if err != nil {
		log.Fatalf("Invalid port: %v", err)
	}
	flag.Parse()

	go func() {
		addr := fmt.Sprintf(":%d", portInt)
		http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
		fmt.Printf("OSP3 Exporter: listening on %s\n", addr)
		log.Fatal(http.ListenAndServe(addr, nil))
	}()

	for {
		if _, err := os.Stat(*devicePtr); os.IsNotExist(err) {
			fmt.Printf("Waiting for device %s...\n", *devicePtr)
			time.Sleep(5 * time.Second)
			continue
		}

		mode := &serial.Mode{BaudRate: baudInt}
		port, err := serial.Open(*devicePtr, mode)
		if err != nil {
			fmt.Printf("Device found but failed to open: %v. Retrying...\n", err)
			time.Sleep(5 * time.Second)
			continue
		}

		fmt.Printf("Successfully connected to %s\n", *devicePtr)

		scanner := bufio.NewScanner(port)
		for scanner.Scan() {
			processLine(scanner.Text())
		}

		fmt.Println("Lost connection to device. Reconnecting...")
		port.Close()
		time.Sleep(2 * time.Second)
	}
}

func processLine(line string) {
	parts := strings.Split(line, ",")
	if len(parts) >= 14 {
		updateMetric(voltage, "input", parts[1], 1000.0)
		updateMetric(current, "input", parts[2], 1000.0)
		updateMetric(power, "input", parts[3], 1000.0)
		updateMetric(status, "input", parts[4], 1.0)

		updateMetric(voltage, "0", parts[5], 1000.0)
		updateMetric(current, "0", parts[6], 1000.0)
		updateMetric(power, "0", parts[7], 1000.0)
		updateMetric(status, "0", parts[8], 1.0)

		updateMetric(voltage, "1", parts[10], 1000.0)
		updateMetric(current, "1", parts[11], 1000.0)
		updateMetric(power, "1", parts[12], 1000.0)
		updateMetric(status, "1", parts[13], 1.0)
	}
}

func updateMetric(gauge *prometheus.GaugeVec, channel string, valueStr string, divider float64) {
	cleanStr := strings.TrimSpace(valueStr)
	val, err := strconv.ParseFloat(cleanStr, 64)
	if err == nil {
		gauge.WithLabelValues(channel).Set(val / divider)
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
