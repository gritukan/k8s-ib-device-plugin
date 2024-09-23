module github.com/gritukan/k8s-ib-device-plugin

go 1.22.5

require google.golang.org/grpc v1.67.0

require github.com/gogo/protobuf v1.3.2 // indirect

require (
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/sys v0.24.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240814211410-ddb44dafa142 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	k8s.io/kubelet v0.31.1
)
