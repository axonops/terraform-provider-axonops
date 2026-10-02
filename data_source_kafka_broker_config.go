package main

import (
	"context"
	"fmt"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*kafkaBrokerConfigDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*kafkaBrokerConfigDataSource)(nil)

type kafkaBrokerConfigDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewKafkaBrokerConfigDataSource() datasource.DataSource {
	return &kafkaBrokerConfigDataSource{}
}

func (d *kafkaBrokerConfigDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*axonopsClient.AxonopsHttpClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *axonopsClient.AxonopsHttpClient, got: %T.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *kafkaBrokerConfigDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kafka_broker_config"
}

var brokerConfigEntryAttrTypes = map[string]attr.Type{
	"name":              types.StringType,
	"value":             types.StringType,
	"source":            types.StringType,
	"is_default":        types.BoolType,
	"is_explicitly_set": types.BoolType,
	"is_read_only":      types.BoolType,
	"is_sensitive":      types.BoolType,
}

func (d *kafkaBrokerConfigDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the configuration of one Kafka broker. Read-only: the AxonOps API cannot change broker configs.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Kafka cluster.",
			},
			"broker_id": schema.Int64Attribute{
				Required:    true,
				Description: "The Kafka broker ID.",
				Validators:  []validator.Int64{int64validator.AtLeast(0)},
			},
			"config_names": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Config names to return (e.g. log.retention.hours). Omit to return all configs.",
				Validators:  []validator.List{listvalidator.SizeAtLeast(1)},
			},
			"address": schema.StringAttribute{
				Computed:    true,
				Description: "The broker address (host:port).",
			},
			"rack": schema.StringAttribute{
				Computed:    true,
				Description: "The broker rack, if set.",
			},
			"values": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Config name to value. Configs without a value (e.g. sensitive ones) are left out.",
			},
			"configs": schema.ListNestedAttribute{
				Computed:    true,
				Description: "All returned configs with their metadata.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":              schema.StringAttribute{Computed: true, Description: "The config name."},
						"value":             schema.StringAttribute{Computed: true, Description: "The config value. Null when unset or sensitive."},
						"source":            schema.StringAttribute{Computed: true, Description: "Where the value comes from (e.g. STATIC_BROKER_CONFIG, DEFAULT_CONFIG)."},
						"is_default":        schema.BoolAttribute{Computed: true, Description: "True when the value is the Kafka default."},
						"is_explicitly_set": schema.BoolAttribute{Computed: true, Description: "True when the value is set explicitly."},
						"is_read_only":      schema.BoolAttribute{Computed: true, Description: "True when the config cannot be changed dynamically."},
						"is_sensitive":      schema.BoolAttribute{Computed: true, Description: "True when the config is sensitive."},
					},
				},
			},
		},
	}
}

type kafkaBrokerConfigDataSourceData struct {
	ClusterName types.String `tfsdk:"cluster_name"`
	BrokerID    types.Int64  `tfsdk:"broker_id"`
	ConfigNames types.List   `tfsdk:"config_names"`
	Address     types.String `tfsdk:"address"`
	Rack        types.String `tfsdk:"rack"`
	Values      types.Map    `tfsdk:"values"`
	Configs     types.List   `tfsdk:"configs"`
}

func (d *kafkaBrokerConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data kafkaBrokerConfigDataSourceData
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var names []string
	if !data.ConfigNames.IsNull() {
		resp.Diagnostics.Append(data.ConfigNames.ElementsAs(ctx, &names, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	info, err := d.client.GetKafkaBroker(ctx, data.ClusterName.ValueString(), data.BrokerID.ValueInt64(), names)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Kafka broker config: %s", err))
		return
	}

	data.Address = types.StringValue(info.Address)
	data.Rack = types.StringPointerValue(info.Rack)

	values := map[string]string{}
	entries := make([]attr.Value, 0, len(info.Configs))
	for _, c := range info.Configs {
		if c.Value != nil {
			values[c.Name] = *c.Value
		}
		obj, diags := types.ObjectValue(brokerConfigEntryAttrTypes, map[string]attr.Value{
			"name":              types.StringValue(c.Name),
			"value":             types.StringPointerValue(c.Value),
			"source":            types.StringValue(c.Source),
			"is_default":        types.BoolValue(c.IsDefaultValue),
			"is_explicitly_set": types.BoolValue(c.IsExplicitlySet),
			"is_read_only":      types.BoolValue(c.IsReadOnly),
			"is_sensitive":      types.BoolValue(c.IsSensitive),
		})
		resp.Diagnostics.Append(diags...)
		entries = append(entries, obj)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	valuesMap, diags := types.MapValueFrom(ctx, types.StringType, values)
	resp.Diagnostics.Append(diags...)
	configsList, diags := types.ListValue(types.ObjectType{AttrTypes: brokerConfigEntryAttrTypes}, entries)
	resp.Diagnostics.Append(diags...)
	data.Values = valuesMap
	data.Configs = configsList
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
