package talos

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"gopkg.in/yaml.v3"
)

type nofileFoundError struct{}

type missingParamError struct{}

type talosMachineSecretsDataSource struct{}

type talosMachineSecretsDataSourceModelV1 struct {
	ID                  types.String        `tfsdk:"id"`
	TalosVersion        types.String        `tfsdk:"talos_version"`
	PathToSecrets       types.String        `tfsdl:"path_to_secrets"`
	MachineSecrets      machineSecrets      `tfsdk:"machine_secrets"`
	ClientConfiguration clientConfiguration `tfsdk:"client_configuration"`
}

var _ datasource.DataSource = &talosMachineSecretsDataSource{}

func NewTalosMachineSecretsDataSource() datasource.DataSource {
	return &talosMachineSecretsDataSource{}
}

func (d *talosMachineSecretsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_machine_secrets"
}

func (d *talosMachineSecretsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Read machine secrets for Talos cluster.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The computed ID of the Talos cluster",
				Computed:    true,
			},
			"talos_version": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The version of talos features to use in reading machine configuration",
				Validators: []validator.String{
					talosVersionValid(),
				},
			},
			"path_to_secrets": schema.StringAttribute{
				Required:    true,
				Description: "The path to the `secrets.yaml` file containing the externally generated secrets",
			},
			"machine_secrets": schema.SingleNestedAttribute{
				Description: "The secrets for the talos cluster",
				Attributes: map[string]schema.Attribute{
					"cluster": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"id": schema.StringAttribute{
								Description: "The cluster ID",
								Required:    true,
							},
							"secret": schema.StringAttribute{
								Description: "The cluster secret",
								Sensitive:   true,
								Required:    true,
							},
						},
						Description: "The cluster secrets",
						Required:    true,
					},
					"secrets": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"bootstrap_token": schema.StringAttribute{
								Description: "The bootstrap token",
								Sensitive:   true,
								Required:    true,
							},
							"secretbox_encryption_secret": schema.StringAttribute{
								Description: "The secretbox encryption secret",
								Sensitive:   true,
								Required:    true,
							},
							"aescbc_encryption_secret": schema.StringAttribute{
								Description: "The AES-CBC encryption secret",
								Sensitive:   true,
								Required:    true,
							},
						},
						Description: "kubernetes cluster secrets",
						Required:    true,
					},
					"trustdinfo": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"token": schema.StringAttribute{
								Description: "The trustd token",
								Sensitive:   true,
								Required:    true,
							},
						},
						Description: "trustd secrets",
						Required:    true,
					},
					"certs": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"etcd":           certSchema(),
							"k8s":            certSchema(),
							"k8s_aggregator": certSchema(),
							"k8s_serviceaccount": schema.SingleNestedAttribute{
								Attributes: map[string]schema.Attribute{
									"key": schema.StringAttribute{
										Description: "The service account key",
										Sensitive:   true,
										Required:    true,
									},
								},
								Description: "The service account secrets",
								Required:    true,
							},
							"os": certSchema(),
						},
						Required: true,
					},
				},
				Required: true,
			},
			"client_configuration": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"ca_certificate": schema.StringAttribute{
						Description: "The client CA certificate",
						Required:    true,
					},
					"client_certificate": schema.StringAttribute{
						Description: "The client certificate",
						Required:    true,
					},
					"client_key": schema.StringAttribute{
						Sensitive:   true,
						Required:    true,
						Description: "The client key",
					},
				},
				Description: "The read client configuration data",
				Required:    true,
			},
		},
	}
}

func (d *talosMachineSecretsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var obj types.Object

	diags := req.Config.Get(ctx, &obj)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	var state talosMachineSecretsDataSourceModelV1

	diags = obj.As(ctx, &state, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	if state.PathToSecrets.IsNull() {
		resp.Diagnostics.AddError("empty file path", "no file found in the path provided or the path is empty")
		return
	}

	secretsFile, err := os.ReadFile(state.PathToSecrets.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid file path", "no file found in the path provided or the path is invalid")
		return
	}
	err = yaml.Unmarshal(secretsFile, &state)
	if err != nil {
		//TODO: handle error more gracefully
		resp.Diagnostics.AddError("corrupted secrets file", "the secrets file you provided is either corrupted or invalid")
		return
	}

	state.ID = basetypes.NewStringValue("machine_secrets")

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

}
