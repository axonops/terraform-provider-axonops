package main

import (
	"context"
	"fmt"
	"regexp"
	"sort"

	axonopsClient "terraform-provider-axonops/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*schemasDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*schemasDataSource)(nil)
var _ datasource.DataSourceWithValidateConfig = (*schemasDataSource)(nil)

type schemasDataSource struct {
	client *axonopsClient.AxonopsHttpClient
}

func NewSchemasDataSource() datasource.DataSource {
	return &schemasDataSource{}
}

func (d *schemasDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, &resp.Diagnostics); client != nil {
		d.client = client
	}
}

func (d *schemasDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_schemas"
}

func (d *schemasDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists the schema registry subjects of a Kafka cluster. " +
			"Use `axonops_schema` to read the schema of a single subject.",
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the Kafka cluster.",
			},
			"subject_regex": schema.StringAttribute{
				Optional: true,
				Description: "Only return subjects matching this RE2 regular expression. " +
					"The match is unanchored; use `^` and `$` for a full match (for example `^orders-.*-value$`).",
			},
			"include_deleted": schema.BoolAttribute{
				Optional:    true,
				Description: "Include soft-deleted subjects. Default: false.",
			},
			"subjects": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Sorted list of matching subject names.",
			},
		},
	}
}

type schemasDataSourceData struct {
	ClusterName    types.String `tfsdk:"cluster_name"`
	SubjectRegex   types.String `tfsdk:"subject_regex"`
	IncludeDeleted types.Bool   `tfsdk:"include_deleted"`
	Subjects       types.List   `tfsdk:"subjects"`
}

func (d *schemasDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	var data schemasDataSourceData
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() || data.SubjectRegex.IsNull() || data.SubjectRegex.IsUnknown() {
		return
	}
	if _, err := regexp.Compile(data.SubjectRegex.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("subject_regex"), "Invalid Regular Expression",
			fmt.Sprintf("subject_regex is not a valid RE2 expression: %s", err))
	}
}

func (d *schemasDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data schemasDataSourceData
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var re *regexp.Regexp
	if !data.SubjectRegex.IsNull() {
		var err error
		re, err = regexp.Compile(data.SubjectRegex.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("subject_regex"), "Invalid Regular Expression", err.Error())
			return
		}
	}

	subjects, err := d.client.GetSchemaSubjects(ctx, data.ClusterName.ValueString(), data.IncludeDeleted.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list schema subjects: %s", err))
		return
	}

	matched := []string{}
	for _, s := range subjects {
		if re == nil || re.MatchString(s) {
			matched = append(matched, s)
		}
	}
	sort.Strings(matched)
	data.Subjects = stringListValue(matched)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
