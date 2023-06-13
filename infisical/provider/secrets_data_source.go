// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	infisical "terraform-provider-infisical/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &SecretsDataSource{}

func NewSecretDataSource() datasource.DataSource {
	return &SecretsDataSource{}
}

// SecretDataSource defines the data source implementation.
type SecretsDataSource struct {
	client *infisical.Client
}

// ExampleDataSourceModel describes the data source data model.
type SecretDataSourceModel struct {
	FolderPath types.String                      `tfsdk:"folder_path"`
	Secrets    map[string]InfisicalSecretDetails `tfsdk:"secrets"`
}

type InfisicalSecretDetails struct {
	Value      types.String `tfsdk:"value"`
	Comment    types.String `tfsdk:"comment"`
	SecretType types.String `tfsdk:"secret_type"`
}

func (d *SecretsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secrets"
}

func (d *SecretsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get secrets from Infisical",

		Attributes: map[string]schema.Attribute{
			"folder_path": schema.StringAttribute{
				Description: "The path to the folder from where secrets should be fetched from",
				Optional:    true,
				Computed:    false,
			},
			"secrets": schema.MapNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"value": schema.StringAttribute{
							Computed:    true,
							Description: "The secret value",
						},
						"comment": schema.StringAttribute{
							Computed:    true,
							Description: "The secret comment",
						},
						"secret_type": schema.StringAttribute{
							Computed:    true,
							Description: "The secret type (shared or personal)",
						},
					},
				},
			},
		},
	}
}

func (d *SecretsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*infisical.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *SecretsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SecretDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	plainTextSecrets, _, err := d.client.GetPlainTextSecretsViaServiceToken(data.FolderPath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Something went wrong while fetching secrets",
			"If the error is not clear, please get in touch at infisical.slack.com\n\n"+
				"Infisical Client Error: "+err.Error(),
		)
	}

	if data.FolderPath.IsNull() {
		data.FolderPath = types.StringValue("/")
	}

	data.Secrets = make(map[string]InfisicalSecretDetails)

	for _, secret := range plainTextSecrets {
		data.Secrets[secret.Key] = InfisicalSecretDetails{Value: types.StringValue(secret.Value), Comment: types.StringValue(secret.Comment), SecretType: types.StringValue(secret.Type)}
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
