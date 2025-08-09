package main

import (
	context "context"
	"cloud.google.com/go/compute/apiv1"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
	"time"
)

type InstanceController struct {
	Project      string
	Zone         string
	InstanceName string
}

func NewInstanceController(project, zone, instanceName string) *InstanceController {
	return &InstanceController{
		Project:      project,
		Zone:         zone,
		InstanceName: instanceName,
	}
}

func (ic *InstanceController) waitForStatus(target string, successMsg string) string {
	ctx := context.Background()
	client, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return "Failed to create client: " + err.Error()
	}
	defer client.Close()

	req := &computepb.GetInstanceRequest{
		Project:  ic.Project,
		Zone:     ic.Zone,
		Instance: ic.InstanceName,
	}
	for i := 0; i < 30; i++ {
		instance, err := client.Get(ctx, req)
		if err != nil {
			return "Error while polling instance status: " + err.Error()
		}
		if instance.GetStatus() == target {
			return successMsg
		}
		time.Sleep(10 * time.Second)
	}
	return "Timeout: Instance did not reach status " + target + " within expected time."
}

func (ic *InstanceController) Start() string {
	ctx := context.Background()
	client, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return "Failed to create client: " + err.Error()
	}
	defer client.Close()

	req := &computepb.StartInstanceRequest{
		Project:  ic.Project,
		Zone:     ic.Zone,
		Instance: ic.InstanceName,
	}
	_, err = client.Start(ctx, req)
	if err != nil {
		return "Failed to launch: " + err.Error()
	}
	return ic.waitForStatus("RUNNING", "A server launched.")
}

func (ic *InstanceController) Stop() string {
	ctx := context.Background()
	client, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return "Failed to create client: " + err.Error()
	}
	defer client.Close()

	req := &computepb.StopInstanceRequest{
		Project:  ic.Project,
		Zone:     ic.Zone,
		Instance: ic.InstanceName,
	}
	_, err = client.Stop(ctx, req)
	if err != nil {
		return "Failed to stop: " + err.Error()
	}
	return ic.waitForStatus("TERMINATED", "A server stopped.")
}

func (ic *InstanceController) GetExternalIP() string {
	ctx := context.Background()
	client, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return "Failed to create client: " + err.Error()
	}
	defer client.Close()

	req := &computepb.GetInstanceRequest{
		Project:  ic.Project,
		Zone:     ic.Zone,
		Instance: ic.InstanceName,
	}
	instance, err := client.Get(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	ifaces := instance.GetNetworkInterfaces()
	if len(ifaces) == 0 {
		return "No network interfaces"
	}
	for _, iface := range ifaces {
		accessConfigs := iface.GetAccessConfigs()
		for _, ac := range accessConfigs {
			ip := ac.GetNatIP()
			if ip != "" {
				return ip
			}
		}
	}
	return "No external IP"
}
