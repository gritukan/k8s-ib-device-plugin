package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	serverSock   = "/var/lib/kubelet/device-plugins/ib-device-plugin.sock"
	ibDevicePath = "/dev/infiniband"

	defaultResourceName   = "ib.plugin/infiniband"
	defaultResourceAmount = 1
)

type IbDevicePlugin struct {
	devs    []*v1beta1.Device
	devices []string
	server  *grpc.Server
}

func NewIbDevicePlugin(resourceAmount int) *IbDevicePlugin {
	devs := make([]*v1beta1.Device, resourceAmount)
	for i := 0; i < resourceAmount; i++ {
		devs[i] = &v1beta1.Device{
			ID:     fmt.Sprintf("ib-plugin/infiniband-%d", i),
			Health: v1beta1.Healthy,
		}
	}

	devices := []string{}
	stat, err := os.Stat(ibDevicePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("InfiniBand device path does not exist")
		} else {
			log.Fatalf("Failed to stat InfiniBand device path: %v", err)
		}
	} else if stat.IsDir() {
		files, err := os.ReadDir(ibDevicePath)
		if err != nil {
			log.Fatalf("Failed to read InfiniBand device path: %v", err)
		}
		for _, file := range files {
			devices = append(devices, path.Join(ibDevicePath, file.Name()))
		}

		log.Println("InfiniBand devices found: ", devices)
	} else {
		log.Printf("InfiniBand device path is not a directory")
	}

	return &IbDevicePlugin{
		devs:    devs,
		devices: devices,
	}
}

func dial(unixSocketPath string, timeout time.Duration) (*grpc.ClientConn, error) {
	c, err := grpc.Dial(unixSocketPath, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithTimeout(timeout),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)

	if err != nil {
		return nil, err
	}

	return c, nil
}

func (p *IbDevicePlugin) Start(resourceName string) error {
	if _, err := os.Stat(serverSock); err == nil {
		if err := os.Remove(serverSock); err != nil {
			return fmt.Errorf("failed to remove pre-existing socket file: %v", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat socket file: %v", err)
	}

	log.Println("Starting Infiniband device plugin")

	lis, err := net.Listen("unix", serverSock)
	if err != nil {
		return err
	}
	p.server = grpc.NewServer([]grpc.ServerOption{}...)
	v1beta1.RegisterDevicePluginServer(p.server, p)

	go p.server.Serve(lis)

	// Wait for server to start by launching a blocking connection
	conn, err := dial(serverSock, 5*time.Second)
	if err != nil {
		return err
	}
	conn.Close()

	log.Println("Infiniband device plugin started")

	log.Println("Registering Infinitband device plugin with Kubelet")

	conn, err = dial(v1beta1.KubeletSocket, 5*time.Second)

	if err != nil {
		return err
	}
	defer conn.Close()

	client := v1beta1.NewRegistrationClient(conn)
	req := &v1beta1.RegisterRequest{
		Version:      v1beta1.Version,
		Endpoint:     path.Base(serverSock),
		ResourceName: resourceName,
	}

	_, err = client.Register(context.Background(), req)

	log.Println("Infiniband device plugin registered")

	if err != nil {
		return err
	}

	return nil
}

func (p *IbDevicePlugin) Stop() error {
	if p.server != nil {
		p.server.Stop()
	}
	return os.Remove(serverSock)
}

func (p *IbDevicePlugin) GetDevicePluginOptions(ctx context.Context, e *v1beta1.Empty) (*v1beta1.DevicePluginOptions, error) {
	return &v1beta1.DevicePluginOptions{
		PreStartRequired:                false,
		GetPreferredAllocationAvailable: false,
	}, nil
}

func (p *IbDevicePlugin) ListAndWatch(e *v1beta1.Empty, s v1beta1.DevicePlugin_ListAndWatchServer) error {
	s.Send(&v1beta1.ListAndWatchResponse{Devices: p.devs})

	for {
		time.Sleep(10 * time.Second)
		for _, dev := range p.devs {
			dev.Health = v1beta1.Healthy
		}
		s.Send(&v1beta1.ListAndWatchResponse{Devices: p.devs})
	}
}

func (p *IbDevicePlugin) GetPreferredAllocation(ctx context.Context, req *v1beta1.PreferredAllocationRequest) (*v1beta1.PreferredAllocationResponse, error) {
	return &v1beta1.PreferredAllocationResponse{}, nil
}

func (p *IbDevicePlugin) Allocate(ctx context.Context, reqs *v1beta1.AllocateRequest) (*v1beta1.AllocateResponse, error) {
	responses := make([]*v1beta1.ContainerAllocateResponse, len(reqs.ContainerRequests))
	for i, containerReq := range reqs.ContainerRequests {
		responses[i] = &v1beta1.ContainerAllocateResponse{}
		if len(containerReq.DevicesIDs) == 0 {
			continue
		}

		log.Println("Allocate Infiniband device for container: ", containerReq.String())

		devices := make([]*v1beta1.DeviceSpec, len(p.devices))
		for i, device := range p.devices {
			devices[i] = &v1beta1.DeviceSpec{
				HostPath:      device,
				ContainerPath: device,
				Permissions:   "rw",
			}
		}
		responses[i].Devices = devices
	}

	return &v1beta1.AllocateResponse{ContainerResponses: responses}, nil
}

func (p *IbDevicePlugin) PreStartContainer(ctx context.Context, req *v1beta1.PreStartContainerRequest) (*v1beta1.PreStartContainerResponse, error) {
	return &v1beta1.PreStartContainerResponse{}, nil
}

func main() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagResourceName := flag.String("resource-name", defaultResourceName, "Define the resource name.")
	flagResourceAmount := flag.Int("resource-amount", defaultResourceAmount, "Define the resource amount.")
	flag.Parse()

	plugin := NewIbDevicePlugin(*flagResourceAmount)
	if err := plugin.Start(*flagResourceName); err != nil {
		log.Fatalf("Failed to start InfiniBand device plugin: %v", err)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		plugin.Stop()
		os.Exit(0)
	}()

	log.Println("Infiniband device plugin started")
	select {}
}
