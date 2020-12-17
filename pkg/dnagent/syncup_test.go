package dnagent

import (
	"context"
	"testing"
	"time"

	"github.com/virtual-disk-array/vda/pkg/lib"
	"github.com/virtual-disk-array/vda/pkg/mocks/mockspdk"
	pbdn "github.com/virtual-disk-array/vda/pkg/proto/dnagentapi"
)

const (
	freeCluster      = 100
	clusterSize      = 4 * 1024 * 1024
	totalDataCluster = 200
	blockSize        = 4096
	reqId            = "427a4eb5-bd7c-418f-944d-ad675f946d72"
	dnId             = "072267fa39c74c52af2d11b53627e93a"
	pdId             = "baee605455c8418e89674af4b5e792e3"
	nvmeTrAddr       = "00:0d.0"
	vdId             = "8dc0e06935734f7e81aff80ddea2b5da"
	vdSize           = uint64(1024 * 1024 * 10)
	cntlrId          = "8e5ac8cc7b824f2eb118281c286c8713"
)

func mockNvmfGetSubsystems(params interface{}) (*mockspdk.SpdkErr, interface{}) {
	return nil, []map[string]interface{}{}
}

func mockBdevGetBdevs(params interface{}) (*mockspdk.SpdkErr, interface{}) {
	if params == nil {
		return nil, make([]map[string]interface{}, 0)
	} else {
		return &mockspdk.SpdkErr{
			Code:    -19,
			Message: "no such device",
		}, nil
	}
}

func mockBdevLvolGetLvstores(params interface{}) (*mockspdk.SpdkErr, interface{}) {
	return nil, []map[string]interface{}{
		map[string]interface{}{
			"uuid":               "xxx",
			"base_bdev":          "xxx",
			"free_clusters":      freeCluster,
			"cluster_size":       clusterSize,
			"total_data_cluster": totalDataCluster,
			"block_size":         blockSize,
			"name":               "xxx",
		},
	}
}

var (
	simpalSpdk = map[string]func(params interface{}) (*mockspdk.SpdkErr, interface{}){
		"nvmf_get_subsystems":    mockNvmfGetSubsystems,
		"bdev_get_bdevs":         mockBdevGetBdevs,
		"bdev_lvol_get_lvstores": mockBdevLvolGetLvstores,
	}
)

func TestNormalSyncup(t *testing.T) {
	sockPath := "/tmp/vdatest.sock"
	sockTimeout := 10
	lisConf := &lib.LisConf{
		TrType:  "tcp",
		TrAddr:  "127.0.0.1",
		AdrFam:  "ipv4",
		TrSvcId: "4420",
	}
	trConf := map[string]interface{}{
		"trtype": "TCP",
	}
	s, err := mockspdk.NewMockSpdkServer("unix", sockPath, t)
	if err != nil {
		return
	}
	for k, v := range simpalSpdk {
		s.AddMethod(k, v)
	}
	go s.Run()
	time.Sleep(time.Second)
	dnAgent := newDnAgentServer(sockPath, sockTimeout, lisConf, trConf)
	ctx := context.Background()
	req := &pbdn.SyncupDnRequest{
		ReqId:    reqId,
		Revision: int64(1),
		DnReq: &pbdn.DnReq{
			DnId: dnId,
			PdReqList: []*pbdn.PdReq{
				&pbdn.PdReq{
					PdId: pdId,
					PdConf: &pbdn.PdConf{
						BdevType: &pbdn.PdConf_BdevNvme{
							BdevNvme: &pbdn.BdevNvme{
								TrAddr: nvmeTrAddr,
							},
						},
					},
					VdBeReqList: []*pbdn.VdBeReq{
						&pbdn.VdBeReq{
							VdId: vdId,
							VdBeConf: &pbdn.VdBeConf{
								Size: vdSize,
								Qos: &pbdn.BdevQos{
									RwIosPerSec:    uint64(0),
									RwMbytesPerSec: uint64(0),
									RMbytesPerSec:  uint64(0),
									WMbytesPerSec:  uint64(0),
								},
								CntlrId: cntlrId,
							},
						},
					},
				},
			},
		},
	}
	reply, err := dnAgent.SyncupDn(ctx, req)
	if err != nil {
		t.Errorf("SyncupDn failed: %v", err)
	}
	replyInfo := reply.ReplyInfo
	if replyInfo.ReplyCode != lib.DnSucceedCode {
		t.Errorf("SyncupDn ReplyCode error: %v", replyInfo.ReplyCode)
	}
	if replyInfo.ReplyMsg != lib.DnSucceedMsg {
		t.Errorf("SyncupDn ReplyMsg error: %v", replyInfo.ReplyMsg)
	}
	dnRsp := reply.DnRsp
	if dnRsp.DnId != dnId {
		t.Errorf("DnId mismatch: %v", dnRsp.DnId)
	}
	if dnRsp.DnInfo.ErrInfo.IsErr {
		t.Errorf("DnRsp has err: %v", dnRsp.DnInfo.ErrInfo)
	}
	if len(dnRsp.PdRspList) != 1 {
		t.Errorf("PdRspList length is wrong: %v", len(dnRsp.PdRspList))
	}
	pdRsp := dnRsp.PdRspList[0]
	if pdRsp.PdId != pdId {
		t.Errorf("PdId mismatch: %v", pdRsp.PdId)
	}
	if pdRsp.PdInfo.ErrInfo.IsErr {
		t.Errorf("PdRsp has err: %v", pdRsp.PdInfo.ErrInfo)
	}
	if pdRsp.PdCapacity.TotalSize != totalDataCluster*clusterSize {
		t.Errorf("Pd TotalSize mismatch: %d", pdRsp.PdCapacity.TotalSize)
	}
	if pdRsp.PdCapacity.FreeSize != freeCluster*clusterSize {
		t.Errorf("Pd FreeSize mismatch: %d", pdRsp.PdCapacity.FreeSize)
	}
	if len(pdRsp.VdBeRspList) != 1 {
		t.Errorf("VdBeRspList length is wrong: %v", len(pdRsp.VdBeRspList))
	}
	vdBeRsp := pdRsp.VdBeRspList[0]
	if vdBeRsp.VdId != vdId {
		t.Errorf("VdId mismatch: %v", vdBeRsp.VdId)
	}
	if vdBeRsp.VdBeInfo.ErrInfo.IsErr {
		t.Errorf("VdBeRsp has err: %v", vdBeRsp.VdBeInfo.ErrInfo)
	}
}

