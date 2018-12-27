package telefonicaopencloud

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/huaweicloud/golangsdk/openstack/networking/v2/subnets"
)

func dataSourceNetworkingSubnetV2() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceNetworkingSubnetV2Read,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"dhcp_enabled": &schema.Schema{
				Type:          schema.TypeBool,
				ConflictsWith: []string{"dhcp_disabled"},
				Optional:      true,
			},

			"dhcp_disabled": &schema.Schema{
				Type:          schema.TypeBool,
				ConflictsWith: []string{"dhcp_enabled"},
				Optional:      true,
			},

			"network_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"OS_TENANT_ID",
					"OS_PROJECT_ID",
				}, ""),
				Description: descriptions["tenant_id"],
			},

			"ip_version": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(int)
					if value != 4 && value != 6 {
						errors = append(errors, fmt.Errorf(
							"Only 4 and 6 are supported values for 'ip_version'"))
					}
					return
				},
			},

			"gateway_ip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"cidr": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"subnet_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			// Computed values
			"allocation_pools": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"start": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"end": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"enable_dhcp": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},

			"dns_nameservers": &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"host_routes": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"destination_cidr": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"next_hop": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceNetworkingSubnetV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d, config))

	listOpts := subnets.ListOpts{}

	if v, ok := d.GetOk("name"); ok {
		listOpts.Name = v.(string)
	}

	if _, ok := d.GetOk("dhcp_enabled"); ok {
		enableDHCP := true
		listOpts.EnableDHCP = &enableDHCP
	}

	if _, ok := d.GetOk("dhcp_disabled"); ok {
		enableDHCP := false
		listOpts.EnableDHCP = &enableDHCP
	}

	if v, ok := d.GetOk("network_id"); ok {
		listOpts.NetworkID = v.(string)
	}

	if v, ok := d.GetOk("tenant_id"); ok {
		listOpts.TenantID = v.(string)
	}

	if v, ok := d.GetOk("ip_version"); ok {
		listOpts.IPVersion = v.(int)
	}

	if v, ok := d.GetOk("gateway_ip"); ok {
		listOpts.GatewayIP = v.(string)
	}

	if v, ok := d.GetOk("cidr"); ok {
		listOpts.CIDR = v.(string)
	}

	if v, ok := d.GetOk("subnet_id"); ok {
		listOpts.ID = v.(string)
	}

	pages, err := subnets.List(networkingClient, listOpts).AllPages()
	if err != nil {
		return fmt.Errorf("Unable to retrieve subnets: %s", err)
	}

	allSubnets, err := subnets.ExtractSubnets(pages)
	if err != nil {
		return fmt.Errorf("Unable to extract subnets: %s", err)
	}

	if len(allSubnets) < 1 {
		return fmt.Errorf("Your query returned no results. " +
			"Please change your search criteria and try again.")
	}

	if len(allSubnets) > 1 {
		return fmt.Errorf("Your query returned more than one result." +
			" Please try a more specific search criteria")
	}

	subnet := allSubnets[0]

	log.Printf("[DEBUG] Retrieved Subnet %s: %+v", subnet.ID, subnet)
	d.SetId(subnet.ID)

	d.Set("name", subnet.Name)
	d.Set("tenant_id", subnet.TenantID)
	d.Set("network_id", subnet.NetworkID)
	d.Set("cidr", subnet.CIDR)
	d.Set("ip_version", subnet.IPVersion)
	d.Set("gateway_ip", subnet.GatewayIP)
	d.Set("enable_dhcp", subnet.EnableDHCP)
	d.Set("region", GetRegion(d, config))

	err = d.Set("dns_nameservers", subnet.DNSNameservers)
	if err != nil {
		log.Printf("[DEBUG] Unable to set dns_nameservers: %s", err)
	}

	err = d.Set("host_routes", subnet.HostRoutes)
	if err != nil {
		log.Printf("[DEBUG] Unable to set host_routes: %s", err)
	}

	// Set the allocation_pools
	var allocationPools []map[string]interface{}
	for _, v := range subnet.AllocationPools {
		pool := make(map[string]interface{})
		pool["start"] = v.Start
		pool["end"] = v.End

		allocationPools = append(allocationPools, pool)
	}
	err = d.Set("allocation_pools", allocationPools)
	if err != nil {
		log.Printf("[DEBUG] Unable to set allocation_pools: %s", err)
	}

	return nil
}
