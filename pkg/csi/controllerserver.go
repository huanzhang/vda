package csi

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"

	pbpo "github.com/virtual-disk-array/vda/pkg/proto/portalapi"
)

type ControllerServer struct {
	csi.UnimplementedControllerServer
	vdaClient pbpo.PortalClient
}

func (cs *ControllerServer) CreateVolume(
	ctx context.Context,
	req *csi.CreateVolumeRequest) (
	*csi.CreateVolumeResponse, error) {
	name := req.GetName()
	size := req.GetCapacityRange().GetRequiredBytes()
	contentSource := req.GetVolumeContentSource()
	klog.Infof("CreateVolume, name: %v size: %v contentSource: %v", name, size, contentSource)
	request := pbpo.CreateDaRequest{
		DaName:       name,
		Description:  "csi created volume",
		Size:         uint64(size),
		PhysicalSize: uint64(size),
		CntlrCnt:     1,
		DaConf: &pbpo.DaConf{
			StripCnt:    1,
			StripSizeKb: 64,
		},
	}
	klog.Infof("CreateDa request: %v", request)
	reply, err := cs.vdaClient.CreateDa(ctx, &request)
	if err != nil {
		klog.Errorf("CreateDa failed: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	// FIXME: check reply info
	klog.Infof("CreateDa reply: %v", reply)
	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			VolumeId:      name,
			CapacityBytes: size,
			ContentSource: contentSource,
		},
	}, nil
}

func (cs *ControllerServer) DeleteVolume(
	ctx context.Context,
	req *csi.DeleteVolumeRequest) (
	*csi.DeleteVolumeResponse, error) {
	volumeId := req.GetVolumeId()
	klog.Infof("DeleteVolume, volumeId: %v", volumeId)
	request := pbpo.DeleteDaRequest{
		DaName: volumeId,
	}
	klog.Infof("DeleteDa request: %v", request)
	reply, err := cs.vdaClient.DeleteDa(ctx, &request)
	if err != nil {
		klog.Errorf("DeleteDa failed: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	// FIXME: check reply info
	klog.Infof("DeleteDa reply: %v", reply)
	return &csi.DeleteVolumeResponse{}, nil
}

func (cs *ControllerServer) ControllerGetCapabilities(
	ctx context.Context,
	req *csi.ControllerGetCapabilitiesRequest) (
	*csi.ControllerGetCapabilitiesResponse, error) {
	klog.Infof("Using default ControllerGetCapabilities")

	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: []*csi.ControllerServiceCapability{
			&csi.ControllerServiceCapability{
				Type: &csi.ControllerServiceCapability_Rpc{
					Rpc: &csi.ControllerServiceCapability_RPC{
						Type: csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
					},
				},
			},
		},
	}, nil
}

func newControllerServer(vdaClient pbpo.PortalClient) *ControllerServer {
	return &ControllerServer{
		vdaClient: vdaClient,
	}
}