func TestOldSyncup(t *testing.T) {
	sockPath := "/tmp/vdatest.sock"
	sockTimeout := 10
	lisConf := &lib.LisConf{
		TrType:  "tcp",
		TrAddr:  "127.0.0.1",
		AdrFam:  "ipv4",
		TrSvcId: "4420",
	}
	trConf := map[string]interface{}{
		"trtype": "TCP",
	}

	s, err := mockspdk.NewMockSpdkServer("unix", sockPath, t)
	if err != nil {
		return
	}
	for k, v := range simpalSpdk {
		s.AddMethod(k, v)
	}
	go s.Run()
	time.Sleep(time.Second)

	dnAgent := newDnAgentServer(sockPath, sockTimeout, lisConf, trConf)
	ctx := context.Background()
	revOld := int64(1)
	revNew := int64(2)

	req := &pbdn.SyncupDnRequest{
		ReqId:    reqId,
		Revision: revNew,
		DnReq: &pbdn.DnReq{
			DnId:      dnId,
			PdReqList: []*pbdn.PdReq{},
		},
	}
	reply, err := dnAgent.SyncupDn(ctx, req)
	if err != nil {
		t.Errorf("SyncupDn failed: %v", err)
	}
	replyInfo := reply.ReplyInfo
	if replyInfo.ReplyCode != lib.DnSucceedCode {
		t.Errorf("SyncupDn ReplyCode error: %v", replyInfo.ReplyCode)
	}
	if replyInfo.ReplyMsg != lib.DnSucceedMsg {
		t.Errorf("SyncupDn ReplyMsg error: %v", replyInfo.ReplyMsg)
	}

	req = &pbdn.SyncupDnRequest{
		ReqId:    reqId,
		Revision: revOld,
		DnReq: &pbdn.DnReq{
			DnId:      dnId,
			PdReqList: []*pbdn.PdReq{},
		},
	}
	reply, err = dnAgent.SyncupDn(ctx, req)
	if err != nil {
		t.Errorf("SyncupDn failed: %v", err)
	}
	replyInfo = reply.ReplyInfo
	if replyInfo.ReplyCode != lib.DnOldRevErrCode {
		t.Errorf("SyncupDn ReplyCode error: %v", replyInfo.ReplyCode)
	}
	if replyInfo.ReplyMsg != lib.DnOldRevErrMsg {
		t.Errorf("SyncupDn ReplyMsg error: %v", replyInfo.ReplyMsg)
	}
}
