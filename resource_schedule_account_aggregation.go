package main

import (
	"context"
	"fmt"


	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func resourceScheduleAccountAggregation() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceAccountAggregationScheduleCreateUpdate,
		ReadContext:   resourceAccountAggregationScheduleRead,
		UpdateContext: resourceAccountAggregationScheduleCreateUpdate,
		DeleteContext: resourceAccountAggregationScheduleDelete,

		Schema: accountAggregationScheduleFields(),
	}
}

func resourceAccountAggregationScheduleCreateUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	accountAggregationSchedule, err := expandAccountAggregationSchedule(d)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Info(ctx, "Performing Account Aggregation Schedule", map[string]interface{}{"source_id": accountAggregationSchedule.SourceID})

	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	newAccountAggregationSchedule, err := client.ManageAccountAggregationSchedule(ctx, accountAggregationSchedule, true)
	if err != nil {
		return diag.FromErr(err)
	}

	newAccountAggregationSchedule.SourceID = accountAggregationSchedule.SourceID

	err = flattenAccountAggregationSchedule(d, newAccountAggregationSchedule)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceAccountAggregationScheduleRead(ctx, d, m)
}

func resourceAccountAggregationScheduleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Refreshing Account Aggregation Schedule for source", map[string]interface{}{"id": d.Id()})
	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	accountAggregationSchedule, err := client.GetAccountAggregationSchedule(ctx, d.Id())
	if accountAggregationSchedule.CronExpressions != nil {
		accountAggregationSchedule.SourceID = d.Id()
	}
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Account Aggregation Schedule for Source not found", map[string]interface{}{"id": d.Id()})
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	err = flattenAccountAggregationSchedule(d, accountAggregationSchedule)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceAccountAggregationScheduleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	tflog.Info(ctx, "Deleting Account Aggregation for Source", map[string]interface{}{"id": d.Id()})

	client, err := m.(*Config).IdentityNowClient(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	accountAggregationSchedule, err := client.GetAccountAggregationSchedule(ctx, d.Id())
	if err != nil {
		// non-panicking type assertion, 2nd arg is boolean indicating type match
		_, notFound := err.(*NotFoundError)
		if notFound {
			tflog.Debug(ctx, "Account Aggregation Schedule for source not found", map[string]interface{}{"id": d.Id()})
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	if accountAggregationSchedule.CronExpressions != nil {
		accountAggregationSchedule.SourceID = d.Id()
		_, err = client.ManageAccountAggregationSchedule(ctx, accountAggregationSchedule, false)
		if err != nil {
			return diag.FromErr(fmt.Errorf("Error removing Account Aggregation Schedule for source ID: %s. \nError: %s", d.Id(), err))
		}

		d.SetId("")
	}

	return nil
}
