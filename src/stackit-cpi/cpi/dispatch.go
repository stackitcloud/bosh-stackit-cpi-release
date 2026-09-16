package cpi

import (
	"github.com/stackitcloud/stackit-cpi/cpi/lib"
)

// Dispatch processes the RPC request and dispatches it to the appropriate handler
func (c *CPI) Dispatch(req lib.RPCRequest) (response *lib.RPCResponse, err error) {
	switch req.Method {
	case lib.Info:
		return c.Info(req)
	case lib.CreateStemcell:
		return c.CreateStemcell(req)
	case lib.DeleteStemcell:
		return c.DeleteStemcell(req)
	case lib.CreateVM:
		return c.CreateVM(req)
	case lib.DeleteVM:
		return c.DeleteVM(req)
	case lib.SetVMMetadata:
		return c.SetVMMetadata(req)
	case lib.HasVM:
		return c.HasVM(req)
	case lib.CalculateVMCloudProperties:
		return c.CalculateVMCloudProperties(req)
	case lib.RebootVM:
		return c.RebootVM(req)
	case lib.CreateDisk:
		return c.CreateDisk(req)
	case lib.DeleteDisk:
		return c.DeleteDisk(req)
	case lib.AttachDisk:
		return c.AttachDisk(req)
	case lib.DetachDisk:
		return c.DetachDisk(req)
	case lib.GetDisks:
		return c.GetDisks(req)
	case lib.HasDisk:
		return c.HasDisk(req)
	case lib.ResizeDisk:
		return c.ResizeDisk(req)
	case lib.UpdateDisk:
		return c.UpdateDisk(req)
	case lib.SetDiskMetadata:
		return c.SetDiskMetadata(req)
	case lib.SnapshotDisk:
		return c.SnapshotDisk(req)
	case lib.DeleteSnapshot:
		return c.DeleteSnapshot(req)
	case lib.CreateNetwork:
		return c.CreateNetwork(req)
	case lib.DeleteNetwork:
		return c.DeleteNetwork(req)
	default:
		return lib.WrapErrorInResponse(lib.MethodNotImplementedErrorMessage, req.GetLoggingContext())
	}
}
