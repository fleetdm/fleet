package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"strconv"
	"terraform-provider-fleetdm/fleetdm_client"
)

// This file implements the "fleetdm_teams" Terraform resource.

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &teamsResource{}
	_ resource.ResourceWithConfigure = &teamsResource{
		client: nil,
	}
	_ resource.ResourceWithImportState = &teamsResource{}
)

// NewTeamsResource is a helper function to simplify the provider implementation.
func NewTeamsResource() resource.Resource {
	return &teamsResource{
		client: nil,
	}
}

// teamsResource is the resource implementation.
type teamsResource struct {
	client *fleetdm_client.FleetDMClient
}

// Configure adds the provider configured client to the resource.
func (r *teamsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*fleetdm_client.FleetDMClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *FleetDMClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

// Metadata returns the resource type name.
func (r *teamsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_teams"
}

// Schema defines the schema for the resource.
func (r *teamsResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = TeamResourceSchema(ctx)
}

// teamModelToTF is a handy function to take the results of the API call
// to the FleetDM API and convert it to the Terraform state.
func teamModelToTF(ctx context.Context, tm *fleetdm_client.TeamGetResponse, tf *TeamModel) error {
	tf.Id = types.Int64Value(tm.Team.ID)
	tf.Name = types.StringValue(tm.Team.Name)
	tf.Description = types.StringValue(tm.Team.Description)
	tf.Secrets = basetypes.NewListNull(NewSecretsValueNull().Type(ctx))
	// Re-marshal agent_options into a string so TF can store it sanely.
	aoBytes, err := json.Marshal(tm.Team.AgentOptions)
	if err != nil {
		return fmt.Errorf("failed to re-marshal agent options: %w", err)
	}
	tf.AgentOptions = types.StringValue(string(aoBytes))

	return nil
}

// Create creates the resource from the plan and sets the initial Terraform state.
func (r *teamsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan TeamModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	newTeam, err := r.client.CreateTeam(plan.Name.ValueString(), plan.Description.ValueString())
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic(
			"Failed to create team",
			fmt.Sprintf("Failed to create team: %s", err)))
		return
	}

	if !plan.AgentOptions.IsNull() && !plan.AgentOptions.IsUnknown() {
		aoPlan := plan.AgentOptions.ValueString()
		if aoPlan != "" {
			newTeam, err = r.client.UpdateAgentOptions(newTeam.Team.ID, aoPlan)
			if err != nil {
				resp.Diagnostics.Append(diag.NewErrorDiagnostic(
					"failed to create agent options",
					fmt.Sprintf("failed to save agent options: %s", err)))
				// This is a problem. The interface terraform presents is that
				// team creation with agent options is atomic, however under the
				// hood it's two api calls. We need to clean up from the first
				// call here, but this isn't atomic and it might fail.
				err = r.client.DeleteTeam(newTeam.Team.ID)
				if err != nil {
					resp.Diagnostics.Append(diag.NewErrorDiagnostic(
						"failed to clean up after failed team creation",
						fmt.Sprintf("failed to delete team %s while cleaning up "+
							"failure setting agent options: %s. Team will need to be "+
							"manually deleted.", plan.Name.ValueString(), err)))
				}
				return
			}
		}
	}

	err = teamModelToTF(ctx, newTeam, &plan)
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic(
			"failed to convert fleet api return to TF structs",
			fmt.Sprintf("failed to convert fleet api return to TF structs: %s", err)))
		_ = r.client.DeleteTeam(newTeam.Team.ID) // Problematic. :-/
		return
	}

	// Set our state to match that of the plan, now that we have
	// completed successfully
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read queries the FleetDM API and sets the TF state to what it finds.
func (r *teamsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TeamModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiTeam, err := r.client.GetTeam(state.Id.ValueInt64())
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic(
			"Failed to get team",
			fmt.Sprintf("Failed to get team: %s", err)))
		return
	}

	if state.Id != types.Int64Value(apiTeam.Team.ID) {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic(
			"ID mismatch",
			fmt.Sprintf("ID mismatch: %s != %s",
				state.Id, strconv.FormatInt(apiTeam.Team.ID, 10))))
		return
	}

	err = teamModelToTF(ctx, apiTeam, &state)
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic(
			"failed to convert fleet api return to TF structs",
			fmt.Sprintf("failed to convert fleet api return to TF structs: %s", err)))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update will compare the plan and the current state and see how they
// differ. It will then update the team in FleetDM to match the plan,
// and then update the Terraform state to match that plan.
func (r *teamsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state TeamModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var plan TeamModel
	diags = req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var name, description *string
	name = nil
	description = nil
	if !plan.Name.Equal(state.Name) {
		n := plan.Name.ValueString()
		name = &n
	}
	if !plan.Description.Equal(state.Description) {
		d := plan.Description.ValueString()
		description = &d
	}

	if name == nil && description == nil && plan.AgentOptions.Equal(state.AgentOptions) {
		// Nothing to do
		return
	}

	var upTeam *fleetdm_client.TeamGetResponse
	var err error

	// Deal with agent options first because it has a higher chance of failure
	if !plan.AgentOptions.Equal(state.AgentOptions) {
		ao := plan.AgentOptions.ValueString()
		if ao != "" {
			upTeam, err = r.client.UpdateAgentOptions(state.Id.ValueInt64(), ao)
			if err != nil {
				resp.Diagnostics.Append(diag.NewErrorDiagnostic(
					"Failed to update agent options",
					fmt.Sprintf("Failed to update agent options: %s", err)))
				return
			}
		}
	}

	if name != nil || description != nil {
		upTeam, err = r.client.UpdateTeam(state.Id.ValueInt64(), name, description)
		if err != nil {
			resp.Diagnostics.Append(diag.NewErrorDiagnostic(
				"Failed to update team",
				fmt.Sprintf("Failed to update team: %s", err)))
			return
		}
	}

	err = teamModelToTF(ctx, upTeam, &state)
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic(
			"failed to convert fleet api return to TF structs",
			fmt.Sprintf("failed to convert fleet api return to TF structs: %s", err)))
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// ImportState enables import of teams. We accept the name of the team
// as input, query the FleetDM API to get the ID, and then set the ID.
// Terraform will turn around and call Read() on that id.
func (r *teamsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := r.client.TeamNameToID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to convert team name to ID",
			fmt.Sprintf("Failed to convert team name to ID: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *teamsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TeamModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteTeam(state.Id.ValueInt64())
	if err != nil {
		resp.Diagnostics.Append(diag.NewErrorDiagnostic(
			"Failed to delete team",
			fmt.Sprintf("Failed to delete team: %s", err)))
		return
	}

	resp.State.RemoveResource(ctx)
}
